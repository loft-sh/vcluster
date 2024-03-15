package deploy

import (
	"fmt"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetTarballPath(t *testing.T) {
	testTable := []struct {
		desc    string
		repo    string
		name    string
		version string

		expectedTarballPath string
	}{
		{
			desc:    "no repo",
			repo:    "",
			name:    "abc",
			version: "v1.2",

			expectedTarballPath: "/tmp/abc-v1.2.tgz",
		},
		{
			desc:    "with repo",
			repo:    "http://helmrepo.io/foo",
			name:    "abc",
			version: "v1.2",

			expectedTarballPath: fmt.Sprintf("/tmp/%s/abc-v1.2.tgz", "17e8c9faf0"),
		},
	}
	for _, testCase := range testTable {
		t.Logf("running test: %q", testCase.desc)
		tarballPath, _ := getTarballPath("/tmp/", testCase.repo, testCase.name, testCase.version)
		assert.Equal(t, tarballPath, testCase.expectedTarballPath)
		assert.Equal(t, filepath.Dir(tarballPath), filepath.Dir(testCase.expectedTarballPath))
	}
}
