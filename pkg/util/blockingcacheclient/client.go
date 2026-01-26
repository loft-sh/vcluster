package blockingcacheclient

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// CacheClient makes sure that the Create/Update/Patch/Delete functions block until the local cache is updated
type CacheClient struct {
	client.Client
	scheme *runtime.Scheme
}

func NewCacheClient(config *rest.Config, options client.Options) (client.Client, error) {
	// create a normal manager cache client
	cachedClient, err := defaultNewClient(config, options)
	if err != nil {
		return nil, err
	}

	return &CacheClient{
		Client: cachedClient,
		scheme: options.Scheme,
	}, nil
}

// defaultNewClient creates the default caching client
func defaultNewClient(config *rest.Config, options client.Options) (client.Client, error) {
	if options.Cache == nil {
		return nil, fmt.Errorf("blockingcacheclient should always be created with a cache (options.Cache)")
	}
	options.Cache.Unstructured = true

	return client.New(config, options)
}

func (c *CacheClient) poll(ctx context.Context, obj runtime.Object, condition func(newObj client.Object, oldAccessor metav1.Object) (bool, error)) error {
	_, ok := obj.(*unstructured.Unstructured)
	if ok {
		return nil
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return nil
	}

	newObj, err := c.scheme.New(gvk)
	if err != nil {
		return nil
	}

	return wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*2, true, func(context.Context) (bool, error) {
		return condition(newObj.(client.Object), accessor)
	})
}

func (c *CacheClient) blockCreate(ctx context.Context, obj client.Object) error {
	return c.poll(ctx, obj, func(newObj client.Object, oldAccessor metav1.Object) (bool, error) {
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: oldAccessor.GetNamespace(), Name: oldAccessor.GetName()}, newObj)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				return true, nil
			} else if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		}

		return true, nil
	})
}

func (c *CacheClient) blockUpdate(ctx context.Context, obj client.Object) error {
	return c.poll(ctx, obj, func(newObj client.Object, oldAccessor metav1.Object) (bool, error) {
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: oldAccessor.GetNamespace(), Name: oldAccessor.GetName()}, newObj)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				return true, nil
			} else if !kerrors.IsNotFound(err) {
				return false, err
			}

			return true, nil
		}

		newAccessor, err := meta.Accessor(newObj)
		if err != nil {
			return false, err
		}

		return oldAccessor.GetUID() != newAccessor.GetUID() || newAccessor.GetResourceVersion() >= oldAccessor.GetResourceVersion(), nil
	})
}

func (c *CacheClient) blockDelete(ctx context.Context, obj runtime.Object) error {
	return c.poll(ctx, obj, func(newObj client.Object, oldAccessor metav1.Object) (bool, error) {
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: oldAccessor.GetNamespace(), Name: oldAccessor.GetName()}, newObj)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				return true, nil
			} else if !kerrors.IsNotFound(err) {
				return false, err
			}

			return true, nil
		}

		newAccessor, err := meta.Accessor(newObj)
		if err != nil {
			return false, err
		}
		return oldAccessor.GetUID() != newAccessor.GetUID() || newAccessor.GetDeletionTimestamp() != nil, nil
	})
}

func (c *CacheClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	err := c.Client.Create(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.blockCreate(ctx, obj)
}

func (c *CacheClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	err := c.Client.Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}

	return c.blockUpdate(ctx, obj)
}

func (c *CacheClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	err := c.Client.Update(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.blockUpdate(ctx, obj)
}

func (c *CacheClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	err := c.Client.Delete(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.blockDelete(ctx, obj)
}

// blockApply waits until the applied object appears in the cache with the expected state.
// It ensures that subsequent reads will see the changes made by the Apply operation.
//
// The function polls the cache until one of the following conditions is met:
//   - The object's UID has changed (object was recreated)
//   - The object's Generation has increased (spec was updated)
//   - The object's ResourceVersion has changed (any update occurred)
func (c *CacheClient) blockApply(ctx context.Context, obj runtime.ApplyConfiguration) error {
	clientObj, err := util.ExtractClientObjectFromApplyConfiguration(obj)
	if err != nil {
		// If we can't extract metadata, we can't verify cache updates.
		// Return nil to avoid blocking indefinitely for a valid apply we just can't track.
		return nil
	}

	// Poll the cache until the condition is met.
	// The poll function will call our condition function repeatedly until it returns true or an error.
	return c.poll(ctx, clientObj, func(newObj client.Object, oldAccessor metav1.Object) (bool, error) {
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: oldAccessor.GetNamespace(), Name: oldAccessor.GetName()}, newObj)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				// If the type is not registered in the scheme, we consider it a success
				// to avoid blocking indefinitely.
				return true, nil
			} else if !kerrors.IsNotFound(err) {
				// Return other errors (e.g. connection issues, permission errors) to stop polling.
				return false, err
			}
			// If the object is not found, keep polling (return false, nil).
			// For an Apply operation, we expect the object to exist eventually.
			return false, nil
		}

		newAccessor, err := meta.Accessor(newObj)
		if err != nil {
			return false, err
		}

		// Condition 1: UID changed - object was deleted and recreated
		// Condition 2: Generation increased - spec was updated
		// Condition 3: ResourceVersion changed - any update occurred (metadata, spec, or status)
		// If any of these conditions are true, the Apply operation is reflected in the cache.
		return oldAccessor.GetUID() != newAccessor.GetUID() ||
			newAccessor.GetGeneration() > oldAccessor.GetGeneration() ||
			newAccessor.GetResourceVersion() != oldAccessor.GetResourceVersion(), nil
	})
}

// TODO: implement DeleteAllOf

func (c *CacheClient) Status() client.StatusWriter {
	return &CacheStatusClient{
		Cache: c,
	}
}

// CacheStatusClient makes sure that the Update/Patch functions block until the local cache is updated
type CacheStatusClient struct {
	Cache *CacheClient
}

func (c *CacheStatusClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	err := c.Cache.Client.Status().Create(ctx, obj, subResource, opts...)
	if err != nil {
		return err
	}

	return c.Cache.blockCreate(ctx, obj)
}

func (c *CacheStatusClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	err := c.Cache.Client.Status().Update(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.Cache.blockUpdate(ctx, obj)
}

func (c *CacheStatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	err := c.Cache.Client.Status().Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}

	return c.Cache.blockUpdate(ctx, obj)
}

func (c *CacheStatusClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.SubResourceApplyOption) error {
	err := c.Cache.Client.Status().Apply(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.Cache.blockApply(ctx, obj)
}
