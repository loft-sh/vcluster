package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/certs"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
)

const (
	waitForClientTimeout = time.Minute * 10
)

func WaitForEtcdClient(parentCtx context.Context, certificatesDir string, endpoints ...string) (*clientv3.Client, error) {
	var etcdClient *clientv3.Client
	var err error
	waitErr := wait.PollUntilContextTimeout(parentCtx, time.Second, waitForClientTimeout, true, func(ctx context.Context) (bool, error) {
		etcdClient, err = GetEtcdClient(parentCtx, certificatesDir, endpoints...)
		if err == nil {
			_, err = etcdClient.MemberList(ctx)
			if err == nil {
				return true, nil
			}

			_ = etcdClient.Close()
		}

		klog.Infof("Couldn't connect to embedded etcd (will retry in a second): %v", err)
		return false, nil
	})
	if waitErr != nil {
		return nil, fmt.Errorf("error waiting for etcdclient: %w", err)
	}

	return etcdClient, nil
}

// GetEtcdClient returns an etcd client connected to the specified endpoints.
// If no endpoints are provided, endpoints are retrieved from the provided runtime config.
// If the runtime config does not list any endpoints, the default endpoint is used.
// The returned client should be closed when no longer needed, in order to avoid leaking GRPC
// client goroutines.
func GetEtcdClient(ctx context.Context, certificatesDir string, endpoints ...string) (*clientv3.Client, error) {
	cfg, err := getClientConfig(ctx, certificatesDir, endpoints...)
	if err != nil {
		return nil, err
	}

	return clientv3.New(*cfg)
}

// getClientConfig generates an etcd client config connected to the specified endpoints.
// If no endpoints are provided, getEndpoints is called to provide defaults.
func getClientConfig(ctx context.Context, certificatesDir string, endpoints ...string) (*clientv3.Config, error) {
	config := &clientv3.Config{
		Endpoints:            endpoints,
		Context:              ctx,
		DialTimeout:          2 * time.Second,
		DialKeepAliveTime:    30 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
		AutoSyncInterval:     10 * time.Second,
		Logger:               zap.L().Named("etcd-client"),
		PermitWithoutStream:  true,
	}

	var err error
	if strings.HasPrefix(endpoints[0], "https://") && certificatesDir != "" {
		config.TLS, err = toTLSConfig(certificatesDir)
	}
	return config, err
}

func toTLSConfig(certificatesDir string) (*tls.Config, error) {
	clientCert, err := tls.LoadX509KeyPair(
		filepath.Join(certificatesDir, certs.APIServerEtcdClientCertName),
		filepath.Join(certificatesDir, certs.APIServerEtcdClientKeyName),
	)
	if err != nil {
		return nil, err
	}

	pool, err := certutil.NewPool(filepath.Join(certificatesDir, certs.EtcdCACertName))
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCert},
	}, nil
}
