package v2

import (
	"context"
	"errors"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/loft-sh/vcluster/pkg/plugin/v2/pluginv2"
	"google.golang.org/grpc"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion: 1,

	MagicCookieKey:   "VCLUSTER_PLUGIN",
	MagicCookieValue: "vcluster",
}

// GRPCProviderPlugin is an implementation of the
// github.com/hashicorp/go-plugin#Plugin and
// github.com/hashicorp/go-plugin#GRPCPlugin interfaces
type GRPCProviderPlugin struct{}

// Server always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCProviderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("vcluster-plugin only implements gRPC clients")
}

// Client always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *GRPCProviderPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("vcluster-plugin only implements gRPC clients")
}

// GRPCClient always returns an error; we're only implementing the server half
// of the interface.
func (p *GRPCProviderPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, clientConn *grpc.ClientConn) (interface{}, error) {
	return pluginv2.NewPluginClient(clientConn), nil
}

// GRPCServer registers the gRPC provider server with the gRPC server that
// go-plugin is standing up.
func (p *GRPCProviderPlugin) GRPCServer(_ *plugin.GRPCBroker, _ *grpc.Server) error {
	return errors.New("vcluster-plugin only implements gRPC clients")
}
