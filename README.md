Research project to explore distributed system testing techniques in Kubernetes.


## Workflow

For now, we seek to statically extract dependencies between reflectors/informers in Kubernetes. It has only been tested with the scheduler.

### Collector
Kubetorch will first collect all callings to the `AddEventHandlers` and use
the corresponding handlers as the starting point for the tracker.
For the below example, collector will recognize `sched.addPodToSchedulingQueue`
as the handler for `ADD` of the `podInformer`.
```
podInformer.Informer().AddEventHandler(
    cache.FilteringResourceEventHandler{
        FilterFunc: ...,
        Handler: cache.ResourceEventHandlerFuncs{
            AddFunc:    sched.addPodToSchedulingQueue,
            UpdateFunc: sched.updatePodInSchedulingQueue,
            DeleteFunc: sched.deletePodFromSchedulingQueue,
        },
    },
)
```


### Tracker
Tracker will start from analyzing the handlers.

For each handler, we identify all the writing point of all the non-local variables.
Basically, tracker pushes all the non-local variables which will be modified inside
the handler to a queue.

In the below example, `sched.SchedulingQueue` will be pushed into the queue.
```
func (sched *Scheduler) addPodToSchedulingQueue(obj interface{}) {
	pod := obj.(*v1.Pod)
	klog.V(3).Infof("add event for unscheduled pod %s/%s", pod.Namespace, pod.Name)
	if err := sched.SchedulingQueue.Add(pod); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to queue %T: %v", obj, err))
	}
}
```

For each variable in the quue, then we identify the reading point.
Basically, tracker finds where the the variables are used.

In the below example, `sched.NextPod()` is reading from the `sched.SchedulingQueue` (some details are omitted here).
```
func (sched *Scheduler) scheduleOne(ctx context.Context) {
    podInfo := sched.NextPod()
    // pod could be nil when schedulerQueue is closed
    if podInfo == nil || podInfo.Pod == nil {
        return
    }
    pod := podInfo.Pod
    ...
}
```
Then we perform taint analysis from the reading point. For the above example, we start from
`podInfo` and tracks all the variables tainted by it.

Tracker keeps tracking until it hits the predefined termination. Those terminations will be
the methods which calls a RESTful api to change some resource on apiserver.
For the below example, `pod` is tainted by the previous `podInfo` and `extender.Bind` will
call RESTful POST and change pod resource. (There wil be more details about how we identify
those terminations)

```
func (sched *Scheduler) extendersBinding(pod *v1.Pod, node string) (bool, error) {
    for _, extender := range sched.Algorithm.Extenders() {
        if !extender.IsBinder() || !extender.IsInterested(pod) {
            continue
        }
        return true, extender.Bind(&v1.Binding{
            ObjectMeta: metav1.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name, UID: pod.UID},
            Target:     v1.ObjectReference{Kind: "Node", Name: node},
        })
    }
    return false, nil
}
```
After that, we have a chain starting from the handler `addPodToSchedulingQueue` ending at
the RESTful POST call `extendersBinding`. Combining the result of the collector, we know that
the `ADD` event of the `podInformer` will lead to `extendersBinding` which changes the pod resources
in the system.

## How to run
