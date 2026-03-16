package azure

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/spf13/pflag"
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

func (o *Options) FillCredentials(ctx context.Context, tryToCreateSAS bool) error {
	if o.ContainsSAS() || !tryToCreateSAS {
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

func (o *Options) ContainsSAS() bool {
	return blobURLContainsSAS(o.BlobURL) || o.SAS != ""
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

	blobInfo, err := getBlobInfo(options.BlobURL)
	if err != nil {
		return "", fmt.Errorf("failed to get blob info: %w", err)
	}
	var storageKey string
	if key := os.Getenv(StorageKeyEnvVar); key != "" {
		storageKey = key
	} else {
		storageKey, err = getStorageKeyFromAzure(ctx, options.GetSubscriptionID(), options.GetResourceGroup(), blobInfo.AccountName)
		if err != nil {
			return "", fmt.Errorf("failed to get storage key from Azure: %w", err)
		}
	}

	// Create shared key credential
	credential, err := blob.NewSharedKeyCredential(blobInfo.AccountName, storageKey)
	if err != nil {
		return "", fmt.Errorf("failed to create shared key credential: %w", err)
	}

	// Set start time to 5 minutes in the past to account for clock skew
	startTime := time.Now().UTC().Add(-5 * time.Minute)
	// Set expiry time to 1 hour from now
	expiryTime := time.Now().UTC().Add(time.Hour)

	// Create BlobSignatureValues with SAS parameters
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,                                                                       // --https-only
		StartTime:     startTime,                                                                               // --start
		ExpiryTime:    expiryTime,                                                                              // --expiry
		Permissions:   to.Ptr(sas.BlobPermissions{Create: true, Write: true, Read: true, List: true}).String(), // --permissions "cw"
		ContainerName: blobInfo.ContainerName,                                                                  // --container-name
		BlobName:      blobInfo.BlobName,                                                                       // --name
	}.SignWithSharedKey(credential)
	if err != nil {
		return "", fmt.Errorf("failed to sign SAS token: %w", err)
	}

	// Return the SAS token query string (without the leading "?")
	return sasQueryParams.Encode(), nil
}

// AddFlags adds CLI flags required for working with Azure storage.
func AddFlags(flags *pflag.FlagSet, options *Options) {
	flags.StringVar(&options.SubscriptionID, "azure-subscription-id", "", "Azure subscription ID where the storage account is located")
	flags.StringVar(&options.ResourceGroup, "azure-resource-group", "", "Azure resource group where the storage account is located")
}
