package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/etcd"
)

var mappingsPrefix = "/vcluster/mappings/"

func NewEtcdBackend(etcdClient etcd.Client) Backend {
	return &etcdBackend{
		etcdClient: etcdClient,
	}
}

type etcdBackend struct {
	etcdClient etcd.Client
}

func (m *etcdBackend) List(ctx context.Context) ([]*Mapping, error) {
	mappings, err := m.etcdClient.List(ctx, mappingsPrefix, 0)
	if err != nil {
		return nil, fmt.Errorf("list mappings")
	}

	retMappings := make([]*Mapping, 0, len(mappings))
	for _, kv := range mappings {
		retMapping := &Mapping{}
		err = json.Unmarshal(kv.Data, retMapping)
		if err != nil {
			return nil, fmt.Errorf("parse mapping %s: %w", string(kv.Key), err)
		}

		retMappings = append(retMappings, retMapping)
	}

	return retMappings, nil
}

func (m *etcdBackend) Watch(_ context.Context) <-chan BackendWatchResponse {
	return nil
}

func (m *etcdBackend) Save(ctx context.Context, mapping *Mapping) error {
	mappingBytes, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	return m.etcdClient.Put(ctx, mappingToKey(mapping.String()), mappingBytes)
}

func (m *etcdBackend) Delete(ctx context.Context, mapping *Mapping) error {
	return m.etcdClient.Delete(ctx, mappingToKey(mapping.String()), 0)
}

func mappingToKey(key string) string {
	sha := sha256.Sum256([]byte(key))
	return strings.ToLower(fmt.Sprintf("%s", hex.EncodeToString(sha[0:])[0:20]))
}
