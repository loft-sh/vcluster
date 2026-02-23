package etcd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
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
	Put(ctx context.Context, key string, value []byte) (int64, error)
	Delete(ctx context.Context, key string) error
	DeletePrefix(ctx context.Context, prefix string) error
	Compact(ctx context.Context, revision int64) error
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
			CaCert:     filepath.Join(constants.PKIDir, "etcd", "ca.crt"),
			ServerCert: filepath.Join(constants.PKIDir, "apiserver-etcd-client.crt"),
			ServerKey:  filepath.Join(constants.PKIDir, "apiserver-etcd-client.key"),
		}

		if vConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			etcdEndpoints = "https://127.0.0.1:2379"
		} else if vConfig.ControlPlane.BackingStore.Etcd.Deploy.Service.Enabled {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd:2379"
		} else {
			etcdEndpoints = "https://" + vConfig.Name + "-etcd-headless:2379"
		}
	} else if vConfig.ControlPlane.BackingStore.Etcd.External.Enabled {
		etcdEndpoints = "https://" + strings.TrimPrefix(vConfig.ControlPlane.BackingStore.Etcd.External.Endpoint, "https://")
		etcdCertificates = &Certificates{
			CaCert:     vConfig.ControlPlane.BackingStore.Etcd.External.TLS.CaFile,
			ServerCert: vConfig.ControlPlane.BackingStore.Etcd.External.TLS.CertFile,
			ServerKey:  vConfig.ControlPlane.BackingStore.Etcd.External.TLS.KeyFile,
		}
	} else {
		etcdEndpoints = constants.K8sKineEndpoint
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

	etcdClient, err := GetEtcdClient(ctx, zap.L().Named("etcd-client"), certificates, endpoints...)
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

func (c *client) ListStream(ctx context.Context, prefix string) <-chan *ValueOrError {
	return listStream(ctx, prefix, c.c.Get)
}

func listStream(
	ctx context.Context,
	prefix string,
	getFn func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error),
) <-chan *ValueOrError {
	retChan := make(chan *ValueOrError, 1000)

	go func() {
		defer close(retChan)

		var revision int64
		first := true
		rangeEnd := clientv3.GetPrefixRangeEnd(prefix)
		startKey := prefix

		for {
			options := []clientv3.OpOption{
				clientv3.WithLimit(1000),
				clientv3.WithRange(rangeEnd),
			}

			if first {
				// read at current revision
				options = append(options, clientv3.WithRev(0))
			} else {
				options = append(options, clientv3.WithRev(revision))
			}

			resp, err := getFn(ctx, startKey, options...)
			if err != nil {
				retChan <- &ValueOrError{Error: err}
				return
			} else if len(resp.Kvs) == 0 {
				return
			}
			if first {
				revision = resp.Header.Revision
				first = false
			}

			for _, kv := range resp.Kvs {
				retChan <- &ValueOrError{
					Value: Value{
						Key:      kv.Key,
						Data:     kv.Value,
						Modified: kv.ModRevision,
					},
				}
			}
			// move to the next page
			// advance past last key to avoid duplicates
			startKey = nextStartKey(resp.Kvs[len(resp.Kvs)-1].Key)

			if !resp.More {
				break
			}
		}
	}()

	return retChan
}

func (c *client) Compact(ctx context.Context, revision int64) error {
	_, err := c.c.Compact(ctx, revision, clientv3.WithCompactPhysical())
	return err
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

func (c *client) Put(ctx context.Context, key string, value []byte) (int64, error) {
	val, err := c.Get(ctx, key)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return 0, err
	}
	if val.Modified == 0 {
		return c.Create(ctx, key, value)
	}
	return c.Update(ctx, key, val.Modified, value)
}

func (c *client) Create(ctx context.Context, key string, value []byte) (int64, error) {
	resp, err := c.c.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(value))).
		Commit()
	if err != nil {
		return 0, err
	}
	if !resp.Succeeded {
		return 0, errors.New("key exists")
	}

	return resp.Header.Revision, nil
}

func (c *client) Update(ctx context.Context, key string, revision int64, value []byte) (int64, error) {
	resp, err := c.c.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", revision)).
		Then(clientv3.OpPut(key, string(value))).
		Else(clientv3.OpGet(key)).
		Commit()
	if err != nil {
		return 0, err
	}
	if !resp.Succeeded {
		return 0, fmt.Errorf("revision %d doesnt match", revision)
	}

	return resp.Header.Revision, nil
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
	_, err := c.c.Delete(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(int64(0)))
	return err
}

func (c *client) Close() error {
	return c.c.Close()
}

func nextStartKey(key []byte) string {
	b := make([]byte, len(key)+1)
	copy(b, key)
	// Compute the next lexicographic key strictly after the current one.
	// Example: "foo" -> "foo\x00". This is used for snapshot pagination to avoid duplicate keys between pages.
	b[len(key)] = 0x00
	return string(b)
}
