package framework

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"k8s.io/client-go/tools/clientcmd"
	uexec "k8s.io/utils/exec"
)

// Adopted from k8s test suite - https://github.com/kubernetes/kubernetes/blob/f2576efecdf2d902b12a3fedae7995311d4febfa/test/e2e/framework/kubectl/kubectl_utils.go#L43-L100
// TestKubeconfig is a struct containing the needed attributes from TestContext and Framework(Namespace).
type TestKubeconfig struct {
	CertDir     string
	Host        string
	KubeConfig  string
	KubeContext string
	KubectlPath string
	Namespace   string // Every test has at least one namespace unless creation is skipped
}

// NewTestKubeconfig returns a new Kubeconfig struct instance.
func NewTestKubeconfig(kubeconfig, namespace string) *TestKubeconfig {
	return &TestKubeconfig{
		KubeConfig:  kubeconfig,
		Namespace:   namespace,
		KubectlPath: "kubectl",
	}
}

// KubectlCmd runs the kubectl executable through the wrapper script.
func (tk *TestKubeconfig) KubectlCmd(args ...string) *exec.Cmd {
	defaultArgs := []string{}

	// Reference a --server option so tests can run anywhere.
	if tk.Host != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.FlagAPIServer+"="+tk.Host)
	}
	if tk.KubeConfig != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.RecommendedConfigPathFlag+"="+tk.KubeConfig)

		// Reference the KubeContext
		if tk.KubeContext != "" {
			defaultArgs = append(defaultArgs, "--"+clientcmd.FlagContext+"="+tk.KubeContext)
		}

	} else {
		if tk.CertDir != "" {
			defaultArgs = append(defaultArgs,
				fmt.Sprintf("--certificate-authority=%s", filepath.Join(tk.CertDir, "ca.crt")),
				fmt.Sprintf("--client-certificate=%s", filepath.Join(tk.CertDir, "kubecfg.crt")),
				fmt.Sprintf("--client-key=%s", filepath.Join(tk.CertDir, "kubecfg.key")))
		}
	}
	if tk.Namespace != "" {
		defaultArgs = append(defaultArgs, fmt.Sprintf("--namespace=%s", tk.Namespace))
	}
	kubectlArgs := append(defaultArgs, args...)

	//We allow users to specify path to kubectl, so you can test either "kubectl" or "cluster/kubectl.sh"
	//and so on.
	cmd := exec.Command(tk.KubectlPath, kubectlArgs...)

	//caller will invoke this and wait on it.
	return cmd
}

// Adopted from k8s test suite - https://github.com/kubernetes/kubernetes/blob/f2576efecdf2d902b12a3fedae7995311d4febfa/test/e2e/framework/util.go#L552-L687
// KubectlBuilder is used to build, customize and execute a kubectl Command.
// Add more functions to customize the builder as needed.
type KubectlBuilder struct {
	cmd     *exec.Cmd
	timeout <-chan time.Time
}

// NewKubectlCommand returns a KubectlBuilder for running kubectl.
func NewKubectlCommand(kubeconfigPath, namespace string, args ...string) *KubectlBuilder {
	b := new(KubectlBuilder)
	tk := NewTestKubeconfig(kubeconfigPath, namespace)
	b.cmd = tk.KubectlCmd(args...)
	return b
}

// WithEnv sets the given environment and returns itself.
func (b *KubectlBuilder) WithEnv(env []string) *KubectlBuilder {
	b.cmd.Env = env
	return b
}

// WithTimeout sets the given timeout and returns itself.
func (b *KubectlBuilder) WithTimeout(t <-chan time.Time) *KubectlBuilder {
	b.timeout = t
	return b
}

// WithStdinData sets the given data to stdin and returns itself.
func (b KubectlBuilder) WithStdinData(data string) *KubectlBuilder {
	b.cmd.Stdin = strings.NewReader(data)
	return &b
}

// WithStdinReader sets the given reader and returns itself.
func (b KubectlBuilder) WithStdinReader(reader io.Reader) *KubectlBuilder {
	b.cmd.Stdin = reader
	return &b
}

// ExecOrDie runs the kubectl executable or dies if error occurs.
func (b KubectlBuilder) ExecOrDie(namespace string) string {
	str, err := b.Exec()
	// In case of i/o timeout error, try talking to the apiserver again after 2s before dying.
	// Note that we're still dying after retrying so that we can get visibility to triage it further.
	if isTimeout(err) {
		log.GetInstance().Infof("Hit i/o timeout error, talking to the server 2s later to see if it's temporary.")
		time.Sleep(2 * time.Second)
		retryStr, retryErr := RunKubectl(namespace, "version")
		log.GetInstance().Infof("stdout: %q", retryStr)
		log.GetInstance().Infof("err: %v", retryErr)
	}
	ExpectNoError(err)
	return str
}

func isTimeout(err error) bool {
	switch err := err.(type) {
	case *url.Error:
		if err, ok := err.Err.(net.Error); ok && err.Timeout() {
			return true
		}
	case net.Error:
		if err.Timeout() {
			return true
		}
	}
	return false
}

// Exec runs the kubectl executable.
func (b KubectlBuilder) Exec() (string, error) {
	stdout, _, err := b.ExecWithFullOutput()
	return stdout, err
}

// ExecWithFullOutput runs the kubectl executable, and returns the stdout and stderr.
func (b KubectlBuilder) ExecWithFullOutput() (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := b.cmd
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	log.GetInstance().Infof("Running '%s %s'", cmd.Path, strings.Join(cmd.Args[1:], " ")) // skip arg[0] as it is printed separately
	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("error starting %v:\nCommand stdout:\n%v\nstderr:\n%v\nerror:\n%v", cmd, cmd.Stdout, cmd.Stderr, err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()
	select {
	case err := <-errCh:
		if err != nil {
			var rc = 127
			if ee, ok := err.(*exec.ExitError); ok {
				rc = int(ee.Sys().(syscall.WaitStatus).ExitStatus())
				log.GetInstance().Infof("rc: %d", rc)
			}
			return stdout.String(), stderr.String(), uexec.CodeExitError{
				Err:  fmt.Errorf("error running %v:\nCommand stdout:\n%v\nstderr:\n%v\nerror:\n%v", cmd, cmd.Stdout, cmd.Stderr, err),
				Code: rc,
			}
		}
	case <-b.timeout:
		_ = b.cmd.Process.Kill()
		return "", "", fmt.Errorf("timed out waiting for command %v:\nCommand stdout:\n%v\nstderr:\n%v", cmd, cmd.Stdout, cmd.Stderr)
	}
	log.GetInstance().Infof("stderr: %q", stderr.String())
	log.GetInstance().Infof("stdout: %q", stdout.String())
	return stdout.String(), stderr.String(), nil
}

// RunKubectlOrDie is a convenience wrapper over kubectlBuilder
func RunKubectlOrDie(kubeconfigPath, namespace string, args ...string) string {
	return NewKubectlCommand(kubeconfigPath, namespace, args...).ExecOrDie(namespace)
}

// RunKubectl is a convenience wrapper over kubectlBuilder
func RunKubectl(kubeconfigPath, namespace string, args ...string) (string, error) {
	return NewKubectlCommand(kubeconfigPath, namespace, args...).Exec()
}

// RunKubectlWithFullOutput is a convenience wrapper over kubectlBuilder
// It will also return the command's stderr.
func RunKubectlWithFullOutput(kubeconfigPath, namespace string, args ...string) (string, string, error) {
	return NewKubectlCommand(kubeconfigPath, namespace, args...).ExecWithFullOutput()
}

// RunKubectlOrDieInput is a convenience wrapper over kubectlBuilder that takes input to stdin
func RunKubectlOrDieInput(kubeconfigPath, namespace string, data string, args ...string) string {
	return NewKubectlCommand(kubeconfigPath, namespace, args...).WithStdinData(data).ExecOrDie(namespace)
}

// RunKubectlInput is a convenience wrapper over kubectlBuilder that takes input to stdin
func RunKubectlInput(kubeconfigPath, namespace string, data string, args ...string) (string, error) {
	return NewKubectlCommand(kubeconfigPath, namespace, args...).WithStdinData(data).Exec()
}
