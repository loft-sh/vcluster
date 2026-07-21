package registry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/image/copy"
	"github.com/loft-sh/image/transports/alltransports"
	"github.com/loft-sh/image/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PushOptions struct {
	*flags.GlobalFlags

	Architecture string

	Images   []string
	Archives []string

	HelmCharts          []string
	HelmChartRepository string

	Log log.Logger
}

func NewPushCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	o := &PushOptions{
		GlobalFlags: globalFlags,

		Log: log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push a docker image, archive or helm chart into vCluster registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&o.Architecture, "architecture", runtime.GOARCH, "Architecture of the image. E.g. amd64, arm64, etc. Only valid if used together with an image argument. E.g. vcluster registry push nginx --architecture amd64. Use 'all' to push all architectures.")
	cmd.Flags().StringSliceVar(&o.Archives, "archive", []string{}, "Path to the archive.tar file. Can also be a directory with .tar files. Needs to have the format registry_repository+tag.tar")
	cmd.Flags().StringSliceVar(&o.HelmCharts, "helm-chart", []string{}, "Path to the helm chart. Can also be a directory with .tgz files.")
	cmd.Flags().StringVar(&o.HelmChartRepository, "helm-chart-repository", "charts", "Repository in the vCluster registry to push the helm chart to. E.g. charts will allow you to use the helm chart with oci://<vcluster-host>/charts/my-chart-name:version.")

	return cmd
}

func (o *PushOptions) Run(ctx context.Context, args []string) error {
	if len(args) > 0 {
		o.Images = args
	}

	// validate flags
	if len(o.Images) == 0 && len(o.Archives) == 0 && len(o.HelmCharts) == 0 {
		return fmt.Errorf("either image or --archive or --helm-chart is required")
	} else if (len(o.Images) > 0 || len(o.Archives) > 0) && len(o.HelmCharts) > 0 {
		return fmt.Errorf("cannot use --helm-chart with --image or --archive")
	}

	// get the client config
	restConfig, err := getConfig(ctx, o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}

	// start the reverse proxy
	localPort := clihelper.RandomPort()
	if err := startReverseProxy(restConfig, localPort, o.Log); err != nil {
		return fmt.Errorf("failed to start reverse proxy: %w", err)
	}

	// push helm charts
	if len(o.HelmCharts) > 0 {
		if err := o.pushHelmCharts(ctx, localPort); err != nil {
			return fmt.Errorf("failed to push helm charts: %w", err)
		}

		return nil
	}

	// push images
	if len(o.Images) > 0 {
		// push images directly to vCluster registry
		if err := o.pushImages(ctx, localPort, o.Images); err != nil {
			return fmt.Errorf("failed to push images: %w", err)
		}
	}

	// push archives
	if len(o.Archives) > 0 {
		if err := o.pushArchives(ctx, localPort, o.Archives); err != nil {
			return fmt.Errorf("failed to push archives: %w", err)
		}
	}

	return nil
}

func (o *PushOptions) pushImages(ctx context.Context, localPort int, images []string) error {
	for _, image := range images {
		srcRef, err := alltransports.ParseImageName("docker://" + image)
		if err != nil {
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		// push the image
		o.Log.Infof("Pushing %s to vCluster at %s", image, fmt.Sprintf("127.0.0.1:%d", localPort))
		if err := o.pushImage(ctx, srcRef, srcRef.DockerReference().String(), localPort); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushArchives(ctx context.Context, localPort int, archives []string) error {
	for _, archive := range archives {
		stat, err := os.Stat(archive)
		if err != nil {
			return fmt.Errorf("failed to stat archive: %w", err)
		}

		// if the archive is a directory, push all tar and tar.gz files in the directory
		if stat.IsDir() {
			files, err := os.ReadDir(archive)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			// push all tar and tar.gz files in the directory
			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".tar") {
					continue
				}

				if err := o.pushArchive(ctx, localPort, filepath.Join(archive, file.Name())); err != nil {
					return err
				}
			}
		} else if err := o.pushArchive(ctx, localPort, archive); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushArchive(ctx context.Context, localPort int, archive string) error {
	imageReference := filepath.Base(archive)
	imageReference = strings.TrimSuffix(imageReference, filepath.Ext(imageReference))
	imageReference = strings.ReplaceAll(imageReference, "_", "/")
	imageReference = strings.ReplaceAll(imageReference, "+", ":")

	// parse the source reference
	srcRef, err := alltransports.ParseImageName(fmt.Sprintf("oci-archive:%s", archive))
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	// push the image
	o.Log.Infof("Pushing %s to %s", archive, imageReference)
	return o.pushImage(ctx, srcRef, imageReference, localPort)
}

func (o *PushOptions) pushHelmCharts(ctx context.Context, localPort int) error {
	for _, helmChart := range o.HelmCharts {
		stat, err := os.Stat(helmChart)
		if err != nil {
			return fmt.Errorf("failed to stat helm chart: %w", err)
		}

		// if the helm chart is a directory, push all tgz files in the directory
		if stat.IsDir() {
			files, err := os.ReadDir(helmChart)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			// push all tgz files in the directory
			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".tgz") {
					continue
				}

				if err := o.pushHelmChart(ctx, filepath.Join(helmChart, file.Name()), localPort); err != nil {
					return err
				}
			}
		} else if err := o.pushHelmChart(ctx, helmChart, localPort); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushHelmChart(ctx context.Context, helmChart string, localPort int) error {
	remoteRef := fmt.Sprintf("oci://127.0.0.1:%d/%s", localPort, o.HelmChartRepository)
	o.Log.Infof("Pushing helm chart %s to %s", helmChart, remoteRef)
	err := runCommand(
		ctx,
		"helm",
		"push",
		"--plain-http",
		helmChart,
		remoteRef,
	)
	if err != nil {
		return fmt.Errorf("failed to push helm chart %s: %w", helmChart, err)
	}

	return nil
}

func (o *PushOptions) pushImage(ctx context.Context, srcRef types.ImageReference, destImageName string, localPort int) error {
	srcContext := &types.SystemContext{
		OSChoice:                    "linux",
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}
	destContext := &types.SystemContext{
		OSChoice:                    "linux",
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}

	// check if the image is a digest
	isDigest := strings.Contains(destImageName, "@")
	imageListSelection := copy.CopySystemImage
	if isDigest || o.Architecture == "all" {
		imageListSelection = copy.CopyAllImages
	} else {
		srcContext.ArchitectureChoice = o.Architecture
		destContext.ArchitectureChoice = o.Architecture
	}

	// replace the registry with the vCluster registry
	parts := strings.Split(destImageName, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid destImageName: %s", destImageName)
	}
	parts[0] = fmt.Sprintf("127.0.0.1:%d", localPort)
	destImageName = strings.Join(parts, "/")
	destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", destImageName))
	if err != nil {
		return fmt.Errorf("failed to parse destRef: %w", err)
	}

	// copy the image
	_, err = copy.Image(ctx, destRef, srcRef, &copy.Options{
		SourceCtx:      srcContext,
		DestinationCtx: destContext,

		PreserveDigests:    isDigest,
		ImageListSelection: imageListSelection,

		RemoveSignatures: true,

		ReportWriter: os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("failed to copy image: %w", err)
	}

	return nil
}

func isRegistryEnabled(ctx context.Context, restConfig *rest.Config) (bool, error) {
	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return false, fmt.Errorf("failed to get transport: %w", err)
	}

	client := &http.Client{Transport: transport}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v2/", restConfig.Host), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func runCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getConfig(ctx context.Context, flags *flags.GlobalFlags) (*rest.Config, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})

	// get the client config
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	// check if registry is enabled
	registryEnabled, err := isRegistryEnabled(ctx, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to check if registry is enabled: %w", err)
	} else if !registryEnabled {
		return nil, fmt.Errorf("vCluster registry is not enabled or the target cluster is not a vCluster. Please make sure to enable the registry in the vCluster config and run `vcluster connect` to connect to the vCluster before pushing images")
	}

	return restConfig, nil
}
