package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log/survey"

	"github.com/sirupsen/logrus"
)

// StreamLogger logs all messages to a stream
type StreamLogger struct {
	logMutex sync.Mutex
	level    logrus.Level

	stream io.Writer
}

// NewStreamLogger creates a new stream logger
func NewStreamLogger(stream io.Writer, level logrus.Level) *StreamLogger {
	return &StreamLogger{
		level: level,

		stream: stream,
	}
}

type fnStringTypeInformation struct {
	tag      string
	logLevel logrus.Level
}

var fnStringTypeInformationMap = map[logFunctionType]*fnStringTypeInformation{
	debugFn: {
		tag:      "Debug: ",
		logLevel: logrus.DebugLevel,
	},
	infoFn: {
		tag:      "Info: ",
		logLevel: logrus.InfoLevel,
	},
	warnFn: {
		tag:      "Warn: ",
		logLevel: logrus.WarnLevel,
	},
	errorFn: {
		tag:      "Error: ",
		logLevel: logrus.ErrorLevel,
	},
	fatalFn: {
		tag:      "Fatal: ",
		logLevel: logrus.FatalLevel,
	},
	panicFn: {
		tag:      "Panic: ",
		logLevel: logrus.PanicLevel,
	},
	doneFn: {
		tag:      "Done: ",
		logLevel: logrus.InfoLevel,
	},
	failFn: {
		tag:      "Fail: ",
		logLevel: logrus.ErrorLevel,
	},
}

func (s *StreamLogger) writeMessage(fnType logFunctionType, message string) {
	fnInformation := fnStringTypeInformationMap[fnType]

	if s.level >= fnInformation.logLevel {
		_, err := s.stream.Write([]byte(fnInformation.tag))
		if err != nil {
			panic(err)
		}

		_, err = s.stream.Write([]byte(message))
		if err != nil {
			panic(err)
		}
	}
}

// StartWait prints a wait message until StopWait is called
func (s *StreamLogger) StartWait(message string) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	if s.level >= logrus.InfoLevel {
		_, err := s.stream.Write([]byte("Wait: "))
		if err != nil {
			panic(err)
		}

		_, err = s.stream.Write([]byte(message + "\n"))
		if err != nil {
			panic(err)
		}
	}
}

// StopWait prints a wait message until StopWait is called
func (s *StreamLogger) StopWait() {

}

// Debug implements interface
func (s *StreamLogger) Debug(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(debugFn, fmt.Sprintln(args...))
}

// Debugf implements interface
func (s *StreamLogger) Debugf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(debugFn, fmt.Sprintf(format, args...)+"\n")
}

// Info implements interface
func (s *StreamLogger) Info(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(infoFn, fmt.Sprintln(args...))
}

// Infof implements interface
func (s *StreamLogger) Infof(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(infoFn, fmt.Sprintf(format, args...)+"\n")
}

// Warn implements interface
func (s *StreamLogger) Warn(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(warnFn, fmt.Sprintln(args...))
}

// Warnf implements interface
func (s *StreamLogger) Warnf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(warnFn, fmt.Sprintf(format, args...)+"\n")
}

// Error implements interface
func (s *StreamLogger) Error(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(errorFn, fmt.Sprintln(args...))
}

// Errorf implements interface
func (s *StreamLogger) Errorf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(errorFn, fmt.Sprintf(format, args...)+"\n")
}

// Fatal implements interface
func (s *StreamLogger) Fatal(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fatalFn, fmt.Sprintln(args...))
	os.Exit(1)
}

// Fatalf implements interface
func (s *StreamLogger) Fatalf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fatalFn, fmt.Sprintf(format, args...)+"\n")
	os.Exit(1)
}

// Panic implements interface
func (s *StreamLogger) Panic(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(panicFn, fmt.Sprintln(args...))
	panic(fmt.Sprintln(args...))
}

// Panicf implements interface
func (s *StreamLogger) Panicf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(panicFn, fmt.Sprintf(format, args...)+"\n")
	panic(fmt.Sprintf(format, args...))
}

// Done implements interface
func (s *StreamLogger) Done(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(doneFn, fmt.Sprintln(args...))
}

// Donef implements interface
func (s *StreamLogger) Donef(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(doneFn, fmt.Sprintf(format, args...)+"\n")
}

// Fail implements interface
func (s *StreamLogger) Fail(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(failFn, fmt.Sprintln(args...))
}

// Failf implements interface
func (s *StreamLogger) Failf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(failFn, fmt.Sprintf(format, args...)+"\n")
}

// Print implements interface
func (s *StreamLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Info(args...)
	case logrus.DebugLevel:
		s.Debug(args...)
	case logrus.WarnLevel:
		s.Warn(args...)
	case logrus.ErrorLevel:
		s.Error(args...)
	case logrus.PanicLevel:
		s.Panic(args...)
	case logrus.FatalLevel:
		s.Fatal(args...)
	}
}

// Printf implements interface
func (s *StreamLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Infof(format, args...)
	case logrus.DebugLevel:
		s.Debugf(format, args...)
	case logrus.WarnLevel:
		s.Warnf(format, args...)
	case logrus.ErrorLevel:
		s.Errorf(format, args...)
	case logrus.PanicLevel:
		s.Panicf(format, args...)
	case logrus.FatalLevel:
		s.Fatalf(format, args...)
	}
}

// SetLevel implements interface
func (s *StreamLogger) SetLevel(level logrus.Level) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.level = level
}

// GetLevel implements interface
func (s *StreamLogger) GetLevel() logrus.Level {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	return s.level
}

func (s *StreamLogger) Write(message []byte) (int, error) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	return s.stream.Write(message)
}

// WriteString implements interface
func (s *StreamLogger) WriteString(message string) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	_, err := s.stream.Write([]byte(message))
	if err != nil {
		panic(err)
	}
}

// Question asks a new question
func (s *StreamLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", errors.New("questions in discard logger not supported")
}
