package gateways

import (
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	testGatewayName      = "testgateway"
	testGatewayNamespace = "test"
	testGatewayClassName = "test-gateway-class"
	testCertRefName      = "certrefsecretname"
)

func init() {
	_ = gatewayv1.Install(scheme.Scheme)
}

func TestSync(t *testing.T) {
	vBaseSpec := gatewaySpec()
	pBaseSpec := hostGatewaySpec()
	changedGatewayStatus := gatewayv1.GatewayStatus{
		Addresses: []gatewayv1.GatewayStatusAddress{
			{
				Value: "123.123.123.123",
			},
		},
	}
	vObjectMeta := virtualGatewayMeta()
	pObjectMeta := hostGatewayMeta()
	baseGateway := gateway(vObjectMeta, vBaseSpec)
	createdGateway := gateway(pObjectMeta, pBaseSpec)
	noUpdateGateway := gateway(vObjectMeta, vBaseSpec, withStatus(changedGatewayStatus))
	backwardUpdateGateway := gateway(
		pObjectMeta,
		gatewayv1.GatewaySpec{GatewayClassName: "backwardsupdatedgatewayclass"},
		withStatus(changedGatewayStatus),
	)
	pBackwardUpdatedGateway := gateway(pObjectMeta, pBaseSpec, withStatus(changedGatewayStatus))
	pBackwardUpdatedGateway.Spec.GatewayClassName = "backwardsupdatedgatewayclass"
	backwardNoUpdateGateway := gateway(pObjectMeta, gatewayv1.GatewaySpec{})
	backwardUpdatedGateway := gateway(vObjectMeta, vBaseSpec, withStatus(changedGatewayStatus))
	backwardUpdatedGateway.Spec.GatewayClassName = "backwardsupdatedgatewayclass"

	syncertesting.RunTestsWithContext(t, newGatewayRegisterContext, []*syncertesting.SyncTest{
		{
			Name:                "Create forward",
			InitialVirtualState: []runtime.Object{baseGateway.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {baseGateway.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {createdGateway.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*gatewaySyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseGateway.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{baseGateway.DeepCopy()},
			InitialPhysicalState: []runtime.Object{gateway(pObjectMeta, gatewayv1.GatewaySpec{})},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {baseGateway.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {createdGateway.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				pGateway := gateway(pObjectMeta, gatewayv1.GatewaySpec{})
				pGateway.ResourceVersion = "999"

				_, err := syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(pGateway, pGateway, baseGateway.DeepCopy(), baseGateway.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseGateway.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdGateway.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {baseGateway.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {createdGateway.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				vGateway := noUpdateGateway.DeepCopy()
				vGateway.ResourceVersion = "999"

				_, err := syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(createdGateway.DeepCopy(), createdGateway.DeepCopy(), vGateway, vGateway))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backwards",
			InitialVirtualState:  []runtime.Object{baseGateway.DeepCopy()},
			InitialPhysicalState: []runtime.Object{backwardUpdateGateway.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {backwardUpdatedGateway.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {pBackwardUpdatedGateway.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				backwardUpdateGateway := backwardUpdateGateway.DeepCopy()
				vGateway := baseGateway.DeepCopy()
				vGateway.ResourceVersion = "999"

				_, err := syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(baseGateway.DeepCopy(), backwardUpdateGateway, vGateway, vGateway))
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Namespace: vGateway.Namespace, Name: vGateway.Name}, vGateway)
				assert.NilError(t, err)

				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Namespace: backwardUpdateGateway.Namespace, Name: backwardUpdateGateway.Name}, backwardUpdateGateway)
				assert.NilError(t, err)

				_, err = syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(backwardUpdateGateway, backwardUpdateGateway, vGateway, vGateway))
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Namespace: vGateway.Namespace, Name: vGateway.Name}, vGateway)
				assert.NilError(t, err)

				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Namespace: backwardUpdateGateway.Namespace, Name: backwardUpdateGateway.Name}, backwardUpdateGateway)
				assert.NilError(t, err)

				_, err = syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(backwardUpdateGateway, backwardUpdateGateway, vGateway, vGateway))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backwards not needed",
			InitialVirtualState:  []runtime.Object{baseGateway.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdGateway.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {baseGateway.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {createdGateway.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				pGateway := backwardNoUpdateGateway.DeepCopy()
				pGateway.ResourceVersion = "999"

				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(pGateway, pGateway, baseGateway.DeepCopy(), baseGateway.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Translate annotation",
			InitialVirtualState: []runtime.Object{
				gatewayWithMeta(metav1.ObjectMeta{
					Name:      baseGateway.Name,
					Namespace: baseGateway.Namespace,
					Labels:    baseGateway.Labels,
					Annotations: map[string]string{
						"gateway.example.com/owner": "team-a",
					},
				}),
			},
			InitialPhysicalState: []runtime.Object{
				gatewayWithMeta(metav1.ObjectMeta{
					Name:      createdGateway.Name,
					Namespace: createdGateway.Namespace,
					Labels:    createdGateway.Labels,
				}),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {
					gatewayWithMeta(metav1.ObjectMeta{
						Name:      baseGateway.Name,
						Namespace: baseGateway.Namespace,
						Labels:    baseGateway.Labels,
						Annotations: map[string]string{
							"gateway.example.com/owner": "team-a",
						},
					}),
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.Gateways(): {
					gatewayWithMeta(metav1.ObjectMeta{
						Name:      createdGateway.Name,
						Namespace: createdGateway.Namespace,
						Labels:    createdGateway.Labels,
						Annotations: map[string]string{
							"gateway.example.com/owner":         "team-a",
							"vcluster.loft.sh/object-name":      baseGateway.Name,
							"vcluster.loft.sh/object-namespace": baseGateway.Namespace,
							translate.UIDAnnotation:             "",
							translate.KindAnnotation:            mappings.Gateways().String(),
							translate.HostNamespaceAnnotation:   createdGateway.Namespace,
							translate.HostNameAnnotation:        createdGateway.Name,
						},
					}),
				},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)

				vGateway := &gatewayv1.Gateway{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Name: baseGateway.Name, Namespace: baseGateway.Namespace}, vGateway)
				assert.NilError(t, err)

				pGateway := &gatewayv1.Gateway{}
				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: createdGateway.Name, Namespace: createdGateway.Namespace}, pGateway)
				assert.NilError(t, err)

				_, err = syncer.(*gatewaySyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(pGateway, pGateway, baseGateway.DeepCopy(), vGateway))
				assert.NilError(t, err)
			},
		},
	})
}

func TestSyncCertificateRefsRejectUnsupportedGroupAndKind(t *testing.T) {
	tests := []struct {
		name        string
		certRef     gatewayv1.SecretObjectReference
		expectedErr string
	}{
		{
			name: "Unsupported certificateRef group",
			certRef: gatewayv1.SecretObjectReference{
				Name:  gatewayv1.ObjectName("certrefsecretname"),
				Group: ptr.To(gatewayv1.Group("gateway.networking.k8s.io")),
				Kind:  ptr.To(gatewayv1.Kind("Secret")),
			},
			expectedErr: `group "gateway.networking.k8s.io" is not supported for certificateRefs`,
		},
		{
			name: "Unsupported certificateRef kind",
			certRef: gatewayv1.SecretObjectReference{
				Name:  gatewayv1.ObjectName("certrefsecretname"),
				Group: ptr.To(gatewayv1.Group("")),
				Kind:  ptr.To(gatewayv1.Kind("ConfigMap")),
			},
			expectedErr: `kind "ConfigMap" not supported for certificateRefs`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vGateway := gatewayWithCertificateRefs(tc.certRef)
			syncCtx, syncer := startGatewaySyncer(t, nil, []runtime.Object{vGateway}, nil)
			_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vGateway.DeepCopy()))
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestSkipSync(t *testing.T) {
	selector := rootconfig.StandardLabelSelector{MatchLabels: map[string]string{"sync": "yes"}}
	enableGatewayClassSync := func(vConfig *config.VirtualClusterConfig) {
		vConfig.Sync.FromHost.GatewayClasses.Enabled = true
		vConfig.Sync.FromHost.GatewayClasses.Selector = selector
	}

	tests := []struct {
		name         string
		className    gatewayv1.ObjectName
		hostObjects  []runtime.Object
		adjustConfig func(*config.VirtualClusterConfig)
		expectedSkip bool
	}{
		{
			name:         "GatewayClass sync disabled",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			adjustConfig: func(vConfig *config.VirtualClusterConfig) { vConfig.Sync.FromHost.GatewayClasses.Selector = selector },
			expectedSkip: false,
		},
		{
			name:         "empty GatewayClass selector",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			adjustConfig: func(vConfig *config.VirtualClusterConfig) { vConfig.Sync.FromHost.GatewayClasses.Enabled = true },
			expectedSkip: false,
		},
		{
			name:         "Gateway without class",
			adjustConfig: enableGatewayClassSync,
			expectedSkip: false,
		},
		{
			name:         "missing host GatewayClass",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			adjustConfig: enableGatewayClassSync,
			expectedSkip: true,
		},
		{
			name:         "deleted host GatewayClass",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			hostObjects:  []runtime.Object{gatewayClass(map[string]string{"sync": "yes"}, withGatewayClassDeleted())},
			adjustConfig: enableGatewayClassSync,
			expectedSkip: true,
		},
		{
			name:         "GatewayClass outside selector",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			hostObjects:  []runtime.Object{gatewayClass(map[string]string{"sync": "no"})},
			adjustConfig: enableGatewayClassSync,
			expectedSkip: true,
		},
		{
			name:         "GatewayClass matches selector",
			className:    gatewayv1.ObjectName(testGatewayClassName),
			hostObjects:  []runtime.Object{gatewayClass(map[string]string{"sync": "yes"})},
			adjustConfig: enableGatewayClassSync,
			expectedSkip: false,
		},
		{
			name:      "invalid GatewayClass selector",
			className: gatewayv1.ObjectName(testGatewayClassName),
			hostObjects: []runtime.Object{
				gatewayClass(map[string]string{"sync": "yes"}),
			},
			adjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.FromHost.GatewayClasses.Enabled = true
				vConfig.Sync.FromHost.GatewayClasses.Selector = rootconfig.StandardLabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "sync",
							Operator: "Invalid",
							Values:   []string{"yes"},
						},
					},
				}
			},
			expectedSkip: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vGateway := gateway(virtualGatewayMeta(), gatewayv1.GatewaySpec{GatewayClassName: tc.className})
			syncCtx, syncer := startGatewaySyncer(t, tc.hostObjects, []runtime.Object{vGateway}, tc.adjustConfig)
			assert.Equal(t, syncer.skipSync(syncCtx, vGateway), tc.expectedSkip)
		})
	}
}

func newGatewayRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	vConfig.Sync.ToHost.Gateways.Enabled = true
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)

	mapper, err := generic.NewMapper(registerContext, &gatewayv1.Gateway{}, translate.Default.HostName)
	if err != nil {
		panic(err)
	}

	err = registerContext.Mappings.AddMapper(mapper)
	if err != nil {
		panic(err)
	}

	err = mapper.Migrate(registerContext, mapper)
	if err != nil {
		panic(err)
	}

	return registerContext
}

func startGatewaySyncer(
	t *testing.T,
	initialPhysicalState []runtime.Object,
	initialVirtualState []runtime.Object,
	adjustConfig func(*config.VirtualClusterConfig),
) (*synccontext.SyncContext, *gatewaySyncer) {
	t.Helper()

	pClient := testingutil.NewFakeClient(scheme.Scheme, initialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, initialVirtualState...)
	vConfig := testingutil.NewFakeConfig()
	if adjustConfig != nil {
		adjustConfig(vConfig)
	}

	registerContext := newGatewayRegisterContext(vConfig, pClient, vClient)
	syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
	return syncCtx, syncer.(*gatewaySyncer)
}

type gatewayOption func(*gatewayv1.Gateway)

func withStatus(status gatewayv1.GatewayStatus) gatewayOption {
	return func(gateway *gatewayv1.Gateway) {
		gateway.Status = status
	}
}

func gateway(meta metav1.ObjectMeta, spec gatewayv1.GatewaySpec, opts ...gatewayOption) *gatewayv1.Gateway {
	ret := &gatewayv1.Gateway{
		ObjectMeta: meta,
		Spec:       spec,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

func gatewayWithMeta(meta metav1.ObjectMeta) *gatewayv1.Gateway {
	return gateway(meta, gatewayv1.GatewaySpec{})
}

func gatewaySpec() gatewayv1.GatewaySpec {
	return gatewayv1.GatewaySpec{
		GatewayClassName: testGatewayClassName,
		Listeners: []gatewayv1.Listener{
			{
				Name:     "http",
				Hostname: gatewayHostname("example.com"),
				Port:     gatewayv1.PortNumber(80),
				Protocol: gatewayv1.HTTPProtocolType,
			},
			gatewayHTTPSListener(secretCertificateRef(testCertRefName)),
		},
	}
}

func hostGatewaySpec() gatewayv1.GatewaySpec {
	spec := gatewaySpec()
	ret := *spec.DeepCopy()
	ret.Listeners[1].TLS.CertificateRefs[0].Name = "certrefsecretname-x-test-x-suffix"
	ret.Listeners[1].TLS.CertificateRefs[0].Namespace = ptr.To(gatewayv1.Namespace(testGatewayNamespace))
	return ret
}

func gatewayWithCertificateRefs(refs ...gatewayv1.SecretObjectReference) *gatewayv1.Gateway {
	return gateway(virtualGatewayMeta(), gatewayv1.GatewaySpec{
		GatewayClassName: testGatewayClassName,
		Listeners:        []gatewayv1.Listener{gatewayHTTPSListener(refs...)},
	})
}

func gatewayHTTPSListener(refs ...gatewayv1.SecretObjectReference) gatewayv1.Listener {
	return gatewayv1.Listener{
		Name:     "https",
		Hostname: gatewayHostname("secure.example.com"),
		Port:     gatewayv1.PortNumber(443),
		Protocol: gatewayv1.HTTPSProtocolType,
		TLS: &gatewayv1.ListenerTLSConfig{
			CertificateRefs: refs,
		},
	}
}

func secretCertificateRef(name string) gatewayv1.SecretObjectReference {
	return gatewayv1.SecretObjectReference{
		Name:  gatewayv1.ObjectName(name),
		Group: ptr.To(gatewayv1.Group("")),
		Kind:  ptr.To(gatewayv1.Kind("Secret")),
	}
}

func virtualGatewayMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testGatewayName,
		Namespace: testGatewayNamespace,
	}
}

func hostGatewayMeta() metav1.ObjectMeta {
	hostName := translate.Default.HostName(nil, testGatewayName, testGatewayNamespace).Name
	return metav1.ObjectMeta{
		Name:      hostName,
		Namespace: testGatewayNamespace,
		Annotations: map[string]string{
			translate.NameAnnotation:          testGatewayName,
			translate.NamespaceAnnotation:     testGatewayNamespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          mappings.Gateways().String(),
			translate.HostNamespaceAnnotation: testGatewayNamespace,
			translate.HostNameAnnotation:      hostName,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: testGatewayNamespace,
		},
		ResourceVersion: "999",
	}
}

type gatewayClassOption func(*gatewayv1.GatewayClass)

func withGatewayClassDeleted() gatewayClassOption {
	return func(gatewayClass *gatewayv1.GatewayClass) {
		now := metav1.Now()
		gatewayClass.Finalizers = []string{"test-finalizer"}
		gatewayClass.DeletionTimestamp = &now
	}
}

func gatewayClass(labels map[string]string, opts ...gatewayClassOption) *gatewayv1.GatewayClass {
	ret := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   testGatewayClassName,
			Labels: labels,
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController("example.com/gateway-controller"),
		},
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

func gatewayHostname(str string) *gatewayv1.Hostname {
	ret := gatewayv1.Hostname(str)
	return &ret
}
