package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
)

const (
	waitForClientTimeout = time.Minute * 10
)

type Certificates struct {
	CaCert     string
	ServerCert string
	ServerKey  string
}

func EndpointsAndCertificatesFromFlags(flags []string) ([]string, *Certificates, error) {
	certificates := &Certificates{}
	endpoints := []string{}

	// parse flags
	for _, arg := range flags {
		if strings.HasPrefix(arg, "--etcd-servers=") {
			endpoints = strings.Split(strings.TrimPrefix(arg, "--etcd-servers="), ",")
		} else if strings.HasPrefix(arg, "--etcd-cafile=") {
			certificates.CaCert = strings.TrimPrefix(arg, "--etcd-cafile=")
		} else if strings.HasPrefix(arg, "--etcd-certfile=") {
			certificates.ServerCert = strings.TrimPrefix(arg, "--etcd-certfile=")
		} else if strings.HasPrefix(arg, "--etcd-keyfile=") {
			certificates.ServerKey = strings.TrimPrefix(arg, "--etcd-keyfile=")
		}
	}

	// fail if etcd servers is not found
	if len(endpoints) == 0 {
		return nil, nil, fmt.Errorf("couldn't find flag --etcd-servers within api-server flags")
	} else if certificates.CaCert == "" || certificates.ServerCert == "" || certificates.ServerKey == "" {
		return endpoints, nil, nil
	}
	return endpoints, certificates, nil
}

func WaitForEtcdClient(parentCtx context.Context, certificates *Certificates, endpoints ...string) (*clientv3.Client, error) {
	var etcdClient *clientv3.Client
	var err error
	waitErr := wait.PollUntilContextTimeout(parentCtx, time.Second, waitForClientTimeout, true, func(ctx context.Context) (bool, error) {
		etcdClient, err = GetEtcdClient(parentCtx, certificates, endpoints...)
		if err == nil {
			defer func() {
				_ = etcdClient.Close()
			}()

			_, err = etcdClient.MemberList(ctx)
			if err == nil {
				return true, nil
			}
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
func GetEtcdClient(ctx context.Context, certificates *Certificates, endpoints ...string) (*clientv3.Client, error) {
	cfg, err := getClientConfig(ctx, certificates, endpoints...)
	if err != nil {
		return nil, err
	}

	return clientv3.New(*cfg)
}

// getClientConfig generates an etcd client config connected to the specified endpoints.
// If no endpoints are provided, getEndpoints is called to provide defaults.
func getClientConfig(ctx context.Context, certificates *Certificates, endpoints ...string) (*clientv3.Config, error) {
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
	if strings.HasPrefix(endpoints[0], "https://") && certificates != nil {
		config.TLS, err = toTLSConfig(certificates)
	}
	return config, err
}

func toTLSConfig(certificates *Certificates) (*tls.Config, error) {
	clientCert, err := tls.LoadX509KeyPair(
		certificates.ServerCert,
		certificates.ServerKey,
	)
	if err != nil {
		return nil, err
	}

	pool, err := certutil.NewPool(certificates.CaCert)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{clientCert},
	}, nil
}
