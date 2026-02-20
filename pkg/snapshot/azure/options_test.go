package azure

import (
	"context"
	"testing"
)

func TestGetSubscriptionID(t *testing.T) {
	tests := []struct {
		name        string
		structField string
		envVar      string
		expected    string
	}{
		{
			name:        "returns SubscriptionID field when set",
			structField: "sub-from-struct",
			envVar:      "",
			expected:    "sub-from-struct",
		},
		{
			name:        "falls back to env var when SubscriptionID field is empty",
			structField: "",
			envVar:      "sub-from-env",
			expected:    "sub-from-env",
		},
		{
			name:        "SubscriptionID field takes priority over env var",
			structField: "sub-from-struct",
			envVar:      "sub-from-env",
			expected:    "sub-from-struct",
		},
		{
			name:        "returns empty string when both are unset",
			structField: "",
			envVar:      "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AZURE_SUBSCRIPTION_ID", tt.envVar)
			o := &Options{SubscriptionID: tt.structField}
			if got := o.GetSubscriptionID(); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestGetResourceGroup(t *testing.T) {
	tests := []struct {
		name        string
		structField string
		envVar      string
		expected    string
	}{
		{
			name:        "returns ResourceGroup field when set",
			structField: "rg-from-struct",
			envVar:      "",
			expected:    "rg-from-struct",
		},
		{
			name:        "falls back to env var when ResourceGroup field is empty",
			structField: "",
			envVar:      "rg-from-env",
			expected:    "rg-from-env",
		},
		{
			name:        "ResourceGroup field takes priority over env var",
			structField: "rg-from-struct",
			envVar:      "rg-from-env",
			expected:    "rg-from-struct",
		},
		{
			name:        "returns empty string when both are unset",
			structField: "",
			envVar:      "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AZURE_RESOURCE_GROUP", tt.envVar)
			o := &Options{ResourceGroup: tt.structField}
			if got := o.GetResourceGroup(); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestGetBlobURLWithSAS(t *testing.T) {
	tests := []struct {
		name     string
		blobURL  string
		sas      string
		expected string
	}{
		{
			name:     "returns blob URL unchanged when SAS field is empty",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz",
			sas:      "",
			expected: "https://account.blob.core.windows.net/container/blob.tar.gz",
		},
		{
			name:     "appends SAS token to blob URL",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz",
			sas:      "sv=2022-11-02&sig=abc123",
			expected: "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&sig=abc123",
		},
		{
			name:     "returns empty string when blob URL and SAS are both empty",
			blobURL:  "",
			sas:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{BlobURL: tt.blobURL, SAS: tt.sas}
			if got := o.GetBlobURLWithSAS(); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestContainsSAS(t *testing.T) {
	tests := []struct {
		name     string
		blobURL  string
		sas      string
		expected bool
	}{
		{
			name:     "false when blob URL and SAS field are both empty",
			blobURL:  "",
			sas:      "",
			expected: false,
		},
		{
			name:     "false when blob URL has no SAS params and SAS field is empty",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz",
			sas:      "",
			expected: false,
		},
		{
			name:     "false when blob URL has query params but no sig",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&se=2023-01-01",
			sas:      "",
			expected: false,
		},
		{
			name:     "true when SAS field is set",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz",
			sas:      "sv=2022-11-02&sig=abc123",
			expected: true,
		},
		{
			name:     "true when blob URL contains sig query parameter",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&sig=abc123",
			sas:      "",
			expected: true,
		},
		{
			name:     "true when both blob URL contains SAS and SAS field is set",
			blobURL:  "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&sig=def456",
			sas:      "sv=2022-11-02&sig=def456",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{BlobURL: tt.blobURL, SAS: tt.sas}
			if got := o.ContainsSAS(); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestFillCredentials(t *testing.T) {
	// Paths tested here do not call Azure SDK:
	//   1. ContainsSAS()=true                             → early return in FillCredentials
	//   2. tryToCreateSAS=false                           → early return in FillCredentials
	//   3. AZURE_STORAGE_BLOB_SAS env var set             → early return in getStorageSAS
	// The path where tryToCreateSAS=true, ContainsSAS()=false, and the env var is unset requires Azure SDK — not tested.
	tests := []struct {
		name           string
		options        Options
		envSAS         string
		tryToCreateSAS bool
		wantSAS        string
	}{
		{
			name: "no-op when tryToCreateSAS is false",
			options: Options{
				BlobURL: "https://account.blob.core.windows.net/container/blob.tar.gz",
			},
			tryToCreateSAS: false,
		},
		{
			name: "no-op when SAS field is already set",
			options: Options{
				BlobURL: "https://account.blob.core.windows.net/container/blob.tar.gz",
				SAS:     "sv=2022-11-02&sig=abc123",
			},
			tryToCreateSAS: true,
			wantSAS:        "sv=2022-11-02&sig=abc123",
		},
		{
			name: "no-op when blob URL already contains SAS",
			options: Options{
				BlobURL: "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&sig=abc123",
			},
			tryToCreateSAS: true,
		},
		{
			name: "populates SAS from AZURE_STORAGE_BLOB_SAS env var",
			options: Options{
				BlobURL: "https://account.blob.core.windows.net/container/blob.tar.gz",
			},
			envSAS:         "sv=2022-11-02&sig=fromenv",
			tryToCreateSAS: true,
			wantSAS:        "sv=2022-11-02&sig=fromenv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(StorageBlobSASEnvVar, tt.envSAS)
			o := tt.options
			if err := o.FillCredentials(context.Background(), tt.tryToCreateSAS); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if o.SAS != tt.wantSAS {
				t.Fatalf("expected SAS %q, got %q", tt.wantSAS, o.SAS)
			}
		})
	}
}
