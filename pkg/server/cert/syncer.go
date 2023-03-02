package cert

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"

	ctrlcontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Syncer interface {
	dynamiccertificates.Notifier
	dynamiccertificates.ControllerRunner
	dynamiccertificates.CertKeyContentProvider
}

func NewSyncer(currentNamespace string, currentNamespaceClient client.Client, options *ctrlcontext.VirtualClusterOptions) (Syncer, error) {
	return &syncer{
		clusterDomain: options.ClusterDomain,

		serverCaKey:  options.ServerCaKey,
		serverCaCert: options.ServerCaCert,

		addSANs:   options.TLSSANs,
		listeners: []dynamiccertificates.Listener{},

		serviceName:           options.ServiceName,
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

func (s *syncer) getSANs() ([]string, error) {

	retSANs := []string{
		s.serviceName,
		s.serviceName + "." + s.currentNamespace, "*." + translate.Suffix + "." + s.currentNamespace + "." + constants.NodeSuffix,
	}

	// get cluster ip of target service
	svc := &corev1.Service{}
	err := s.currentNamespaceCient.Get(context.TODO(), types.NamespacedName{
		Namespace: s.currentNamespace,
		Name:      s.serviceName,
	}, svc)
	if err != nil {
		return nil, fmt.Errorf("error getting vcluster service %s/%s: %v", s.currentNamespace, s.serviceName, err)
	} else if svc.Spec.ClusterIP == "" {
		return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", s.currentNamespace, s.serviceName)
	}

	// get load balancer ip
	// currently, the load balancer service is named <serviceName>-lb, but the syncer image might run in legacy environments
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

	retSANs = append(retSANs, svc.Spec.ClusterIP)

	// add pod IP
	podIP := os.Getenv("POD_IP")
	if podIP != "" {
		retSANs = append(retSANs, podIP)
	}

	// get cluster ip of load balancer service
	lbSVCName := translate.GetLoadBalancerSVCName(s.serviceName)
	lbSVC := &corev1.Service{}
	err = s.currentNamespaceCient.Get(context.TODO(), types.NamespacedName{
		Namespace: s.currentNamespace,
		Name:      lbSVCName,
	}, lbSVC)
	// proceed only if load balancer service exists
	if !errors.IsNotFound(err) {
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster load balancer service %s/%s: %v", s.currentNamespace, lbSVCName, err)
		} else if lbSVC.Spec.ClusterIP == "" {
			return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", s.currentNamespace, lbSVCName)
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
		retSANs = append(retSANs,
			lbSVCName,
			lbSVCName+"."+s.currentNamespace, "*."+translate.Suffix+"."+s.currentNamespace+"."+constants.NodeSuffix,
		)
	}

	// make sure other sans are there as well
	retSANs = append(retSANs, s.addSANs...)
	sort.Strings(retSANs)

	return retSANs, nil
}

func (s *syncer) RunOnce(ctx context.Context) error {
	s.currentCertMutex.Lock()
	defer s.currentCertMutex.Unlock()

	extraSANs, err := s.getSANs()
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

func (s *syncer) Run(ctx context.Context, workers int) {
	wait.JitterUntil(func() {
		extraSANs, err := s.getSANs()
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
	}, time.Second*2, 1.25, true, ctx.Done())
}
