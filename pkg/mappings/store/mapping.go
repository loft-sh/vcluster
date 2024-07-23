package store

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

type Mapping struct {
	synccontext.NameMapping `json:",inline"`

	Labels        []synccontext.LabelMapping `json:"labels,omitempty"`
	LabelsCluster []synccontext.LabelMapping `json:"labelsCluster,omitempty"`
	References    []synccontext.NameMapping  `json:"references,omitempty"`

	changed bool `json:"-"`
}

func (m Mapping) String() string {
	return m.NameMapping.String()
}
