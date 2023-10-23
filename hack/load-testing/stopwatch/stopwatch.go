package stopwatch

import (
	"time"

	"github.com/go-logr/logr"
)

// Timer defines the timer interface.
type Timer interface {
	Reset()
	Stop(string, ...interface{})
}

// NewTimer creates a new timer.
func NewTimer(logger logr.Logger) Timer {
	return &StopWatch{
		logger:   logger,
		lastTime: time.Now(),
	}
}

type StopWatch struct {
	logger logr.Logger

	lastTime time.Time
}

func (s *StopWatch) Reset() {
	s.lastTime = time.Now()
}

func (s *StopWatch) Stop(msg string, keysAndValues ...interface{}) {
	elapsed := time.Since(s.lastTime)

	// You can add a threshold here to control when to log.
	// For example, only log if the elapsed time is longer than a certain duration.
	if elapsed > time.Millisecond {
		s.logger.Info(msg, append([]interface{}{"elapsed", elapsed}, keysAndValues...)...)
	}

	s.lastTime = time.Now()
}

func (s *StopWatch) StopWithCallback(msg string, callback func(elapsed time.Duration)) {
	elapsed := time.Since(s.lastTime)

	// You can add a threshold here to control when to execute the callback.
	// For example, only execute the callback if the elapsed time is longer than a certain duration.
	if elapsed > time.Millisecond {
		s.logger.Info(msg, "elapsed", elapsed)
		if callback != nil {
			callback(elapsed)
		}
	}

	s.lastTime = time.Now()
}
