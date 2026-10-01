package synccontext

import (
	"errors"
	"fmt"

	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/version"
)

// ErrNilClusterVersionInfo is returned when no version info is passed. Callers that treat a
// missing version as "not discovered" should skip the call instead.
var ErrNilClusterVersionInfo = errors.New("nil cluster version info")

// ParseClusterVersion converts a discovered cluster version into a comparable version.
func ParseClusterVersion(info *version.Info) (*utilversion.Version, error) {
	if info == nil {
		return nil, ErrNilClusterVersionInfo
	}

	parsed, err := utilversion.ParseGeneric(info.String())
	if err != nil {
		return nil, fmt.Errorf("parse cluster version %q: %w", info.String(), err)
	}
	return parsed, nil
}
