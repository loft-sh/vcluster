package testing

import "k8s.io/apimachinery/pkg/runtime"

type FakeEventRecorder struct {
}

func (f *FakeEventRecorder) Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
}
