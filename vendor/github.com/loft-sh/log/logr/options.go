package logr

import "os"

type options struct {
	componentName     string
	logEncoding       string
	logLevel          string
	development       bool
	disableStacktrace bool
	globalKlog        bool
	globalZap         bool
	logFullCallerPath bool
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
