package azure

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type BlobInfo struct {
	ContainerName string
	BlobName      string
	AccountName   string
}

func GetBlobInfo(blobURL string) (BlobInfo, error) {
	if blobURL == "" {
		return BlobInfo{}, fmt.Errorf("blob URL is empty")
	}
	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return BlobInfo{}, fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Extract the storage account name from host (format: {account}.blob.core.windows.net)
	hostParts := strings.Split(parsedURL.Host, ".")
	if len(hostParts) < 1 {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}
	accountName := hostParts[0]

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
		AccountName:   accountName,
	}, nil
}

// NewBlobClient creates an Azure blob client from BlobInfo and a full blob URL with SAS token
func NewBlobClient(ctx context.Context, subscriptionID, resourceGroup string, info BlobInfo, blobURL string, useDefaultCredentials bool) (*blockblob.Client, error) {
	var blobClient *blockblob.Client
	var err error
	if useDefaultCredentials {
		// TODO use default Azure credentials to create blob client
		// Use default Azure credentials for authentication. From Azure SDK go docs:
		//
		// DefaultAzureCredential attempts to authenticate with each of these credential types, in the following order,
		// stopping when one provides a token:
		//    1. EnvironmentCredential
		//    2. WorkloadIdentityCredential, if environment variable configuration is set by the Azure workload identity webhook.
		//    3. ManagedIdentityCredential
		//    4. AzureCLICredential
		//    5. AzureDeveloperCLICredential
		//    6. AzurePowerShellCredential
		//
		// More details in go docs here https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential.
		//
		// TODO: if using default Azure credentials does not work here, then try share key approach that requires using storage key
		//
		var storageKey string
		if key := os.Getenv(StorageKeyEnvVar); key != "" {
			storageKey = key
		} else {
			var defaultCredential *azidentity.DefaultAzureCredential
			defaultCredential, err = azidentity.NewDefaultAzureCredential(nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create default Azure credential: %w", err)
			}
			// create the storage account client to get the shared key
			clientFactory, err := armstorage.NewClientFactory(subscriptionID, defaultCredential, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create client factory: %w", err)
			}
			storageAccountClient := clientFactory.NewAccountsClient()
			// List storage account keys
			resp, err := storageAccountClient.ListKeys(ctx, resourceGroup, info.AccountName, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to list storage account keys: %w", err)
			}

			// Return the first key (equivalent to [0].value in Azure CLI)
			if resp.Keys == nil || len(resp.Keys) == 0 {
				return nil, fmt.Errorf("no keys found for storage account %s", info.AccountName)
			}

			if resp.Keys[0].Value == nil {
				return nil, fmt.Errorf("key value is nil for storage account %s", info.AccountName)
			}

			storageKey = *resp.Keys[0].Value
		}

		// Create shared key credential
		sharedKeyCredential, err := blob.NewSharedKeyCredential(info.AccountName, storageKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		// blobClient, err = blockblob.NewClient(blobURL, defaultCredential, nil)
		blobClient, err = blockblob.NewClientWithSharedKeyCredential(blobURL, sharedKeyCredential, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with default Azure credentials: %w", err)
		}
	} else {
		// Create the block blob client with SAS token (no credentials needed, token is in URL)
		blobClient, err = blockblob.NewClientWithNoCredential(blobURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with SAS token: %w", err)
		}
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

	// Build container URL with SAS token (reconstruct account URL from account name)
	accountURL := fmt.Sprintf("%s://%s.blob.core.windows.net", parsedURL.Scheme, info.AccountName)
	containerURL := fmt.Sprintf("%s/%s?%s", accountURL, info.ContainerName, parsedURL.RawQuery)

	containerClient, err := container.NewClientWithNoCredential(containerURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create container client: %w", err)
	}

	return containerClient, info.BlobName, nil
}
