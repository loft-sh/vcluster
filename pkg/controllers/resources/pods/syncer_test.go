package pods

import (
	"fmt"
	"testing"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/pod-security-admission/api"
	"k8s.io/utils/pointer"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(generictesting.DefaultTestTargetNamespace)

	PodLogsVolumeName := "pod-logs"
	LogsVolumeName := "logs"
	KubeletPodVolumeName := "kubelet-pods"
	HostpathPodName := "test-hostpaths"

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
			Name:      generictesting.DefaultTestVclusterServiceName,
			Namespace: generictesting.DefaultTestCurrentNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}
	translate.Suffix = generictesting.DefaultTestVclusterName
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
			podtranslate.LabelsAnnotation:             "",
			podtranslate.NameAnnotation:               vObjectMeta.Name,
			podtranslate.NamespaceAnnotation:          vObjectMeta.Namespace,
			translate.NameAnnotation:                  vObjectMeta.Name,
			translate.NamespaceAnnotation:             vObjectMeta.Namespace,
			podtranslate.ServiceAccountNameAnnotation: "",
			podtranslate.UIDAnnotation:                string(vObjectMeta.UID),
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.Suffix,
		},
	}
	pPodBase := &corev1.Pod{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: pointer.Bool(false),
			EnableServiceLinks:           pointer.Bool(false),
			HostAliases: []corev1.HostAlias{{
				IP:        pVclusterService.Spec.ClusterIP,
				Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
			}},
			Hostname: vObjectMeta.Name,
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
	nodeSelectorOption := "labelB=enforcedB,otherLabel=abc"
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

	vHostPath := fmt.Sprintf(podtranslate.VirtualPathTemplate, generictesting.DefaultTestCurrentNamespace, generictesting.DefaultTestVclusterName)

	pHostPathPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(vHostPathPod.Name, generictesting.DefaultTestCurrentNamespace),
			Namespace: generictesting.DefaultTestTargetNamespace,

			Annotations: map[string]string{
				podtranslate.ClusterAutoScalerAnnotation:  "false",
				podtranslate.LabelsAnnotation:             "",
				podtranslate.NameAnnotation:               vHostPathPod.Name,
				podtranslate.NamespaceAnnotation:          vHostPathPod.Namespace,
				translate.NameAnnotation:                  vHostPathPod.Name,
				translate.NamespaceAnnotation:             vHostPathPod.Namespace,
				podtranslate.ServiceAccountNameAnnotation: "",
				podtranslate.UIDAnnotation:                string(vHostPathPod.UID),
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vHostPathPod.Namespace,
				translate.MarkerLabel:    translate.Suffix,
			},
			// CreationTimestamp: metav1.Time{},
			// ResourceVersion:   "999",
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: pointer.Bool(false),
			EnableServiceLinks:           pointer.Bool(false),
			HostAliases: []corev1.HostAlias{{
				IP:        pVclusterService.Spec.ClusterIP,
				Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
			}},
			Hostname: vHostPathPod.Name,
			Containers: []corev1.Container{
				{
					Name:  "nginx-placeholder",
					Image: "nginx",
					Env:   pPodContainerEnv,
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
				ctx.Options.EnforceNodeSelector = true
				ctx.Options.NodeSelector = nodeSelectorOption
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncDown(syncCtx, vPodWithNodeSelector.DeepCopy())
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
				_, err := syncer.(*podSyncer).SyncDown(syncCtx, vPodPSS.DeepCopy())
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
				ctx.Options.EnforcePodSecurityStandard = string(api.LevelPrivileged)
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncDown(syncCtx, vPodPSS.DeepCopy())
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
				ctx.Options.EnforcePodSecurityStandard = string(api.LevelRestricted)
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncDown(syncCtx, vPodPSSR.DeepCopy())
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
				ctx.Options.RewriteHostPaths = true
				synccontext, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncDown(synccontext, vHostPathPod.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}
