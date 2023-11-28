package k3s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var tokenPath = "/data/server/token"

const VClusterCommandEnv = "VCLUSTER_COMMAND"

type k3sCommand struct {
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

func StartK3S(ctx context.Context, serviceCIDR, k3sToken string) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer writer.Close()

	command := &k3sCommand{}
	err = yaml.Unmarshal([]byte(os.Getenv(VClusterCommandEnv)), command)
	if err != nil {
		return fmt.Errorf("parsing k3s command %s: %w", os.Getenv(VClusterCommandEnv), err)
	}

	// add service cidr and k3s token
	command.Args = append(
		command.Args,
		"--service-cidr", serviceCIDR,
		"--token", strings.TrimSpace(k3sToken),
	)
	args := append(command.Command, command.Args...)

	// start func
	done := make(chan struct{})
	go func() {
		defer close(done)

		// make sure we scan the output correctly
		scan := scanner.NewScanner(reader)
		for scan.Scan() {
			line := scan.Text()
			if len(line) == 0 {
				continue
			}

			// print to our logs
			args := []interface{}{"component", "k3s"}
			loghelper.PrintKlogLine(line, args)
		}
	}()

	// start the command
	klog.InfoS("Starting k3s", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = writer
	cmd.Stderr = writer
	err = cmd.Run()

	// make sure we wait for scanner to be done
	_ = writer.Close()
	<-done

	// regular stop case
	if err != nil && err.Error() != "signal: killed" {
		return err
	}
	return nil
}

func EnsureK3SToken(ctx context.Context, currentNamespaceClient kubernetes.Interface, currentNamespace, vClusterName string) (string, error) {
	// check if secret exists
	secretName := fmt.Sprintf("vc-k3s-%s", vClusterName)
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return string(secret.Data["token"]), nil
	} else if !kerrors.IsNotFound(err) {
		return "", err
	}

	// try to read token file (migration case)
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		token = []byte(random.String(64))
	}

	// create k3s secret
	secret, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: currentNamespace,
		},
		Data: map[string][]byte{
			"token": token,
		},
		Type: corev1.SecretTypeOpaque,
	}, metav1.CreateOptions{})

	if kerrors.IsAlreadyExists(err) {
		// retrieve k3s secret again
		secret, err = currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return string(secret.Data["token"]), nil
}
