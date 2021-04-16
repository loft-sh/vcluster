package testing

import "k8s.io/apimachinery/pkg/runtime"

type FakeEventRecorder struct {
}

func (f *FakeEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {}

func (f *FakeEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}

func (f *FakeEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
}
