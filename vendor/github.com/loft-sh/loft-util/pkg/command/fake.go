package command

import "io"

// FakeCommand is used for testing
type FakeCommand struct {
	OutputBytes []byte
}

// CombinedOutput runs the command and returns the stdout and stderr
func (f *FakeCommand) CombinedOutput() ([]byte, error) {
	return f.OutputBytes, nil
}

// Output runs the command and returns the stdout
func (f *FakeCommand) Output() ([]byte, error) {
	return f.OutputBytes, nil
}

// RunWithEnv Run implements interface
func (f *FakeCommand) RunWithEnv(stdout io.Writer, stderr io.Writer, stdin io.Reader, dir string, extraEnvVars map[string]string) error {
	return nil
}

// Run implements interface
func (f *FakeCommand) Run(workingDirectory string, stdout io.Writer, stderr io.Writer, stdin io.Reader) error {
	return nil
}
