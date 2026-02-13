package azure

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/go-logr/logr"
	"github.com/loft-sh/vcluster/pkg/snapshot/types"
)

type ObjectStore struct {
	log           logr.Logger
	blobClient    *blockblob.Client
	containerName string
	blobName      string
	accountURL    string
	blobURL       string
}

var _ types.Storage = &ObjectStore{}

func NewStore(ctx context.Context, options *Options, logger logr.Logger) (*ObjectStore, error) {
	objectStore := &ObjectStore{log: logger}
	err := objectStore.init(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to init Azure object store: %w", err)
	}

	return objectStore, nil
}

func (o *ObjectStore) init(ctx context.Context, options *Options) error {
	if options.BlobURL == "" {
		return fmt.Errorf("blob URL is required")
	}
	var useDefaultCredentials bool
	if !options.ContainsSAS() {
		useDefaultCredentials = true
	}

	// Get the blob URL with SAS token appended
	o.blobURL = options.GetBlobURLWithSAS()

	// Extract information from blob URL
	info, err := GetBlobInfo(o.blobURL)
	if err != nil {
		return fmt.Errorf("failed to get blob info: %w", err)
	}

	// Create the blob client
	o.blobClient, err = NewBlobClient(info, o.blobURL, useDefaultCredentials)
	if err != nil {
		return fmt.Errorf("failed to create blob client: %w", err)
	}

	o.containerName = info.ContainerName
	o.blobName = info.BlobName
	o.accountURL = info.AccountURL

	return nil
}

func (o *ObjectStore) Target() string {
	// Return URL without SAS token for display purposes
	return fmt.Sprintf("%s/%s/%s", o.accountURL, o.containerName, o.blobName)
}

func (o *ObjectStore) PutObject(ctx context.Context, body io.Reader) error {
	_, err := o.blobClient.UploadStream(ctx, body, nil)
	if err != nil {
		return fmt.Errorf("failed to upload blob %s: %w", o.blobName, err)
	}
	return nil
}

func (o *ObjectStore) GetObject(ctx context.Context) (io.ReadCloser, error) {
	resp, err := o.blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob %s: %w", o.blobName, err)
	}
	return resp.Body, nil
}
func (o *ObjectStore) List(ctx context.Context) ([]types.Snapshot, error) {
	// Create the container client for listing blobs
	containerClient, blobName, err := NewContainerClient(o.blobURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create container client: %w", err)
	}

	// Determine prefix for listing
	prefix := blobName
	if strings.HasSuffix(prefix, "tar.gz") {
		// Use the "parent dir" as the prefix if a file was given
		prefix = filepath.Dir(prefix)

		// Handle if the blob is at the root of the container
		if prefix == "." {
			prefix = ""
		}
	}

	// Add trailing slash if the prefix is not empty
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// List blobs with pagination
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	snapshots := make([]types.Snapshot, 0)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blobItem := range page.Segment.BlobItems {
			if blobItem.Name == nil || blobItem.Properties == nil || blobItem.Properties.LastModified == nil {
				continue
			}

			// Skip non *.tar.gz objects
			if !strings.HasSuffix(*blobItem.Name, "tar.gz") {
				continue
			}

			// Skip blobs not in the "current directory"
			id := strings.TrimPrefix(*blobItem.Name, prefix)
			if filepath.Dir(id) != "." {
				continue
			}

			// Build blob URL without SAS token
			blobURL := fmt.Sprintf("%s/%s/%s", o.accountURL, o.containerName, *blobItem.Name)

			snapshots = append(snapshots, types.Snapshot{
				ID:        id,
				URL:       blobURL,
				Timestamp: *blobItem.Properties.LastModified,
			})
		}
	}

	return snapshots, nil
}

func (o *ObjectStore) Delete(ctx context.Context) error {
	_, err := o.blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blob %s: %w", o.blobName, err)
	}
	return nil
}
