package k3s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/util/commandwriter"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var TokenPath = "/data/server/token"

func StartK3S(ctx context.Context, vConfig *config.VirtualClusterConfig, serviceCIDR, k3sToken string) error {
	// build args
	args := []string{}
	if len(vConfig.ControlPlane.Distro.K3S.Command) > 0 {
		args = append(args, vConfig.ControlPlane.Distro.K3S.Command...)
	} else {
		args = append(args, "/binaries/k3s")
		args = append(args, "server")
		args = append(args, "--write-kubeconfig=/data/k3s-config/kube-config.yaml")
		args = append(args, "--data-dir=/data")
		args = append(args, "--service-cidr="+serviceCIDR)
		args = append(args, "--token="+strings.TrimSpace(k3sToken))
		args = append(args, "--disable=traefik,servicelb,metrics-server,local-storage,coredns")
		args = append(args, "--disable-network-policy")
		args = append(args, "--disable-agent")
		args = append(args, "--disable-cloud-controller")
		args = append(args, "--egress-selector-mode=disabled")
		args = append(args, "--flannel-backend=none")
		args = append(args, "--kube-apiserver-arg=bind-address=127.0.0.1")
		if vConfig.ControlPlane.Advanced.VirtualScheduler.Enabled {
			args = append(args, "--kube-controller-manager-arg=controllers=*,-nodeipam,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
			args = append(args, "--kube-apiserver-arg=endpoint-reconciler-type=none")
			args = append(args, "--kube-controller-manager-arg=node-monitor-grace-period=1h")
			args = append(args, "--kube-controller-manager-arg=node-monitor-period=1h")
		} else {
			args = append(args, "--disable-scheduler")
			args = append(args, "--kube-controller-manager-arg=controllers=*,-nodeipam,-nodelifecycle,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle,-ttl")
			args = append(args, "--kube-apiserver-arg=endpoint-reconciler-type=none")
		}
		if vConfig.ControlPlane.BackingStore.Etcd.Deploy.Enabled {
			// wait until etcd is up and running
			_, err := etcd.WaitForEtcdClient(ctx, &etcd.Certificates{
				CaCert:     "/data/pki/etcd/ca.crt",
				ServerCert: "/data/pki/apiserver-etcd-client.crt",
				ServerKey:  "/data/pki/apiserver-etcd-client.key",
			}, "https://"+vConfig.Name+"-etcd:2379")
			if err != nil {
				return err
			}

			args = append(args, "--datastore-endpoint=https://"+vConfig.Name+"-etcd:2379")
			args = append(args, "--datastore-cafile=/data/pki/etcd/ca.crt")
			args = append(args, "--datastore-certfile=/data/pki/apiserver-etcd-client.crt")
			args = append(args, "--datastore-keyfile=/data/pki/apiserver-etcd-client.key")
		} else if vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			args = append(args, "--datastore-endpoint=https://localhost:2379")
			args = append(args, "--datastore-cafile=/data/pki/etcd/ca.crt")
			args = append(args, "--datastore-certfile=/data/pki/apiserver-etcd-client.crt")
			args = append(args, "--datastore-keyfile=/data/pki/apiserver-etcd-client.key")
		} else if vConfig.EmbeddedDatabase() && vConfig.Config.ControlPlane.BackingStore.Database.Embedded.DataSource != "" {
			args = append(args, "--datastore-endpoint="+vConfig.Config.ControlPlane.BackingStore.Database.Embedded.DataSource)
			args = append(args, "--datastore-cafile="+vConfig.Config.ControlPlane.BackingStore.Database.Embedded.CaFile)
			args = append(args, "--datastore-certfile="+vConfig.Config.ControlPlane.BackingStore.Database.Embedded.CertFile)
			args = append(args, "--datastore-keyfile="+vConfig.Config.ControlPlane.BackingStore.Database.Embedded.KeyFile)
		} else if vConfig.Config.ControlPlane.BackingStore.Database.External.Enabled {
			args = append(args, "--datastore-endpoint="+vConfig.Config.ControlPlane.BackingStore.Database.External.DataSource)
			args = append(args, "--datastore-cafile="+vConfig.Config.ControlPlane.BackingStore.Database.External.CaFile)
			args = append(args, "--datastore-certfile="+vConfig.Config.ControlPlane.BackingStore.Database.External.CertFile)
			args = append(args, "--datastore-keyfile="+vConfig.Config.ControlPlane.BackingStore.Database.External.KeyFile)
		}
	}

	// add extra args
	args = append(args, vConfig.ControlPlane.Distro.K3S.ExtraArgs...)

	// check what writer we should use
	writer, err := commandwriter.NewCommandWriter("k3s", false)
	if err != nil {
		return err
	}
	defer writer.Close()

	// start the command
	klog.InfoS("Starting k3s", "args", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = writer.Writer()
	cmd.Stderr = writer.Writer()
	err = cmd.Run()

	// make sure we wait for scanner to be done
	writer.CloseAndWait(ctx, err)

	// regular stop case
	if err != nil && err.Error() != "signal: killed" {
		return err
	}
	return nil
}

func EnsureK3SToken(ctx context.Context, currentNamespaceClient kubernetes.Interface, currentNamespace, vClusterName string, options *config.VirtualClusterConfig) (string, error) {
	// check if token is set externally
	if options.ControlPlane.Distro.K3S.Token != "" {
		return options.ControlPlane.Distro.K3S.Token, nil
	}

	// check if secret exists
	secretName := fmt.Sprintf("vc-k3s-%s", vClusterName)
	secret, err := currentNamespaceClient.CoreV1().Secrets(currentNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return string(secret.Data["token"]), nil
	} else if !kerrors.IsNotFound(err) {
		return "", err
	}

	// try to read token file (migration case)
	token, err := os.ReadFile(TokenPath)
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
