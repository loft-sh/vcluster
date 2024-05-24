package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func ProjectNamespace(projectName string) string {
	return "loft-p-" + projectName
}

func SafeConcatName(name ...string) string {
	return SafeConcatNameMax(name, 63)
}

func SafeConcatNameMax(name []string, max int) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > max {
		digest := sha256.Sum256([]byte(fullPath))
		return fullPath[0:max-8] + "-" + hex.EncodeToString(digest[0:])[0:7]
	}
	return fullPath
}
