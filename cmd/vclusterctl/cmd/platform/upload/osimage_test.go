package upload

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestMissingImageHint(t *testing.T) {
	notFound := kerrors.NewNotFound(schema.GroupResource{Group: "management.loft.sh", Resource: "osimages"}, "ubuntu-24.04")

	hinted := missingImageHint(fmt.Errorf("request upload targets from part 1: %w", notFound), "ubuntu-24.04")
	assert.ErrorContains(t, hinted, `osimages.management.loft.sh "ubuntu-24.04" not found`)
	assert.ErrorContains(t, hinted, "create it in the platform UI or with kubectl")

	// Anything else is the platform's to explain, including the preconditions the upload
	// subresource checks, so it reaches the user untouched.
	invalid := kerrors.NewInvalid(schema.GroupKind{}, "ubuntu-24.04", nil)
	assert.Equal(t, invalid, missingImageHint(invalid, "ubuntu-24.04"))
	assert.Equal(t, "boom", missingImageHint(errors.New("boom"), "ubuntu-24.04").Error())
}

// The checksum is calculated after the parts are up, so the file is no longer at its start
// if anything ever reads it sequentially. Hashing a partial file would record a wrong
// checksum on an otherwise complete image.
func TestFileChecksumHashesTheWholeFileFromAnyOffset(t *testing.T) {
	contents := []byte("os image bytes")
	path := filepath.Join(t.TempDir(), "image.qcow2")
	assert.NilError(t, os.WriteFile(path, contents, 0o600))

	file, err := os.Open(path)
	assert.NilError(t, err)
	defer file.Close()

	_, err = file.Read(make([]byte, 5))
	assert.NilError(t, err)

	checksum, err := fileChecksum(file)
	assert.NilError(t, err)

	expected := sha256.Sum256(contents)
	assert.Equal(t, hex.EncodeToString(expected[:]), checksum)
}
