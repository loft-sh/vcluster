package exec

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/vladimirvivien/gexe/vars"
)

type CommandPolicy byte

const (
	ExitOnErrPolicy CommandPolicy = 1 << iota
	ConcurrentExecPolicy
)

// CommandResult stores results of executed commands using the CommandBuilder
type CommandResult struct {
	mu       sync.RWMutex
	workChan chan *Proc
	procs    []*Proc
	errProcs []*Proc
}

// Procs return all executed processes
func (cr *CommandResult) Procs() []*Proc {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.procs
}

// ErrProcs returns errored processes
func (cr *CommandResult) ErrProcs() []*Proc {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.errProcs
}

// Errs returns all errors
func (cr *CommandResult) Errs() (errs []error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	for _, proc := range cr.errProcs {
		errs = append(errs, fmt.Errorf("%s: %s", proc.Err(), proc.Result()))
	}
	return
}

// ErrStrings returns errors as []string
func (cr *CommandResult) ErrStrings() (errStrings []string) {
	errs := cr.Errs()
	for _, err := range errs {
		errStrings = append(errStrings, err.Error())
	}
	return
}

// PipedCommandResult stores results of piped commands
type PipedCommandResult struct {
	procs    []*Proc
	errProcs []*Proc
	lastProc *Proc
	err      error
}

// Procs return all executed processes in pipe
func (cr *PipedCommandResult) Procs() []*Proc {
	return cr.procs
}

// ErrProcs returns errored piped processes
func (cr *PipedCommandResult) ErrProcs() []*Proc {
	return cr.errProcs
}

// Errs returns all errors
func (cr *PipedCommandResult) Errs() (errs []error) {
	for _, proc := range cr.errProcs {
		errs = append(errs, fmt.Errorf("%s: %s", proc.Err(), proc.Result()))
	}
	return
}

// ErrStrings returns errors as []string
func (cr *PipedCommandResult) ErrStrings() (errStrings []string) {
	errs := cr.Errs()
	for _, err := range errs {
		errStrings = append(errStrings, err.Error())
	}
	return
}

// LastProc executes last executed process
func (cr *PipedCommandResult) LastProc() *Proc {
	procLen := len(cr.procs)
	if procLen == 0 {
		return nil
	}
	return cr.procs[procLen-1]
}

// CommandBuilder is a batch command builder that
// can execute commands using different execution policies (i.e. serial, piped, concurrent)
type CommandBuilder struct {
	cmdPolicy CommandPolicy
	procs     []*Proc
	vars      *vars.Variables
	err       error
	stdout    io.Writer
	stderr    io.Writer
}

// CommandsWithContextVars creates a *CommandBuilder with the specified context and session variables.
// The resulting *CommandBuilder is used to execute command strings.
func CommandsWithContextVars(ctx context.Context, variables *vars.Variables, cmds ...string) *CommandBuilder {
	cb := new(CommandBuilder)
	cb.vars = variables
	for _, cmd := range cmds {
		cb.procs = append(cb.procs, NewProcWithContextVars(ctx, cmd, variables))
	}
	return cb
}

// CommandsWithContext creates a *CommandBuilder, with specified context, used to collect
// command strings to be executed.
func CommandsWithContext(ctx context.Context, cmds ...string) *CommandBuilder {
	return CommandsWithContextVars(ctx, &vars.Variables{}, cmds...)
}

// Commands creates a *CommandBuilder used to collect
// command strings to be executed.
func Commands(cmds ...string) *CommandBuilder {
	return CommandsWithContext(context.Background(), cmds...)
}

// CommandsWithVars creates a new CommandBuilder and sets session varialbes for it
func CommandsWithVars(variables *vars.Variables, cmds ...string) *CommandBuilder {
	return CommandsWithContextVars(context.Background(), variables, cmds...)
}

// WithPolicy sets one or more command policy mask values, i.e. (CmdOnErrContinue | CmdExecConcurrent)
func (cb *CommandBuilder) WithPolicy(policyMask CommandPolicy) *CommandBuilder {
	cb.cmdPolicy = policyMask
	return cb
}

// Add adds a new command string to the builder
func (cb *CommandBuilder) Add(cmds ...string) *CommandBuilder {
	for _, cmd := range cmds {
		cb.procs = append(cb.procs, NewProc(cb.vars.Eval(cmd)))
	}
	return cb
}

// WithStdout sets the standard output stream for the builder
func (cb *CommandBuilder) WithStdout(out io.Writer) *CommandBuilder {
	cb.stdout = out
	return cb
}

// WithStderr sets the standard output err stream for the builder
func (cb *CommandBuilder) WithStderr(err io.Writer) *CommandBuilder {
	cb.stderr = err
	return cb
}

// WithWorkDir sets the working directory for all defined commands
func (cb *CommandBuilder) WithWorkDir(dir string) *CommandBuilder {
	for _, proc := range cb.procs {
		proc.cmd.Dir = dir
	}
	return cb
}

// Run executes all commands successively and waits for all of the result. The result of each individual
// command can be accessed from CommandResult.Procs[] after the execution completes. If policy == ExitOnErrPolicy, the
// execution will stop on the first error encountered, otherwise it will continue. Processes with errors can be accessed
// from CommandResult.ErrProcs.
func (cb *CommandBuilder) Run() *CommandResult {
	var result CommandResult
	for _, p := range cb.procs {
		result.procs = append(result.procs, p)
		if err := cb.runCommand(p); err != nil {
			result.errProcs = append(result.errProcs, p)
			if hasPolicy(cb.cmdPolicy, ExitOnErrPolicy) {
				break
			}
			continue
		}
	}

	return &result
}

// Start starts all processes sequentially by default, or concurrently if ConcurrentExecPolicy is set, and does not wait for the commands
// to complete. Use CommandResult.Wait to wait for the processes to complete. Then, the result of each command can be accessed
// from CommandResult.Procs[] or CommandResult.ErrProcs to access failed processses. If policy == ExitOnErrPolicy, the execution will halt
// on the first error encountered, otherwise it will continue.
func (cb *CommandBuilder) Start() *CommandResult {
	result := &CommandResult{workChan: make(chan *Proc, len(cb.procs))}
	go func(builder *CommandBuilder, cr *CommandResult) {
		defer close(cr.workChan)

		// start with concurrently and wait for all procs to launch
		if hasPolicy(builder.cmdPolicy, ConcurrentExecPolicy) {
			var gate sync.WaitGroup
			for _, proc := range builder.procs {
				cr.mu.Lock()
				cr.procs = append(cr.procs, proc)
				cr.mu.Unlock()

				// setup standard output/err
				proc.cmd.Stdout = cb.stdout
				if cb.stdout == nil {
					proc.cmd.Stdout = proc.result
				}

				proc.cmd.Stderr = cb.stderr
				if cb.stderr == nil {
					proc.cmd.Stderr = proc.result
				}

				gate.Add(1)
				go func(conProc *Proc, conResult *CommandResult) {
					conResult.mu.Lock()
					defer conResult.mu.Unlock()
					defer gate.Done()
					if err := conProc.Start().Err(); err != nil {
						cr.errProcs = append(cr.errProcs, conProc)
						return
					}
					conResult.workChan <- conProc
				}(proc, cr)
			}
			gate.Wait()
			return
		}

		// start sequentially
		for _, proc := range builder.procs {
			cr.mu.Lock()
			cr.procs = append(cr.procs, proc)
			cr.mu.Unlock()

			// setup standard output/err
			proc.cmd.Stdout = cb.stdout
			if cb.stdout == nil {
				proc.cmd.Stdout = proc.result
			}

			proc.cmd.Stderr = cb.stderr
			if cb.stderr == nil {
				proc.cmd.Stderr = proc.result
			}

			// start sequentially
			if err := proc.Start().Err(); err != nil {
				cr.mu.Lock()
				cr.errProcs = append(cr.errProcs, proc)
				cr.mu.Unlock()
				if hasPolicy(builder.cmdPolicy, ExitOnErrPolicy) {
					break
				}
				continue
			}

			cr.workChan <- proc
		}
	}(cb, result)

	return result
}

// Concurr starts all processes concurrently and does not wait for the commands
// to complete. It is equivalent to Commands(...).WithPolicy(ConcurrentExecPolicy).Start().
func (cb *CommandBuilder) Concurr() *CommandResult {
	cb.cmdPolicy = ConcurrentExecPolicy
	return cb.Start()
}

// Pipe executes each command serially chaining the combinedOutput of previous command to the inputPipe of next command.
func (cb *CommandBuilder) Pipe() *PipedCommandResult {
	if cb.err != nil {
		return &PipedCommandResult{err: cb.err}
	}

	var result PipedCommandResult
	procLen := len(cb.procs)
	if procLen == 0 {
		return &PipedCommandResult{}
	}

	// wire last proc to combined output
	last := procLen - 1
	result.lastProc = cb.procs[last]

	// setup standard output/err for last proc in pipe
	result.lastProc.cmd.Stdout = cb.stdout
	if cb.stdout == nil {
		result.lastProc.cmd.Stdout = result.lastProc.result
	}

	result.lastProc.cmd.Stderr = cb.stderr
	if cb.stderr == nil {
		result.lastProc.cmd.Stderr = result.lastProc.result
	}

	result.lastProc.cmd.Stdout = result.lastProc.result
	for i, p := range cb.procs[:last] {
		pipeout, err := p.cmd.StdoutPipe()
		if err != nil {
			p.err = err
			return &PipedCommandResult{err: err, errProcs: []*Proc{p}}
		}

		cb.procs[i+1].cmd.Stdin = pipeout
	}

	// start each process (but, not wait for result)
	// to ensure data flow between successive processes start
	for _, p := range cb.procs {
		result.procs = append(result.procs, p)
		if err := p.Start().Err(); err != nil {
			result.errProcs = append(result.errProcs, p)
			return &result
		}
	}

	// wait and access processes result
	for _, p := range cb.procs {
		if err := p.Wait().Err(); err != nil {
			result.errProcs = append(result.errProcs, p)
			break
		}
	}

	return &result
}

func (cb *CommandBuilder) runCommand(proc *Proc) error {
	// setup standard out and standard err

	proc.cmd.Stdout = cb.stdout
	if cb.stdout == nil {
		proc.cmd.Stdout = proc.result
	}

	proc.cmd.Stderr = cb.stderr
	if cb.stderr == nil {
		proc.cmd.Stderr = proc.result
	}

	if err := proc.Start().Err(); err != nil {
		return err
	}

	if err := proc.Wait().Err(); err != nil {
		return err
	}
	return nil
}

func (cr *CommandResult) Wait() *CommandResult {
	for proc := range cr.workChan {
		if err := proc.Wait().Err(); err != nil {
			cr.errProcs = append(cr.errProcs, proc)
		}
	}
	return cr
}

func hasPolicy(mask, pol CommandPolicy) bool {
	return (mask & pol) != 0
}
