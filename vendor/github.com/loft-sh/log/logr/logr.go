package logr

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	zaphook "github.com/Sytten/logrus-zap-hook"
	"github.com/go-logr/logr"
	"github.com/loft-sh/log/logr/zapr"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
)

// NewLogger creates a new logr.Logger and sets it as configuration for other
// global logger packages.
func NewLogger(component string) (logr.Logger, error) {
	path, _ := os.Getwd()
	path = fmt.Sprintf("%s/", path)

	loftLogEncoding, err := GetEncoding()
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to get log encoding: %w", err)
	}

	logFullCallerPath := LogFullCallerPath()

	atomicLevel, kubernetesVerbosityLevel, err := GetLogLevel()
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to get log level: %w", err)
	}

	// -- Config --
	config := zap.NewProductionConfig()

	if os.Getenv("DEVELOPMENT") == "true" {
		config = zap.NewDevelopmentConfig()
	}

	// -- Set log encoding --
	config.Encoding = loftLogEncoding

	config.DisableStacktrace = os.Getenv("LOFT_LOG_DISABLE_STACKTRACE") == "" || os.Getenv("LOFT_LOG_DISABLE_STACKTRACE") != "false"

	if config.Encoding == "console" {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	}

	// -- Set log caller format --
	if logFullCallerPath {
		config.EncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(strings.TrimPrefix(caller.String(), path))
		}
	}

	// -- Set log level --
	config.Level = atomicLevel

	// -- Build config --
	zapLog, err := config.Build(zap.Fields(zap.String("component", component)))
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to build zap logger: %w", err)
	}

	// Zap global logger
	_ = zap.ReplaceGlobals(zapLog)

	// logr
	kvl, err := strconv.Atoi(kubernetesVerbosityLevel)
	if err != nil {
		kvl = 0
	}
	log := zapr.NewLoggerWithOptions(zapLog, zapr.VerbosityLevel(kvl))

	// Klog global logger
	err = SetGlobalKlog(log, kubernetesVerbosityLevel)
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to set global klog logger: %w", err)
	}

	// Logrus
	logrus.SetReportCaller(true) // So Zap reports the right caller
	logrus.SetOutput(io.Discard) // Prevent logrus from writing its logs

	hook, err := zaphook.NewZapHook(zapLog)
	if err != nil {
		return logr.Logger{}, fmt.Errorf("failed to create logrus hook: %w", err)
	}
	logrus.AddHook(hook)

	return log, nil
}

// SetGlobalKlog sets the global klog logger
func SetGlobalKlog(logger logr.Logger, kubernetesVerbosityLevel string) error {
	klog.ClearLogger()

	klogFlagSet := &flag.FlagSet{}
	klog.InitFlags(klogFlagSet)
	if err := klogFlagSet.Set("v", kubernetesVerbosityLevel); err != nil {
		return fmt.Errorf("failed to set klog verbosity level: %w", err)
	}
	if err := klogFlagSet.Parse([]string{}); err != nil {
		return fmt.Errorf("failed to parse klog flags: %w", err)
	}

	klog.SetLoggerWithOptions(logger, klog.ContextualLogger(true))

	return nil
}

// GetLogLevel returns the zap log level and the kubernetes verbosity level
func GetLogLevel() (zap.AtomicLevel, string, error) {
	logLevel := os.Getenv("LOFT_LOG_LEVEL") // debug, info, warn, error, dpanic, panic, fatal
	if logLevel == "" {
		logLevel = "info"
	}

	kubernetesVerbosityLevel := os.Getenv("KUBERNETES_VERBOSITY_LEVEL") // numerical values increasing: 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
	if kubernetesVerbosityLevel == "" {
		kubernetesVerbosityLevel = "0"
	}
	if kubernetesVerbosityLevel != "0" {
		logLevel = "debug"
	}
	if logLevel == "debug" && kubernetesVerbosityLevel == "0" {
		kubernetesVerbosityLevel = "1"
	}

	atomicLevel, err := zap.ParseAtomicLevel(logLevel)

	return atomicLevel, kubernetesVerbosityLevel, err
}

// GetEncoding returns the log encoding; "console" or "json". (default: console)
func GetEncoding() (string, error) {
	loftLogEncoding := os.Getenv("LOFT_LOG_ENCODING") // json or console
	if loftLogEncoding == "" {
		loftLogEncoding = "console"
	}
	if loftLogEncoding != "json" && loftLogEncoding != "console" {
		return "", fmt.Errorf("invalid log encoding: %s", loftLogEncoding)
	}

	return loftLogEncoding, nil
}

// LogFullCallerPath returns true if the full caller path should be logged
func LogFullCallerPath() bool {
	logFullCallerPath := os.Getenv("LOFT_LOG_FULL_CALLER_PATH") // true or false
	if logFullCallerPath == "" {
		logFullCallerPath = "false"
	}

	return logFullCallerPath == "true"
}

// FromContextOrGlobal returns a logr.Logger from the given context or the global logger
func FromContextOrGlobal(ctx context.Context) logr.Logger {
	if ctx == nil {
		return klog.Background()
	}

	if logger, err := logr.FromContext(ctx); err == nil {
		return logger
	}

	return klog.Background()
}
