package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	remotev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	orasremote "oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	configRef = "config"
	etcdRef   = "etcd"
)

func Push(
	ctx context.Context,
	vClusterConfig *VClusterConfig,
	etcdSnapshot string,
	registry,
	repository,
	tag string,
	username string,
	password string,
) error {
	// create a file store
	fs, err := file.New("/tmp/")
	if err != nil {
		return err
	}
	defer fs.Close()

	// create descriptor array
	descriptors := []v1.Descriptor{}

	// vCluster config
	vClusterConfigFile, err := writeJSONFile(vClusterConfig)
	if err != nil {
		return err
	}
	defer os.Remove(vClusterConfigFile)
	vClusterConfigDescriptor, err := fs.Add(ctx, configRef, ConfigMediaType, vClusterConfigFile)
	if err != nil {
		return fmt.Errorf("add config %s to image", vClusterConfigFile)
	}
	descriptors = append(descriptors, vClusterConfigDescriptor)

	// etcd
	etcdDescriptor, err := fs.Add(ctx, etcdRef, EtcdLayerMediaType, etcdSnapshot)
	if err != nil {
		return fmt.Errorf("add etcd snapshot %s to image", etcdSnapshot)
	}
	descriptors = append(descriptors, etcdDescriptor)

	// pack the files and tag the packed manifest
	manifestDescriptor, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1_RC4, ArtifactType, oras.PackManifestOptions{
		Layers: descriptors,
	})
	if err != nil {
		return fmt.Errorf("pack vCluster: %w", err)
	}

	// tag the image
	if tag != "" {
		if err = fs.Tag(ctx, manifestDescriptor, tag); err != nil {
			return fmt.Errorf("tag vCluster: %w", err)
		}
	}

	// create client
	repo, err := createClient(registry, repository, username, password)
	if err != nil {
		return err
	}

	// copy from the file store to the remote repository
	_, err = oras.Copy(ctx, fs, tag, repo, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("push vCluster image: %w", err)
	}

	return nil
}

func Pull(
	ctx context.Context,
	target string,
	username string,
	password string,
) (io.ReadCloser, error) {
	ref, err := name.ParseReference(target)
	if err != nil {
		return nil, err
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuth(&authn.Basic{
		Username: username,
		Password: password,
	}))
	if err != nil {
		return nil, err
	}

	etcdReader, err := FindLayerWithMediaType(img, EtcdLayerMediaType)
	if err != nil {
		return nil, err
	}

	return etcdReader, nil
}

func FindLayerWithMediaType(img remotev1.Image, mediaType string) (io.ReadCloser, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	// search config layer
	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			return nil, fmt.Errorf("get layer: %w", err)
		}

		// is config layer?
		if mediaType == string(mt) {
			reader, err := layer.Uncompressed()
			if err != nil {
				return nil, fmt.Errorf("read config layer: %w", err)
			}

			return reader, nil
		}
	}

	return nil, fmt.Errorf("couldn't find layer with type %s", mediaType)
}

func createClient(registry, repository, username, password string) (*orasremote.Repository, error) {
	// connect to a remote repository
	repo, err := orasremote.NewRepository(registry + "/" + repository)
	if err != nil {
		return nil, fmt.Errorf("create repository")
	}

	// Note: The below code can be omitted if authentication is not required
	authClient := &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.DefaultCache,
	}
	if username != "" {
		authClient.Credential = auth.StaticCredential(registry, auth.Credential{
			Username: username,
			Password: password,
		})
	}
	repo.Client = authClient
	return repo, nil
}

func writeJSONFile(data interface{}) (string, error) {
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}

	err = json.NewEncoder(tempFile).Encode(data)
	if err != nil {
		_ = os.Remove(tempFile.Name())
		return "", err
	}

	_ = tempFile.Close()
	return tempFile.Name(), nil
}
