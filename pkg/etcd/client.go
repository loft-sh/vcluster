package etcd

import (
	"context"
	"errors"
	"fmt"

	vconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Value struct {
	Key  []byte
	Data []byte
}

var ErrNotFound = errors.New("etcdwrapper: key not found")

type Client interface {
	List(ctx context.Context, key string) ([]Value, error)
	Watch(ctx context.Context, key string) clientv3.WatchChan
	Get(ctx context.Context, key string) (Value, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	Close() error
}

type client struct {
	c *clientv3.Client
}

func NewFromConfig(ctx context.Context, vConfig *config.VirtualClusterConfig) (Client, error) {
	// start kine embedded or external
	var (
		etcdEndpoints    string
		etcdCertificates *Certificates
	)

	// handle different backing store's
	if vConfig.ControlPlane.BackingStore.Etcd.Deploy.Enabled || vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
		// embedded or deployed etcd
		etcdCertificates = &Certificates{
			CaCert:     "/data/pki/etcd/ca.crt",
			ServerCert: "/data/pki/apiserver-etcd-client.crt",
			ServerKey:  "/data/pki/apiserver-etcd-client.key",
		}
		if vConfig.Distro() == vconfig.K0SDistro {
			etcdCertificates = &Certificates{
				CaCert:     "/data/k0s/pki/etcd/ca.crt",
				ServerCert: "/data/k0s/pki/apiserver-etcd-client.crt",
				ServerKey:  "/data/k0s/pki/apiserver-etcd-client.key",
			}
		}

		if vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			etcdEndpoints = "https://127.0.0.1:2379"
		} else {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd:2379"
		}
	} else if vConfig.Distro() == vconfig.K8SDistro {
		etcdEndpoints = constants.K8sKineEndpoint
	} else if vConfig.Distro() == vconfig.K3SDistro {
		etcdEndpoints = constants.K3sKineEndpoint
	} else if vConfig.Distro() == vconfig.K0SDistro {
		if (vConfig.ControlPlane.BackingStore.Database.Embedded.Enabled && vConfig.ControlPlane.BackingStore.Database.Embedded.DataSource != "") ||
			vConfig.ControlPlane.BackingStore.Database.External.Enabled {
			etcdEndpoints = constants.K0sKineEndpoint
		} else {
			etcdEndpoints = "https://127.0.0.1:2379"
			etcdCertificates = &Certificates{
				CaCert:     "/data/k0s/pki/etcd/ca.crt",
				ServerCert: "/data/k0s/pki/apiserver-etcd-client.crt",
				ServerKey:  "/data/k0s/pki/apiserver-etcd-client.key",
			}
		}
	}

	return New(ctx, etcdCertificates, etcdEndpoints)
}

func New(ctx context.Context, certificates *Certificates, endpoints ...string) (Client, error) {
	err := WaitForEtcd(ctx, certificates, endpoints...)
	if err != nil {
		return nil, err
	}

	etcdClient, err := GetEtcdClient(ctx, certificates, endpoints...)
	if err != nil {
		return nil, err
	}

	return &client{
		c: etcdClient,
	}, nil
}

func (c *client) Watch(ctx context.Context, key string) clientv3.WatchChan {
	return c.c.Watch(ctx, key, clientv3.WithPrefix(), clientv3.WithPrevKV(), clientv3.WithProgressNotify())
}

func (c *client) List(ctx context.Context, key string) ([]Value, error) {
	resp, err := c.c.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithRev(0))
	if err != nil {
		return nil, err
	}

	var vals []Value
	for _, kv := range resp.Kvs {
		vals = append(vals, Value{
			Key:  kv.Key,
			Data: kv.Value,
		})
	}

	return vals, nil
}

func (c *client) Get(ctx context.Context, key string) (Value, error) {
	resp, err := c.c.Get(ctx, key, clientv3.WithRev(0))
	if err != nil {
		return Value{}, fmt.Errorf("etcd get: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return Value{}, ErrNotFound
	}

	if len(resp.Kvs) == 1 {
		return Value{
			Key:  resp.Kvs[0].Key,
			Data: resp.Kvs[0].Value,
		}, nil
	}

	highestRevision := &mvccpb.KeyValue{ModRevision: -1}
	for _, kv := range resp.Kvs {
		if kv.ModRevision > highestRevision.ModRevision {
			highestRevision = kv
		}
	}

	return Value{
		Key:  highestRevision.Key,
		Data: highestRevision.Value,
	}, nil
}

func (c *client) Put(ctx context.Context, key string, value []byte) error {
	_, err := c.Get(ctx, key)
	if err == nil {
		_, err := c.c.Put(ctx, key, string(value), clientv3.WithIgnoreLease())
		return err
	} else {
		_, err := c.c.Put(ctx, key, string(value))
		return err
	}
}

func (c *client) Delete(ctx context.Context, key string) error {
	_, err := c.c.Delete(ctx, key)
	return err
}

func (c *client) Close() error {
	return c.c.Close()
}
