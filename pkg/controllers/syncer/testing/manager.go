package testing

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/loft-sh/vcluster/pkg/util/log"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func newFakeManager(client *testingutil.FakeIndexClient) ctrl.Manager {
	return &fakeManager{client: client}
}

type fakeManager struct {
	client *testingutil.FakeIndexClient
}

func (f *fakeManager) SetFields(interface{}) error { return nil }

func (f *fakeManager) GetConfig() *rest.Config { return &rest.Config{Host: "127.0.0.1"} }

func (f *fakeManager) GetScheme() *runtime.Scheme { return f.client.Scheme() }

func (f *fakeManager) GetClient() client.Client { return f.client }

func (f *fakeManager) GetFieldIndexer() client.FieldIndexer { return f.client }

func (f *fakeManager) GetCache() cache.Cache { return nil }

func (f *fakeManager) GetEventRecorderFor(name string) record.EventRecorder {
	return &fakeEventBroadcaster{}
}

func (f *fakeManager) GetRESTMapper() meta.RESTMapper { return nil }

func (f *fakeManager) GetAPIReader() client.Reader { return f.client }

func (f *fakeManager) Start(ctx context.Context) error { return nil }

func (f *fakeManager) Add(manager.Runnable) error { return nil }

func (f *fakeManager) Elected() <-chan struct{} { return make(chan struct{}) }

func (f *fakeManager) AddMetricsExtraHandler(path string, handler http.Handler) error { return nil }

func (f *fakeManager) AddHealthzCheck(name string, check healthz.Checker) error { return nil }

func (f *fakeManager) AddReadyzCheck(name string, check healthz.Checker) error { return nil }

func (f *fakeManager) GetWebhookServer() *webhook.Server { return nil }

func (f *fakeManager) GetLogger() logr.Logger { return log.NewLog(0) }

func (f *fakeManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return v1alpha1.ControllerConfigurationSpec{}
}
