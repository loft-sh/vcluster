package podhelper

import (
	"bytes"
	"io"
	"net/http"

	dockerterm "github.com/moby/term"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/kubectl/pkg/util/term"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	clientspdy "k8s.io/client-go/transport/spdy"
	kubectlExec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/scheme"
)

// SubResource specifies with sub resources should be used for the container connection (exec or attach)
type SubResource string

const (
	// SubResourceExec creates a new process in the container and attaches to that
	SubResourceExec SubResource = "exec"

	// SubResourceAttach attaches to the top process of the container
	SubResourceAttach SubResource = "attach"
)

// ExecStreamWithTransportOptions are the options used for executing a stream
type ExecStreamWithTransportOptions struct {
	ExecStreamOptions

	Transport   http.RoundTripper
	Upgrader    clientspdy.Upgrader
	SubResource SubResource
}

// ExecStreamWithTransport executes a kubectl exec with given transport round tripper and upgrader
func ExecStreamWithTransport(client kubernetes.Interface, options *ExecStreamWithTransportOptions) error {
	var (
		t             term.TTY
		sizeQueue     remotecommand.TerminalSizeQueue
		streamOptions remotecommand.StreamOptions
		tty           = options.TTY
	)

	execRequest := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.Pod).
		Namespace(options.Namespace).
		SubResource(string(options.SubResource))

	if tty {
		tty, t = SetupTTY(options.Stdin, options.Stdout)
		if options.ForceTTY || tty {
			tty = true
			if t.Raw && options.TerminalSizeQueue == nil {
				// this call spawns a goroutine to monitor/update the terminal size
				sizeQueue = t.MonitorSize(t.GetSize())
			} else if options.TerminalSizeQueue != nil {
				sizeQueue = options.TerminalSizeQueue
				t.Raw = true
			}

			// unset options.Stderr if it was previously set because both stdout and stderr
			// go over t.Out when tty is true
			options.Stderr = nil
			streamOptions = remotecommand.StreamOptions{
				Stdin:             t.In,
				Stdout:            t.Out,
				Tty:               t.Raw,
				TerminalSizeQueue: sizeQueue,
			}
		}
	}
	if !tty {
		streamOptions = remotecommand.StreamOptions{
			Stdin:  options.Stdin,
			Stdout: options.Stdout,
			Stderr: options.Stderr,
		}
	}

	if options.SubResource == SubResourceExec {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Command:   options.Command,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	} else if options.SubResource == SubResourceAttach {
		execRequest.VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)
	}

	exec, err := remotecommand.NewSPDYExecutorForTransports(options.Transport, options.Upgrader, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	return t.Safe(func() error {
		return exec.Stream(streamOptions)
	})
}

// ExecStreamOptions are the options for ExecStream
type ExecStreamOptions struct {
	Pod       string
	Namespace string
	Container string
	Command   []string

	ForceTTY          bool
	TTY               bool
	TerminalSizeQueue remotecommand.TerminalSizeQueue

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecStream executes a command and streams the output to the given streams
func ExecStream(kubeConfig *rest.Config, options *ExecStreamOptions) error {
	wrapper, upgradeRoundTripper, err := GetUpgraderWrapper(kubeConfig)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	return ExecStreamWithTransport(kubeClient, &ExecStreamWithTransportOptions{
		ExecStreamOptions: *options,
		Transport:         wrapper,
		Upgrader:          upgradeRoundTripper,
		SubResource:       SubResourceExec,
	})
}

// ExecBuffered executes a command for kubernetes and returns the output and error buffers
func ExecBuffered(kubeConfig *rest.Config, namespace, pod, container string, command []string, stdin io.Reader) ([]byte, []byte, error) {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}

	kubeExecError := ExecStream(kubeConfig, &ExecStreamOptions{
		Pod:       pod,
		Namespace: namespace,
		Container: container,
		Command:   command,
		Stdin:     stdin,
		Stdout:    stdoutBuffer,
		Stderr:    stderrBuffer,
	})
	if kubeExecError != nil {
		if _, ok := kubeExecError.(kubectlExec.CodeExitError); !ok {
			return nil, nil, kubeExecError
		}
	}

	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), kubeExecError
}

// GetUpgraderWrapper returns an upgrade wrapper for the given config @Factory
func GetUpgraderWrapper(restConfig *rest.Config) (http.RoundTripper, clientspdy.Upgrader, error) {
	wrapper, upgradeRoundTripper, err := clientspdy.RoundTripperFor(restConfig)
	if err != nil {
		return nil, nil, err
	}

	return wrapper, upgradeRoundTripper, nil
}

// SetupTTY creates a term.TTY (docker)
func SetupTTY(stdin io.Reader, stdout io.Writer) (bool, term.TTY) {
	t := term.TTY{
		Out: stdout,
		In:  stdin,
	}

	if !t.IsTerminalIn() {
		return false, t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	newStdin, newStdout, _ := dockerterm.StdStreams()
	t.In = newStdin
	if stdout != nil {
		t.Out = newStdout
	}

	return true, t
}
