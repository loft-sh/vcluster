package testutil

import (
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"k8s.io/client-go/tools/events"
)

type eventRecordingTranslator struct {
	syncertypes.GenericTranslator
	recorder events.EventRecorder
}

func WithFakeEventRecorder(translator syncertypes.GenericTranslator) (syncertypes.GenericTranslator, *events.FakeRecorder) {
	recorder := events.NewFakeRecorder(20)
	return &eventRecordingTranslator{
		GenericTranslator: translator,
		recorder:          recorder,
	}, recorder
}

func (t *eventRecordingTranslator) EventRecorder() events.EventRecorder {
	return t.recorder
}

func NextEvent(recorder *events.FakeRecorder) (string, bool) {
	select {
	case event := <-recorder.Events:
		return event, true
	default:
		return "", false
	}
}
