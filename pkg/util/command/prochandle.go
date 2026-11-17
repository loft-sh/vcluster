package command

// A handle to a running process. May be used to inspect the process properties
// and terminate it.
type procHandle interface {
	// Reads and returns the process's command line.
	cmdline() ([]string, error)

	// Reads and returns the process's environment.
	environ() ([]string, error)

	// Terminates the process gracefully.
	terminateGracefully() error

	// Terminates the process forcibly.
	terminateForcibly() error
}
