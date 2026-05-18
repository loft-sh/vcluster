package snapshot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/loft-sh/api/v4/pkg/snapshot"
	"gotest.tools/v3/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectedOptions snapshot.Options
		expectedError   string
	}{
		{
			name: "s3",
			url:  fmt.Sprintf("s3://my-bucket/my-key?region=eu-west-1&access-key-id=%s&secret-access-key=%s", base64.StdEncoding.EncodeToString([]byte("my-access-key-id")), base64.StdEncoding.EncodeToString([]byte("my-secret-access-key"))),
			expectedOptions: snapshot.Options{
				Type: "s3",
				S3: snapshot.S3Options{
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
			expectedOptions: snapshot.Options{
				Type:      "container",
				Container: snapshot.ContainerOptions{Path: "/my-path"},
			},
		},
		{
			name: "oci",
			url:  "oci://my-registry.com/my-repo?skip-client-credentials=true",
			expectedOptions: snapshot.Options{
				Type: "oci",
				OCI: snapshot.OCIOptions{
					Repository:            "my-registry.com/my-repo",
					SkipClientCredentials: true,
				},
			},
		},
		{
			name: "azure",
			url:  "https://mysnapshotstorage.blob.core.windows.net/my-cluster-snapshots/snap-1.tar.gz",
			expectedOptions: snapshot.Options{
				Type: "azure",
				Azure: snapshot.AzureOptions{
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
			snapshotOptions := &snapshot.Options{}
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
