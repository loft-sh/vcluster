package translator

import (
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PrintChanges(oldObject, newObject client.Object, log loghelper.Logger) {
	if os.Getenv("DEBUG") == "true" {
		rawPatch, err := client.MergeFrom(oldObject).Data(newObject)
		if err == nil {
			log.Debugf("Updating object with: %v", string(rawPatch))
		}
	}
}
