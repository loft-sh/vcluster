/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kind

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/utils"
	"sigs.k8s.io/e2e-framework/support"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var kindVersion = "v0.26.0"

type Cluster struct {
	path        string
	name        string
	kubecfgFile string
	version     string
	image       string
	rc          *rest.Config
}

// Enforce Type check always to avoid future breaks
var _ support.E2EClusterProvider = &Cluster{}

func NewCluster(name string) *Cluster {
	return &Cluster{name: name}
}

func NewProvider() support.E2EClusterProvider {
	return &Cluster{}
}

func WithImage(image string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		k, ok := c.(*Cluster)
		if ok {
			k.image = image
		}
	}
}

func WithPath(path string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		k, ok := c.(*Cluster)
		if ok {
			k.path = path
		}
	}
}

func (k *Cluster) SetDefaults() support.E2EClusterProvider {
	if k.path == "" {
		k.path = "kind"
	}
	return k
}

func (k *Cluster) WithName(name string) support.E2EClusterProvider {
	k.name = name
	return k
}

func (k *Cluster) WithPath(path string) support.E2EClusterProvider {
	k.path = path
	return k
}

func (k *Cluster) WithVersion(ver string) support.E2EClusterProvider {
	k.version = ver
	return k
}

func (k *Cluster) WithOpts(opts ...support.ClusterOpts) support.E2EClusterProvider {
	for _, o := range opts {
		o(k)
	}
	return k
}

func (k *Cluster) getKubeconfig() (string, error) {
	kubecfg := fmt.Sprintf("%s-kubecfg", k.name)

	var stdout, stderr bytes.Buffer
	err := utils.RunCommandWithSeperatedOutput(fmt.Sprintf(`%s get kubeconfig --name %s`, k.path, k.name), &stdout, &stderr)
	if err != nil {
		return "", fmt.Errorf("kind get kubeconfig: stderr: %s: %w", stderr.String(), err)
	}
	log.V(4).Info("kind get kubeconfig stderr \n", stderr.String())

	file, err := os.CreateTemp("", fmt.Sprintf("kind-cluster-%s", kubecfg))
	if err != nil {
		return "", fmt.Errorf("kind kubeconfig file: %w", err)
	}
	defer file.Close()

	k.kubecfgFile = file.Name()

	if n, err := io.WriteString(file, stdout.String()); n == 0 || err != nil {
		return "", fmt.Errorf("kind kubecfg file: bytes copied: %d: %w]", n, err)
	}

	return file.Name(), nil
}

func (k *Cluster) clusterExists(name string) (string, bool) {
	clusters := utils.FetchCommandOutput(fmt.Sprintf("%s get clusters", k.path))
	for _, c := range strings.Split(clusters, "\n") {
		if c == name {
			return clusters, true
		}
	}
	return clusters, false
}

func (k *Cluster) CreateWithConfig(ctx context.Context, kindConfigFile string) (string, error) {
	var args []string
	if kindConfigFile != "" {
		args = append(args, "--config", kindConfigFile)
	}
	return k.Create(ctx, args...)
}

func (k *Cluster) Create(ctx context.Context, args ...string) (string, error) {
	log.V(4).Info("Creating kind cluster ", k.name)
	if err := k.findOrInstallKind(); err != nil {
		return "", err
	}

	if _, ok := k.clusterExists(k.name); ok {
		log.V(4).Info("Skipping Kind Cluster.Create: cluster already created: ", k.name)
		kConfig, err := k.getKubeconfig()
		if err != nil {
			return "", err
		}
		return kConfig, k.initKubernetesAccessClients()
	}

	if k.image != "" {
		args = append(args, "--image", k.image)
	}

	command := fmt.Sprintf(`%s create cluster --name %s`, k.path, k.name)
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	log.V(4).Info("Launching:", command)
	p := utils.RunCommand(command)
	if p.Err() != nil {
		outBytes, err := io.ReadAll(p.Out())
		if err != nil {
			log.ErrorS(err, "failed to read data from the kind create process output due to an error")
		}
		return "", fmt.Errorf("kind: failed to create cluster %q: %s: %s: %s", k.name, p.Err(), p.Result(), string(outBytes))
	}
	clusters, ok := k.clusterExists(k.name)
	if !ok {
		return "", fmt.Errorf("kind Cluster.Create: cluster %v still not in 'cluster list' after creation: %v", k.name, clusters)
	}
	log.V(4).Info("kind clusters available: ", clusters)

	kConfig, err := k.getKubeconfig()
	if err != nil {
		return "", err
	}
	return kConfig, k.initKubernetesAccessClients()
}

func (k *Cluster) initKubernetesAccessClients() error {
	cfg, err := conf.New(k.kubecfgFile)
	if err != nil {
		return err
	}
	k.rc = cfg
	return nil
}

func (k *Cluster) GetKubeconfig() string {
	return k.kubecfgFile
}

func (k *Cluster) GetKubectlContext() string {
	return fmt.Sprintf("kind-%s", k.name)
}

// ExportLogs export all cluster logs to the provided path.
func (k *Cluster) ExportLogs(ctx context.Context, dest string) error {
	log.V(4).Info("Exporting kind cluster logs to ", dest)
	if err := k.findOrInstallKind(); err != nil {
		return err
	}

	p := utils.RunCommand(fmt.Sprintf(`%s export logs %s --name %s`, k.path, dest, k.name))
	if p.Err() != nil {
		return fmt.Errorf("kind: export cluster %v logs failed: %s: %s", k.name, p.Err(), p.Result())
	}

	return nil
}

func (k *Cluster) Destroy(ctx context.Context) error {
	log.V(4).Info("Destroying kind cluster ", k.name)
	if err := k.findOrInstallKind(); err != nil {
		return err
	}

	p := utils.RunCommand(fmt.Sprintf(`%s delete cluster --name %s`, k.path, k.name))
	if p.Err() != nil {
		outBytes, err := io.ReadAll(p.Out())
		if err != nil {
			log.ErrorS(err, "failed to read data from the kind delete process output due to an error")
		}
		return fmt.Errorf("kind: failed to delete cluster %q: %s: %s: %s", k.name, p.Err(), p.Result(), string(outBytes))
	}

	log.V(4).Info("Removing kubeconfig file ", k.kubecfgFile)
	if err := os.RemoveAll(k.kubecfgFile); err != nil {
		return fmt.Errorf("kind: remove kubefconfig %v failed: %w", k.kubecfgFile, err)
	}

	return nil
}

func (k *Cluster) findOrInstallKind() error {
	if k.version != "" {
		kindVersion = k.version
	}
	path, err := utils.FindOrInstallGoBasedProvider(k.path, "kind", "sigs.k8s.io/kind", kindVersion)
	if path != "" {
		k.path = path
	}
	return err
}

func (k *Cluster) LoadImage(ctx context.Context, image string, args ...string) error {
	p := utils.RunCommand(fmt.Sprintf(`%s load docker-image --name %s %s`, k.path, k.name, image))
	if p.Err() != nil {
		return fmt.Errorf("kind: load docker-image %v failed: %s: %s", image, p.Err(), p.Result())
	}
	return nil
}

func (k *Cluster) LoadImageArchive(ctx context.Context, imageArchive string, args ...string) error {
	p := utils.RunCommand(fmt.Sprintf(`%s load image-archive --name %s %s`, k.path, k.name, imageArchive))
	if p.Err() != nil {
		return fmt.Errorf("kind: load image-archive %v failed: %s: %s", imageArchive, p.Err(), p.Result())
	}
	return nil
}

func (k *Cluster) WaitForControlPlane(ctx context.Context, client klient.Client) error {
	r, err := resources.New(client.RESTConfig())
	if err != nil {
		return err
	}
	for _, sl := range []metav1.LabelSelectorRequirement{
		{Key: "component", Operator: metav1.LabelSelectorOpIn, Values: []string{"etcd", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}},
		{Key: "k8s-app", Operator: metav1.LabelSelectorOpIn, Values: []string{"kindnet", "kube-dns", "kube-proxy"}},
	} {
		selector, err := metav1.LabelSelectorAsSelector(
			&metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					sl,
				},
			},
		)
		if err != nil {
			return err
		}
		err = wait.For(conditions.New(r).ResourceListN(&v1.PodList{}, len(sl.Values), resources.WithLabelSelector(selector.String())))
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Cluster) KubernetesRestConfig() *rest.Config {
	return k.rc
}
