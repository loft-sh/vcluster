package kubeexec

import (
	"context"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecStreamOptions are the options for ExecStream
type ExecStreamOptions struct {
	Pod       string
	Namespace string
	Container string

	Command []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// SubResource specifies with sub resources should be used for the container connection (exec or attach)
type SubResource string

const (
	// SubResourceExec creates a new process in the container and attaches to that
	SubResourceExec SubResource = "exec"
)

// ExecCombinedOutput prints the combined output stdout & stderr
func ExecCombinedOutput(ctx context.Context, restConfig *rest.Config, pod, namespace, container string, command []string) ([]byte, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = writer.Close()
	}()

	err = Exec(ctx, restConfig, &ExecStreamOptions{
		Pod:       pod,
		Namespace: namespace,
		Container: container,
		Command:   command,
		Stdout:    writer,
		Stderr:    writer,
	})
	if err != nil {
		return nil, err
	}

	_ = writer.Close()
	return io.ReadAll(reader)
}

// Exec executes a kubectl exec with given transport round tripper and upgrader
func Exec(ctx context.Context, restConfig *rest.Config, options *ExecStreamOptions) error {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	execRequest := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.Pod).
		Namespace(options.Namespace).
		SubResource(string(SubResourceExec)).
		VersionedParams(&corev1.PodExecOptions{
			Container: options.Container,
			Command:   options.Command,
			Stdin:     options.Stdin != nil,
			Stdout:    options.Stdout != nil,
			Stderr:    options.Stderr != nil,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:  options.Stdin,
			Stdout: options.Stdout,
			Stderr: options.Stderr,
		})
	}()

	select {
	case <-ctx.Done():
		<-errChan
		return nil
	case err = <-errChan:
		return err
	}
}
