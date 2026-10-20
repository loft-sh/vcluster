package v2

import (
	"fmt"
	"io"

	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
)

func newPluginLogger(logger *zap.Logger) hclog.Logger {
	interceptLogger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: io.Discard,
		Level:  hclog.Info,
	})
	interceptLogger.RegisterSink(&zapHclogSink{logger: logger})
	return interceptLogger
}

type zapHclogSink struct {
	logger *zap.Logger
}

func (z *zapHclogSink) Accept(name string, level hclog.Level, message string, args ...interface{}) {
	if level == hclog.Debug || level == hclog.Trace {
		return
	}

	logger := z.logger
	if name != "" {
		logger = logger.Named(name)
	}

	fields := make([]zap.Field, 0, (len(args)+1)/2)
	for len(args) >= 2 {
		fields = append(fields, zap.Any(fmt.Sprint(args[0]), args[1]))
		args = args[2:]
	}
	if len(args) == 1 {
		fields = append(fields, zap.Any("extra", args[0]))
	}

	logger = logger.With(fields...)
	switch level {
	case hclog.Error:
		logger.Error(message)
	case hclog.Warn:
		logger.Warn(message)
	case hclog.Info, hclog.NoLevel:
		logger.Info(message)
	case hclog.Debug, hclog.Trace:
		logger.Debug(message)
	}
}
