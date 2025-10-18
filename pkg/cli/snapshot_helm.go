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
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	minSnapshotVersion      = "0.23.0-alpha.8"
	minAsyncSnapshotVersion = "0.29.0-alpha.1"
)

func CreateSnapshot(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshotOpts *snapshot.Options, podOptions *pod.Options, log log.Logger, async bool) error {
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

	// creating snapshot request with 'vcluster snapshot create' command
	err = createSnapshotRequest(ctx, vCluster, kubeClient, snapshotOpts, log)
	if err != nil {
		return err
	}
	return nil
}

func GetSnapshots(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshotOpts *snapshot.Options, log log.Logger) error {
	// init kube client and vCluster
	vCluster, kubeClient, restConfig, err := initSnapshotCommand(ctx, args, globalFlags, snapshotOpts, log)
	if err != nil {
		return fmt.Errorf("failed to init snapshot command: %w", err)
	}

	if snapshotOpts.Type == "container" {
		podOptions := &pod.Options{
			Exec: true,
		}
		err = pod.RunSnapshotPod(ctx, restConfig, kubeClient, []string{"/vcluster", "snapshot", "get"}, vCluster, podOptions, snapshotOpts, log)
		if err != nil {
			return fmt.Errorf("failed to run snapshot pod: %w", err)
		}
		return nil
	}

	err = snapshot.GetSnapshots(ctx, vCluster.Namespace, snapshotOpts, kubeClient, log)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}
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

func createSnapshotRequest(ctx context.Context, vCluster *find.VCluster, kubeClient *kubernetes.Clientset, snapshotOpts *snapshot.Options, log log.Logger) error {
	err := checkIfVClusterSupportsSnapshotRequests(vCluster, log)
	if err != nil {
		return fmt.Errorf("vCluster version check failed: %w", err)
	}
	vClusterConfig, err := getVClusterConfig(ctx, vCluster, kubeClient, snapshotOpts)
	if err != nil {
		return fmt.Errorf("failed to get vcluster config: %w", err)
	}
	// Create snapshot request resources
	_, err = snapshot.CreateSnapshotRequestResources(ctx, vCluster.Namespace, vCluster.Name, vClusterConfig, snapshotOpts, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to create snapshot request resources: %w", err)
	}
	log.Infof("Beginning snapshot creation... Check the snapshot status by running `vcluster snapshot get %s %s`", vCluster.Name, snapshotOpts.GetURL())
	return nil
}

func checkIfVClusterSupportsSnapshotRequests(vCluster *find.VCluster, log log.Logger) error {
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if the version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minAsyncSnapshotVersion)) {
			return fmt.Errorf("command `vcluster snapshot create` can be used with vCluster version %s and newer, but specified virtual cluster uses vCluster version %s", minAsyncSnapshotVersion, vCluster.Version)
		}
	}
	return nil
}

func getVClusterConfig(ctx context.Context, vCluster *find.VCluster, kubeClient *kubernetes.Clientset, snapshotOpts *snapshot.Options) (*vclusterconfig.VirtualClusterConfig, error) {
	var err error
	var vClusterConfig *vclusterconfig.VirtualClusterConfig
	if snapshotOpts.Release.Values != nil {
		vClusterConfig, err = vclusterconfig.ParseConfigBytes(snapshotOpts.Release.Values, vCluster.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vcluster config: %w", err)
		}
	} else {
		// get vCluster config
		var configSecretName string
		var volumesToCheck []corev1.Volume
		if vCluster.StatefulSet != nil {
			volumesToCheck = vCluster.StatefulSet.Spec.Template.Spec.Volumes
		} else if vCluster.Deployment != nil {
			volumesToCheck = vCluster.Deployment.Spec.Template.Spec.Volumes
		} else {
			return nil, fmt.Errorf("vcluster %s is not a statefulset or deployment", vCluster.Name)
		}
		for _, volume := range volumesToCheck {
			if volume.Name == "vcluster-config" {
				if volume.Secret == nil {
					return nil, fmt.Errorf("vCluster %s does not have a volume vcluster-config with Secret as a source", vCluster.Name)
				}
				configSecretName = volume.Secret.SecretName
				break
			}
		}
		configSecret, err := kubeClient.CoreV1().Secrets(vCluster.Namespace).Get(ctx, configSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get vCluster config secret: %w", err)
		}
		configBytes := configSecret.Data["config.yaml"]
		if configBytes == nil {
			return nil, fmt.Errorf("vCluster %s config secret does not have vCluster config set in 'config.yaml' data key", vCluster.Name)
		}
		vClusterConfig, err = vclusterconfig.ParseConfigBytes(configBytes, vCluster.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vcluster config: %w", err)
		}
	}

	return vClusterConfig, nil
}
