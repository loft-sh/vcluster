package services

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/types"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testservice",
		Namespace: "testns",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, "testservice", "testns").Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Service").String(),
			translate.HostNamespaceAnnotation: "test",
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testservice", "testns").Name,
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
	}
	vKubernetesObjectMeta := metav1.ObjectMeta{
		Name:      "kubernetes",
		Namespace: "default",
	}
	baseService := &corev1.Service{
		ObjectMeta: vObjectMeta,
	}
	createdService := &corev1.Service{
		ObjectMeta: pObjectMeta,
	}
	kubernetesService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
	}
	createdByServerService := createdService.DeepCopy()
	createdByServerService.Annotations[ServiceBlockDeletion] = "true"
	updateForwardSpec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "somePort",
				Port: 123,
			},
		},
		PublishNotReadyAddresses: true,
		Type:                     corev1.ServiceTypeNodePort,
		ExternalName:             "external",
		ExternalTrafficPolicy:    corev1.ServiceExternalTrafficPolicyTypeLocal,
		SessionAffinity:          corev1.ServiceAffinityClientIP,
		LoadBalancerSourceRanges: []string{"backwardRange"},
		SessionAffinityConfig: &corev1.SessionAffinityConfig{
			ClientIP: &corev1.ClientIPConfig{},
		},
		HealthCheckNodePort: 112,
	}
	updateForwardService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vObjectMeta.Name,
			Namespace:   vObjectMeta.Namespace,
			Annotations: map[string]string{"a": "b"},
		},
		Spec: updateForwardSpec,
	}
	updatedForwardService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pObjectMeta.Name,
			Namespace: pObjectMeta.Namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:               vObjectMeta.Name,
				translate.NamespaceAnnotation:          vObjectMeta.Namespace,
				translate.UIDAnnotation:                "",
				translate.KindAnnotation:               corev1.SchemeGroupVersion.WithKind("Service").String(),
				translate.HostNamespaceAnnotation:      pObjectMeta.Namespace,
				translate.HostNameAnnotation:           pObjectMeta.Name,
				translate.ManagedAnnotationsAnnotation: "a",
				"a":                                    "b",
			},
			Labels: pObjectMeta.Labels,
		},
		Spec: updateForwardSpec,
	}
	updateBackwardSpec := corev1.ServiceSpec{
		ExternalName:   "backwardExternal",
		ExternalIPs:    []string{"123:221:123:221"},
		LoadBalancerIP: "123:213:123:213",
	}
	updateBackwardRecreateSpec := corev1.ServiceSpec{
		ClusterIP:      "123:123:123:123",
		ExternalName:   updateBackwardSpec.ExternalName,
		ExternalIPs:    updateBackwardSpec.ExternalIPs,
		LoadBalancerIP: updateBackwardSpec.LoadBalancerIP,
	}
	updateBackwardSpecService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			Labels:      pObjectMeta.Labels,
			Annotations: pObjectMeta.Annotations,
		},
		Spec: corev1.ServiceSpec{
			ExternalName:   "backwardExternal",
			ExternalIPs:    []string{"123:221:123:221"},
			LoadBalancerIP: "123:213:123:213",
		},
	}
	updatedBackwardSpecService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalName:   "backwardExternal",
			ExternalIPs:    []string{"123:221:123:221"},
			LoadBalancerIP: "123:213:123:213",
		},
	}
	updateBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			Labels:      pObjectMeta.Labels,
			Annotations: pObjectMeta.Annotations,
		},
		Spec: updateBackwardRecreateSpec,
	}
	updatedBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:      "123:123:123:123",
			ExternalIPs:    []string{"123:221:123:221"},
			LoadBalancerIP: "123:213:123:213",
		},
	}
	updateBackwardStatus := corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					IP:       "121:121:121:121",
					Hostname: "ingresshost",
				},
			},
		},
	}
	updateBackwardStatusService := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Status:     updateBackwardStatus,
	}
	updatedBackwardStatusService := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Status:     updateBackwardStatus,
	}
	kubernetesWithClusterIPService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
		Spec: corev1.ServiceSpec{
			ClusterIP: "121:212:121:212",
		},
	}
	kubernetesWithPortsService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "k8port",
					Port: 4321,
				},
			},
		},
	}

	vServicePorts1 := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test",
					Port:       123,
					NodePort:   567,
					TargetPort: intstr.FromInt(10),
				},
			},
		},
	}
	vServicePorts1Synced := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test",
					Port:       123,
					NodePort:   456,
					TargetPort: intstr.FromInt(10),
				},
			},
		},
	}
	pServicePorts1 := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test",
					Port:       123,
					NodePort:   456,
					TargetPort: intstr.FromInt(10),
				},
			},
		},
	}
	pServicePorts2 := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test123",
					Port:       123,
					NodePort:   456,
					TargetPort: intstr.FromInt(10),
				},
			},
		},
	}
	pServicePorts2Synced := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "test",
					Port:       123,
					NodePort:   567,
					TargetPort: intstr.FromInt(10),
				},
			},
		},
	}
	vServiceClusterIPFromExternal := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Spec: corev1.ServiceSpec{
			ExternalName: "test.com",
			Type:         corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	pServiceExternal := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			ExternalName: "test.com",
			Type:         corev1.ServiceTypeExternalName,
		},
	}
	pServiceClusterIPFromExternal := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			ExternalName: "test.com",
			Type:         corev1.ServiceTypeClusterIP,
			Ports:        vServiceClusterIPFromExternal.Spec.Ports,
		},
	}
	selectorKey := "test"
	vServiceNodePortFromExternal := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{selectorKey: "test-key"},
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
	pServiceNodePortFromExternal := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				translate.HostLabel(selectorKey): vServiceNodePortFromExternal.Spec.Selector[selectorKey],
				translate.NamespaceLabel:         vServiceNodePortFromExternal.Namespace,
				translate.MarkerLabel:            translate.VClusterName,
			},
			Type:  corev1.ServiceTypeNodePort,
			Ports: vServiceNodePortFromExternal.Spec.Ports,
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                "Create Forward",
			InitialVirtualState: []runtime.Object{baseService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseService))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync node ports physical -> virtual",
			InitialVirtualState:  []runtime.Object{vServicePorts1.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pServicePorts1.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {vServicePorts1Synced.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {pServicePorts1.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(pServicePorts1.DeepCopy(), vServicePorts1.DeepCopy(), synccontext.SyncEventSourceHost))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync ports virtual -> physical",
			InitialVirtualState:  []runtime.Object{vServicePorts1.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pServicePorts2.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {vServicePorts1.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {pServicePorts2Synced.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pServicePorts2.DeepCopy(), vServicePorts1.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{updateForwardService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdByServerService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateForwardService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedForwardService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(createdByServerService.DeepCopy(), updateForwardService.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(createdService.DeepCopy(), baseService.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward spec no recreation",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				baseService := baseService.DeepCopy()
				updateBackwardSpecService := updateBackwardSpecService.DeepCopy()
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(updateBackwardSpecService, baseService, synccontext.SyncEventSourceHost))
				assert.NilError(t, err)

				err = ctx.VirtualManager.GetClient().Get(ctx, types.NamespacedName{Namespace: baseService.Namespace, Name: baseService.Name}, baseService)
				assert.NilError(t, err)

				err = ctx.PhysicalManager.GetClient().Get(ctx, types.NamespacedName{Namespace: updateBackwardSpecService.Namespace, Name: updateBackwardSpecService.Name}, updateBackwardSpecService)
				assert.NilError(t, err)

				baseService.Spec.ExternalName = updateBackwardSpecService.Spec.ExternalName
				_, err = syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(updateBackwardSpecService.DeepCopy(), baseService.DeepCopy(), synccontext.SyncEventSourceHost))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward spec with recreation",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecRecreateService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecRecreateService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecRecreateService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				baseService := baseService.DeepCopy()
				updateBackwardSpecRecreateService := updateBackwardSpecRecreateService.DeepCopy()
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(updateBackwardSpecRecreateService, baseService))
				assert.NilError(t, err)

				err = ctx.VirtualManager.GetClient().Get(ctx, types.NamespacedName{Namespace: baseService.Namespace, Name: baseService.Name}, baseService)
				assert.NilError(t, err)

				err = ctx.PhysicalManager.GetClient().Get(ctx, types.NamespacedName{Namespace: updateBackwardSpecRecreateService.Namespace, Name: updateBackwardSpecRecreateService.Name}, updateBackwardSpecRecreateService)
				assert.NilError(t, err)

				baseService.Spec.ExternalName = updateBackwardSpecService.Spec.ExternalName
				_, err = syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(updateBackwardSpecRecreateService.DeepCopy(), baseService.DeepCopy(), synccontext.SyncEventSourceHost))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward status",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardStatusService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardStatusService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardStatusService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(updateBackwardStatusService.DeepCopy(), baseService.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward not needed",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(createdService.DeepCopy(), baseService.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync not existent physical kubernetes service",
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				err := specialservices.SyncKubernetesService(ctx.ToSyncContext("sync-kubernetes-service"), "default", "kubernetes", types.NamespacedName{
					Name:      specialservices.DefaultKubernetesSVCName,
					Namespace: specialservices.DefaultKubernetesSVCNamespace,
				}, TranslateServicePorts)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync not existent virtual kubernetes service",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{kubernetesService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				err := specialservices.SyncKubernetesService(ctx.ToSyncContext("sync-kubernetes-service"), "default", "kubernetes", types.NamespacedName{
					Name:      specialservices.DefaultKubernetesSVCName,
					Namespace: specialservices.DefaultKubernetesSVCNamespace,
				}, TranslateServicePorts)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync kubernetes service with recreation",
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{kubernetesWithClusterIPService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				err := specialservices.SyncKubernetesService(ctx.ToSyncContext("sync-kubernetes-service"), "default", "kubernetes", types.NamespacedName{
					Name:      specialservices.DefaultKubernetesSVCName,
					Namespace: specialservices.DefaultKubernetesSVCNamespace,
				}, TranslateServicePorts)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync kubernetes service without recreation",
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{kubernetesWithPortsService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				err := specialservices.SyncKubernetesService(ctx.ToSyncContext("sync-kubernetes-service"), "default", "kubernetes", types.NamespacedName{
					Name:      specialservices.DefaultKubernetesSVCName,
					Namespace: specialservices.DefaultKubernetesSVCNamespace,
				}, TranslateServicePorts)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync kubernetes service change type ExternalName to ClusterIP",
			InitialVirtualState:  []runtime.Object{vServiceClusterIPFromExternal.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pServiceExternal.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {vServiceClusterIPFromExternal.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {pServiceClusterIPFromExternal.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pServiceExternal.DeepCopy(), vServiceClusterIPFromExternal.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync kubernetes service change type ExternalName to NodePort",
			InitialVirtualState:  []runtime.Object{vServiceNodePortFromExternal.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pServiceExternal.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {vServiceNodePortFromExternal.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {pServiceNodePortFromExternal.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pServiceExternal.DeepCopy(), vServiceNodePortFromExternal.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
