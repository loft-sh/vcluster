package upload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/osimage"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// OSImageCmd holds the cmd flags
type OSImageCmd struct {
	*flags.GlobalFlags

	File string

	SkipWait bool

	Log log.Logger
}

// newOSImageCmd creates a new command
func newOSImageCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &OSImageCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("upload osimage", `
Uploads a file to an existing OS image, which holds its bytes
in the image store the OS image names.
Example:
vcluster platform upload osimage ubuntu-24.04 --file ubuntu.qcow2
########################################################
	`)

	useLine, validator := util.NamedPositionalArgsValidator(true, true, "OSIMAGE_NAME")
	c := &cobra.Command{
		Use:   "osimage" + useLine,
		Short: "Uploads a file to an existing OS image",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.File, "file", "", "The local image file to upload")
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the image is ready")
	_ = c.MarkFlagRequired("file")

	return c
}

// Run executes the command
func (cmd *OSImageCmd) Run(ctx context.Context, args []string) error {
	imageName := args[0]

	file, err := os.Open(cmd.File)
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("read image file: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("image file %s is empty", cmd.File)
	}

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// Nothing about the image is checked here. The upload subresource applies every
	// precondition, including ones a client cannot see, and the first call reports them.
	uploader := osimage.NewUploader(managementClient.Loft().ManagementV1().OSImages(), cmd.Log)
	if err := uploader.Upload(ctx, imageName, file, info.Size()); err != nil {
		return missingImageHint(err, imageName)
	}

	// Hashed only once the parts are up, so a rejected upload does not cost a full read of a
	// multi gigabyte file first.
	cmd.Log.Infof("Calculating checksum of %s...", cmd.File)
	checksum, err := fileChecksum(file)
	if err != nil {
		return err
	}

	if err := uploader.Finalize(ctx, imageName, checksum); err != nil {
		return err
	}

	if cmd.SkipWait {
		cmd.Log.Donef("Successfully uploaded os image %s, the platform is verifying it", imageName)
		return nil
	}

	cmd.Log.Infof("Waiting for os image %s to become ready...", imageName)
	if err := uploader.WaitForReady(ctx, imageName, clihelper.Timeout()); err != nil {
		return err
	}

	cmd.Log.Donef("Successfully uploaded os image %s", imageName)
	return nil
}

// missingImageHint names the way out of a not found, which is the one thing the platform
// cannot say: the CLI creates no images, so the image comes from the UI or kubectl.
func missingImageHint(err error, imageName string) error {
	if !kerrors.IsNotFound(err) {
		return err
	}
	return fmt.Errorf("%w, create it in the platform UI or with kubectl before uploading to it", err)
}

func fileChecksum(file io.ReadSeeker) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("calculate checksum: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("calculate checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
