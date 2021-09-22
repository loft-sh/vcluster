package loghelper

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/loft-sh/vcluster/pkg/util/log"
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
	l := ctrl.Log.WithName(name)
	withDepthLogger, ok := l.(log.WithDepth)
	if ok {
		l = withDepthLogger.WithDepth(2)
	}

	return &logger{
		l,
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
	l := ctrl.Log.WithName("")
	withDepthLogger, ok := l.(log.WithDepth)
	if ok {
		l = withDepthLogger.WithDepth(2)
	}

	(&logger{l}).Infof(format, a...)
}

func Errorf(format string, a ...interface{}) {
	l := ctrl.Log.WithName("")
	withDepthLogger, ok := l.(log.WithDepth)
	if ok {
		l = withDepthLogger.WithDepth(2)
	}

	(&logger{l}).Errorf(format, a...)
}
