package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/loft-sh/vcluster/pkg/etcd"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"k8s.io/klog/v2"
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
	mappings, err := m.etcdClient.List(ctx, mappingsPrefix)
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

func (m *etcdBackend) Watch(ctx context.Context) <-chan BackendWatchResponse {
	responseChan := make(chan BackendWatchResponse)
	watchChan := m.etcdClient.Watch(ctx, mappingsPrefix)
	go func() {
		defer close(responseChan)

		for event := range watchChan {
			if event.Canceled {
				responseChan <- BackendWatchResponse{
					Err: event.Err(),
				}
			} else if len(event.Events) > 0 {
				retEvents := make([]*BackendWatchEvent, 0, len(event.Events))
				for _, singleEvent := range event.Events {
					var eventType BackendWatchEventType
					if singleEvent.Type == mvccpb.PUT {
						eventType = BackendWatchEventTypeUpdate
					} else if singleEvent.Type == mvccpb.DELETE {
						eventType = BackendWatchEventTypeDelete
					} else {
						continue
					}

					// parse mapping
					retMapping := &Mapping{}
					err := json.Unmarshal(singleEvent.Kv.Value, retMapping)
					if err != nil {
						klog.FromContext(ctx).Info("Error decoding event", "key", string(singleEvent.Kv.Key), "error", err.Error())
						continue
					}

					retEvents = append(retEvents, &BackendWatchEvent{
						Type:    eventType,
						Mapping: retMapping,
					})
				}

				responseChan <- BackendWatchResponse{
					Events: retEvents,
				}
			}
		}
	}()

	return responseChan
}

func (m *etcdBackend) Save(ctx context.Context, mapping *Mapping) error {
	mappingBytes, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	return m.etcdClient.Put(ctx, mappingToKey(mapping), mappingBytes)
}

func (m *etcdBackend) Delete(ctx context.Context, mapping *Mapping) error {
	return m.etcdClient.Delete(ctx, mappingToKey(mapping))
}

func mappingToKey(mapping *Mapping) string {
	nameNamespace := mapping.VirtualName.Name
	if mapping.VirtualName.Namespace != "" {
		nameNamespace = mapping.VirtualName.Namespace + "/" + nameNamespace
	}

	return path.Join(mappingsPrefix, mapping.GroupVersion().String(), strings.ToLower(mapping.Kind), nameNamespace)
}
