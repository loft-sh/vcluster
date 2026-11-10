package setup

import (
	"errors"
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestSetHostClusterVersionSkippedForPrivateNodes(t *testing.T) {
	// with private nodes (and standalone, which implies private nodes) there might not
	// be a reachable host cluster, so the host client must not be touched at all
	options := &pkgconfig.VirtualClusterConfig{}
	options.PrivateNodes.Enabled = true
	options.HostClient = nil

	controllerContext := &synccontext.ControllerContext{}
	if err := setHostClusterVersion(controllerContext, options); err != nil {
		t.Fatalf("expected no error for private nodes with nil HostClient, got %v", err)
	}
	if controllerContext.HostClusterVersion != nil {
		t.Fatalf("expected HostClusterVersion to stay nil for private nodes, got %#v", controllerContext.HostClusterVersion)
	}
}

func TestSetHostClusterVersionFetchesVersion(t *testing.T) {
	fakeClient := fake.NewClientset()
	fakeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "34",
		GitVersion: "v1.34.0",
	}

	options := &pkgconfig.VirtualClusterConfig{}
	options.HostClient = fakeClient

	controllerContext := &synccontext.ControllerContext{}
	if err := setHostClusterVersion(controllerContext, options); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if controllerContext.HostClusterVersion == nil || controllerContext.HostClusterVersion.String() != "1.34.0" {
		t.Fatalf("expected HostClusterVersion 1.34.0, got %#v", controllerContext.HostClusterVersion)
	}
}

func TestSetHostClusterVersionUnparsableVersion(t *testing.T) {
	// a discovered version that does not parse must fail startup instead of silently
	// dropping version-dependent behavior
	fakeClient := fake.NewClientset()
	fakeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		GitVersion: "vendor-build-xyz",
	}

	options := &pkgconfig.VirtualClusterConfig{}
	options.HostClient = fakeClient

	controllerContext := &synccontext.ControllerContext{}
	err := setHostClusterVersion(controllerContext, options)
	if err == nil {
		t.Fatal("expected error for unparsable host version, got nil")
	}
	if !strings.Contains(err.Error(), "parse host cluster version") {
		t.Fatalf("expected wrapped parse error, got %v", err)
	}
	if controllerContext.HostClusterVersion != nil {
		t.Fatalf("expected HostClusterVersion to stay nil on error, got %#v", controllerContext.HostClusterVersion)
	}
}

func TestSetHostClusterVersionDiscoveryError(t *testing.T) {
	fakeClient := fake.NewClientset()
	fakeClient.PrependReactor("get", "version", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("host unreachable")
	})

	options := &pkgconfig.VirtualClusterConfig{}
	options.HostClient = fakeClient

	controllerContext := &synccontext.ControllerContext{}
	err := setHostClusterVersion(controllerContext, options)
	if err == nil {
		t.Fatal("expected error when host discovery fails, got nil")
	}
	if !strings.Contains(err.Error(), "get host cluster version") {
		t.Fatalf("expected wrapped host cluster version error, got %v", err)
	}
	if controllerContext.HostClusterVersion != nil {
		t.Fatalf("expected HostClusterVersion to stay nil on error, got %#v", controllerContext.HostClusterVersion)
	}
}

func TestSetHostClusterVersionNilHostClient(t *testing.T) {
	options := &pkgconfig.VirtualClusterConfig{}
	options.HostClient = nil

	controllerContext := &synccontext.ControllerContext{}
	err := setHostClusterVersion(controllerContext, options)
	if err == nil {
		t.Fatal("expected error for nil HostClient in shared mode, got nil")
	}
}
