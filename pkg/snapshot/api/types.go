package api

import "github.com/loft-sh/vcluster/pkg/snapshot"

type OptionsRequest struct {
	Options *snapshot.Options `json:"options,omitempty"`
}
