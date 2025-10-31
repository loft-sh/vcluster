/*
Copyright 2024 The Kubernetes Authors.

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

package vcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vladimirvivien/gexe"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	log "k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/utils"
	"sigs.k8s.io/e2e-framework/support"
	"sigs.k8s.io/yaml"
)

const (
	vclusterVersion = "v0.20.0"
	vclusterPath    = "vclusterctl"
)

type Cluster struct {
	path            string
	name            string
	kubecfgFile     string // kubeconfig file for the vcluster
	version         string
	namespace       string // namespace to create the vcluster in
	hostKubeCfg     string // kubeconfig file for the host cluster
	hostKubeContext string // kubeconfig context for the host cluster
	rc              *rest.Config
}

// Enforce Type check always to avoid future breaks
var _ support.E2EClusterProvider = &Cluster{}

func NewCluster(name string) *Cluster {
	return &Cluster{name: name}
}

func NewProvider() support.E2EClusterProvider {
	return &Cluster{}
}

func WithPath(path string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		v, ok := c.(*Cluster)
		if ok {
			v.WithPath(path)
		}
	}
}

func WithNamespace(ns string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		v, ok := c.(*Cluster)
		if ok {
			v.namespace = ns
		}
	}
}

func WithHostKubeConfig(kubeconfig string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		v, ok := c.(*Cluster)
		if ok {
			v.hostKubeCfg = kubeconfig
		}
	}
}

func WithHostKubeContext(kubeContext string) support.ClusterOpts {
	return func(c support.E2EClusterProvider) {
		v, ok := c.(*Cluster)
		if ok {
			v.hostKubeContext = kubeContext
		}
	}
}

func (c *Cluster) WithName(name string) support.E2EClusterProvider {
	c.name = name
	return c
}

func (c *Cluster) WithVersion(version string) support.E2EClusterProvider {
	c.version = version
	return c
}

func (c *Cluster) WithPath(path string) support.E2EClusterProvider {
	c.path = path
	return c
}

func (c *Cluster) WithOpts(opts ...support.ClusterOpts) support.E2EClusterProvider {
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Cluster) SetDefaults() support.E2EClusterProvider {
	if c.path == "" {
		c.path = vclusterPath
	}
	return c
}

func (c *Cluster) Create(ctx context.Context, args ...string) (string, error) {
	log.V(4).Info("Creating vcluster ", c.name)
	if err := c.findOrInstallVcluster(); err != nil {
		return "", err
	}

	if _, exists := c.clusterExists(c.name); exists {
		log.V(4).Info("Skipping vcluster Cluster.Create: cluster already created: ", c.name)
		kConfig, err := c.getKubeconfig()
		if err != nil {
			return "", err
		}
		return kConfig, c.initKubernetesAccessClients()
	}

	if c.namespace != "" {
		args = append(args, "--namespace", c.namespace)
	}

	if c.hostKubeContext != "" {
		args = append(args, "--context", c.hostKubeContext)
	}

	command := fmt.Sprintf("%s create %s --connect=false --update-current=false", c.path, c.name)
	if len(args) > 0 {
		command = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}
	log.V(4).Info("Launching:", command)
	echo := gexe.New()
	if c.hostKubeCfg != "" {
		echo.SetEnv("KUBECONFIG", c.hostKubeCfg)
	}

	p := echo.RunProc(command)
	if p.Err() != nil {
		outBytes, err := io.ReadAll(p.Out())
		if err != nil {
			log.ErrorS(err, "failed to read data from the vcluster create process output due to an error")
		}
		return "", fmt.Errorf("vcluster: failed to create cluster %q: %s: %s: %s", c.name, p.Err(), p.Result(), string(outBytes))
	}
	clusters, ok := c.clusterExists(c.name)
	if !ok {
		return "", fmt.Errorf("vcluster Cluster.Create: cluster %v still not in 'cluster list' after creation: %v", c.name, clusters)
	}
	log.V(4).Info("vcluster clusters available: ", clusters)

	kConfig, err := c.getKubeconfig()
	if err != nil {
		return "", err
	}

	return kConfig, nil
}

func (c *Cluster) CreateWithConfig(ctx context.Context, configFile string) (string, error) {
	var args []string
	if configFile != "" {
		args = append(args, "--values", configFile)
	}
	return c.Create(ctx, args...)
}

func (c *Cluster) GetKubeconfig() string {
	return c.kubecfgFile
}

type kubeconfig struct {
	CurrentContext string `json:"current-context"`
}

func (c *Cluster) GetKubectlContext() string {
	kc := &kubeconfig{}
	raw, err := os.ReadFile(c.kubecfgFile)
	if err != nil {
		return ""
	}
	if err := yaml.Unmarshal(raw, kc); err != nil {
		return ""
	}
	return kc.CurrentContext
}

func (c *Cluster) ExportLogs(ctx context.Context, dest string) error {
	// Not implemented
	return nil
}

func (c *Cluster) Destroy(ctx context.Context) error {
	log.V(4).Info("Destroying vcluster ", c.name)
	if err := c.findOrInstallVcluster(); err != nil {
		return err
	}

	command := fmt.Sprintf("%s delete %s", c.path, c.name)
	p := utils.RunCommand(command)
	if p.Err() != nil {
		outBytes, err := io.ReadAll(p.Out())
		if err != nil {
			log.ErrorS(err, "failed to read data from the vcluster delete process output due to an error")
		}
		return fmt.Errorf("vcluster: failed to delete cluster %q: %s: %s: %s", c.name, p.Err(), p.Result(), string(outBytes))
	}

	log.V(4).Info("Removing kubeconfig file ", c.kubecfgFile)
	if err := os.Remove(c.kubecfgFile); err != nil {
		return fmt.Errorf("vcluster: failed to remove kubeconfig file %q: %w", c.kubecfgFile, err)
	}
	return nil
}

func (c *Cluster) WaitForControlPlane(ctx context.Context, client klient.Client) error {
	r, err := resources.New(client.RESTConfig())
	if err != nil {
		return err
	}

	// vcluster seem to name the dns pod with label vcluster-kube-dns.
	// Wish there was a way to avoid hard coding this slightly better.
	for _, sl := range []metav1.LabelSelectorRequirement{
		{Key: "k8s-app", Operator: metav1.LabelSelectorOpIn, Values: []string{"vcluster-kube-dns"}},
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

func (c *Cluster) KubernetesRestConfig() *rest.Config {
	return c.rc
}

// helpers to implement support.E2EClusterProvider
func (c *Cluster) findOrInstallVcluster() error {
	version := c.version
	if c.version == "" {
		version = vclusterVersion
	}
	path, err := utils.FindOrInstallGoBasedProvider(c.path, vclusterPath, "github.com/loft-sh/vcluster/cmd/vclusterctl", version)
	if path != "" {
		c.path = path
	}
	return err
}

type clusterItem struct {
	Name string `json:"Name"`
}

func (c *Cluster) clusterExists(name string) (string, bool) {
	raw := utils.FetchCommandOutput(fmt.Sprintf("%s list --output json", c.path))
	clusters := []clusterItem{}
	if err := json.Unmarshal([]byte(raw), &clusters); err != nil {
		return raw, false
	}
	for _, c := range clusters {
		if c.Name == name {
			return raw, true
		}
	}
	return raw, false
}

func (c *Cluster) getKubeconfig() (string, error) {
	kubecfg := fmt.Sprintf("%s-kubecfg", c.name)

	var stdout, stderr bytes.Buffer
	err := utils.RunCommandWithSeperatedOutput(fmt.Sprintf(`%s connect %s --print`, c.path, c.name), &stdout, &stderr)
	if err != nil {
		return "", fmt.Errorf("vcluster connect: stderr: %s: %w", stderr.String(), err)
	}
	log.V(4).Info("vcluster connect stderr \n", stderr.String())

	file, err := os.CreateTemp("", fmt.Sprintf("vcluster-cluster-%s", kubecfg))
	if err != nil {
		return "", fmt.Errorf("vcluster kubeconfig file: %w", err)
	}
	defer file.Close()

	c.kubecfgFile = file.Name()

	if n, err := io.WriteString(file, stdout.String()); n == 0 || err != nil {
		return "", fmt.Errorf("vcluster kubecfg file: bytes copied: %d: %w]", n, err)
	}
	return file.Name(), nil
}

func (c *Cluster) initKubernetesAccessClients() error {
	cfg, err := conf.New(c.kubecfgFile)
	if err != nil {
		return err
	}
	c.rc = cfg
	return nil
}
