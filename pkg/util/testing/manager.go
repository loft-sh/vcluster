package testing

import (
	"context"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/loft-sh/vcluster/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func NewFakeManager(client *FakeIndexClient) ctrl.Manager {
	return &fakeManager{client: client}
}

type fakeManager struct {
	client *FakeIndexClient
}

func (f *fakeManager) SetFields(_ interface{}) error { return nil }

func (f *fakeManager) GetConfig() *rest.Config { return &rest.Config{Host: "unit-test-client"} }

func (f *fakeManager) GetScheme() *runtime.Scheme { return f.client.Scheme() }

func (f *fakeManager) GetClient() client.Client { return f.client }

func (f *fakeManager) GetFieldIndexer() client.FieldIndexer { return f.client }

func (f *fakeManager) GetCache() cache.Cache { return &fakeCache{FakeIndexClient: f.client} }

func (f *fakeManager) GetEventRecorderFor(string) record.EventRecorder {
	return &fakeEventBroadcaster{}
}

func (f *fakeManager) GetRESTMapper() meta.RESTMapper { return nil }

func (f *fakeManager) GetAPIReader() client.Reader { return f.client }

func (f *fakeManager) Start(context.Context) error { return nil }

func (f *fakeManager) Add(manager.Runnable) error { return nil }

func (f *fakeManager) Elected() <-chan struct{} { return make(chan struct{}) }

func (f *fakeManager) AddMetricsExtraHandler(string, http.Handler) error { return nil }

func (f *fakeManager) AddHealthzCheck(string, healthz.Checker) error { return nil }

func (f *fakeManager) AddReadyzCheck(string, healthz.Checker) error { return nil }

func (f *fakeManager) GetWebhookServer() webhook.Server { return webhook.NewServer(webhook.Options{}) }

func (f *fakeManager) GetLogger() logr.Logger { return log.NewLog(0) }

func (f *fakeManager) GetControllerOptions() config.Controller {
	return config.Controller{}
}

func (f *fakeManager) GetHTTPClient() *http.Client {
	return &http.Client{}
}

func (f *fakeManager) AddMetricsServerExtraHandler(_ string, _ http.Handler) error {
	return nil
}

type fakeCache struct {
	*FakeIndexClient
}

func (f *fakeCache) GetInformer(_ context.Context, _ client.Object, _ ...cache.InformerGetOption) (cache.Informer, error) {
	return &fakeInformer{}, nil
}

func (f *fakeCache) GetInformerForKind(_ context.Context, _ schema.GroupVersionKind, _ ...cache.InformerGetOption) (cache.Informer, error) {
	return &fakeInformer{}, nil
}

func (f *fakeCache) RemoveInformer(_ context.Context, _ client.Object) error {
	return nil
}

func (f *fakeCache) Start(_ context.Context) error {
	return nil
}

func (f *fakeCache) WaitForCacheSync(_ context.Context) bool {
	return true
}

func (f *fakeCache) IndexField(ctx context.Context, obj client.Object, key string, extractValue client.IndexerFunc) error {
	return f.FakeIndexClient.IndexField(ctx, obj, key, extractValue)
}

type fakeInformer struct{}

func (f *fakeInformer) AddEventHandler(_ toolscache.ResourceEventHandler) (toolscache.ResourceEventHandlerRegistration, error) {
	//nolint:nilnil
	return nil, nil
}

func (f *fakeInformer) AddEventHandlerWithResyncPeriod(_ toolscache.ResourceEventHandler, _ time.Duration) (toolscache.ResourceEventHandlerRegistration, error) {
	//nolint:nilnil
	return nil, nil
}

func (f *fakeInformer) RemoveEventHandler(_ toolscache.ResourceEventHandlerRegistration) error {
	return nil
}

func (f *fakeInformer) AddIndexers(_ toolscache.Indexers) error {
	return nil
}

func (f *fakeInformer) HasSynced() bool {
	return true
}

func (f *fakeInformer) IsStopped() bool {
	return false
}

type fakeEventBroadcaster struct{}

func (f *fakeEventBroadcaster) StartEventWatcher(_ func(*corev1.Event)) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartRecordingToSink(_ record.EventSink) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartLogging(_ func(format string, args ...interface{})) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) StartStructuredLogging(_ klog.Level) watch.Interface {
	return nil
}

func (f *fakeEventBroadcaster) NewRecorder(_ *runtime.Scheme, _ corev1.EventSource) record.EventRecorder {
	return f
}

func (f *fakeEventBroadcaster) Shutdown() {}

func (f *fakeEventBroadcaster) Event(_ runtime.Object, _, _, _ string) {}

func (f *fakeEventBroadcaster) Eventf(_ runtime.Object, _, _, _ string, _ ...interface{}) {
}

func (f *fakeEventBroadcaster) AnnotatedEventf(_ runtime.Object, _ map[string]string, _, _, _ string, _ ...interface{}) {
}
