package cli

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// fixFileOwnershipUnderSudo corrects ownership of a file and its parent directory
// when running under sudo. This handles the corner case where "sudo vcluster create"
// is run on a machine without an existing ~/.kube/config — the newly created file
// and directory are root-owned, making them unusable by the actual user. When the
// file already exists, overwriting preserves the original ownership (POSIX behavior),
// so this is a no-op in the common case.
//
// Only paths under the invoking user's home directory (resolved via os/user from
// SUDO_USER) are modified. System paths like /etc or /tmp are never touched.
func fixFileOwnershipUnderSudo(filePath string) {
	filePath, _ = filepath.Abs(filePath)

	sudoUID := os.Getenv("SUDO_UID")
	sudoGID := os.Getenv("SUDO_GID")
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUID == "" || sudoGID == "" || sudoUser == "" {
		return
	}

	uid, err := strconv.Atoi(sudoUID)
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(sudoGID)
	if err != nil {
		return
	}

	// Resolve the real user's home from the system user database (passwd/LDAP/
	// directory services). This avoids hardcoding /home or /Users and handles
	// non-standard home layouts.
	u, err := user.Lookup(sudoUser)
	if err != nil || u.HomeDir == "" {
		return
	}

	// Only fix ownership for paths under the user's home directory.
	// Anything outside (e.g. /etc/kubernetes/admin.conf, /tmp) is left untouched.
	if !strings.HasPrefix(filePath, u.HomeDir+string(os.PathSeparator)) {
		return
	}

	_ = os.Chown(filePath, uid, gid)
	_ = os.Chown(filepath.Dir(filePath), uid, gid)
}
