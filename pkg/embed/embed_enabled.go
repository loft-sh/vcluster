//go:build embed_chart

package embed

import "embed"

//go:generate ../../hack/embed-chart.sh
//go:embed chart/*.tgz
var Charts embed.FS
