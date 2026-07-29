package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vladimirvivien/gexe/vars"
)

// Proc stores process info when running a process
type Proc struct {
	id         int
	err        error
	userid     *int
	groupid    *int
	state      *os.ProcessState
	result     *bytes.Buffer
	outputPipe io.ReadCloser
	errorPipe  io.ReadCloser
	inputPipe  io.WriteCloser
	cmd        *osexec.Cmd
	process    *os.Process
	vars       *vars.Variables
}

// NewProcWithContext sets up command string to be started as an OS process using the specified context.
// However, it does not start the process. The process must be started using a subsequent call to
// Proc.StartXXX() or Proc.RunXXX() method.
func NewProcWithContext(ctx context.Context, cmdStr string) *Proc {
	words, err := parse(cmdStr)
	if err != nil {
		return &Proc{err: err}
	}

	command := osexec.CommandContext(ctx, words[0], words[1:]...)

	return &Proc{
		cmd:    command,
		result: new(bytes.Buffer),
		vars:   &vars.Variables{},
	}

}

// NewProc sets up command string to be started as an OS process, however
// does not start the process. The process must be started using a subsequent call to
// Proc.StartXXX() or Proc.RunXXX() method.
func NewProc(cmdStr string) *Proc {
	return NewProcWithContext(context.Background(), cmdStr)
}

// NewProcWithVars sets up new command string and session variables for a new proc
func NewProcWithVars(cmdStr string, variables *vars.Variables) *Proc {
	p := NewProcWithContext(context.Background(), variables.Eval(cmdStr))
	p.vars = variables
	return p
}

// NewProcWithContextVars is a convenient function to create new Proc with context and variables.
func NewProcWithContextVars(ctx context.Context, cmdStr string, variables *vars.Variables) *Proc {
	proc := NewProcWithContext(ctx, variables.Eval(cmdStr))
	proc.vars = variables
	return proc
}

// StartProcWithContext creates and starts an OS process (with combined stdout/stderr) using the specified context.
// The function does not wait for the process to complete and must be followed by proc.Wait() to wait for process completion.
// Then, call proc.Out() or proc.Result() to access the process' result.
func StartProcWithContext(ctx context.Context, cmdStr string) *Proc {
	proc := NewProcWithContext(ctx, cmdStr)
	proc.cmd.Stdout = proc.result
	proc.cmd.Stderr = proc.result

	if proc.Err() != nil {
		return proc
	}
	return proc.Start()
}

// StartProc creates and starts an OS process using StartProcWithContext using a default context.
func StartProc(cmdStr string) *Proc {
	return StartProcWithContext(context.Background(), cmdStr)
}

// StartProcWithVars sets session variables and calls StartProc to create and start a process.
func StartProcWithVars(cmdStr string, variables *vars.Variables) *Proc {
	proc := StartProcWithContext(context.Background(), variables.Eval(cmdStr))
	proc.vars = variables
	return proc
}

// StartProcWithContextVars is a convenient function that creates and starts a process with a context and variables.
func StartProcWithContextVars(ctx context.Context, cmdStr string, variables *vars.Variables) *Proc {
	proc := StartProcWithContext(ctx, variables.Eval(cmdStr))
	proc.vars = variables
	return proc
}

// RunProcWithContext creates, starts, and runs an OS process using the specified context.
// It then waits for a new process (with combined stdout/stderr) to complete.
// Use Proc.Out() to access the command's output as an io.Reader, or use Proc.Result()
// to access the commands output as a string.
func RunProcWithContext(ctx context.Context, cmdStr string) *Proc {
	proc := StartProcWithContext(ctx, cmdStr)
	if err := proc.Err(); err != nil {
		return proc
	}
	if err := proc.Wait().Err(); err != nil {
		proc.err = err
		return proc
	}
	return proc
}

// RunProc creates, starts, and runs for a new process using RunProcWithContext with a default context.
func RunProc(cmdStr string) *Proc {
	return RunProcWithContext(context.Background(), cmdStr)
}

// RunProcWithVars sets session variables and calls RunProc
func RunProcWithVars(cmdStr string, variables *vars.Variables) *Proc {
	proc := RunProcWithContext(context.Background(), variables.Eval(cmdStr))
	proc.vars = variables
	return proc
}

// RunProcWithContextVars runs a process with a context and session variables
func RunProcWithContextVars(ctx context.Context, cmdStr string, variables *vars.Variables) *Proc {
	proc := RunProcWithContext(ctx, variables.Eval(cmdStr))
	proc.vars = variables
	return proc
}

// RunWithContext creates and runs a new process using the specified context.
// It waits for its result (combined stdin,stderr) and makes it availble as a string value.
// This is equivalent to calling Proc.RunProcWithContext() followed by Proc.Result().
func RunWithContext(ctx context.Context, cmdStr string) string {
	return RunProcWithContext(ctx, cmdStr).Result()
}

// Runs is a convenient shortcut to calling RunWithContext with a default context
func Run(cmdStr string) (result string) {
	return RunProcWithContext(context.Background(), cmdStr).Result()
}

// RunWithVars creates and runs a new process with a specified session variables.
func RunWithVars(cmdStr string, variables *vars.Variables) string {
	return RunProcWithVars(cmdStr, variables).Result()
}

// RunWithContextVars creates and runs a new process with a specified context and session variables.
func RunWithContextVars(ctx context.Context, cmdStr string, variables *vars.Variables) string {
	return RunProcWithContextVars(ctx, cmdStr, variables).Result()
}

// Start starts the associated command as an OS process and does not wait for its result.
// This call should follow a process creation using NewProc.
// If you don't want to use the internal combined output streams, make sure to configure access
// to the process' input/output (stdin,stdout,stderr) prior to calling Proc.Start().
func (p *Proc) Start() *Proc {
	if p.err != nil {
		return p
	}

	if p.hasStarted() {
		return p
	}

	if p.cmd == nil {
		p.err = fmt.Errorf("cmd is nill")
		return p
	}

	// wire an output if none was provided
	if p.cmd.Stdout == nil {
		p.cmd.Stdout = p.result
	}
	if p.cmd.Stderr == nil {
		p.cmd.Stderr = p.result
	}

	// apply user id and user grp
	var procCred *syscall.Credential
	if p.userid != nil {
		procCred = &syscall.Credential{
			Uid: uint32(*p.userid),
		}
	}
	if p.groupid != nil {
		if procCred == nil {
			procCred = new(syscall.Credential)
		}
		procCred.Uid = uint32(*p.groupid)
	}
	if procCred != nil {
		if p.cmd.SysProcAttr == nil {
			p.cmd.SysProcAttr = new(syscall.SysProcAttr)
		}
		p.cmd.SysProcAttr.Credential = procCred
	}

	if err := p.cmd.Start(); err != nil {
		p.err = err
		return p
	}

	p.process = p.cmd.Process
	p.id = p.cmd.Process.Pid
	p.state = p.cmd.ProcessState

	return p
}

// SetVars sets session variables for Proc
func (p *Proc) SetVars(variables *vars.Variables) *Proc {
	p.vars = variables
	return p
}

// Command returns the os/exec.Cmd that started the process
func (p *Proc) Command() *osexec.Cmd {
	return p.cmd
}

// SetUserid looks up the user by a numerical id or
// by a name to be used for the process when launched.
func (p *Proc) SetUserid(user string) *Proc {
	if p.err != nil {
		return p
	}
	uid, err := lookupUserID(user)
	if err != nil {
		p.err = err
		return p
	}

	p.userid = &uid

	return p
}

// SetGroupid looks up the group by a numerical id or
// by a name to be used for the process when launched.
func (p *Proc) SetGroupid(grp string) *Proc {
	if p.err != nil {
		return p
	}

	gid, err := lookupGroupID(grp)
	if err != nil {
		p.err = err
		return p
	}

	p.groupid = &gid

	return p
}

// SetWorkDir sets the working directory for the command
func (p *Proc) SetWorkDir(path string) *Proc {
	p.cmd.Dir = path
	return p
}

// Peek attempts to read process state information
func (p *Proc) Peek() *Proc {
	p.state = p.cmd.ProcessState
	return p
}

// Wait waits for a previously started process to complete.
// Wait should follow Proc.StartXXX() methods to ensure completion.
func (p *Proc) Wait() *Proc {
	if p.err != nil {
		return p
	}

	if !p.hasStarted() {
		p.err = fmt.Errorf("process not started")
		return p
	}

	if p.cmd == nil {
		p.err = fmt.Errorf("command is nill")
		return p
	}
	if err := p.cmd.Wait(); err != nil {
		p.err = err
		// use return below to get proc info
	}
	return p.Peek()
}

// Run starts and waits for a process to complete.
func (p *Proc) Run() *Proc {
	if p.err != nil {
		return p
	}

	if startErr := p.Start().Err(); startErr != nil {
		p.err = startErr
		return p
	}
	return p.Wait()
}

// ID returns process id
func (p *Proc) ID() int {
	return p.id
}

// Exited returns true if process exits ok
func (p *Proc) Exited() bool {
	if p.state == nil {
		return false
	}
	return p.state.Exited()
}

// ExitCode returns process exit code
func (p *Proc) ExitCode() int {
	if p.state == nil {
		return -1
	}
	return p.state.ExitCode()
}

// IsSuccess returns true if proc exit ok
func (p *Proc) IsSuccess() bool {
	if p.state == nil {
		return false
	}
	return p.state.Success()
}

// SysTime returns proc system cpu time
func (p *Proc) SysTime() time.Duration {
	if p.state == nil {
		return -1
	}
	return p.state.SystemTime()
}

// UserTime returns proc user cpu time
func (p *Proc) UserTime() time.Duration {
	if p.state == nil {
		return -1
	}
	return p.state.UserTime()
}

// Err returns any execution error
func (p *Proc) Err() error {
	return p.err
}

// Kill halts the process
func (p *Proc) Kill() *Proc {
	if p.err != nil {
		return p
	}

	if err := p.cmd.Process.Kill(); err != nil {
		p.err = err
	}
	return p
}

// Out returns the combined result (Stdout/Stderr) as a single reader if StartProc, RunProc, or Run
// package function was used to initiate the process. If Stdout/Stderr was set independently
// (i.e. with proc.Setstdout(...)) proc.Out will be nil.
//
// NB: Out used to start/wait the process if necessary. However, that behavior has been deprecated.
// You must ensure the process has been properly initiated and wait for completion prior to calling Out.
func (p *Proc) Out() io.Reader {
	return p.result
}

// Result returns the combined stdout and stderr (see Proc.Out()) result as a string value.
// If there was a previous error in the call chain, this will return the error as a string.
func (p *Proc) Result() string {
	if p.result == nil {
		return "result <nil>"
	}
	result := strings.TrimSpace(p.result.String())
	if err := p.Err(); err != nil && result == "" {
		return err.Error()
	}
	return result
}

// Stdin returns the standard input stream for the process
func (p *Proc) Stdin() io.Reader {
	return p.cmd.Stdin
}

// SetStdin sets a stream for the process to read its input from
func (p *Proc) SetStdin(in io.Reader) {
	p.cmd.Stdin = in
}

// GetInputPipe returns a stream where the process input can be written to
// Deprecated: conflicts with the way the underlying exe.Command works
func (p *Proc) GetInputPipe() io.Writer {
	return p.inputPipe
}

// Stdout returns the standard output stream for the process
func (p *Proc) Stdout() io.Writer {
	return p.cmd.Stdout
}

// SetStdout sets a stream where the process can write its output to
func (p *Proc) SetStdout(out io.Writer) {
	p.cmd.Stdout = out
}

// GetOutputPipe returns a stream where the process output can be read from
// Deprecated: conflicts with the way the underlying exe.Command works
func (p *Proc) GetOutputPipe() io.Reader {
	return p.outputPipe
}

// Stderr returns the standard error stream for the process
func (p *Proc) Stderr() io.Writer {
	return p.cmd.Stderr
}

// SetStderr sets a stream where the process can write its errors to
func (p *Proc) SetStderr(out io.Writer) {
	p.cmd.Stderr = out
}

// GetErrorPipe returns a stream where the process error can be read from
// Deprecated: conflicts with the way the underlying exe.Command works
func (p *Proc) GetErrorPipe() io.Reader {
	return p.errorPipe
}

func (p *Proc) hasStarted() bool {
	return (p.cmd.Process != nil && p.cmd.Process.Pid != 0)
}

// Parse parses the command string and returns its tokens
func Parse(cmd string) ([]string, error) {
	return parse(cmd)
}

func lookupUserID(userid string) (int, error) {
	var uid int
	var usr *user.User
	var err error

	// assume userid is a valid numerical user id
	usr, err = user.LookupId(userid)
	if err == nil {
		uid, _ = strconv.Atoi(usr.Uid)
		return uid, nil
	}

	// if not numercal id, lookup by username
	usr, err = user.Lookup(userid)
	if err == nil {
		uid, _ = strconv.Atoi(usr.Uid)
		return uid, nil
	}

	return 0, err
}

func lookupGroupID(grpid string) (int, error) {
	var gid int
	var grp *user.Group
	var err error

	// assume grpid is a valid numerical group id
	grp, err = user.LookupGroupId(grpid)
	if err == nil {
		gid, _ = strconv.Atoi(grp.Gid)
		return gid, nil
	}

	// if not numercal id, lookup by groupname
	grp, err = user.LookupGroup(grpid)
	if err == nil {
		gid, _ = strconv.Atoi(grp.Gid)
		return gid, nil
	}

	return 0, err
}
