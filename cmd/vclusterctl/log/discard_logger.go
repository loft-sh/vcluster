package log

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/survey"

	"github.com/sirupsen/logrus"
)

// DiscardLogger just discards every log statement
type DiscardLogger struct {
	PanicOnExit bool
}

// Debug implements logger interface
func (d *DiscardLogger) Debug(...interface{}) {}

// Debugf implements logger interface
func (d *DiscardLogger) Debugf(string, ...interface{}) {}

// Info implements logger interface
func (d *DiscardLogger) Info(...interface{}) {}

// Infof implements logger interface
func (d *DiscardLogger) Infof(string, ...interface{}) {}

// Warn implements logger interface
func (d *DiscardLogger) Warn(...interface{}) {}

// Warnf implements logger interface
func (d *DiscardLogger) Warnf(string, ...interface{}) {}

// Error implements logger interface
func (d *DiscardLogger) Error(...interface{}) {}

// Errorf implements logger interface
func (d *DiscardLogger) Errorf(string, ...interface{}) {}

// Fatal implements logger interface
func (d *DiscardLogger) Fatal(args ...interface{}) {
	if d.PanicOnExit {
		d.Panic(args...)
	}

	os.Exit(1)
}

// Fatalf implements logger interface
func (d *DiscardLogger) Fatalf(format string, args ...interface{}) {
	if d.PanicOnExit {
		d.Panicf(format, args...)
	}

	os.Exit(1)
}

// Panic implements logger interface
func (d *DiscardLogger) Panic(args ...interface{}) {
	panic(fmt.Sprint(args...))
}

// Panicf implements logger interface
func (d *DiscardLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

// Done implements logger interface
func (d *DiscardLogger) Done(...interface{}) {}

// Donef implements logger interface
func (d *DiscardLogger) Donef(string, ...interface{}) {}

// Fail implements logger interface
func (d *DiscardLogger) Fail(...interface{}) {}

// Failf implements logger interface
func (d *DiscardLogger) Failf(string, ...interface{}) {}

// Print implements logger interface
func (d *DiscardLogger) Print(logrus.Level, ...interface{}) {}

// Printf implements logger interface
func (d *DiscardLogger) Printf(logrus.Level, string, ...interface{}) {}

// StartWait implements logger interface
func (d *DiscardLogger) StartWait(string) {}

// StopWait implements logger interface
func (d *DiscardLogger) StopWait() {}

// SetLevel implements logger interface
func (d *DiscardLogger) SetLevel(logrus.Level) {}

// GetLevel implements logger interface
func (d *DiscardLogger) GetLevel() logrus.Level { return logrus.FatalLevel }

// Write implements logger interface
func (d *DiscardLogger) Write(message []byte) (int, error) {
	return len(message), nil
}

// WriteString implements logger interface
func (d *DiscardLogger) WriteString(string) {}

// Question asks a new question
func (d *DiscardLogger) Question(*survey.QuestionOptions) (string, error) {
	return "", SurveyError{}
}

// SurveyError is used to identify errors where questions were asked in the discard logger
type SurveyError struct{}

// Error implements error interface
func (s SurveyError) Error() string {
	return "Asking questions is not possible in silenced mode"
}
