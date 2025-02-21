package pro

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

type FeatureCheck func(ctx *synccontext.ControllerContext) error

var FeatureChecks []FeatureCheck

func CheckFeatures(ctx *synccontext.ControllerContext) error {
	for _, check := range FeatureChecks {
		if err := check(ctx); err != nil {
			return err
		}
	}

	return nil
}
