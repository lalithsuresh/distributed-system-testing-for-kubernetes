package testdata

import "handler"

type handlerManager struct {
}

func (h *handlerManager) addHandler(obj interface{}) {

}

func (h *handlerManager) updateHandler(oldObj, newObj interface{}) {

}

func (h *handlerManager) deleteHandler(obj interface{}) {

}

func addResourceEventHandlerFuncs(h *handlerManager) {
	handler.AddEventHandler( // want "call: &{handler AddEventHandler}"
		handler.ResourceEventHandlerFuncs{
			AddFunc:    h.addHandler,    // want "handler: h.addHandler"
			UpdateFunc: h.updateHandler, // want "handler: h.updateHandler"
			DeleteFunc: h.deleteHandler, // want "handler: h.deleteHandler"
		},
	)
}

func addFilteringResourceEventHandler(h *handlerManager) {
	handler.AddEventHandler( // want "call: &{handler AddEventHandler}"
		handler.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				return true
			},
			Handler: handler.ResourceEventHandlerFuncs{
				AddFunc:    h.addHandler,    // want "handler: h.addHandler"
				UpdateFunc: h.updateHandler, // want "handler: h.updateHandler"
				DeleteFunc: h.deleteHandler, // want "handler: h.deleteHandler"
			},
		},
	)
}
