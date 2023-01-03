package translator

import (
	"os"
	"reflect"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
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

func NewIfNil[T interface {
	DeepCopy() T
}](updated, obj T) T {
	if reflect.ValueOf(updated).IsNil() {
		return obj.DeepCopy()
	}

	return updated
}
