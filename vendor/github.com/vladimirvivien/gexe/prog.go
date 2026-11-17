package gexe

import (
	"github.com/vladimirvivien/gexe/prog"
)

// Prog makes info available about currently executing program
func (e *Echo) Prog() *prog.Info {
	return e.prog
}

// Workdir returns the current program's working directory
func (e *Echo) Workdir() string {
	return e.Prog().Workdir()
}
