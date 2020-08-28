// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package testdata

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
	Handler    ResourceEventHandler
}

func (f FilteringResourceEventHandler) do(obj interface{}) {

}

type manager interface {
	AddEventHandler(handler ResourceEventHandler)
}

type handlerManager struct {
}

func (h *handlerManager) AddEventHandler(handler ResourceEventHandler) {

}

func (h *handlerManager) addHandler(obj interface{}) {

}

func (h *handlerManager) updateHandler(oldObj, newObj interface{}) {

}

func (h *handlerManager) deleteHandler(obj interface{}) {

}

func addAllEventHandlers(m manager, h handlerManager) {
	m.AddEventHandler(
		ResourceEventHandlerFuncs{
			AddFunc:    h.addHandler,
			UpdateFunc: h.updateHandler,
			DeleteFunc: h.deleteHandler,
		},
	)
	m.AddEventHandler(
		FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				return true
			},
			Handler: ResourceEventHandlerFuncs{
				AddFunc:    h.addHandler,
				UpdateFunc: h.updateHandler,
				DeleteFunc: h.deleteHandler,
			},
		},
	)
}
