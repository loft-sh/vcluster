package controllerhelper

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type ObjectEventHelper struct {
	Name           string
	OnObjectChange func(obj runtime.Object) error
	OnObjectDelete func(obj runtime.Object) error
}

func (o *ObjectEventHelper) OnAdd(obj interface{}) {
	if o.OnObjectChange != nil {
		if obj == nil {
			return
		}

		runtimeObj, ok := obj.(runtime.Object)
		if !ok {
			klog.Infof("%s: got a non runtime object in OnAdd: %#+v", o.Name, obj)
			return
		}

		err := o.OnObjectChange(runtimeObj)
		if err != nil {
			klog.Warningf("%s: error in OnObjectChange in OnAdd: %v", o.Name, err)
		}
	}
}

func (o *ObjectEventHelper) OnUpdate(_, newObj interface{}) {
	if o.OnObjectChange != nil {
		if newObj == nil {
			return
		}

		runtimeObj, ok := newObj.(runtime.Object)
		if !ok {
			klog.Infof("got a non runtime object in OnUpdate %s: %#+v", o.Name, newObj)
			return
		}

		err := o.OnObjectChange(runtimeObj)
		if err != nil {
			klog.Warningf("error in OnObjectChange in OnUpdate %s: %v", o.Name, err)
		}
	}
}

func (o *ObjectEventHelper) OnDelete(obj interface{}) {
	if o.OnObjectDelete != nil {
		if obj == nil {
			return
		}

		runtimeObj, ok := obj.(runtime.Object)
		if !ok {
			klog.Infof("got a non runtime object in OnDelete %s: %#+v", o.Name, obj)
			return
		}

		err := o.OnObjectDelete(runtimeObj)
		if err != nil {
			klog.Warningf("error in OnObjectDelete in OnDelete %s: %v", o.Name, err)
		}
	}
}
