package pods

import (
	"fmt"
	"testing"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/util/maps"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/pod-security-admission/api"
	"k8s.io/utils/ptr"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(generictesting.DefaultTestTargetNamespace)
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	PodLogsVolumeName := "pod-logs"
	LogsVolumeName := "logs"
	KubeletPodVolumeName := "kubelet-pods"
	HostpathPodName := "test-hostpaths"
	NotInjectedPodName := "test-not-injected"

	pPodContainerEnv := []corev1.EnvVar{
		{
			Name:  "KUBERNETES_PORT",
			Value: "tcp://1.2.3.4:443",
		},
		{
			Name:  "KUBERNETES_PORT_443_TCP",
			Value: "tcp://1.2.3.4:443",
		},
		{
			Name:  "KUBERNETES_PORT_443_TCP_ADDR",
			Value: "1.2.3.4",
		},
		{
			Name:  "KUBERNETES_PORT_443_TCP_PORT",
			Value: "443",
		}, {
			Name:  "KUBERNETES_PORT_443_TCP_PROTO",
			Value: "tcp",
		}, {
			Name:  "KUBERNETES_SERVICE_HOST",
			Value: "1.2.3.4",
		}, {
			Name:  "KUBERNETES_SERVICE_PORT",
			Value: "443",
		}, {
			Name:  "KUBERNETES_SERVICE_PORT_HTTPS",
			Value: "443",
		},
	}

	pVclusterService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generictesting.DefaultTestVClusterServiceName,
			Namespace: generictesting.DefaultTestCurrentNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}
	translate.VClusterName = generictesting.DefaultTestVClusterName
	pDNSService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName("kube-dns", "kube-system"),
			Namespace: generictesting.DefaultTestTargetNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "2.2.2.2",
		},
	}
	vNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testns",
		},
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testpod",
		Namespace: vNamespace.Name,
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.PhysicalName("testpod", "testns"),
		Namespace: "test",
		Annotations: map[string]string{
			podtranslate.ClusterAutoScalerAnnotation:  "false",
			podtranslate.VClusterLabelsAnnotation:     "",
			podtranslate.NameAnnotation:               vObjectMeta.Name,
			podtranslate.NamespaceAnnotation:          vObjectMeta.Namespace,
			translate.NameAnnotation:                  vObjectMeta.Name,
			translate.UIDAnnotation:                   "",
			translate.NamespaceAnnotation:             vObjectMeta.Namespace,
			podtranslate.ServiceAccountNameAnnotation: "",
			podtranslate.UIDAnnotation:                string(vObjectMeta.UID),
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
	}
	pPodBase := &corev1.Pod{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: ptr.To(false),
			EnableServiceLinks:           ptr.To(false),
			HostAliases: []corev1.HostAlias{{
				IP:        pVclusterService.Spec.ClusterIP,
				Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
			}},
			ServiceAccountName: "vc-workload-vcluster",
			Hostname:           vObjectMeta.Name,
		},
	}
	vPodWithNodeName := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			NodeName: "test123",
		},
	}
	pPodWithNodeName := pPodBase.DeepCopy()
	pPodWithNodeName.Spec.NodeName = "test456"

	vPodWithNodeSelector := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"labelA": "valueA",
				"labelB": "valueB",
			},
		},
	}
	nodeSelectorOption := map[string]string{
		"labelB":     "enforcedB",
		"otherLabel": "abc",
	}
	pPodWithNodeSelector := pPodBase.DeepCopy()
	pPodWithNodeSelector.Spec.NodeSelector = map[string]string{
		"labelA":     "valueA",
		"labelB":     "enforcedB",
		"otherLabel": "abc",
	}

	// pod security standards test objects
	vPodPSS := &corev1.Pod{
		ObjectMeta: vObjectMeta,
	}

	pPodPss := pPodBase.DeepCopy()

	vPodPSSR := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container",
					Ports: []corev1.ContainerPort{
						{HostPort: 80},
					},
				},
			},
		},
	}

	vHostpathNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: generictesting.DefaultTestCurrentNamespace,
		},
	}

	vHostPathPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HostpathPodName,
			Namespace: generictesting.DefaultTestCurrentNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx-placeholder",
					Image: "nginx",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      PodLogsVolumeName,
							MountPath: podtranslate.PodLoggingHostPath,
						},
						{
							Name:      LogsVolumeName,
							MountPath: podtranslate.LogHostPath,
						},
						{
							Name:      KubeletPodVolumeName,
							MountPath: podtranslate.KubeletPodPath,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: PodLogsVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.PodLoggingHostPath,
						},
					},
				},
				{
					Name: LogsVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.LogHostPath,
						},
					},
				},
				{
					Name: KubeletPodVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.KubeletPodPath,
						},
					},
				},
			},
		},
	}

	vHostPath := fmt.Sprintf(podtranslate.VirtualPathTemplate, generictesting.DefaultTestCurrentNamespace, generictesting.DefaultTestVClusterName)

	hostToContainer := corev1.MountPropagationHostToContainer
	pHostPathPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(vHostPathPod.Name, generictesting.DefaultTestCurrentNamespace),
			Namespace: generictesting.DefaultTestTargetNamespace,

			Annotations: map[string]string{
				podtranslate.ClusterAutoScalerAnnotation:  "false",
				podtranslate.VClusterLabelsAnnotation:     "",
				podtranslate.NameAnnotation:               vHostPathPod.Name,
				podtranslate.NamespaceAnnotation:          vHostPathPod.Namespace,
				translate.NameAnnotation:                  vHostPathPod.Name,
				translate.NamespaceAnnotation:             vHostPathPod.Namespace,
				translate.UIDAnnotation:                   "",
				podtranslate.ServiceAccountNameAnnotation: "",
				podtranslate.UIDAnnotation:                string(vHostPathPod.UID),
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vHostPathPod.Namespace,
				translate.MarkerLabel:    translate.VClusterName,
			},
			// CreationTimestamp: metav1.Time{},
			// ResourceVersion:   "999",
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: ptr.To(false),
			EnableServiceLinks:           ptr.To(false),
			HostAliases: []corev1.HostAlias{{
				IP:        pVclusterService.Spec.ClusterIP,
				Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
			}},
			Hostname:           vHostPathPod.Name,
			ServiceAccountName: "vc-workload-vcluster",
			Containers: []corev1.Container{
				{
					Name:  "nginx-placeholder",
					Image: "nginx",
					Env:   pPodContainerEnv,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:             PodLogsVolumeName,
							MountPath:        podtranslate.PodLoggingHostPath,
							MountPropagation: &hostToContainer,
						},
						{
							Name:             LogsVolumeName,
							MountPath:        podtranslate.LogHostPath,
							MountPropagation: &hostToContainer,
						},
						{
							Name:             KubeletPodVolumeName,
							MountPath:        podtranslate.KubeletPodPath,
							MountPropagation: &hostToContainer,
						},
						{
							Name:      fmt.Sprintf("%s-%s", PodLogsVolumeName, podtranslate.PhysicalVolumeNameSuffix),
							MountPath: podtranslate.PhysicalPodLogVolumeMountPath,
						},
						{
							Name:      fmt.Sprintf("%s-%s", LogsVolumeName, podtranslate.PhysicalVolumeNameSuffix),
							MountPath: podtranslate.PhysicalLogVolumeMountPath,
						},
						{
							Name:      fmt.Sprintf("%s-%s", KubeletPodVolumeName, podtranslate.PhysicalVolumeNameSuffix),
							MountPath: podtranslate.PhysicalKubeletVolumeMountPath,
						},
					},
				},
			},

			Volumes: []corev1.Volume{
				{
					Name: PodLogsVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: vHostPath + "/log/pods",
						},
					},
				},
				{
					Name: LogsVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: vHostPath + "/log",
						},
					},
				},
				{
					Name: KubeletPodVolumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: vHostPath + "/kubelet/pods",
						},
					},
				},
				{
					Name: fmt.Sprintf("%s-%s", PodLogsVolumeName, podtranslate.PhysicalVolumeNameSuffix),
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.PodLoggingHostPath,
						},
					},
				},
				{
					Name: fmt.Sprintf("%s-%s", LogsVolumeName, podtranslate.PhysicalVolumeNameSuffix),
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.LogHostPath,
						},
					},
				},
				{
					Name: fmt.Sprintf("%s-%s", KubeletPodVolumeName, podtranslate.PhysicalVolumeNameSuffix),
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: podtranslate.KubeletPodPath,
						},
					},
				},
			},
		},
	}

	vInjectedPodNamespace := vHostpathNamespace.DeepCopy()
	vNotInjectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NotInjectedPodName,
			Namespace: generictesting.DefaultTestCurrentNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx-not-injected",
					Image: "nginx",
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "nginx-not-injected",
				},
			},
		},
	}

	pInjectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(NotInjectedPodName, generictesting.DefaultTestCurrentNamespace),
			Namespace: generictesting.DefaultTestTargetNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx-not-injected",
					Image: "nginx",
				},
				{
					Name:  "nginx-injected",
					Image: "nginx",
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "nginx-not-injected",
				},
				{
					Name: "nginx-injected",
				},
			},
		},
	}

	syncLabelsWildcard := "test.sh/*"
	testLabels := map[string]string{
		"test.sh/label1": "true",
		"test.sh/label2": "true",
	}

	vPodWithLabels := &corev1.Pod{
		ObjectMeta: vObjectMeta,
	}
	vPodWithLabels.Labels = testLabels

	pPodWithLabels := pPodBase.DeepCopy()
	maps.Copy(pPodWithLabels.Labels, testLabels)
	maps.Copy(pPodWithLabels.Labels, convertLabelKeyWithPrefix(testLabels))
	pPodWithLabels.Annotations[podtranslate.VClusterLabelsAnnotation] = podtranslate.LabelsAnnotation(vPodWithLabels)

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Delete virtual pod",
			InitialVirtualState:  []runtime.Object{vPodWithNodeName.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pPodWithNodeName},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {
					pPodWithNodeName,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).Sync(syncCtx, pPodWithNodeName.DeepCopy(), vPodWithNodeName)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync and enforce NodeSelector",
			InitialVirtualState:  []runtime.Object{vPodWithNodeSelector.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodWithNodeSelector.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {
					pPodWithNodeSelector,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.Nodes.Selector.Labels = nodeSelectorOption
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncCtx, vPodWithNodeSelector.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "SyncDown pods without any pod security standards",
			InitialVirtualState:  []runtime.Object{vPodPSS.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodPSS.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pPodPss.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncCtx, vPodPSS.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Enforce privileged pod security standard",
			InitialVirtualState:  []runtime.Object{vPodPSS.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodPSS.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pPodPss.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Policies.PodSecurityStandard = string(api.LevelPrivileged)
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncCtx, vPodPSS.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Enforce restricted pod security standard",
			InitialVirtualState:  []runtime.Object{vPodPSSR.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodPSSR.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Policies.PodSecurityStandard = string(api.LevelRestricted)
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncCtx, vPodPSSR.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Map hostpaths",
			InitialVirtualState:  []runtime.Object{vHostPathPod, vHostpathNamespace},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vHostPathPod.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pHostPathPod.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.ControlPlane.HostPathMapper.Enabled = true
				synccontext, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(synccontext, vHostPathPod.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Check injected sidecars",
			InitialVirtualState:  []runtime.Object{vNotInjectedPod, vInjectedPodNamespace},
			InitialPhysicalState: []runtime.Object{pInjectedPod.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vNotInjectedPod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				synccontext, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).Sync(synccontext, pInjectedPod.DeepCopy(), vNotInjectedPod.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Check Syncer with sync labels wildcard",
			InitialVirtualState:  []runtime.Object{vPodWithLabels.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodWithLabels.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {
					pPodWithLabels.DeepCopy(),
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Experimental.SyncSettings.SyncLabels = []string{syncLabelsWildcard}
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncCtx, vPodWithLabels.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}

func convertLabelKeyWithPrefix(labels map[string]string) map[string]string {
	ret := make(map[string]string, len(labels))

	for k, v := range labels {
		ret[translate.ConvertLabelKeyWithPrefix(translate.LabelPrefix, k)] = v
	}

	return ret
}
