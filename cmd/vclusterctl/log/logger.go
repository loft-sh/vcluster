package log

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/survey"
	"github.com/sirupsen/logrus"
)

// Level type
type logFunctionType uint32

const (
	panicFn logFunctionType = iota
	fatalFn
	errorFn
	warnFn
	infoFn
	debugFn
	failFn
	doneFn
)

// Logger defines the common logging interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	Done(args ...interface{})
	Donef(format string, args ...interface{})

	Fail(args ...interface{})
	Failf(format string, args ...interface{})

	StartWait(message string)
	StopWait()

	Print(level logrus.Level, args ...interface{})
	Printf(level logrus.Level, format string, args ...interface{})

	Write(message []byte) (int, error)
	WriteString(message string)

	Question(params *survey.QuestionOptions) (string, error)

	SetLevel(level logrus.Level)
	GetLevel() logrus.Level
}
