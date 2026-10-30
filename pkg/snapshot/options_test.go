package snapshot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"gotest.tools/v3/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectedOptions snapshotapi.Options
		expectedError   string
	}{
		{
			name: "s3",
			url:  fmt.Sprintf("s3://my-bucket/my-key?region=eu-west-1&access-key-id=%s&secret-access-key=%s", base64.StdEncoding.EncodeToString([]byte("my-access-key-id")), base64.StdEncoding.EncodeToString([]byte("my-secret-access-key"))),
			expectedOptions: snapshotapi.Options{
				Type: "s3",
				S3: snapshotapi.S3Options{
					Bucket:          "my-bucket",
					Key:             "my-key",
					Region:          "eu-west-1",
					AccessKeyID:     "my-access-key-id",
					SecretAccessKey: "my-secret-access-key",
				},
			},
		},
		{
			name: "container",
			url:  "container:///my-path",
			expectedOptions: snapshotapi.Options{
				Type:      "container",
				Container: snapshotapi.ContainerOptions{Path: "/my-path"},
			},
		},
		{
			name: "oci",
			url:  "oci://my-registry.com/my-repo?skip-client-credentials=true",
			expectedOptions: snapshotapi.Options{
				Type: "oci",
				OCI: snapshotapi.OCIOptions{
					Repository:            "my-registry.com/my-repo",
					SkipClientCredentials: true,
				},
			},
		},
		{
			name: "azure",
			url:  "https://mysnapshotstorage.blob.core.windows.net/my-cluster-snapshots/snap-1.tar.gz",
			expectedOptions: snapshotapi.Options{
				Type: "azure",
				Azure: snapshotapi.AzureOptions{
					BlobURL: "https://mysnapshotstorage.blob.core.windows.net/my-cluster-snapshots/snap-1.tar.gz",
				},
			},
		},
		{
			name:          "s3 unexpected option",
			url:           "s3://my-bucket/my-key?region=eu-west-1&unexpected-option=true",
			expectedError: "error parsing options: unknown parameter in url: unexpected-option",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshotOptions := &snapshotapi.Options{}
			err := Parse(test.url, snapshotOptions)
			if test.expectedError != "" {
				assert.Error(t, err, test.expectedError)
				return
			}
			assert.NilError(t, err)

			optionsRaw, err := json.Marshal(snapshotOptions)
			assert.NilError(t, err)

			expectedRaw, err := json.Marshal(test.expectedOptions)
			assert.NilError(t, err)

			assert.Equal(t, string(expectedRaw), string(optionsRaw))
		})
	}
}
