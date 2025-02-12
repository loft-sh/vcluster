package etcd

import (
	"context"
	"errors"
	"fmt"

	vconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Value struct {
	Key      []byte
	Data     []byte
	Modified int64
}

var ErrNotFound = errors.New("etcdwrapper: key not found")

type Client interface {
	List(ctx context.Context, key string) ([]Value, error)
	ListStream(ctx context.Context, key string) <-chan *ValueOrError
	Watch(ctx context.Context, key string) clientv3.WatchChan
	Get(ctx context.Context, key string) (Value, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
	Close() error
}

type client struct {
	c *clientv3.Client
}

func GetEtcdEndpoint(vConfig *config.VirtualClusterConfig) (string, *Certificates) {
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
		} else if vConfig.ControlPlane.BackingStore.Etcd.Deploy.Service.Enabled {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd:2379"
		} else {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd-headless:2379"
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

	return etcdEndpoints, etcdCertificates
}

func NewFromConfig(ctx context.Context, vConfig *config.VirtualClusterConfig) (Client, error) {
	etcdEndpoints, etcdCertificates := GetEtcdEndpoint(vConfig)
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

type ValueOrError struct {
	Value Value
	Error error
}

func (c *client) ListStream(ctx context.Context, key string) <-chan *ValueOrError {
	retChan := make(chan *ValueOrError, 1000)

	originalKey := key
	go func() {
		defer close(retChan)

		first := true
		for {
			options := []clientv3.OpOption{clientv3.WithRev(0), clientv3.WithLimit(1000)}
			if first {
				options = append(options, clientv3.WithPrefix())
			} else {
				options = append(options, clientv3.WithRange(string(getPrefix([]byte(originalKey)))))
			}

			resp, err := c.c.Get(
				ctx,
				key,
				options...,
			)
			if err != nil {
				retChan <- &ValueOrError{Error: err}
				return
			} else if len(resp.Kvs) == 0 {
				return
			}

			for _, kv := range resp.Kvs {
				retChan <- &ValueOrError{
					Value: Value{
						Key:      kv.Key,
						Data:     kv.Value,
						Modified: kv.ModRevision,
					},
				}

				key = string(kv.Key)
				first = false
			}

			if !resp.More {
				break
			}
		}
	}()

	return retChan
}

func (c *client) Watch(ctx context.Context, key string) clientv3.WatchChan {
	return c.c.Watch(ctx, key, clientv3.WithPrefix(), clientv3.WithPrevKV(), clientv3.WithProgressNotify())
}

func (c *client) List(ctx context.Context, key string) ([]Value, error) {
	resp, err := c.c.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithRev(int64(0)))
	if err != nil {
		return nil, err
	}

	var vals []Value
	for _, kv := range resp.Kvs {
		vals = append(vals, Value{
			Key:      kv.Key,
			Data:     kv.Value,
			Modified: kv.ModRevision,
		})
	}

	return vals, nil
}

func (c *client) Get(ctx context.Context, key string) (Value, error) {
	resp, err := c.c.Get(ctx, key)
	if err != nil {
		return Value{}, err
	}

	if len(resp.Kvs) == 1 {
		return Value{
			Key:      resp.Kvs[0].Key,
			Data:     resp.Kvs[0].Value,
			Modified: resp.Kvs[0].ModRevision,
		}, nil
	}

	return Value{}, ErrNotFound
}

func (c *client) Put(ctx context.Context, key string, value []byte) error {
	val, err := c.Get(ctx, key)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if val.Modified == 0 {
		return c.Create(ctx, key, value)
	}
	return c.Update(ctx, key, val.Modified, value)
}

func (c *client) Create(ctx context.Context, key string, value []byte) error {
	resp, err := c.c.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(value))).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("key exists")
	}

	return nil
}

func (c *client) Update(ctx context.Context, key string, revision int64, value []byte) error {
	resp, err := c.c.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", revision)).
		Then(clientv3.OpPut(key, string(value))).
		Else(clientv3.OpGet(key)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return fmt.Errorf("revision %d doesnt match", revision)
	}

	return nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	_, err := c.c.Txn(ctx).
		Then(
			clientv3.OpGet(key),
			clientv3.OpDelete(key),
		).
		Commit()
	return err
}

func (c *client) DeletePrefix(ctx context.Context, prefix string) error {
	_, err := c.c.Delete(ctx, prefix, clientv3.WithPrefix())
	return err
}

func (c *client) Close() error {
	return c.c.Close()
}

func getPrefix(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return end
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to WithFromKey policy
	return []byte{0}
}
