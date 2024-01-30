package pluginhookclient

import (
	"context"
	"strings"

	"github.com/loft-sh/vcluster/pkg/plugin"
	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func WrapPhysicalClient(delegate client.Client) client.Client {
	return wrapClient(false, delegate)
}

func WrapVirtualClient(delegate client.Client) client.Client {
	return wrapClient(true, delegate)
}

func NewPhysicalPluginClientFactory(delegate client.NewClientFunc) client.NewClientFunc {
	return NewPluginClient(false, delegate)
}

func NewVirtualPluginClientFactory(delegate client.NewClientFunc) client.NewClientFunc {
	return NewPluginClient(true, delegate)
}

func NewPluginClient(virtual bool, delegate client.NewClientFunc) client.NewClientFunc {
	return func(config *rest.Config, options client.Options) (client.Client, error) {
		if !plugin.DefaultManager.HasPlugins() {
			return delegate(config, options)
		}

		innerClient, err := delegate(config, options)
		if err != nil {
			return nil, err
		}

		return wrapClient(virtual, innerClient), nil
	}
}

func wrapClient(virtual bool, innerClient client.Client) client.Client {
	suffix := "Physical"
	if virtual {
		suffix = "Virtual"
	}

	return &Client{
		Client: innerClient,
		suffix: suffix,
		scheme: innerClient.Scheme(),
	}
}

// Client makes sure that the Create/Update/Patch/Delete functions block until the local cache is updated
type Client struct {
	client.Client
	suffix string
	scheme *runtime.Scheme
}

func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Get(ctx, key, obj, opts...)
	}

	err := c.Client.Get(ctx, key, obj, opts...)
	if err != nil {
		return err
	}

	return plugin.DefaultManager.MutateObject(ctx, obj, "Get"+c.suffix, c.scheme)
}

func (c *Client) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.List(ctx, list, opts...)
	}

	// check if there is a hook for this operation
	gvk, err := apiutil.GVKForObject(list, c.scheme)
	if err != nil {
		return err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")
	if !plugin.DefaultManager.HasClientHooksForType(plugintypes.VersionKindType{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Type:       "Get" + c.suffix,
	}) {
		return c.Client.List(ctx, list, opts...)
	}

	err = c.Client.List(ctx, list, opts...)
	if err != nil {
		return err
	}

	objs, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	for i := range objs {
		err = plugin.DefaultManager.MutateObject(ctx, objs[i].(client.Object), "Get"+c.suffix, c.scheme)
		if err != nil {
			return err
		}
	}

	return meta.SetList(list, objs)
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Create(ctx, obj, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Create"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Create(ctx, obj, opts...)
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Patch(ctx, obj, patch, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Patch(ctx, obj, patch, opts...)
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Update(ctx, obj, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Update(ctx, obj, opts...)
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Delete(ctx, obj, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Delete"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Delete(ctx, obj, opts...)
}

// TODO: implement DeleteAllOf

func (c *Client) Status() client.StatusWriter {
	return &StatusClient{
		Client: c.Client,

		suffix: c.suffix,
		scheme: c.scheme,
	}
}

// StatusClient makes sure that the Update/Patch functions will be mutated if hooks exist
type StatusClient struct {
	Client client.Client

	suffix string
	scheme *runtime.Scheme
}

func (c *StatusClient) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Status().Create(ctx, obj, subResource, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Create"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Status().Create(ctx, obj, subResource, opts...)
}

func (c *StatusClient) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Status().Update(ctx, obj, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Status().Update(ctx, obj, opts...)
}

func (c *StatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Status().Patch(ctx, obj, patch, opts...)
	}

	err := plugin.DefaultManager.MutateObject(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Status().Patch(ctx, obj, patch, opts...)
}
