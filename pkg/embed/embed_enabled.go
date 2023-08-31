//go:build embed_charts

package embed

import "embed"

//go:generate ../../hack/embed-charts.sh
//go:embed charts/*.tgz
var Charts embed.FS
