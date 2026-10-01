package suite

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

type key int

const (
	previewSpecsKey key = iota
)

var (
	focusedLabels = map[string]bool{}
)

func IsFocused(label string) bool {
	return focusedLabels[label]
}

func PreviewSpecsFrom(ctx context.Context) ginkgo.Report {
	report, _ := ctx.Value(previewSpecsKey).(ginkgo.Report)
	return report
}

func PreviewSpecsAroundNode(config types.SuiteConfig) func(ctx context.Context) context.Context {
	report := ginkgo.PreviewSpecs("", config)
	for _, spec := range report.SpecReports {
		if spec.State == types.SpecStatePassed {
			for _, label := range spec.Labels() {
				focusedLabels[label] = true
			}
		}
	}
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, previewSpecsKey, report)
	}
}
