package flags

import (
	flag "github.com/spf13/pflag"
)

// GlobalFlags is the flags that contains the global flags
type GlobalFlags struct {
	Silent    bool
	Debug     bool
	Config    string
	Context   string
	Namespace string
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.StringVar(&globalFlags.Context, "context", "", "The kubernetes config context to use")
	flags.StringVarP(&globalFlags.Namespace, "namespace", "n", "", "The kubernetes namespace to use")
	flags.BoolVarP(&globalFlags.Silent, "silent", "s", false, "Run in silent mode and prevents any vcluster log output except panics & fatals")

	return globalFlags
}
