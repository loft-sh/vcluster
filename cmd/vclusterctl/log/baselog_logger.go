package log

import (
	"github.com/go-logr/logr"
	"github.com/loft-sh/log"
)

type baseloggerLogSink struct {
	name      string
	keyValues map[string]interface{}
	logger    log.BaseLogger
}

var _ logr.LogSink = &baseloggerLogSink{}

func (*baseloggerLogSink) Init(info logr.RuntimeInfo) {
}

func (baseloggerLogSink) Enabled(level int) bool {
	return true
}

func (l baseloggerLogSink) Info(level int, msg string, kvs ...interface{}) {
	l.logger.Infof("%s\t", msg)
	for k, v := range l.keyValues {
		l.logger.Infof("%s: %+v  ", k, v)
	}
	for i := 0; i < len(kvs); i += 2 {
		l.logger.Infof("%s: %+v  ", kvs[i], kvs[i+1])
	}
}

func (l baseloggerLogSink) Error(err error, msg string, kvs ...interface{}) {
	kvs = append(kvs, "error", err)
	l.Info(0, msg, kvs...)
}

func (l baseloggerLogSink) WithName(name string) logr.LogSink {
	return &baseloggerLogSink{
		name:      l.name + "." + name,
		keyValues: l.keyValues,
		logger:    l.logger,
	}
}

func (l baseloggerLogSink) WithValues(kvs ...interface{}) logr.LogSink {
	newMap := make(map[string]interface{}, len(l.keyValues)+len(kvs)/2)
	for k, v := range l.keyValues {
		newMap[k] = v
	}
	for i := 0; i < len(kvs); i += 2 {
		newMap[kvs[i].(string)] = kvs[i+1]
	}
	return &baseloggerLogSink{
		name:      l.name,
		keyValues: newMap,
		logger:    l.logger,
	}
}

func NewBaseLogLogger(baseLogger log.BaseLogger) logr.Logger {
	sink := &baseloggerLogSink{
		logger: baseLogger,
	}

	return logr.New(sink)
}
