// Suite: cli
// Tests vCluster CLI command lifecycle (create, list, delete, pause, resume, etc.).
// No shared vCluster needed — tests manage their own instances.
// Run: just run-e2e 'cli'
package e2e_next

import (
	_ "github.com/loft-sh/vcluster/e2e-next/test_cli"
)
