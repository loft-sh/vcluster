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
		return nil, fmt.Errorf("etcd backend: list mappings: %w", err)
	}

	retMappings := make([]*Mapping, 0, len(mappings))
	for _, kv := range mappings {
		retMapping := &Mapping{}
		err = json.Unmarshal(kv.Data, retMapping)
		if err != nil {
			return nil, fmt.Errorf("etcd backend: parse mapping %s: %w", string(kv.Key), err)
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
			switch {
			case event.Canceled:
				responseChan <- BackendWatchResponse{
					Err: event.Err(),
				}
			case event.IsProgressNotify():
				klog.FromContext(ctx).V(1).Info("received progress notify from etcd")
			case len(event.Events) > 0:
				retEvents := make([]*BackendWatchEvent, 0, len(event.Events))
				for _, singleEvent := range event.Events {
					var eventType BackendWatchEventType
					switch singleEvent.Type {
					case mvccpb.PUT:
						eventType = BackendWatchEventTypeUpdate
					case mvccpb.DELETE:
						eventType = BackendWatchEventTypeDelete
					default:
						continue
					}

					// parse mapping
					retMapping := &Mapping{}

					value := singleEvent.Kv.Value
					if len(value) == 0 && singleEvent.Type == mvccpb.DELETE && singleEvent.PrevKv != nil {
						value = singleEvent.PrevKv.Value
					}

					err := json.Unmarshal(value, retMapping)
					if err != nil {
						klog.FromContext(ctx).Info(
							"etcd backend: Error decoding event",
							"key", string(singleEvent.Kv.Key),
							"singleEventValue", string(singleEvent.Kv.Value),
							"eventType", eventType,
							"error", err.Error(),
						)
						// FIXME(ThomasK33): This leads to mapping leaks. Etcd might have
						// already compacted the previous version. Thus we would never
						// receive any information of the mapping that was deleted apart from its keys.
						// And because there is no mapping, we are omitting deleting it from the mapping stores.
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