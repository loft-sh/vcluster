package azure

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type BlobInfo struct {
	ContainerName string
	BlobName      string
	AccountURL    string
}

func GetBlobInfo(blobURL string) (BlobInfo, error) {
	if blobURL == "" {
		return BlobInfo{}, fmt.Errorf("blob URL is empty")
	}
	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return BlobInfo{}, fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Extract storage account URL (scheme + host)
	accountURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Extract container and blob name from the URL path (format: /container/blob/path)
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}

	containerName := pathParts[0]
	blobName := strings.Join(pathParts[1:], "/")

	return BlobInfo{
		ContainerName: containerName,
		BlobName:      blobName,
		AccountURL:    accountURL,
	}, nil
}

// NewBlobClient creates an Azure blob client from BlobInfo and a full blob URL with SAS token
func NewBlobClient(info BlobInfo, blobURL string) (*blockblob.Client, error) {
	// Create the block blob client with SAS token (no credentials needed, token is in URL)
	blobClient, err := blockblob.NewClientWithNoCredential(blobURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	return blobClient, nil
}

// NewContainerClient creates a container client from blob URL with SAS token
func NewContainerClient(blobURL string) (*container.Client, string, error) {
	info, err := GetBlobInfo(blobURL)
	if err != nil {
		return nil, "", err
	}
	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Build container URL with SAS token
	containerURL := fmt.Sprintf("%s/%s?%s", info.AccountURL, info.ContainerName, parsedURL.RawQuery)

	containerClient, err := container.NewClientWithNoCredential(containerURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create container client: %w", err)
	}

	return containerClient, info.BlobName, nil
}
