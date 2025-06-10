package log

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Logger interface {
	Infof(format string, a ...interface{})
}

type logger struct {
	logr.Logger
}

func New(name string) Logger {
	l := ctrl.Log.WithName(name)
	l.WithCallDepth(2)
	return &logger{
		l,
	}
}

func NewFromExisting(log logr.Logger, name string) Logger {
	return &logger{
		log.WithName(name),
	}
}

func NewWithoutName() Logger {
	return &logger{
		log.Log,
	}
}

func (l *logger) Base() logr.Logger {
	return l.Logger
}

func (l *logger) WithName(name string) Logger {
	return &logger{
		Logger: l.Logger.WithName(name),
	}
}

func (l *logger) Infof(format string, a ...interface{}) {
	l.WithCallDepth(1).Info(fmt.Sprintf(format, a...))
}

func (l *logger) Debugf(format string, a ...interface{}) {
	l.Logger.WithCallDepth(1).V(1).Info(fmt.Sprintf(format, a...))
}

func (l *logger) Errorf(format string, a ...interface{}) {
	l.WithCallDepth(1).Error(fmt.Errorf(format, a...), "")
}
