package serviceaccounts

import (
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-serviceaccount",
			Namespace: "test",
			Annotations: map[string]string{
				"test": "test",
			},
		},
		Secrets: []corev1.ObjectReference{
			{
				Kind: "Test",
			},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test",
			},
		},
	}
	pSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				"test":                                 "test",
				translate.ManagedAnnotationsAnnotation: "test",
				translate.NameAnnotation:               vSA.Name,
				translate.NamespaceAnnotation:          vSA.Namespace,
				translate.UIDAnnotation:                "",
				translate.KindAnnotation:               corev1.SchemeGroupVersion.WithKind("ServiceAccount").String(),
				translate.HostNamespaceAnnotation:      "test",
				translate.HostNameAnnotation:           translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vSA.Namespace,
			},
		},
		AutomountServiceAccountToken: &[]bool{false}[0],
	}

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
		{
			Name: "ServiceAccount sync",
			InitialVirtualState: []runtime.Object{
				vSA,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {pSA},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vSA))
				assert.NilError(t, err)
			},
		},
	})
}

// TestPullSecretsFor verifies the selector logic of pullSecretsFor directly.
func TestPullSecretsFor(t *testing.T) {
	refs := []corev1.LocalObjectReference{{Name: "my-registry-creds"}}

	cases := []struct {
		name    string
		syncer  *serviceAccountSyncer
		sa      *corev1.ServiceAccount
		wantNil bool
	}{
		{
			name: "no imagePullSecrets configured returns nil",
			syncer: &serviceAccountSyncer{
				imagePullSecrets:        nil,
				imagePullSecretSelector: vclusterconfig.StandardLabelSelector{MatchLabels: map[string]string{}},
			},
			sa:      &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa"}},
			wantNil: true,
		},
		{
			name: "empty selector returns nil regardless of imagePullSecrets",
			syncer: &serviceAccountSyncer{
				imagePullSecrets:        refs,
				imagePullSecretSelector: vclusterconfig.StandardLabelSelector{},
			},
			sa:      &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa"}},
			wantNil: true,
		},
		{
			name: "matchLabels empty matches all SAs",
			syncer: &serviceAccountSyncer{
				imagePullSecrets:        refs,
				imagePullSecretSelector: vclusterconfig.StandardLabelSelector{MatchLabels: map[string]string{}},
			},
			sa:      &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa"}},
			wantNil: false,
		},
		{
			name: "label selector matching SA labels returns refs",
			syncer: &serviceAccountSyncer{
				imagePullSecrets:        refs,
				imagePullSecretSelector: vclusterconfig.StandardLabelSelector{MatchLabels: map[string]string{"env": "prod"}},
			},
			sa: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Name:   "sa",
				Labels: map[string]string{"env": "prod"},
			}},
			wantNil: false,
		},
		{
			name: "label selector not matching SA labels returns nil",
			syncer: &serviceAccountSyncer{
				imagePullSecrets:        refs,
				imagePullSecretSelector: vclusterconfig.StandardLabelSelector{MatchLabels: map[string]string{"env": "prod"}},
			},
			sa: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Name:   "sa",
				Labels: map[string]string{"env": "dev"},
			}},
			wantNil: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.syncer.pullSecretsFor(tc.sa)
			if tc.wantNil {
				assert.Assert(t, result == nil, "expected nil, got %v", result)
			} else {
				assert.DeepEqual(t, refs, result)
			}
		})
	}
}

// TestSyncToHostWithImagePullSecrets verifies that SyncToHost writes imagePullSecrets
// onto the host ServiceAccount when the virtual SA matches the configured selector.
func TestSyncToHostWithImagePullSecrets(t *testing.T) {
	const pullSecretName = "my-registry-creds"

	vSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-serviceaccount",
			Namespace: "test",
		},
	}

	// expected host SA without imagePullSecrets
	basePSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vSA.Name,
				translate.NamespaceAnnotation:     vSA.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("ServiceAccount").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vSA.Namespace,
			},
		},
		AutomountServiceAccountToken: &[]bool{false}[0],
	}

	// expected host SA with imagePullSecrets
	pSAWithSecrets := basePSA.DeepCopy()
	pSAWithSecrets.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecretName}}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                "matchLabels empty propagates imagePullSecrets to all SAs",
			InitialVirtualState: []runtime.Object{vSA},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecrets = []vclusterconfig.ImagePullSecretName{{Name: pullSecretName}}
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecretSelector = vclusterconfig.StandardLabelSelector{
					MatchLabels: map[string]string{},
				}
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {pSAWithSecrets},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vSA))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "empty selector does not propagate imagePullSecrets",
			InitialVirtualState: []runtime.Object{vSA},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecrets = []vclusterconfig.ImagePullSecretName{{Name: pullSecretName}}
				// no ImagePullSecretSelector set
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {basePSA},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vSA))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "non-matching selector does not propagate imagePullSecrets",
			InitialVirtualState: []runtime.Object{vSA},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecrets = []vclusterconfig.ImagePullSecretName{{Name: pullSecretName}}
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecretSelector = vclusterconfig.StandardLabelSelector{
					MatchLabels: map[string]string{"inject-pull-secrets": "true"},
				}
				// vSA has no labels, so selector will not match
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {basePSA},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vSA))
				assert.NilError(t, err)
			},
		},
	})
}

// TestSyncWithImagePullSecrets verifies that the Sync (reconcile/update) path enforces
// the configured imagePullSecrets on the host SA, overwriting any stale values.
func TestSyncWithImagePullSecrets(t *testing.T) {
	const pullSecretName = "my-registry-creds"

	vSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-serviceaccount",
			Namespace: "test",
		},
	}

	hostSAName := translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name
	baseAnnotations := map[string]string{
		translate.NameAnnotation:          vSA.Name,
		translate.NamespaceAnnotation:     vSA.Namespace,
		translate.UIDAnnotation:           "",
		translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("ServiceAccount").String(),
		translate.HostNamespaceAnnotation: "test",
		translate.HostNameAnnotation:      hostSAName,
	}
	baseLabels := map[string]string{translate.NamespaceLabel: vSA.Namespace}

	// host SA that was previously synced but has a stale imagePullSecret
	hostSAStale := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hostSAName,
			Namespace:   "test",
			Annotations: baseAnnotations,
			Labels:      baseLabels,
		},
		AutomountServiceAccountToken: &[]bool{false}[0],
		ImagePullSecrets:             []corev1.LocalObjectReference{{Name: "stale-secret"}},
	}

	// after Sync, host SA should have the correct imagePullSecrets
	pSAAfterSync := hostSAStale.DeepCopy()
	pSAAfterSync.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecretName}}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Sync replaces stale imagePullSecrets with configured ones",
			InitialVirtualState:  []runtime.Object{vSA},
			InitialPhysicalState: []runtime.Object{hostSAStale},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecrets = []vclusterconfig.ImagePullSecretName{{Name: pullSecretName}}
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecretSelector = vclusterconfig.StandardLabelSelector{
					MatchLabels: map[string]string{},
				}
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {pSAAfterSync},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(hostSAStale.DeepCopy(), hostSAStale.DeepCopy(), vSA.DeepCopy(), vSA.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync clears imagePullSecrets when selector no longer matches",
			InitialVirtualState:  []runtime.Object{vSA},
			InitialPhysicalState: []runtime.Object{hostSAStale},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecrets = []vclusterconfig.ImagePullSecretName{{Name: pullSecretName}}
				// selector requires a label the vSA does not have
				vConfig.ControlPlane.Advanced.WorkloadServiceAccount.ImagePullSecretSelector = vclusterconfig.StandardLabelSelector{
					MatchLabels: map[string]string{"inject-pull-secrets": "true"},
				}
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {
					func() *corev1.ServiceAccount {
						s := hostSAStale.DeepCopy()
						s.ImagePullSecrets = nil
						return s
					}(),
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(hostSAStale.DeepCopy(), hostSAStale.DeepCopy(), vSA.DeepCopy(), vSA.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
