package blockingcacheclient

import (
	"context"
	"fmt"
	"strconv"
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
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	var newObj client.Object
	if u, ok := obj.(*unstructured.Unstructured); ok {
		// For unstructured (e.g. from ApplyConfiguration), we cannot use c.scheme.New(gvk)
		// as the type may not be in the scheme. Use a new Unstructured with the same GVK
		// so that Get() and the condition can run, and cache blocking actually happens.
		newObj = &unstructured.Unstructured{}
		newObj.GetObjectKind().SetGroupVersionKind(u.GetObjectKind().GroupVersionKind())
	} else {
		gvk, err := apiutil.GVKForObject(obj, c.scheme)
		if err != nil {
			return nil
		}

		created, err := c.scheme.New(gvk)
		if err != nil {
			return nil
		}
		newObj = created.(client.Object)
	}

	return wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*2, true, func(context.Context) (bool, error) {
		return condition(newObj, accessor)
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

func (c *CacheClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	err := c.Client.Apply(ctx, obj, opts...)
	if err != nil {
		return err
	}

	return c.blockApply(ctx, obj)
}

func (c *CacheClient) blockApply(ctx context.Context, obj runtime.ApplyConfiguration) error {
	applied, err := util.ExtractClientObjectFromApplyConfiguration(obj)
	if err != nil {
		return err
	}

	return c.poll(ctx, applied, func(newObj client.Object, appliedAccessor metav1.Object) (bool, error) {
		err := c.Client.Get(ctx, types.NamespacedName{Namespace: appliedAccessor.GetNamespace(), Name: appliedAccessor.GetName()}, newObj)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				return true, nil
			} else if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		}

		newAccessor, err := meta.Accessor(newObj)
		if err != nil {
			return false, err
		}

		return resourceVersionReached(newAccessor.GetResourceVersion(), appliedAccessor.GetResourceVersion()), nil
	})
}

// resourceVersionReached reports whether the cache has observed the write that
// returned the given resource version. kube-apiserver versions are a global
// decimal counter, so those compare numerically, which also settles a delete
// and recreate: an older incarnation has a lower version, a replacement
// written after the apply a higher one. Any other version is opaque, and the
// only ordering-free evidence is the cache serving exactly that version.
func resourceVersionReached(cached, returned string) bool {
	cachedRV, cachedErr := strconv.ParseUint(cached, 10, 64)
	returnedRV, returnedErr := strconv.ParseUint(returned, 10, 64)
	if cachedErr == nil && returnedErr == nil {
		return cachedRV >= returnedRV
	}

	return cached == returned
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
