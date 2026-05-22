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
		ip, err := resolveEntryIP(ctx, e)
		if err != nil {
			errs = append(errs, err)
		} else {
			ips = append(ips, ip)
		}
	}
	return ips, errs
}

func resolveEntryIP(ctx context.Context, e dnsNameserverEntry) (string, error) {
	var svcList corev1.ServiceList
	if err := e.reader.List(ctx, &svcList,
		client.InNamespace(e.namespace),
		client.MatchingLabelsSelector{Selector: e.selector},
	); err != nil {
		return "", fmt.Errorf("list Services in %q: %w", e.namespace, err)
	}
	switch len(svcList.Items) {
	case 0:
		return "", fmt.Errorf("no Service matches selector %q in namespace %q", e.selector, e.namespace)
	case 1:
		ip := svcList.Items[0].Spec.ClusterIP
		if ip == "" || ip == corev1.ClusterIPNone {
			return "", fmt.Errorf("Service %s/%s has no ClusterIP", svcList.Items[0].Namespace, svcList.Items[0].Name)
		}
		return ip, nil
	default:
		names := make([]string, 0, len(svcList.Items))
		for _, s := range svcList.Items {
			names = append(names, s.Name)
		}
		return "", fmt.Errorf("expected exactly one Service, got %d: %v", len(svcList.Items), names)
	}
}

// buildDNSNameserversResolver constructs the resolver and starts a dedicated
// informer cache for each host-scoped entry. Tenant-scoped entries reuse the
// virtual manager's cache. Returns (nil, nil) when no nameservers are configured.
func buildDNSNameserversResolver(ctx *synccontext.RegisterContext) (*dnsNameserversResolver, error) {
	cfgEntries := ctx.Config.Sync.ToHost.Pods.DNS.Nameservers
	if len(cfgEntries) == 0 {
		return nil, nil
	}

	res := &dnsNameserversResolver{}
	hostCaches := map[string]client.Reader{}

	for i, ce := range cfgEntries {
		svc := ce.Service
		// selector validity is guaranteed by validateDNSNameservers at startup
		selector, _ := svc.LabelSelector.ToSelector()

		entry := dnsNameserverEntry{
			namespace: svc.Namespace,
			selector:  selector,
		}

		switch svc.Scope {
		case vclusterconfig.DNSNameserverScopeHost:
			if r, ok := hostCaches[svc.Namespace]; ok {
				entry.reader = r
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
				hostCaches[svc.Namespace] = c
				entry.reader = c
			}

		case vclusterconfig.DNSNameserverScopeTenant:
			entry.reader = ctx.VirtualManager.GetClient()

		default:
			return nil, fmt.Errorf("nameservers[%d]: unknown scope %q", i, svc.Scope)
		}

		res.entries = append(res.entries, entry)
	}

	return res, nil
}

var _ podstranslate.DNSNameserversResolver = (*dnsNameserversResolver)(nil)
