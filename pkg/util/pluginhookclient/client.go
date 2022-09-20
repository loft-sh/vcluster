package pluginhookclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/plugin/remote"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"strings"
	"time"
)

func WrapPhysicalClient(delegate client.Client) client.Client {
	return wrapClient(false, delegate)
}

func WrapVirtualClient(delegate client.Client) client.Client {
	return wrapClient(true, delegate)
}

func NewPhysicalPluginClientFactory(delegate cluster.NewClientFunc) cluster.NewClientFunc {
	return NewPluginClient(false, delegate)
}

func NewVirtualPluginClientFactory(delegate cluster.NewClientFunc) cluster.NewClientFunc {
	return NewPluginClient(true, delegate)
}

func NewPluginClient(virtual bool, delegate cluster.NewClientFunc) cluster.NewClientFunc {
	if !plugin.DefaultManager.HasPlugins() {
		return delegate
	}

	return func(cache cache.Cache, config *rest.Config, options client.Options, uncachedObjects ...client.Object) (client.Client, error) {
		innerClient, err := delegate(cache, config, options, uncachedObjects...)
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

	return executeClientHooksFor(ctx, obj, "Get"+c.suffix, c.scheme)
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
	clientHooks := plugin.DefaultManager.ClientHooksFor(plugin.VersionKindType{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Type:       "Get" + c.suffix,
	})
	if len(clientHooks) == 0 {
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
		err = executeClientHooksFor(ctx, objs[i].(client.Object), "Get"+c.suffix, c.scheme)
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

	err := executeClientHooksFor(ctx, obj, "Create"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Create(ctx, obj, opts...)
}

func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Patch(ctx, obj, patch, opts...)
	}

	err := executeClientHooksFor(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Patch(ctx, obj, patch, opts...)
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Update(ctx, obj, opts...)
	}

	err := executeClientHooksFor(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Update(ctx, obj, opts...)
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Delete(ctx, obj, opts...)
	}

	err := executeClientHooksFor(ctx, obj, "Delete"+c.suffix, c.scheme)
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

func (c *StatusClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Status().Update(ctx, obj, opts...)
	}

	err := executeClientHooksFor(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Status().Update(ctx, obj, opts...)
}

func (c *StatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if !plugin.DefaultManager.HasClientHooks() {
		return c.Client.Status().Patch(ctx, obj, patch, opts...)
	}

	err := executeClientHooksFor(ctx, obj, "Update"+c.suffix, c.scheme)
	if err != nil {
		return err
	}

	return c.Client.Status().Patch(ctx, obj, patch, opts...)
}

func executeClientHooksFor(ctx context.Context, obj client.Object, hookType string, scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return err
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	versionKindType := plugin.VersionKindType{
		APIVersion: apiVersion,
		Kind:       kind,
		Type:       hookType,
	}
	clientHooks := plugin.DefaultManager.ClientHooksFor(versionKindType)
	if len(clientHooks) > 0 {
		encodedObj, err := json.Marshal(obj)
		if err != nil {
			return errors.Wrap(err, "encode obj")
		}

		for _, clientHook := range clientHooks {
			encodedObj, err = mutateObject(ctx, versionKindType, encodedObj, clientHook)
			if err != nil {
				return err
			}
		}

		err = json.Unmarshal(encodedObj, obj)
		if err != nil {
			return errors.Wrap(err, "unmarshal obj")
		}
	}

	return nil
}

func mutateObject(ctx context.Context, versionKindType plugin.VersionKindType, obj []byte, plugin *plugin.Plugin) ([]byte, error) {
	conn, err := grpc.Dial(plugin.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error dialing plugin %s: %v", plugin.Name, err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	loghelper.New("mutate").Debugf("calling plugin %s to mutate object %s %s", plugin.Name, versionKindType.APIVersion, versionKindType.Kind)
	mutateResult, err := remote.NewPluginClient(conn).Mutate(ctx, &remote.MutateRequest{
		ApiVersion: versionKindType.APIVersion,
		Kind:       versionKindType.Kind,
		Object:     string(obj),
		Type:       versionKindType.Type,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "call plugin %s", plugin.Name)
	}

	if mutateResult.Mutated {
		return []byte(mutateResult.Object), nil
	}
	return obj, nil
}
