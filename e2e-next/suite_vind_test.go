// Suite: vind
// Tests the vCluster docker (vind) driver lifecycle.
// No shared vCluster needed — tests manage their own instances via `vcluster create --driver docker`.
// Run:      just run-e2e 'vind'
package e2e_next

import (
	_ "github.com/loft-sh/vcluster/e2e-next/test_vind"
)
