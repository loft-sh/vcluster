package garbagecollect

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

type GarbageCollect interface {
	GarbageCollect(workqueue.RateLimitingInterface) error
}

type Source struct {
	Period time.Duration

	log      loghelper.Logger
	run      GarbageCollect
	stopChan <-chan struct{}
}

func NewGarbageCollectSource(collector GarbageCollect, stopChan <-chan struct{}, log loghelper.Logger) *Source {
	return &Source{
		Period: time.Second * 30,

		log:      log,
		run:      collector,
		stopChan: stopChan,
	}
}

func (gc *Source) Start(ctx context.Context, h handler.EventHandler, queue workqueue.RateLimitingInterface, p ...predicate.Predicate) error {
	if gc.run == nil {
		return fmt.Errorf("no run function defined in garbage collector")
	}

	go func() {
		wait.JitterUntil(func() {
			gc.log.Debugf("garbage collect")
			err := gc.run.GarbageCollect(queue)
			if err != nil {
				gc.log.Error(err, "garbage collect")
			}
		}, gc.Period, 1.25, true, gc.stopChan)
	}()

	return nil
}
