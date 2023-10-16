package events

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/hack/load-testing/tests/framework"
	"github.com/loft-sh/vcluster/pkg/util/random"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEvents(ctx context.Context, kubeClient client.Client, restConfig *rest.Config, amount int64, namespace string) error {
	// create the event recorder
	recorderClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&clientcorev1.EventSinkImpl{Interface: recorderClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(kubeClient.Scheme(), corev1.EventSource{Component: "loading-test"})
	err = framework.CreateNamespace(ctx, kubeClient, namespace)
	if err != nil {
		return err
	}

	for i := int64(0); i < amount; i++ {
		if i%int64(100) == 0 {
			klog.FromContext(ctx).Info("Creating event", "n", i)
			time.Sleep(time.Millisecond * 100)
		}

		recorder.Event(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%v", i),
				Namespace: namespace,
			},
		}, "Warning", "Test", random.String(1024))
	}

	time.Sleep(time.Millisecond * 100)
	return nil
}
