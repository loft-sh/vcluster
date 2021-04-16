package terminal

import (
	"io"
	"os"

	dockerterm "github.com/docker/docker/pkg/term"
	"k8s.io/kubectl/pkg/util/term"
)

// SetupTTY creates a term.TTY (docker)
func SetupTTY(stdin io.Reader, stdout io.Writer) term.TTY {
	t := term.TTY{
		Out: stdout,
		In:  stdin,
	}

	if !t.IsTerminalIn() {
		return t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	stdin, stdout, _ = dockerterm.StdStreams()

	if stdout == os.Stdin {
		t.In = stdin
	}

	if stdout == os.Stdout {
		t.Out = stdout
	}

	return t
}
