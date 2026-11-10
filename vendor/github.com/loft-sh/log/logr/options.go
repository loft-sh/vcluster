package logr

import (
	"os"
	"regexp"
)

type options struct {
	componentName               string
	logEncoding                 string
	logLevel                    string
	logFile                     string
	logFileMode                 os.FileMode
	development                 bool
	disableStacktrace           bool
	globalKlog                  bool
	globalZap                   bool
	logFullCallerPath           bool
	discardMessageMatchingRegex []*regexp.Regexp
}

type Option interface {
	apply(*options)
}

type componentNameOption string

func (c componentNameOption) apply(o *options) {
	o.componentName = string(c)
}

func WithComponentName(name string) Option {
	return componentNameOption(name)
}

type logFileOption string

func (l logFileOption) apply(o *options) {
	o.logFile = string(l)
}

// WithLogFile enables rotating file output.
// Each path uses up to ~600 MB: a 100 MB active file plus five uncompressed backups.
// Rotation and retention are size-based, never age-based.
// Writers are shared for the process lifetime (one log file path per process).
// Options apply in order, so a later WithOptionsFromEnv overrides the path with
// LOFT_LOG_FILE, including when that variable is unset.
func WithLogFile(path string) Option {
	return logFileOption(path)
}

type logFileModeOption os.FileMode

func (m logFileModeOption) apply(o *options) {
	mode := os.FileMode(m).Perm()
	if mode != 0 {
		o.logFileMode = mode
	}
}

// WithLogFileMode sets the mode for newly-created active and rotated log files.
// Modes are enforced on unix only; elsewhere the platform may ignore them (on
// Windows os.Chmod toggles just the read-only bit).
// It defaults to 0644, and mode 0 keeps that default.
// A pre-existing file keeps its own mode, including across rotations, but only where
// it is narrower than requested: wider bits are removed, so a file pre-placed at the
// configured path cannot widen what was asked for.
// Creating another logger for the same path with a different requested mode returns an error.
func WithLogFileMode(mode os.FileMode) Option {
	return logFileModeOption(mode)
}

type logLevelOption string

func (l logLevelOption) apply(o *options) {
	o.logLevel = string(l)
}

func WithLogLevel(logLevel string) Option {
	return logLevelOption(logLevel)
}

type logEncodingOption string

func (l logEncodingOption) apply(o *options) {
	o.logEncoding = string(l)
}

func WithLogEncoding(logEncoding string) Option {
	return logEncodingOption(logEncoding)
}

type logFullCallerPathOption bool

func (l logFullCallerPathOption) apply(o *options) {
	o.logFullCallerPath = bool(l)
}

func WithLogFullCallerPath(logFullCallerPath bool) Option {
	return logFullCallerPathOption(logFullCallerPath)
}

type globalKlogOption bool

func (s globalKlogOption) apply(o *options) {
	o.globalKlog = bool(s)
}

func WithGlobalKlog(global bool) Option {
	return globalKlogOption(global)
}

type globalZapOption bool

func (s globalZapOption) apply(o *options) {
	o.globalZap = bool(s)
}

func WithGlobalZap(global bool) Option {
	return globalZapOption(global)
}

type developmentOption bool

func (d developmentOption) apply(o *options) {
	o.development = bool(d)
}

func WithDevelopment(inDevelopment bool) Option {
	return developmentOption(inDevelopment)
}

type fromEnvOption struct{}

func (fromEnvOption) apply(o *options) {
	o.development = os.Getenv("DEVELOPMENT") == "true"
	o.disableStacktrace = os.Getenv("LOFT_LOG_DISABLE_STACKTRACE") == "" || os.Getenv("LOFT_LOG_DISABLE_STACKTRACE") != "false"
	o.logEncoding = GetEncoding()
	o.logFile = LogFile()
	o.logFullCallerPath = LogFullCallerPath()
	o.logLevel = LoftLogLevel()
}

func WithOptionsFromEnv() Option {
	return fromEnvOption{}
}

type disableStacktraceOption bool

func (d disableStacktraceOption) apply(o *options) {
	o.disableStacktrace = bool(d)
}

func WithDisableStacktrace(disableStacktrace bool) Option {
	return disableStacktraceOption(disableStacktrace)
}

type discardMessageMatchingRegexOption string

func (d discardMessageMatchingRegexOption) apply(o *options) {
	if len(o.discardMessageMatchingRegex) == 0 {
		o.discardMessageMatchingRegex = []*regexp.Regexp{regexp.MustCompile(string(d))}
	} else {
		o.discardMessageMatchingRegex = append(o.discardMessageMatchingRegex, regexp.MustCompile(string(d)))
	}
}

func WithDiscardMessageMatchingRegex(regex string) Option {
	return discardMessageMatchingRegexOption(regex)
}
