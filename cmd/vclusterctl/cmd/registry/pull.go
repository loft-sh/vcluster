package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/image/copy"
	"github.com/loft-sh/image/transports/alltransports"
	"github.com/loft-sh/image/types"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

type PullOptions struct {
	*flags.GlobalFlags

	Destination string

	Architecture string

	Log log.Logger
}

func NewPullCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	o := &PullOptions{
		GlobalFlags: globalFlags,

		Log: log.GetInstance(),
	}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull a docker image from a container registry and save it to a tarball that can be imported into the vCluster registry",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run(cmd.Context(), args)
		},
	}

	cmd.Flags().StringVar(&o.Destination, "destination", "", "Path to the destination directory. If not specified, the images will be pulled to the current directory.")
	cmd.Flags().StringVar(&o.Architecture, "architecture", runtime.GOARCH, "Architecture of the image to pull. If not specified, the architecture will be detected automatically. Use 'all' to pull all architectures.")

	return cmd
}

func (o *PullOptions) Run(ctx context.Context, images []string) error {
	for _, image := range images {
		if err := o.pullImage(ctx, image); err != nil {
			return fmt.Errorf("failed to pull image %s: %w", image, err)
		}
	}

	return nil
}

func (o *PullOptions) pullImage(ctx context.Context, image string) error {
	srcContext := &types.SystemContext{
		OSChoice:                    "linux",
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}
	destContext := &types.SystemContext{
		OSChoice:                    "linux",
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
	}

	// get the source reference
	srcRef, err := alltransports.ParseImageName("docker://" + image)
	if err != nil {
		return fmt.Errorf("invalid source name %s: %w", image, err)
	}

	// get the filename of the image
	fileName := strings.ReplaceAll(strings.ReplaceAll(srcRef.DockerReference().String(), "/", "_"), ":", "+") + ".tar"
	if o.Destination != "" {
		fileName = filepath.Join(o.Destination, fileName)
	}

	// check if the image is a digest
	isDigest := strings.Contains(image, "@")
	imageListSelection := copy.CopySystemImage
	if isDigest || o.Architecture == "all" {
		imageListSelection = copy.CopyAllImages
	} else {
		srcContext.ArchitectureChoice = o.Architecture
		destContext.ArchitectureChoice = o.Architecture
	}

	// this is always an oci-archive for now
	destRef, err := alltransports.ParseImageName(fmt.Sprintf("oci-archive:%s", fileName))
	if err != nil {
		return fmt.Errorf("invalid destination name %s: %w", fileName, err)
	}

	// copy the image
	o.Log.Infof("Pulling image %s to %s", srcRef.DockerReference().Name(), fileName)
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
