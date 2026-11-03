package licenseapi

import (
	"cmp"
	"slices"
)

func New() *License {
	limits := make([]*Limit, 0, len(Limits))
	for _, limit := range Limits {
		limits = append(limits, limit)
	}
	slices.SortFunc(limits, func(a, b *Limit) int {
		return cmp.Compare(a.Name, b.Name)
	})

	// Sorting features by module is not required here. However, to maintain backwards compatibility, the structure of
	// features being contained within a module is still necessary. Therefore, all features are now returned in one module.
	return &License{
		Modules: []*Module{
			{
				DisplayName: "All Features",
				Name:        string(VirtualClusterModule),
				Limits:      limits,
				Features:    GetAllFeatures(),
			},
		},
	}
}
