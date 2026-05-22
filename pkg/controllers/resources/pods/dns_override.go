package pods

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	podstranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

type dnsNameserversResolver struct {
	entries []dnsNameserverEntry

	// hostCaches maps host namespace to the cache.Cache backing the
	// resolver's reads from that namespace. The pod syncer reuses these
	// caches to install Service watches that re-enqueue virtual pods when
	// the selected Service's ClusterIP changes or when the label selector
	// suddenly matches a different Service.
	hostCaches map[string]cache.Cache

	// tenantNamespaces is the set of tenant-cluster namespaces referenced
	// by tenant-scoped entries. The pod syncer uses it to filter Service
	// events from the virtual cache so re-resolution also runs when a
	// tenant Service's ClusterIP or selector match changes.
	tenantNamespaces map[string]struct{}
}

type dnsNameserverEntry struct {
	namespace string
	selector  labels.Selector
	reader    client.Reader
}

func (r *dnsNameserversResolver) Enabled() bool {
	return r != nil && len(r.entries) > 0
}

func (r *dnsNameserversResolver) Resolve(ctx context.Context) ([]string, []error) {
	if !r.Enabled() {
		return nil, nil
	}
	var ips []string
	var errs []error
	for _, e := range r.entries {
		entryIPs, err := resolveEntryIPs(ctx, e)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ips = append(ips, entryIPs...)
	}
	return ips, errs
}

func resolveEntryIPs(ctx context.Context, e dnsNameserverEntry) ([]string, error) {
	if e.selector == nil {
		return nil, fmt.Errorf("nil label selector for namespace %q", e.namespace)
	}
	var svcList corev1.ServiceList
	if err := e.reader.List(ctx, &svcList,
		client.InNamespace(e.namespace),
		client.MatchingLabelsSelector{Selector: e.selector},
	); err != nil {
		return nil, fmt.Errorf("list Services in %q: %w", e.namespace, err)
	}
	if len(svcList.Items) == 0 {
		return nil, fmt.Errorf("no Service matches selector %q in namespace %q", e.selector, e.namespace)
	}

	ips := make([]string, 0, len(svcList.Items))
	skipped := make([]string, 0, len(svcList.Items))
	for _, svc := range svcList.Items {
		ip := svc.Spec.ClusterIP
		if ip == "" || ip == corev1.ClusterIPNone {
			skipped = append(skipped, fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
			continue
		}
		ips = append(ips, ip)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no Service with a ClusterIP matches selector %q in namespace %q (headless or unassigned: %v)", e.selector, e.namespace, skipped)
	}
	return ips, nil
}

// buildDNSNameserversResolver constructs the resolver and starts a dedicated
// informer cache for each host-scoped entry. Tenant-scoped entries reuse the
// virtual manager's cache. Returns (nil, nil) when no nameservers are configured.
func buildDNSNameserversResolver(ctx *synccontext.RegisterContext) (*dnsNameserversResolver, error) {
	cfgEntries := ctx.Config.Sync.ToHost.Pods.DNS.Nameservers
	res := &dnsNameserversResolver{
		hostCaches:       map[string]cache.Cache{},
		tenantNamespaces: map[string]struct{}{},
	}
	if len(cfgEntries) == 0 {
		return res, nil
	}

	for i, ce := range cfgEntries {
		svc := ce.Service
		selector, err := svc.LabelSelector.ToSelector()
		if err != nil {
			return nil, fmt.Errorf("nameservers[%d]: invalid label selector: %w", i, err)
		}

		entry := dnsNameserverEntry{
			namespace: svc.Namespace,
			selector:  selector,
		}

		switch svc.Scope {
		case vclusterconfig.DNSNameserverScopeHost:
			if c, ok := res.hostCaches[svc.Namespace]; ok {
				entry.reader = c
			} else {
				c, err := cache.New(ctx.HostManager.GetConfig(), cache.Options{
					Scheme: ctx.HostManager.GetScheme(),
					Mapper: ctx.HostManager.GetRESTMapper(),
					DefaultNamespaces: map[string]cache.Config{
						svc.Namespace: {},
					},
				})
				if err != nil {
					return nil, fmt.Errorf("nameservers[%d]: build host cache: %w", i, err)
				}
				if err := ctx.HostManager.Add(c); err != nil {
					return nil, fmt.Errorf("nameservers[%d]: attach host cache: %w", i, err)
				}
				res.hostCaches[svc.Namespace] = c
				entry.reader = c
			}

		case vclusterconfig.DNSNameserverScopeTenant:
			entry.reader = ctx.VirtualManager.GetClient()
			res.tenantNamespaces[svc.Namespace] = struct{}{}

		default:
			return nil, fmt.Errorf("nameservers[%d]: unknown scope %q", i, svc.Scope)
		}

		res.entries = append(res.entries, entry)
	}

	return res, nil
}

var _ podstranslate.DNSNameserversResolver = (*dnsNameserversResolver)(nil)
