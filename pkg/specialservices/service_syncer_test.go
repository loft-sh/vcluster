package specialservices

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// apiserverRecreatesClient plays the kube-apiserver that recreates default/kubernetes with its own
// cluster ip as soon as the service is gone, so that the first create after a delete loses the race.
type apiserverRecreatesClient struct {
	client.Client

	recreateWithIP string
	racesLeft      int
	creates        int
}

func (c *apiserverRecreatesClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.creates++
	if c.racesLeft > 0 {
		c.racesLeft--
		recreated := obj.(*corev1.Service).DeepCopy()
		recreated.ResourceVersion = ""
		recreated.Spec.ClusterIP = c.recreateWithIP
		recreated.Spec.ClusterIPs = []string{c.recreateWithIP}
		if err := c.Client.Create(ctx, recreated, opts...); err != nil {
			return err
		}
		return kerrors.NewAlreadyExists(schema.GroupResource{Resource: "services"}, obj.GetName())
	}

	return c.Client.Create(ctx, obj, opts...)
}

func TestSyncKubernetesServiceRecreateLosesRace(t *testing.T) {
	hostService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "vcluster", Namespace: "host"},
		Spec:       corev1.ServiceSpec{ClusterIP: "10.0.0.5", ClusterIPs: []string{"10.0.0.5"}, Ports: []corev1.ServicePort{{Name: "https", Port: 443}}},
	}
	virtualService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: DefaultKubernetesSVCName, Namespace: DefaultKubernetesSVCNamespace},
		Spec:       corev1.ServiceSpec{ClusterIP: "10.96.0.1", ClusterIPs: []string{"10.96.0.1"}, Ports: []corev1.ServicePort{{Name: "https", Port: 443}}},
	}

	for _, testCase := range []struct {
		name      string
		racesLost int
		wantErr   bool
	}{
		{name: "create wins", racesLost: 0},
		{name: "create loses once", racesLost: 1},
		{name: "create loses twice", racesLost: 2},
		{name: "create never wins", racesLost: recreateAttempts, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			virtualClient := &apiserverRecreatesClient{
				Client:         testingutil.NewFakeClient(scheme.Scheme, virtualService.DeepCopy()),
				recreateWithIP: "10.96.0.1",
				racesLeft:      testCase.racesLost,
			}
			syncCtx := &synccontext.SyncContext{
				Context:                context.Background(),
				VirtualClient:          virtualClient,
				CurrentNamespaceClient: testingutil.NewFakeClient(scheme.Scheme, hostService.DeepCopy()),
			}

			err := SyncKubernetesService(syncCtx, "host", "vcluster", DefaultKubernetesSvcKey, func(ports []corev1.ServicePort) []corev1.ServicePort { return ports })
			if testCase.wantErr {
				assert.ErrorContains(t, err, "already exists")
				return
			}
			assert.NilError(t, err)

			got := &corev1.Service{}
			assert.NilError(t, virtualClient.Get(context.Background(), DefaultKubernetesSvcKey, got))
			assert.Equal(t, got.Spec.ClusterIP, "10.0.0.5")
			assert.Equal(t, virtualClient.creates, testCase.racesLost+1)
		})
	}
}
