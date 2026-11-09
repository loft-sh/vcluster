package cert

import (
	"context"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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

func NewSyncer(ctx *synccontext.ControllerContext) (Syncer, error) {
	return &syncer{
		serverCaKey:  ctx.Config.VirtualClusterKubeConfig().ServerCAKey,
		serverCaCert: ctx.Config.VirtualClusterKubeConfig().ServerCACert,

		vClient:                 ctx.VirtualManager.GetClient(),
		workloadNamespaceClient: ctx.HostNamespaceClient,
		vConfig:                 ctx.Config,

		listeners: []dynamiccertificates.Listener{},
	}, nil
}

type syncer struct {
	serverCaCert string
	serverCaKey  string

	vClient                 client.Client
	workloadNamespaceClient client.Client
	vConfig                 *config.VirtualClusterConfig

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

func (s *syncer) RunOnce(ctx context.Context) error {
	s.currentCertMutex.Lock()
	defer s.currentCertMutex.Unlock()

	return s.regen(ctx)
}

func (s *syncer) regen(ctx context.Context) error {
	cert, key, extraSANs, err := GenAPIServerServingCerts(ctx, s.workloadNamespaceClient, s.vClient, s.vConfig, s.serverCaCert, s.serverCaKey, s.currentCert, s.currentKey)
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
		s.currentCertMutex.Lock()
		defer s.currentCertMutex.Unlock()

		if err := s.regen(ctx); err != nil {
			klog.Infof("Error regenerating certificate: %v", err)
			return
		}

		for _, l := range s.listeners {
			l.Enqueue()
		}
	}, time.Second*2, 1.25, true)
}
