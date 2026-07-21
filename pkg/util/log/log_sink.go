package log

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

type WithDepth interface {
	WithDepth(depth int) logr.Logger
}

func NewLog(level int) logr.Logger {
	return logr.New(&logSink{
		level: level,
		depth: 2,
	})
}

type logSink struct {
	current  int
	level    int
	prefixes []string
	depth    int
}

var _ logr.CallDepthLogSink = &logSink{}

func (l *logSink) WithCallDepth(depth int) logr.LogSink {
	return &logSink{
		level:    l.level,
		current:  l.current,
		prefixes: l.prefixes,
		depth:    l.depth + depth,
	}
}

func (l *logSink) WithDepth(depth int) logr.Logger {
	return logr.New(&logSink{
		level:    l.level,
		current:  l.current,
		prefixes: l.prefixes,
		depth:    l.depth + depth,
	})
}

func (l *logSink) Init(logr.RuntimeInfo) {
	// l.depth = info.CallDepth
}

// Info logs a non-error message with the given key/value pairs as context.
//
// The msg argument should be used to add some constant description to
// the log line.  The key/value pairs can then be used to add additional
// variable information.  The key/value pairs should alternate string
// keys and arbitrary values.
func (l *logSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	klog.InfoDepth(l.depth, l.formatMsg(msg, keysAndValues...))
}

// Enabled tests whether this InfoLogger is enabled. For example,
// commandline flags might be used to set the logging verbosity and disable
// some info logs.
func (l *logSink) Enabled(level int) bool {
	return level >= l.level
}

// Error logs an error, with the given message and key/value pairs as context.
// It functions similarly to calling Info with the "error" named value, but may
// have unique behavior, and should be preferred for logging errors (see the
// package documentations for more information).
//
// The msg field should be used to add context to any underlying error,
// while the err field should be used to attach the actual error that
// triggered this log line, if present.
func (l *logSink) Error(err error, msg string, keysAndValues ...interface{}) {
	newKeysAndValues := []interface{}{err}
	newKeysAndValues = append(newKeysAndValues, keysAndValues...)
	klog.ErrorDepth(l.depth, l.formatMsg(msg, newKeysAndValues...))
}

// V returns an InfoLogger value for a specific verbosity level.  A higher
// verbosity level means a log message is less important.  It's illegal to
// pass a log level less than zero.
func (l *logSink) V(level int) logr.Logger {
	if level < l.level {
		return logr.New(&silent{})
	}

	prefixes := []string{}
	prefixes = append(prefixes, l.prefixes...)
	return logr.New(&logSink{
		level:    l.level,
		current:  level,
		prefixes: prefixes,
		depth:    l.depth,
	})
}

// WithValues adds some key-value pairs of context to a logger.
// See Info for documentation on how key/value pairs work.
func (l *logSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	prefixes := []string{}
	prefixes = append(prefixes, l.prefixes...)
	prefixes = append(prefixes, formatKeysAndValues(keysAndValues...))

	return &logSink{
		level:    l.level,
		current:  l.current,
		prefixes: prefixes,
		depth:    l.depth,
	}
}

// WithName adds a new element to the logger's name.
// Successive calls with WithName continue to append
// suffixes to the logger's name.  It's strongly recommended
// that name segments contain only letters, digits, and hyphens
// (see the package documentation for more information).
func (l *logSink) WithName(name string) logr.LogSink {
	if name == "" {
		return &logSink{
			level:    l.level,
			current:  l.current,
			prefixes: l.prefixes,
			depth:    l.depth,
		}
	}

	prefixes := []string{}
	prefixes = append(prefixes, l.prefixes...)
	prefixes = append(prefixes, name)

	return &logSink{
		level:    l.level,
		current:  l.current,
		prefixes: prefixes,
		depth:    l.depth,
	}
}

func (l *logSink) formatMsg(msg string, keysAndValues ...interface{}) string {
	prefixes := strings.Join(l.prefixes, ": ")
	addString := formatKeysAndValues(keysAndValues...)

	retString := msg
	if prefixes != "" {
		retString = prefixes + ": " + retString
	}
	if addString != "" {
		retString += " " + addString
	}
	// if l.current != 0 {
	//	retString = "(" + strconv.Itoa(l.current) + ") " + retString
	// }
	return retString
}

func formatKeysAndValues(keysAndValues ...interface{}) string {
	args := []string{}
	for _, kv := range keysAndValues {
		switch t := kv.(type) {
		case string:
			args = append(args, t)
		case error:
			args = append(args, t.Error())
		default:
			args = append(args, fmt.Sprintf("%#v", kv))
		}
	}

	return strings.Join(args, " ")
}

type silent struct{}

func (s *silent) Init(_ logr.RuntimeInfo)                   {}
func (s *silent) Info(_ int, _ string, _ ...interface{})    {}
func (s *silent) Enabled(_ int) bool                        { return false }
func (s *silent) Error(_ error, _ string, _ ...interface{}) {}
func (s *silent) V(_ int) logr.Logger                       { return logr.New(s) }
func (s *silent) WithValues(_ ...interface{}) logr.LogSink  { return s }
func (s *silent) WithName(_ string) logr.LogSink            { return s }
