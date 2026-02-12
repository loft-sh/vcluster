package azure

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

const (
	// StorageKeyEnvVar is the name of the environment variable that contains the Azure storage account access key that
	// can authenticate vCluster's requests to the storage account
	StorageKeyEnvVar = "AZURE_STORAGE_KEY"

	// StorageBlobSASEnvVar is the name of the environment variable that contains the Azure storage blob SAS token that
	// can be used to authenticate vCluster's requests to the storage blob.
	StorageBlobSASEnvVar = "AZURE_STORAGE_BLOB_SAS"
)

type Options struct {
	BlobURL        string `json:"blob-url,omitempty"`
	SAS            string `json:"sas,omitempty"`
	SubscriptionID string `json:"subscription-id,omitempty"`
	ResourceGroup  string `json:"resource-group,omitempty"`
}

func (o *Options) GetSubscriptionID() string {
	if o.SubscriptionID != "" {
		return o.SubscriptionID
	}
	if subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subscriptionID != "" {
		return subscriptionID
	}
	return ""
}

func (o *Options) GetResourceGroup() string {
	if o.ResourceGroup != "" {
		return o.ResourceGroup
	}
	if resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP"); resourceGroup != "" {
		return resourceGroup
	}
	return ""
}

func (o *Options) FillCredentials(ctx context.Context) error {
	if blobURLContainsSAS(o.BlobURL) {
		return nil
	}

	sasToken, err := getStorageSAS(ctx, *o)
	if err != nil {
		return fmt.Errorf("failed to get SAS token: %w", err)
	}

	o.SAS = sasToken
	return nil
}

// GetBlobURLWithSAS returns the blob URL with SAS token appended
func (o *Options) GetBlobURLWithSAS() string {
	if o.SAS == "" {
		return o.BlobURL
	}
	return o.BlobURL + "?" + o.SAS
}

// blobURLContainsSAS returns true if the given blob URL contains the storage account SAS token
func blobURLContainsSAS(blobURL string) bool {
	if blobURL == "" {
		return false
	}

	parsedURL, err := url.Parse(blobURL)
	if err != nil {
		return false
	}

	// Check for common SAS token parameters
	// Azure SAS tokens typically include 'sig' (signature) parameter
	queryParams := parsedURL.Query()
	return queryParams.Has("sig") || strings.Contains(parsedURL.RawQuery, "sig=")
}

// getStorageSAS creates and returns a SAS token for the given blob URL.
// Returns only the SAS token query string (without the leading "?").
//
// This is equivalent to running Azure CLI command:
//
//	az storage blob generate-sas \
//	  --account-name "$STORAGE_ACCOUNT" \
//	  --container-name "$CONTAINER" \
//	  --name "$BLOB_NAME" \
//	  --https-only \
//	  --permissions "cw" \
//	  --start "$START" \
//	  --expiry "$EXPIRY" \
//	  --account-key "$AZURE_STORAGE_KEY" \
//	  -o tsv
func getStorageSAS(ctx context.Context, options Options) (string, error) {
	if sasToken := os.Getenv(StorageBlobSASEnvVar); sasToken != "" {
		return sasToken, nil
	}

	// Get all required blob information
	blobInfo, err := getBlobInfo(ctx, options)
	if err != nil {
		return "", fmt.Errorf("failed to get blob info: %w", err)
	}

	// Create shared key credential
	credential, err := blob.NewSharedKeyCredential(blobInfo.storageAccountName, blobInfo.storageKey)
	if err != nil {
		return "", fmt.Errorf("failed to create shared key credential: %w", err)
	}

	// Set start time to 5 minutes in the past to account for clock skew
	startTime := time.Now().UTC().Add(-5 * time.Minute)
	// Set expiry time to 1 hour from now
	expiryTime := time.Now().UTC().Add(5 * time.Minute)

	// Create BlobSignatureValues with SAS parameters
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,                                                                       // --https-only
		StartTime:     startTime,                                                                               // --start
		ExpiryTime:    expiryTime,                                                                              // --expiry
		Permissions:   to.Ptr(sas.BlobPermissions{Create: true, Write: true, Read: true, List: true}).String(), // --permissions "cw"
		ContainerName: blobInfo.containerName,                                                                  // --container-name
		BlobName:      blobInfo.blobName,                                                                       // --name
	}.SignWithSharedKey(credential)
	if err != nil {
		return "", fmt.Errorf("failed to sign SAS token: %w", err)
	}

	// Return the SAS token query string (without the leading "?")
	return sasQueryParams.Encode(), nil
}

type blobInfo struct {
	storageAccountName string
	storageKey         string
	containerName      string
	blobName           string
}

// getBlobInfo extracts all blob information from the blob URL and retrieves the storage key.
// It parses the blob URL to extract storage account name, container name, blob name,
// and retrieves the storage key from the environment variable or Azure API.
func getBlobInfo(ctx context.Context, options Options) (blobInfo, error) {
	if options.BlobURL == "" {
		return blobInfo{}, fmt.Errorf("blob URL is empty")
	}

	parsedURL, err := url.Parse(options.BlobURL)
	if err != nil {
		return blobInfo{}, fmt.Errorf("failed to parse blob URL: %w", err)
	}

	// Extract the storage account name from the hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return blobInfo{}, fmt.Errorf("invalid blob URL: missing hostname")
	}
	hostParts := strings.Split(hostname, ".")
	if len(hostParts) < 1 || hostParts[0] == "" {
		return blobInfo{}, fmt.Errorf("invalid blob URL format: cannot extract storage account name from %s", hostname)
	}
	storageAccountName := hostParts[0]

	// Extract the blob container name and the blob name from the path
	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return blobInfo{}, fmt.Errorf("invalid blob URL: missing path")
	}
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return blobInfo{}, fmt.Errorf("invalid blob URL format: path must contain container and blob name")
	}
	containerName := pathParts[0]
	blobName := strings.Join(pathParts[1:], "/")

	// Get the storage key from the environment variable, if set in env, or from Azure API
	var storageKey string
	if key := os.Getenv(StorageKeyEnvVar); key != "" {
		storageKey = key
	} else {
		storageKey, err = getStorageKeyFromAzure(ctx, options.GetSubscriptionID(), options.GetResourceGroup(), storageAccountName)
		if err != nil {
			return blobInfo{}, fmt.Errorf("failed to get storage key from Azure: %w", err)
		}
	}

	return blobInfo{
		storageAccountName: storageAccountName,
		storageKey:         storageKey,
		containerName:      containerName,
		blobName:           blobName,
	}, nil
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
		return "", fmt.Errorf("subscription ID is required")
	}
	if resourceGroup == "" {
		return "", fmt.Errorf("resource group is required")
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
	if resp.Keys == nil || len(resp.Keys) == 0 {
		return "", fmt.Errorf("no keys found for storage account %s", storageAccount)
	}

	if resp.Keys[0].Value == nil {
		return "", fmt.Errorf("key value is nil for storage account %s", storageAccount)
	}

	return *resp.Keys[0].Value, nil
}

// createAzureStorageAccountsClient creates an Azure storage accounts client using Azure CLI credentials
func createAzureStorageAccountsClient(subscriptionID string) (*armstorage.AccountsClient, error) {
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
