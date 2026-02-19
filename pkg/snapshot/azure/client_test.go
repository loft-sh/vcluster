package azure

import (
	"strings"
	"testing"
)

func TestGetBlobInfo(t *testing.T) {
	tests := []struct {
		name          string
		blobURL       string
		expectedInfo  BlobInfo
		expectedError string
	}{
		{
			name:          "empty URL",
			blobURL:       "",
			expectedError: "blob URL is empty",
		},
		{
			name:          "invalid URL",
			blobURL:       "https://[::1",
			expectedError: "failed to parse blob URL",
		},
		{
			name:          "missing hostname",
			blobURL:       "https:///container/blob",
			expectedError: "invalid blob URL format",
		},
		{
			name:          "missing container and blob",
			blobURL:       "https://account.blob.core.windows.net/",
			expectedError: "invalid blob URL format",
		},
		{
			name:          "missing blob name",
			blobURL:       "https://account.blob.core.windows.net/container",
			expectedError: "invalid blob URL format",
		},
		{
			name:    "valid URL with port",
			blobURL: "https://account.blob.core.windows.net:443/container/blob.tar.gz",
			expectedInfo: BlobInfo{
				AccountName:   "account",
				ContainerName: "container",
				BlobName:      "blob.tar.gz",
			},
		},
		{
			name:    "URL with SAS query",
			blobURL: "https://account.blob.core.windows.net/container/blob.tar.gz?sv=2022-11-02&sig=abc123",
			expectedInfo: BlobInfo{
				AccountName:   "account",
				ContainerName: "container",
				BlobName:      "blob.tar.gz",
			},
		},
		{
			name:    "nested blob path keeps double slash",
			blobURL: "https://account.blob.core.windows.net/container/path//to/blob.tar.gz",
			expectedInfo: BlobInfo{
				AccountName:   "account",
				ContainerName: "container",
				BlobName:      "path//to/blob.tar.gz",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, err := getBlobInfo(test.blobURL)
			if test.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", test.expectedError)
				}
				if !strings.Contains(err.Error(), test.expectedError) {
					t.Fatalf("expected error containing %q, got %q", test.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if info != test.expectedInfo {
				t.Fatalf("expected BlobInfo %+v, got %+v", test.expectedInfo, info)
			}
		})
	}
}
