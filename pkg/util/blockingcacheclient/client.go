package blockingcacheclient

import (
	"context"
	"fmt"
	"time"

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
