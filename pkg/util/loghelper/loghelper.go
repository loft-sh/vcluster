package loghelper

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Logger interface {
	logr.Logger
	Infof(format string, a ...interface{})
	Debugf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

type logger struct {
	logr.Logger
}

func New(name string) Logger {
	return &logger{
		ctrl.Log.WithName(name),
	}
}

func (l *logger) Infof(format string, a ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, a...))
}

func (l *logger) Debugf(format string, a ...interface{}) {
	l.Logger.V(1).Info(fmt.Sprintf(format, a...))
}

func (l *logger) Errorf(format string, a ...interface{}) {
	l.Logger.Error(errors.New(fmt.Sprintf(format, a...)), "")
}

func Infof(format string, a ...interface{}) {
	(&logger{ctrl.Log}).Infof(format, a...)
}

func Debugf(format string, a ...interface{}) {
	(&logger{ctrl.Log}).Debugf(format, a...)
}

func Errorf(format string, a ...interface{}) {
	(&logger{ctrl.Log}).Errorf(format, a...)
}
