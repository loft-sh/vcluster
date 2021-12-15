package loghelper

import (
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Logger interface {
	WithName(name string) Logger
	Base() logr.Logger
	Infof(format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
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
func NewWithoutName(log logr.Logger) Logger {
	return &logger{
		log.WithName(""),
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
	l.Logger.Info(fmt.Sprintf(format, a...))
}

func (l *logger) Debugf(format string, a ...interface{}) {
	l.Logger.V(1).Info(fmt.Sprintf(format, a...))
}

func (l *logger) Errorf(format string, a ...interface{}) {
	l.Logger.Error(fmt.Errorf(format, a...), "")
}

func Infof(format string, a ...interface{}) {
	l := ctrl.Log.WithName("")
	l = l.WithCallDepth(2)

	(&logger{l}).Infof(format, a...)
}

func Errorf(format string, a ...interface{}) {
	l := ctrl.Log.WithName("")
	l = l.WithCallDepth(2)

	(&logger{l}).Errorf(format, a...)
}
