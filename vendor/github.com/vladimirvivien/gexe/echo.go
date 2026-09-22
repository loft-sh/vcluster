package gexe

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/vladimirvivien/gexe/prog"
	"github.com/vladimirvivien/gexe/vars"
)

var (
	// DefaultEcho surfaces an Echo session used for all package functions
	DefaultEcho = New()
)

// Echo represents a new Echo session used for accessing
// Gexe types and methods.
type Echo struct {
	err  error
	vars *vars.Variables // session vars
	prog *prog.Info
}

// New creates a new Echo session
func New() *Echo {
	e := &Echo{
		vars: vars.New(),
		prog: prog.Prog(),
	}
	return e
}

// AddExecPath adds an executable path to PATH
func (e *Echo) AddExecPath(execPath string) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s%c%s", oldPath, os.PathListSeparator, e.Eval(execPath)))
}

// ProgAvail returns the full path of the program if found on exec PATH
func (e *Echo) ProgAvail(progName string) string {
	path, err := exec.LookPath(e.Eval(progName))
	if err != nil {
		return ""
	}
	return path
}
