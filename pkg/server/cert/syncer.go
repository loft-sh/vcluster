package cert

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ExtraSANsFunc func(ctx context.Context) ([]string, error)

// ExtraSANs can be used to add extra sans via a function
var ExtraSANs []ExtraSANsFunc

type Syncer interface {
	dynamiccertificates.Notifier
	dynamiccertificates.ControllerRunner
	dynamiccertificates.CertKeyContentProvider
}

func NewSyncer(_ context.Context, currentNamespace string, currentNamespaceClient client.Client, options *config.VirtualClusterConfig) (Syncer, error) {
	return &syncer{
		clusterDomain: options.Networking.Advanced.ClusterDomain,

		serverCaKey:  options.VirtualClusterKubeConfig().ServerCAKey,
		serverCaCert: options.VirtualClusterKubeConfig().ServerCACert,

		fakeKubeletIPs: options.Networking.Advanced.ProxyKubelets.ByIP,

		addSANs:   options.ControlPlane.Proxy.ExtraSANs,
		listeners: []dynamiccertificates.Listener{},

		serviceName:           options.WorkloadService,
		currentNamespace:      currentNamespace,
		currentNamespaceCient: currentNamespaceClient,
	}, nil
}

type syncer struct {
	clusterDomain string

	serverCaCert string
	serverCaKey  string

	addSANs []string

	serviceName           string
	currentNamespace      string
	currentNamespaceCient client.Client

	fakeKubeletIPs bool

	listeners []dynamiccertificates.Listener

	currentCertMutex sync.RWMutex
	currentCert      []byte
	currentKey       []byte
	currentSANs      []string
}

func (s *syncer) Name() string {
	return "apiserver"
}

func (s *syncer) CurrentCertKeyContent() ([]byte, []byte) {
	s.currentCertMutex.RLock()
	defer s.currentCertMutex.RUnlock()

	return s.currentCert, s.currentKey
}

func (s *syncer) AddListener(listener dynamiccertificates.Listener) {
	s.currentCertMutex.Lock()
	defer s.currentCertMutex.Unlock()

	s.listeners = append(s.listeners, listener)
}

func (s *syncer) getSANs(ctx context.Context) ([]string, error) {
	retSANs := []string{
		s.serviceName,
		s.serviceName + "." + s.currentNamespace, "*." + constants.NodeSuffix,
	}

	// get cluster ip of target service
	svc := &corev1.Service{}
	err := s.currentNamespaceCient.Get(ctx, types.NamespacedName{
		Namespace: s.currentNamespace,
		Name:      s.serviceName,
	}, svc)
	if err != nil {
		return nil, fmt.Errorf("error getting vcluster service %s/%s: %w", s.currentNamespace, s.serviceName, err)
	} else if svc.Spec.ClusterIP == "" {
		return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", s.currentNamespace, s.serviceName)
	}

	// get load balancer ip
	// currently, the load balancer service is named <serviceName>, but the syncer image might run in legacy environments
	// where the load balancer service is the same service, the service is only updated if the helm template is rerun,
	// so we are leaving this snippet in, but the load balancer ip will be read via the lbSVC var below
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			retSANs = append(retSANs, ing.IP)
		}
		if ing.Hostname != "" {
			retSANs = append(retSANs, ing.Hostname)
		}
	}

	// add cluster ip
	retSANs = append(retSANs, svc.Spec.ClusterIP)

	// add extra sans
	for _, extraSans := range ExtraSANs {
		extraSansValues, err := extraSans(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting extra sans: %w", err)
		}

		retSANs = append(retSANs, extraSansValues...)
	}

	// add pod IP
	podIP := os.Getenv("POD_IP")
	if podIP != "" {
		retSANs = append(retSANs, podIP)
	}

	// get cluster ip of load balancer service
	lbSVC := &corev1.Service{}
	err = s.currentNamespaceCient.Get(ctx, types.NamespacedName{
		Namespace: s.currentNamespace,
		Name:      s.serviceName,
	}, lbSVC)
	// proceed only if load balancer service exists
	if !kerrors.IsNotFound(err) {
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster load balancer service %s/%s: %w", s.currentNamespace, s.serviceName, err)
		} else if lbSVC.Spec.ClusterIP == "" {
			return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", s.currentNamespace, s.serviceName)
		}

		for _, ing := range lbSVC.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				retSANs = append(retSANs, ing.IP)
			}
			if ing.Hostname != "" {
				retSANs = append(retSANs, ing.Hostname)
			}
		}
		// append hostnames for load balancer service
		retSANs = append(
			retSANs,
			s.serviceName,
			s.serviceName+"."+s.currentNamespace, "*."+translate.VClusterName+"."+s.currentNamespace+"."+constants.NodeSuffix,
		)
	}

	if s.fakeKubeletIPs {
		// get cluster ips of node services
		svcs := &corev1.ServiceList{}
		err = s.currentNamespaceCient.List(ctx, svcs, client.InNamespace(s.currentNamespace), client.MatchingLabels{nodeservice.ServiceClusterLabel: translate.VClusterName})
		if err != nil {
			return nil, err
		}
		for _, svc := range svcs.Items {
			if svc.Spec.ClusterIP == "" {
				continue
			}

			retSANs = append(retSANs, svc.Spec.ClusterIP)
		}
	}

	// make sure other sans are there as well
	retSANs = append(retSANs, s.addSANs...)
	sort.Strings(retSANs)

	return retSANs, nil
}

func (s *syncer) RunOnce(ctx context.Context) error {
	s.currentCertMutex.Lock()
	defer s.currentCertMutex.Unlock()

	extraSANs, err := s.getSANs(ctx)
	if err != nil {
		return err
	}

	return s.regen(extraSANs)
}

func (s *syncer) regen(extraSANs []string) error {
	klog.Infof("Generating serving cert for service ips: %v", extraSANs)

	// GenServingCerts will write generated or updated cert/key to s.currentCert, s.currentKey
	cert, key, _, err := GenServingCerts(s.serverCaCert, s.serverCaKey, s.currentCert, s.currentKey, s.clusterDomain, extraSANs)
	if err != nil {
		return err
	}
	s.currentCert = cert
	s.currentKey = key

	s.currentSANs = extraSANs
	return nil
}

func (s *syncer) Run(ctx context.Context, _ int) {
	wait.JitterUntilWithContext(ctx, func(ctx context.Context) {
		extraSANs, err := s.getSANs(ctx)
		if err != nil {
			klog.Infof("Error retrieving SANs: %v", err)
			return
		}

		s.currentCertMutex.Lock()
		defer s.currentCertMutex.Unlock()

		if !reflect.DeepEqual(extraSANs, s.currentSANs) {
			err = s.regen(extraSANs)
			if err != nil {
				klog.Infof("Error regenerating certificate: %v", err)
				return
			}

			for _, l := range s.listeners {
				l.Enqueue()
			}
		}
	}, time.Second*2, 1.25, true)
}
