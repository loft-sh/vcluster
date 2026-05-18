package snapshot

import (
	"github.com/loft-sh/api/v4/pkg/snapshot"
)

type LongRunningRequest interface {
	GetPhase() snapshot.RequestPhase
}
