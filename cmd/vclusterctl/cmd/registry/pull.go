package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

type PullOptions struct {
	*flags.GlobalFlags

	Destination string

	Log log.Logger
}

func NewPullCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	o := &PullOptions{
		GlobalFlags: globalFlags,

		Log: log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull a local docker image or archive from vCluster registry",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&o.Destination, "destination", "", "Path to the destination directory. If not specified, the images will be pulled to the current directory.")

	return cmd
}

func (o *PullOptions) Run(ctx context.Context, images []string) error {
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
		return fmt.Errorf("vCluster registry is not enabled or the target cluster is not a vCluster. Please make sure to enable the registry in the vCluster config and run `vcluster connect` to connect to the vCluster before pulling images")
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

	// pull the images
	for _, image := range images {
		// parse the image reference
		localRef, err := name.ParseReference(image)
		if err != nil {
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		// pull the image
		if err := o.pullImage(ctx, localRef, remoteRegistry, transport); err != nil {
			return fmt.Errorf("failed to pull image %s: %w", localRef.String(), err)
		}
	}

	return nil
}

func (o *PullOptions) pullImage(ctx context.Context, localRef name.Reference, remoteRegistry name.Registry, transport http.RoundTripper) error {
	remoteRef, err := replaceRegistry(localRef, remoteRegistry)
	if err != nil {
		return err
	}

	// create the progress channel
	progressChan := make(chan v1.Update, 200)
	defer close(progressChan)

	// pull the image
	tarballPath := filepath.Join(o.Destination, refToFilename(localRef))
	o.Log.Infof("Pulling image %s from %s to %s", localRef.Name(), remoteRef.Name(), tarballPath)
	image, err := remote.Image(remoteRef, remote.WithContext(ctx), remote.WithProgress(progressChan), remote.WithTransport(transport))
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// create the tarball file
	tarballFile, err := os.Create(tarballPath)
	if err != nil {
		return fmt.Errorf("failed to create tarball file: %w", err)
	}
	defer tarballFile.Close()

	// write the progress to the console
	go func() {
		for update := range progressChan {
			if update.Error != nil {
				o.Log.Errorf("failed to pull image %s: %v", localRef.String(), update.Error)
				return
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
				o.Log.Errorf("failed to write progress to console: %v", err)
				return
			}
		}
	}()

	// write the image to the tarball file
	err = tarball.Write(localRef, image, tarballFile)
	if err != nil {
		return fmt.Errorf("failed to write image to tarball: %w", err)
	}

	o.Log.Infof("Successfully pulled image %s", localRef.String())
	return nil
}

func refToFilename(ref name.Reference) string {
	return strings.ReplaceAll(strings.ReplaceAll(ref.Name(), "/", "_"), ":", "_") + ".tar"
}
