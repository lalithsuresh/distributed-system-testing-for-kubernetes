package handler

type ResourceEventHandler interface {
	do(obj interface{})
}

type ResourceEventHandlerFuncs struct {
	AddFunc    func(obj interface{})
	UpdateFunc func(oldObj, newObj interface{})
	DeleteFunc func(obj interface{})
}

func (r ResourceEventHandlerFuncs) do(obj interface{}) {

}

type FilteringResourceEventHandler struct {
	FilterFunc func(obj interface{}) bool
	Handler    ResourceEventHandlerFuncs
}

func (f FilteringResourceEventHandler) do(obj interface{}) {

}

func AddEventHandler(handler ResourceEventHandler) {

}
