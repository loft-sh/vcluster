package oci

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/loft-sh/image/copy"
	"github.com/loft-sh/image/transports/alltransports"
	"github.com/loft-sh/image/types"
	"github.com/loft-sh/vcluster/config"
)

func PullImage(ctx context.Context, image, destination string, dockerAuthConfig *types.DockerAuthConfig) error {
	srcContext := &types.SystemContext{
		OSChoice:                    "linux",
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		DockerAuthConfig:            dockerAuthConfig,
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

	// check if the image is a digest
	srcContext.ArchitectureChoice = runtime.GOARCH
	destContext.ArchitectureChoice = runtime.GOARCH

	// this is always an oci-archive for now
	destRef, err := alltransports.ParseImageName(fmt.Sprintf("oci:%s", destination))
	if err != nil {
		return fmt.Errorf("invalid destination name %s: %w", destination, err)
	}

	// copy the image
	buffer := &bytes.Buffer{}
	_, err = copy.Image(ctx, destRef, srcRef, &copy.Options{
		SourceCtx:      srcContext,
		DestinationCtx: destContext,

		RemoveSignatures: true,
		ReportWriter:     buffer,
	})
	if err != nil {
		return fmt.Errorf("failed to copy image: %s - %w", buffer.String(), err)
	}

	return nil
}

func GetDockerAuthConfig(containerdConfig config.ContainerdJoin, registry string) *types.DockerAuthConfig {
	if containerdConfig.Registry.Auth != nil {
		registryParts := strings.Split(registry, "/")
		registryAuth := containerdConfig.Registry.Auth[registryParts[0]]
		if registryAuth.Username != "" {
			return &types.DockerAuthConfig{
				Username:      registryAuth.Username,
				Password:      registryAuth.Password,
				IdentityToken: registryAuth.IdentityToken,
			}
		}
	}

	return nil
}
