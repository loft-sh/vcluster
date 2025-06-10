package testing

import "k8s.io/apimachinery/pkg/runtime"

type FakeEventRecorder struct {
}

func (f *FakeEventRecorder) Event(runtime.Object, string, string, string) {}

func (f *FakeEventRecorder) Eventf(runtime.Object, string, string, string, ...interface{}) {
}

func (f *FakeEventRecorder) AnnotatedEventf(runtime.Object, map[string]string, string, string, string, ...interface{}) {
}
