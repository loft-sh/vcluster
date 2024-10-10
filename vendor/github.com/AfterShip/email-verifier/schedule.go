package emailverifier

import (
	"time"
)

// schedule represents a job schedule
type schedule struct {
	stopCh    chan struct{} // stop channel to control job
	jobFunc   interface{}   //  schedule job handle function
	jobParams []interface{} // params of function
	ticker    *time.Ticker  // ticker sends the time with a period specified by a duration
	running   bool          // running indicates the current running state of schedule.
}

// newSchedule returns a new schedule instance
func newSchedule(period time.Duration, jobFunc interface{}, params ...interface{}) *schedule {
	return &schedule{
		stopCh:    make(chan struct{}),
		jobFunc:   jobFunc,
		jobParams: params,
		ticker:    time.NewTicker(period),
	}
}

// start triggers the schedule job
func (s *schedule) start() {
	if s.running {
		return
	}

	s.running = true
	go func() {
		for {
			select {
			case <-s.ticker.C:
				callJobFuncWithParams(s.jobFunc, s.jobParams)
			case <-s.stopCh:
				s.ticker.Stop()
				return
			}
		}
	}()
}

// stop will stop previously started schedule job
func (s *schedule) stop() {
	if !s.running {
		return
	}
	s.running = false
	s.stopCh <- struct{}{}
}
