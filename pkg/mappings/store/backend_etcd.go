package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/loft-sh/vcluster/pkg/etcd"
	"go.etcd.io/etcd/api/v3/mvccpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

var MappingsPrefix = "/vcluster/mappings/"

func NewEtcdBackend(etcdClient etcd.Client) Backend {
	return &etcdBackend{
		etcdClient: etcdClient,
	}
}

type etcdBackend struct {
	etcdClient etcd.Client
}

func (m *etcdBackend) List(ctx context.Context) ([]*Mapping, error) {
	mappings, err := m.etcdClient.List(ctx, MappingsPrefix)
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
	watchChan := m.etcdClient.Watch(ctx, MappingsPrefix)
	go func() {
		defer close(responseChan)

		for event := range watchChan {
			switch {
			case event.Canceled:
				responseChan <- BackendWatchResponse{
					Err: event.Err(),
				}
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

						// we only send the reconstructed mapping to the consumer
						if eventType == BackendWatchEventTypeDelete {
							retEvents = append(retEvents, &BackendWatchEvent{
								Type:    BackendWatchEventTypeDeleteReconstructed,
								Mapping: reconstructNameMappingFromKey(string(singleEvent.Kv.Key)),
							})
						}

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

	_, err = m.etcdClient.Put(ctx, mappingToKey(mapping), mappingBytes)
	return err
}

func (m *etcdBackend) Delete(ctx context.Context, mapping *Mapping) error {
	return m.etcdClient.Delete(ctx, mappingToKey(mapping))
}

func reconstructNameMappingFromKey(key string) *Mapping {
	retMapping := &Mapping{}
	trimmedKey := strings.TrimPrefix(key, MappingsPrefix)
	splittedKey := strings.Split(trimmedKey, "/")
	if splittedKey[0] == "v1" {
		retMapping.GroupVersionKind = corev1.SchemeGroupVersion.WithKind(splittedKey[1])
		if len(splittedKey) == 4 {
			retMapping.VirtualName = types.NamespacedName{
				Namespace: splittedKey[2],
				Name:      splittedKey[3],
			}
		} else {
			retMapping.VirtualName = types.NamespacedName{
				Name: splittedKey[2],
			}
		}
	} else {
		retMapping.GroupVersionKind = schema.GroupVersionKind{
			Group:   splittedKey[0],
			Version: splittedKey[1],
			Kind:    splittedKey[2],
		}
		if len(splittedKey) == 5 {
			retMapping.VirtualName = types.NamespacedName{
				Namespace: splittedKey[3],
				Name:      splittedKey[4],
			}
		} else {
			retMapping.VirtualName = types.NamespacedName{
				Name: splittedKey[3],
			}
		}
	}

	return retMapping
}

func mappingToKey(mapping *Mapping) string {
	nameNamespace := mapping.VirtualName.Name
	if mapping.VirtualName.Namespace != "" {
		nameNamespace = mapping.VirtualName.Namespace + "/" + nameNamespace
	}

	return path.Join(MappingsPrefix, mapping.GroupVersion().String(), strings.ToLower(mapping.Kind), nameNamespace)
}
