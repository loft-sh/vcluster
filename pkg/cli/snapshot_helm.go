package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/ghodss/yaml"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var minSnapshotVersion = "0.23.0-alpha.8"

func Snapshot(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshotOpts *snapshot.Options, podOptions *pod.Options, log log.Logger, async bool) error {
	// init kube client and vCluster
	vCluster, kubeClient, restConfig, err := initSnapshotCommand(ctx, args, globalFlags, snapshotOpts, log)
	if err != nil {
		return err
	}

	// get vCluster release
	vClusterRelease, err := helm.NewSecrets(kubeClient).Get(ctx, vCluster.Name, vCluster.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get vCluster release: %w", err)
	}

	// set helm release
	if vClusterRelease != nil && vClusterRelease.Chart != nil && vClusterRelease.Chart.Metadata != nil {
		values, _ := yaml.Marshal(vClusterRelease.Config)
		snapshotOpts.Release = &snapshot.HelmRelease{
			ReleaseName:      vClusterRelease.Name,
			ReleaseNamespace: vClusterRelease.Namespace,
			ChartName:        vClusterRelease.Chart.Metadata.Name,
			ChartVersion:     vClusterRelease.Chart.Metadata.Version,
			Values:           values,
		}
	}

	if !async {
		// run snapshot pod
		return pod.RunSnapshotPod(ctx, restConfig, kubeClient, []string{"/vcluster", "snapshot"}, vCluster, podOptions, snapshotOpts, log)
	}

	// create a snapshot request
	snapshotRequest := &snapshot.Request{
		Spec: snapshot.RequestSpec{
			Options: *snapshotOpts,
		},
	}
	configMap, secret, err := snapshot.MarshalSnapshotRequest(vCluster.Namespace, snapshotRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot request: %w", err)
	}
	// create snapshot request Secret
	secret.GenerateName = "snapshot-request-"
	secret, err = kubeClient.CoreV1().Secrets(vCluster.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create a snapshot request secret: %w", err)
	}
	// create snapshot request ConfigMap
	configMap.Name = secret.Name
	_, err = kubeClient.CoreV1().ConfigMaps(vCluster.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create a snapshot request configmap: %w", err)
	}

	log.Infof(
		"Snapshot request %s/%s created, you can check it by running `kubectl get configmap -n %s %s`",
		configMap.Namespace, configMap.Name,
		configMap.Namespace, configMap.Name)

	return nil
}

func fillSnapshotOptions(snapshotURL string, snapshotOptions *snapshot.Options) error {
	// parse snapshot url
	err := snapshot.Parse(snapshotURL, snapshotOptions)
	if err != nil {
		return fmt.Errorf("parse snapshot url: %w", err)
	}

	// storage needs to be either s3 or file
	err = snapshot.Validate(snapshotOptions, false)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	// try to fill in oci options
	snapshotOptions.OCI.FillCredentials(true)
	snapshotOptions.S3.FillCredentials(true)
	return nil
}

func initSnapshotCommand(
	ctx context.Context,
	args []string,
	globalFlags *flags.GlobalFlags,
	snapshotOptions *snapshot.Options,
	log log.Logger,
) (*find.VCluster, *kubernetes.Clientset, *rest.Config, error) {
	if len(args) != 2 {
		return nil, nil, nil, fmt.Errorf("unexpected amount of arguments: %d, need exactly 2 arguments. E.g. vcluster [snapshot|restore] my-vcluster s3://my-bucket/my-key", len(args))
	}

	// parse snapshot url
	err := fillSnapshotOptions(args[1], snapshotOptions)
	if err != nil {
		return nil, nil, nil, err
	}

	// find the vCluster
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// check if snapshot is supported
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return nil, nil, nil, fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
	}

	// build kubernetes client
	restClient, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return nil, nil, nil, err
	}

	return vCluster, kubeClient, restClient, nil
}
