package translator

import (
	"os"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/runtime"
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

func NewIfNil[T any, P interface {
	*T
	runtime.Object
}](updated, obj P) P {
	if updated == nil {
		return obj.DeepCopyObject().(P)
	}

	return updated
}
