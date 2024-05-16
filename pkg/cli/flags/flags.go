package flags

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	flag "github.com/spf13/pflag"
)

// GlobalFlags is the flags that contains the global flags
type GlobalFlags struct {
	Silent    bool
	Debug     bool
	Config    string
	Context   string
	Namespace string
	LogOutput string
}

// SetGlobalFlags applies the global flags
func SetGlobalFlags(flags *flag.FlagSet, log log.Logger) *GlobalFlags {
	globalFlags := &GlobalFlags{}

	defaultConfigPath, err := cliconfig.ConfigFilePath()
	if err != nil {
		log.Fatalf("failed to get vcluster configuration file path: %w", err)
	}

	flags.BoolVar(&globalFlags.Debug, "debug", false, "Prints the stack trace if an error occurs")
	flags.StringVar(&globalFlags.Config, "config", defaultConfigPath, "The vcluster platform config to use (will be created if it does not exist)")
	flags.StringVar(&globalFlags.Context, "context", "", "The kubernetes config context to use")
	flags.StringVarP(&globalFlags.Namespace, "namespace", "n", "", "The kubernetes namespace to use")
	flags.BoolVarP(&globalFlags.Silent, "silent", "s", false, "Run in silent mode and prevents any vcluster log output except panics & fatals")
	flags.StringVar(&globalFlags.LogOutput, "log-output", "plain", "The log format to use. Can be either plain, raw or json")

	return globalFlags
}
