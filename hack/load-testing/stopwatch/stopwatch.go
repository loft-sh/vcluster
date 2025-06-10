package stopwatch

import (
	"time"

	"github.com/go-logr/logr"
)

func New(logger logr.Logger) *StopWatch {
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
	s.logger.Info(msg, append([]interface{}{"elapsed", elapsed}, keysAndValues...)...)
	s.lastTime = time.Now()
}
