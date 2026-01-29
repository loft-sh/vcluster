package init

import (
	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	. "github.com/onsi/ginkgo/v2"
)

// This must be called before any ginkgo DSL evaluation
var _ = AddTreeConstructionNodeArgsTransformer(suite.NodeTransformer)
var _ = AddTreeConstructionNodeArgsTransformer(e2e.ContextualNodeTransformer)
