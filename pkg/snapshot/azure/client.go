package azure

import (
	"context"
	"errors"
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

var (
	// ErrSubscriptionIDNotSet is returned when the Azure subscription ID is not set
	ErrSubscriptionIDNotSet = fmt.Errorf("the Azure subscription ID is required, set the AZURE_SUBSCRIPTION_ID environment variable, or set the --azure-subscription-id flag if you're running vcluster CLI")

	// ErrResourceGroupNotSet is returned when the Azure resource group is not set
	ErrResourceGroupNotSet = fmt.Errorf("the Azure resource group is required, set the AZURE_RESOURCE_GROUP environment variable, or set the --azure-resource-group flag if you're running vcluster CLI")
)

// IsAzureFlagNotSetError returns true if the error is caused by a missing Azure flag
func IsAzureFlagNotSetError(err error) bool {
	return errors.Is(err, ErrSubscriptionIDNotSet) || errors.Is(err, ErrResourceGroupNotSet)
}

type BlobInfo struct {
	ContainerName string
	BlobName      string
	AccountName   string
}

func getBlobInfo(blobURL string) (BlobInfo, error) {
	if blobURL == "" {
		return BlobInfo{}, fmt.Errorf("blob URL is empty")
	}
	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return BlobInfo{}, fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Extract the storage account name from host (format: {account}.blob.core.windows.net)
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}
	hostParts := strings.Split(hostname, ".")
	if len(hostParts) < 1 || hostParts[0] == "" {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}
	accountName := hostParts[0]

	// Extract container and blob name from the URL path (format: /container/blob/path)
	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}
	pathParts := strings.SplitN(path, "/", 2)
	if len(pathParts) < 2 {
		return BlobInfo{}, fmt.Errorf("invalid blob URL format, expected: https://{account}.blob.core.windows.net/{container}/{blob}")
	}

	containerName := pathParts[0]
	blobName := pathParts[1]

	return BlobInfo{
		ContainerName: containerName,
		BlobName:      blobName,
		AccountName:   accountName,
	}, nil
}

// newBlobClient creates an Azure blob client from BlobInfo and a full blob URL with SAS token
func newBlobClient(ctx context.Context, subscriptionID, resourceGroup string, info BlobInfo, blobURL string, useSASTokenFromBlobURL bool) (*blockblob.Client, error) {
	var blobClient *blockblob.Client
	var err error
	if useSASTokenFromBlobURL {
		// Create the block blob client with SAS token (no credentials needed, token is in URL)
		blobClient, err = blockblob.NewClientWithNoCredential(blobURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob client with SAS token: %w", err)
		}
	} else {
		var storageKey string
		if key := os.Getenv(StorageKeyEnvVar); key != "" {
			//
			// Create a blob client with the specified storage key
			//
			storageKey = key
		} else {
			storageKey, err = getStorageKeyFromAzure(ctx, subscriptionID, resourceGroup, info.AccountName)
			if err != nil {
				return nil, fmt.Errorf("failed to get storage key from Azure: %w", err)
			}
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
	}

	return blobClient, nil
}

// newContainerClient creates a container client from blob URL with SAS token
func newContainerClient(blobURL string) (*container.Client, string, error) {
	info, err := getBlobInfo(blobURL)
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

// createAzureStorageAccountsClient creates an Azure storage accounts client using Azure CLI credentials
func createAzureStorageAccountsClient(subscriptionID string) (*armstorage.AccountsClient, error) {
	if subscriptionID == "" {
		return nil, ErrSubscriptionIDNotSet
	}
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
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure CLI credential (make sure you're logged in with 'az login'): %w", err)
	}
	clientFactory, err := armstorage.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create client factory: %w", err)
	}

	return clientFactory.NewAccountsClient(), nil
}

// getStorageKeyFromAzure gets the storage account access key by re-using your existing Azure CLI login.
//
// This is equivalent to running:
//
//	az storage account keys list \
//	  --resource-group "$RG" \
//	  --account-name "$SA" \
//	  --query '[0].value' \
//	  -o tsv
func getStorageKeyFromAzure(ctx context.Context, subscriptionID, resourceGroup, storageAccount string) (string, error) {
	if subscriptionID == "" {
		return "", ErrSubscriptionIDNotSet
	}
	if resourceGroup == "" {
		return "", ErrResourceGroupNotSet
	}
	if storageAccount == "" {
		return "", fmt.Errorf("storage account name is required")
	}

	// Create Azure storage accounts client
	client, err := createAzureStorageAccountsClient(subscriptionID)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure storage client: %w", err)
	}

	// List storage account keys
	resp, err := client.ListKeys(ctx, resourceGroup, storageAccount, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list storage account keys: %w", err)
	}

	// Return the first key (equivalent to [0].value in Azure CLI)
	if len(resp.Keys) == 0 {
		return "", fmt.Errorf("no keys found for storage account %s", storageAccount)
	}

	if resp.Keys[0].Value == nil {
		return "", fmt.Errorf("key value is nil for storage account %s", storageAccount)
	}

	return *resp.Keys[0].Value, nil
}
