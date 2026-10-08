package destroy

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestResourceOrderCoversEveryStorageKind pins that destroy knows every storage.loft.sh kind the
// api module defines. Destroy refuses to start while discovery reports a resource outside
// resourceOrder, so a kind that reaches the api module without a place in the order fails every
// destroy of a platform serving it, as machineconfigtemplates and networkenvironments did. The
// api module is where a new kind first reaches this repo, so this is the earliest the gap shows.
func TestResourceOrderCoversEveryStorageKind(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := storagev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	var missing []string
	for gvk, typ := range scheme.AllKnownTypes() {
		if gvk.Group != storagev1.SchemeGroupVersion.Group {
			continue
		}
		obj := reflect.New(typ).Interface()
		if _, isList := obj.(client.ObjectList); isList {
			continue
		}
		// only a kind carrying object metadata is served by a CRD
		if _, ok := obj.(client.Object); !ok {
			continue
		}
		if name := resourceName(gvk.Kind); !slices.Contains(resourceOrder, name) {
			missing = append(missing, name)
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("storage.loft.sh kinds with no place in resourceOrder, so destroy refuses a platform that serves them: %v", missing)
	}
}

// TestResourceOrderIsWellFormed pins the two invariants the destroy loop relies on: a resource
// is deleted once, and every legacy name is in the order, since the loop only consults the
// legacy list to quieten the log for a resource discovery no longer reports.
func TestResourceOrderIsWellFormed(t *testing.T) {
	seen := map[string]struct{}{}
	for _, name := range resourceOrder {
		if _, dup := seen[name]; dup {
			t.Errorf("%q is listed twice in resourceOrder", name)
		}
		seen[name] = struct{}{}
	}
	for _, name := range legacyResources {
		if _, ok := seen[name]; !ok {
			t.Errorf("legacy resource %q is not in resourceOrder", name)
		}
	}
}

// TestResourceOrderDeletesHoldersFirst pins the relative positions a finalizer makes load-bearing:
// a resource that holds another through a finalizer must be deleted first, or destroy waits on
// the held one until its timeout. A node claim holds its network environment, a provider's
// finalizer waits on its environments, and a tenant storage names its profile.
func TestResourceOrderDeletesHoldersFirst(t *testing.T) {
	pairs := [][2]string{
		{"nodeclaims", "networkenvironments"},
		{"networkenvironments", "nodeproviders"},
		{"tenantstorages", "storageprofiles"},
		{"tenantstorages", "tenants"},
	}
	for _, pair := range pairs {
		first, second := slices.Index(resourceOrder, pair[0]), slices.Index(resourceOrder, pair[1])
		if first < 0 || second < 0 {
			t.Fatalf("%q and %q must both be in resourceOrder, got positions %d and %d", pair[0], pair[1], first, second)
		}
		if first > second {
			t.Errorf("%q must be deleted before %q, which it holds until it is gone", pair[0], pair[1])
		}
	}
}

// resourceName is the plural the API server derives for a kind: lower-cased, "s" appended, with
// the "es" and "ies" endings the storage kinds need (ClusterAccess, and nothing ending in a
// consonant plus "y" today; SSHKey and AccessKey keep their "y").
func resourceName(kind string) string {
	name := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(name, "s"):
		return name + "es"
	case strings.HasSuffix(name, "y") && !strings.HasSuffix(name, "ey"):
		return strings.TrimSuffix(name, "y") + "ies"
	default:
		return name + "s"
	}
}
