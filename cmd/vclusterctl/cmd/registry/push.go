package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/util/archive"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PushOptions struct {
	*flags.GlobalFlags

	Architecture string

	NoDocker bool

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
		Short: "Push a local docker image or archive into vCluster registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&o.Architecture, "architecture", runtime.GOARCH, "Architecture of the image. E.g. amd64, arm64, etc. Only valid if used together with an image argument. E.g. vcluster registry push nginx --architecture amd64")
	cmd.Flags().BoolVar(&o.NoDocker, "no-docker", false, "If true, the images will be downloaded and then pushed to the vCluster registry. This is useful if you haven't installed docker or if you want to push images from a different architecture.")
	cmd.Flags().StringSliceVar(&o.Archives, "archive", []string{}, "Path to the archive.tar or archive.tar.gz file. Can also be a directory with .tar or .tar.gz files.")
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
	restConfig, err := getClient(o.GlobalFlags)
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}

	// get the transport
	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return fmt.Errorf("failed to get transport: %w", err)
	}

	// check if registry is enabled
	registryEnabled, err := isRegistryEnabled(ctx, restConfig.Host, transport)
	if err != nil {
		return fmt.Errorf("failed to check if registry is enabled: %w", err)
	} else if !registryEnabled {
		return fmt.Errorf("vCluster registry is not enabled or the target cluster is not a vCluster. Please make sure to enable the registry in the vCluster config and run `vcluster connect` to connect to the vCluster before pushing images")
	}

	// push helm charts
	if len(o.HelmCharts) > 0 {
		if err := o.pushHelmCharts(ctx, restConfig); err != nil {
			return fmt.Errorf("failed to push helm charts: %w", err)
		}

		return nil
	}

	// get the vCluster host
	vClusterHost, err := url.Parse(restConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to parse vCluster host: %w", err)
	}

	// get the remote registry
	remoteRegistry, err := name.NewRegistry(vClusterHost.Host)
	if err != nil {
		return fmt.Errorf("failed to get remote registry: %w", err)
	}

	// push images
	if len(o.Images) > 0 {
		// check if docker is installed
		if _, err := exec.LookPath("docker"); err != nil {
			o.Log.Infof("Docker is not installed, pushing images to vCluster registry directly without docker save...")
			o.NoDocker = true
		}

		// push images without docker save
		if o.NoDocker {
			// push images directly to vCluster registry
			if err := o.pushImages(ctx, remoteRegistry, transport, o.Images); err != nil {
				return fmt.Errorf("failed to push images: %w", err)
			}
		} else {
			// save images to archive
			o.Log.Infof("Saving image(s) %s to archive...", strings.Join(o.Images, ", "))
			args := []string{"docker", "save", "-o", "image.tar"}
			args = append(args, o.Images...)
			if err := runCommand(ctx, args...); err != nil {
				return fmt.Errorf("failed to save image: %w", err)
			}

			// cleanup
			defer os.Remove("image.tar")
			o.Archives = append(o.Archives, "image.tar")
		}
	}

	// push archives
	for _, archive := range o.Archives {
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
				if !strings.HasSuffix(file.Name(), ".tar") && !strings.HasSuffix(file.Name(), ".tar.gz") {
					continue
				}

				if err := o.pushArchive(ctx, filepath.Join(archive, file.Name()), remoteRegistry, transport); err != nil {
					return err
				}
			}
		} else if err := o.pushArchive(ctx, archive, remoteRegistry, transport); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushImages(ctx context.Context, remoteRegistry name.Registry, transport http.RoundTripper, images []string) error {
	puller, err := remote.NewPuller(remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithPlatform(v1.Platform{Architecture: o.Architecture, OS: "linux"}))
	if err != nil {
		return fmt.Errorf("failed to create puller: %w", err)
	}

	for _, image := range images {
		localRef, err := name.ParseReference(image)
		if err != nil {
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		descriptor, err := puller.Get(ctx, localRef)
		if err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}

		image, err := descriptor.Image()
		if err != nil {
			return fmt.Errorf("failed to get image: %w", err)
		}

		if err := o.pushImage(ctx, localRef, image, remoteRegistry, transport); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushHelmCharts(ctx context.Context, restConfig *rest.Config) error {
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

				if err := o.pushHelmChart(ctx, filepath.Join(helmChart, file.Name()), restConfig); err != nil {
					return err
				}
			}
		} else if err := o.pushHelmChart(ctx, helmChart, restConfig); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushHelmChart(ctx context.Context, helmChart string, restConfig *rest.Config) error {
	// TODO: right now we only support certificate based authentication for pushing helm charts. Which will not work
	// when using platform based authentication.

	keyFile := restConfig.KeyFile
	if keyFile == "" {
		if len(restConfig.KeyData) == 0 {
			return fmt.Errorf("no key data found in client config. Helm cannot use token based authentication for pushing")
		}

		tempFile, err := os.CreateTemp("", "vcluster-helm-chart-key")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())

		if _, err := tempFile.Write(restConfig.KeyData); err != nil {
			return fmt.Errorf("failed to write key data: %w", err)
		}

		keyFile = tempFile.Name()
	}

	certFile := restConfig.CertFile
	if certFile == "" {
		if len(restConfig.CertData) == 0 {
			return fmt.Errorf("no cert data found in client config. Helm cannot use token based authentication for pushing")
		}

		tempFile, err := os.CreateTemp("", "vcluster-helm-chart-cert")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())

		if _, err := tempFile.Write(restConfig.CertData); err != nil {
			return fmt.Errorf("failed to write cert data: %w", err)
		}

		certFile = tempFile.Name()
	}

	endpoint, err := url.Parse(restConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}

	remoteRef := fmt.Sprintf("oci://%s/%s", endpoint.Host, o.HelmChartRepository)
	o.Log.Infof("Pushing helm chart %s to %s", helmChart, remoteRef)
	err = runCommand(
		ctx,
		"helm",
		"push",
		"--insecure-skip-tls-verify",
		"--cert-file", certFile,
		"--key-file", keyFile,
		helmChart,
		remoteRef,
	)
	if err != nil {
		return fmt.Errorf("failed to push helm chart %s: %w", helmChart, err)
	}

	return nil
}

func isRegistryEnabled(ctx context.Context, host string, transport http.RoundTripper) (bool, error) {
	client := &http.Client{Transport: transport}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v2/", host), nil)
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

func (o *PushOptions) pushArchive(ctx context.Context, path string, remoteRegistry name.Registry, transport http.RoundTripper) error {
	// create a temp directory to extract the archive to
	tempDir, err := os.MkdirTemp("", "vcluster-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// extract the archive to the temp directory
	if strings.HasSuffix(path, ".tar.gz") {
		if err := archive.ExtractTarGz(path, tempDir); err != nil {
			return err
		}
	} else if strings.HasSuffix(path, ".tar") {
		if err := archive.ExtractTar(path, tempDir); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported archive format: %s, must be .tar or .tar.gz", path)
	}

	// get the image index from the temp directory
	imageIndex, err := layout.ImageIndexFromPath(tempDir)
	if err != nil {
		return err
	}

	manifest, err := imageIndex.IndexManifest()
	if err != nil {
		return err
	}

	for _, manifest := range manifest.Manifests {
		imageTag := manifest.Annotations["io.containerd.image.name"]
		if imageTag == "" {
			return fmt.Errorf("image tag not found in manifest: %s, annotations: %v", path, manifest.Annotations)
		}

		localRef, err := name.ParseReference(imageTag)
		if err != nil {
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		image, err := imageIndex.Image(manifest.Digest)
		if err != nil {
			return fmt.Errorf("failed to get image: %w", err)
		}

		if err := o.pushImage(ctx, localRef, image, remoteRegistry, transport); err != nil {
			return err
		}
	}

	return nil
}

func (o *PushOptions) pushImage(ctx context.Context, localRef name.Reference, image v1.Image, remoteRegistry name.Registry, transport http.RoundTripper) error {
	remoteRef, err := replaceRegistry(localRef, remoteRegistry)
	if err != nil {
		return err
	}

	o.Log.Infof("Pushing image %s to %s", localRef.String(), remoteRef.String())
	progressChan := make(chan v1.Update, 200)
	errChan := make(chan error, 1)

	// push image to remote registry
	go func() {
		errChan <- remote.Push(
			remoteRef,
			image,
			remote.WithContext(ctx),
			remote.WithProgress(progressChan),
			remote.WithTransport(transport),
		)
	}()

	for update := range progressChan {
		if update.Error != nil {
			return update.Error
		}

		status := "Pushing"
		if update.Complete == update.Total {
			status = "Pushed"
		}

		_, err := fmt.Fprintf(os.Stdout, "%s %s\n", status, (&jsonmessage.JSONProgress{
			Current: update.Complete,
			Total:   update.Total,
		}).String())
		if err != nil {
			return err
		}
	}

	err = <-errChan
	if err != nil {
		return err
	}

	o.Log.Infof("Successfully pushed image %s", remoteRef.String())
	return nil
}

func replaceRegistry(localRef name.Reference, remoteRegistry name.Registry) (name.Reference, error) {
	localTag, err := name.NewTag(localRef.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse tag: %w", err)
	}
	remoteTag := localTag
	remoteTag.Repository.Registry = remoteRegistry
	remoteRef, err := name.ParseReference(remoteTag.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote reference: %w", err)
	}

	return remoteRef, nil
}

func getClient(flags *flags.GlobalFlags) (*rest.Config, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})

	// get the client config
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return restConfig, nil
}
