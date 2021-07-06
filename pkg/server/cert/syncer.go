package cert

import (
	"context"
	"fmt"
	ctrlcontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"sync"
	"time"
)

var (
	certPath = "/var/lib/vcluster/tls"
)

type Syncer interface {
	dynamiccertificates.Notifier
	dynamiccertificates.ControllerRunner
	dynamiccertificates.CertKeyContentProvider
}

func NewSyncer(ctx *ctrlcontext.ControllerContext) Syncer {
	return &syncer{
		clusterDomain: ctx.Options.ClusterDomain,

		serverCaKey:  ctx.Options.ServerCaKey,
		serverCaCert: ctx.Options.ServerCaCert,

		addSANs:     ctx.Options.TlsSANs,
		listeners:   []dynamiccertificates.Listener{},
		serviceName: ctx.Options.ServiceName,

		vClient: ctx.VirtualManager.GetClient(),
		pClient: ctx.LocalManager.GetClient(),
	}
}

type syncer struct {
	clusterDomain string

	serverCaCert string
	serverCaKey  string

	addSANs []string

	serviceName string

	vClient client.Client
	pClient client.Client

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
	retSANs := []string{}

	// get current namespace
	namespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return nil, err
	}

	// get cluster ip of target service
	svc := &corev1.Service{}
	err = s.pClient.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      s.serviceName,
	}, svc)
	if err != nil {
		return nil, fmt.Errorf("error getting vcluster service %s/%s: %v", namespace, s.serviceName, err)
	} else if svc.Spec.ClusterIP == "" {
		return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", namespace, s.serviceName)
	}

	// get load balancer ip
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			retSANs = append(retSANs, ing.IP)
		}
		if ing.Hostname != "" {
			retSANs = append(retSANs, ing.Hostname)
		}
	}

	retSANs = append(retSANs, svc.Spec.ClusterIP)

	// get cluster ips of node services
	svcs := &corev1.ServiceList{}
	err = s.pClient.List(context.TODO(), svcs, client.InNamespace(namespace), client.MatchingLabels{nodeservice.ServiceClusterLabel: translate.Suffix})
	if err != nil {
		return nil, err
	}
	for _, svc := range svcs.Items {
		if svc.Spec.ClusterIP == "" {
			continue
		}

		retSANs = append(retSANs, svc.Spec.ClusterIP)
	}

	// make sure other sans are there as well
	retSANs = append(retSANs, s.addSANs...)
	sort.Strings(retSANs)

	return retSANs, nil
}

func (s *syncer) RunOnce() error {
	s.currentCertMutex.Lock()
	defer s.currentCertMutex.Unlock()

	extraSANs, err := s.getSANs()
	if err != nil {
		return err
	}

	return s.regen(extraSANs)
}

func (s *syncer) regen(extraSANs []string) error {
	err := os.MkdirAll(certPath, 0755)
	if err != nil {
		return err
	}

	klog.Infof("Generating serving cert for service ips: %v", extraSANs)
	tlsCert := filepath.Join(certPath, "serving-tls.crt")
	tlsKey := filepath.Join(certPath, "serving-tls.key")
	_, err = GenServingCerts(s.serverCaCert, s.serverCaKey, tlsCert, tlsKey, s.clusterDomain, extraSANs)
	if err != nil {
		return err
	}

	s.currentCert, err = ioutil.ReadFile(tlsCert)
	if err != nil {
		return err
	}

	s.currentKey, err = ioutil.ReadFile(tlsKey)
	if err != nil {
		return err
	}

	s.currentSANs = extraSANs
	return nil
}

func (s *syncer) Run(workers int, stopCh <-chan struct{}) {
	wait.JitterUntil(func() {
		extraSANs, err := s.getSANs()
		if err != nil {
			klog.Infof("Error retrieving SANs: %v", err)
			return
		}

		s.currentCertMutex.Lock()
		defer s.currentCertMutex.Unlock()

		if reflect.DeepEqual(extraSANs, s.currentSANs) == false {
			err = s.regen(extraSANs)
			if err != nil {
				klog.Infof("Error regenerating certificate: %v", err)
				return
			}

			for _, l := range s.listeners {
				l.Enqueue()
			}
		}
	}, time.Second*2, 1.25, true, stopCh)
}
