package log

import (
	"github.com/go-logr/logr"

	"github.com/loft-sh/utils/pkg/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/survey"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
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
	log.Logger

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	Done(args ...interface{})
	Donef(format string, args ...interface{})

	Fail(args ...interface{})
	Failf(format string, args ...interface{})

	StartWait(message string)
	StopWait()
	Question(params *survey.QuestionOptions) (string, error)

	Write(message []byte) (int, error)
	WriteString(message string)

	// Fulfill pkg/util/loghelper interface... :/
	Base() logr.Logger
	WithName(name string) loghelper.Logger
}
