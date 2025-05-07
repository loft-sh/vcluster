package pods

import (
	"fmt"
	"maps"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/pod-security-admission/api"
	"k8s.io/utils/ptr"
)

var (
	pVclusterService = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testingutil.DefaultTestVClusterServiceName,
			Namespace: testingutil.DefaultTestCurrentNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}
	pDNSService = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, "kube-dns", "kube-system").Name,
			Namespace: testingutil.DefaultTestTargetNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "2.2.2.2",
		},
	}
)

func TestSyncTable(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

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
		Name:      translate.Default.HostName(nil, "testpod", "testns").Name,
		Namespace: "test",
		Annotations: map[string]string{
			podtranslate.ClusterAutoScalerAnnotation:  "false",
			podtranslate.VClusterLabelsAnnotation:     "",
			podtranslate.NameAnnotation:               vObjectMeta.Name,
			podtranslate.NamespaceAnnotation:          vObjectMeta.Namespace,
			translate.NameAnnotation:                  vObjectMeta.Name,
			translate.UIDAnnotation:                   "",
			translate.NamespaceAnnotation:             vObjectMeta.Namespace,
			translate.KindAnnotation:                  corev1.SchemeGroupVersion.WithKind("Pod").String(),
			translate.HostNamespaceAnnotation:         "test",
			translate.HostNameAnnotation:              translate.Default.HostName(nil, "testpod", "testns").Name,
			podtranslate.ServiceAccountNameAnnotation: "",
			podtranslate.UIDAnnotation:                string(vObjectMeta.UID),
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.VClusterName,
		},
	}
	virtualNode := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test123"},
	}
	testCases := []struct {
		name                     string
		expectPhysicalPod        bool
		expectVirtualPod         bool
		withVirtualNode          bool
		virtualPodWithoutNode    bool
		initialVContainers       []corev1.Container
		expectedVContainers      []corev1.Container
		initialPContainers       []corev1.Container
		initialNodeSelector      map[string]string
		expectedNodeSelector     map[string]string
		nodeSelectorOption       map[string]string
		syncToHost               bool
		securityStandard         string
		virtualPodsLabels        map[string]string
		expectPhysicalPodsLabels map[string]string
	}{
		{
			name:              "Delete virtual pod",
			expectPhysicalPod: true,
		},
		{
			name:              "Check injected sidecar",
			expectPhysicalPod: true,
			expectVirtualPod:  true,
			withVirtualNode:   true,
			initialVContainers: []corev1.Container{
				{Name: "nginx-not-injected"},
			},
			expectedVContainers: []corev1.Container{
				{Name: "nginx-not-injected"},
			},
			initialPContainers: []corev1.Container{
				{Name: "nginx-not-injected"},
				{Name: "nginx-injected"},
			},
		},
		{
			name:                  "SyncToHost and enforce NodeSelector",
			expectPhysicalPod:     true,
			syncToHost:            true,
			expectVirtualPod:      true,
			virtualPodWithoutNode: true,
			initialNodeSelector: map[string]string{
				"labelA": "valueA",
				"labelB": "valueB",
			},
			nodeSelectorOption: map[string]string{
				"labelB":     "enforcedB",
				"otherLabel": "abc",
			},
			expectedNodeSelector: map[string]string{
				"labelA":     "valueA",
				"labelB":     "enforcedB",
				"otherLabel": "abc",
			},
		},
		{
			name:                  "SyncToHost without security standard",
			expectPhysicalPod:     true,
			syncToHost:            true,
			expectVirtualPod:      true,
			virtualPodWithoutNode: true,
		},
		{
			name:                  "SyncToHost with privileged security standard",
			expectPhysicalPod:     true,
			syncToHost:            true,
			expectVirtualPod:      true,
			virtualPodWithoutNode: true,
			securityStandard:      string(api.LevelPrivileged),
		},
		{
			name:                  "SyncToHost with restricted security standard",
			expectPhysicalPod:     false,
			syncToHost:            true,
			expectVirtualPod:      true,
			virtualPodWithoutNode: true,
			securityStandard:      string(api.LevelRestricted),
		},
		{
			name:                     "SyncToHost with labels wildcard",
			expectPhysicalPod:        true,
			syncToHost:               true,
			expectVirtualPod:         true,
			virtualPodWithoutNode:    true,
			virtualPodsLabels:        map[string]string{"test.sh/abcd": "yes"},
			expectPhysicalPodsLabels: map[string]string{"test.sh/abcd": "yes"},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			vPodInitial := corev1.Pod{
				ObjectMeta: vObjectMeta,
			}
			if tC.initialVContainers != nil {
				vPodInitial.Spec.Containers = tC.initialVContainers
			}
			if tC.initialNodeSelector != nil {
				vPodInitial.Spec.NodeSelector = tC.initialNodeSelector
			}

			vPodFinal := corev1.Pod{
				ObjectMeta: vObjectMeta,
				Spec: corev1.PodSpec{
					NodeSelector: tC.initialNodeSelector,
				},
			}
			if tC.virtualPodsLabels != nil {
				vPodInitial.Labels = tC.virtualPodsLabels
				vPodFinal.Labels = tC.virtualPodsLabels
			}
			if tC.expectedVContainers != nil {
				vPodFinal.Spec.Containers = tC.expectedVContainers
			}
			pPodInitial := corev1.Pod{
				ObjectMeta: *pObjectMeta.DeepCopy(),
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.To(false),
					EnableServiceLinks:           ptr.To(false),
					HostAliases: []corev1.HostAlias{{
						IP:        pVclusterService.Spec.ClusterIP,
						Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
					}},
					ServiceAccountName: "vc-workload-vcluster",
					Hostname:           vObjectMeta.Name,
					Containers:         tC.initialPContainers,
				},
			}
			pPodFinal := &corev1.Pod{
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
					Containers:         tC.initialPContainers,
					NodeSelector:       tC.expectedNodeSelector,
				},
			}
			if tC.expectPhysicalPodsLabels != nil {
				maps.Copy(pPodFinal.Labels, tC.virtualPodsLabels)
				pPodFinal.Annotations[podtranslate.VClusterLabelsAnnotation] = podtranslate.LabelsAnnotation(vPodInitial.DeepCopy())
			}

			if !tC.virtualPodWithoutNode {
				pPodInitial.Spec.NodeName = "test123"
				pPodFinal.Spec.NodeName = "test123"
				vPodInitial.Spec.NodeName = "test123"
				vPodFinal.Spec.NodeName = "test123"
			}

			initialVirtualObjects := []runtime.Object{vPodInitial.DeepCopy(), vNamespace.DeepCopy()}
			if tC.withVirtualNode {
				initialVirtualObjects = append(initialVirtualObjects, virtualNode.DeepCopy())
			}
			expectedVirtualObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectVirtualPod {
				expectedVirtualObjects[corev1.SchemeGroupVersion.WithKind("Pod")] =
					[]runtime.Object{vPodFinal.DeepCopy()}
			}
			initialPhysicalObjects := []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()}
			if !tC.syncToHost {
				initialPhysicalObjects = append(initialPhysicalObjects, pPodInitial.DeepCopy())
			}
			expectedPhysicalObjects := map[schema.GroupVersionKind][]runtime.Object{}
			if tC.expectPhysicalPod {
				expectedPhysicalObjects[corev1.SchemeGroupVersion.WithKind("Pod")] =
					[]runtime.Object{pPodFinal.DeepCopy()}
			}

			test := syncertesting.SyncTest{
				Name:                  tC.name,
				InitialVirtualState:   initialVirtualObjects,
				InitialPhysicalState:  initialPhysicalObjects,
				ExpectedVirtualState:  expectedVirtualObjects,
				ExpectedPhysicalState: expectedPhysicalObjects,
			}

			// setting up the clients
			pClient, vClient, vConfig := test.Setup()
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)

			registerContext.Config.Networking.Advanced.ProxyKubelets.ByIP = false
			registerContext.Config.Sync.FromHost.Nodes.Selector.Labels = tC.nodeSelectorOption
			if tC.securityStandard != "" {
				registerContext.Config.Policies.PodSecurityStandard = tC.securityStandard
			}

			syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, New)

			var err error
			if tC.syncToHost {
				_, err = syncer.(*podSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPodInitial.DeepCopy()))
			} else {
				_, err = syncer.(*podSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPodInitial.DeepCopy(),
					pPodInitial.DeepCopy(),
					vPodInitial.DeepCopy(),
					vPodInitial.DeepCopy(),
				))
			}
			assert.NilError(t, err)

			test.Validate(t)
		})
	}
}

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

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
			Name:      testingutil.DefaultTestVClusterServiceName,
			Namespace: testingutil.DefaultTestCurrentNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.2.3.4",
		},
	}
	translate.VClusterName = testingutil.DefaultTestVClusterName
	pDNSService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, "kube-dns", "kube-system").Name,
			Namespace: testingutil.DefaultTestTargetNamespace,
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
		Name:      translate.Default.HostName(nil, "testpod", "testns").Name,
		Namespace: "test",
		Annotations: map[string]string{
			podtranslate.ClusterAutoScalerAnnotation:  "false",
			podtranslate.VClusterLabelsAnnotation:     "",
			podtranslate.NameAnnotation:               vObjectMeta.Name,
			podtranslate.NamespaceAnnotation:          vObjectMeta.Namespace,
			translate.NameAnnotation:                  vObjectMeta.Name,
			translate.UIDAnnotation:                   "",
			translate.NamespaceAnnotation:             vObjectMeta.Namespace,
			translate.KindAnnotation:                  corev1.SchemeGroupVersion.WithKind("Pod").String(),
			translate.HostNameAnnotation:              translate.Default.HostName(nil, "testpod", "testns").Name,
			translate.HostNamespaceAnnotation:         "test",
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
	pPodWithNodeName := pPodBase.DeepCopy()
	pPodWithNodeName.Spec.NodeName = "test456"

	vHostpathNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingutil.DefaultTestCurrentNamespace,
		},
	}

	vHostPathPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HostpathPodName,
			Namespace: testingutil.DefaultTestCurrentNamespace,
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

	vHostPath := fmt.Sprintf(podtranslate.VirtualPathTemplate, testingutil.DefaultTestCurrentNamespace, testingutil.DefaultTestVClusterName)

	hostToContainer := corev1.MountPropagationHostToContainer
	pHostPathPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vHostPathPod.Name, testingutil.DefaultTestCurrentNamespace).Name,
			Namespace: testingutil.DefaultTestTargetNamespace,

			Annotations: map[string]string{
				podtranslate.ClusterAutoScalerAnnotation:  "false",
				podtranslate.VClusterLabelsAnnotation:     "",
				podtranslate.NameAnnotation:               vHostPathPod.Name,
				podtranslate.NamespaceAnnotation:          vHostPathPod.Namespace,
				translate.NameAnnotation:                  vHostPathPod.Name,
				translate.NamespaceAnnotation:             vHostPathPod.Namespace,
				translate.UIDAnnotation:                   "",
				translate.KindAnnotation:                  corev1.SchemeGroupVersion.WithKind("Pod").String(),
				translate.HostNamespaceAnnotation:         "test",
				translate.HostNameAnnotation:              translate.Default.HostName(nil, vHostPathPod.Name, testingutil.DefaultTestCurrentNamespace).Name,
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

	testNodeName := "test123"
	pVclusterNodeService := pVclusterService.DeepCopy()
	pVclusterNodeService.Name = translate.SafeConcatName(testingutil.DefaultTestVClusterName, "node", testNodeName)

	pPodFakeKubelet := pPodBase.DeepCopy()
	pPodFakeKubelet.Spec.NodeName = testNodeName
	pPodFakeKubelet.Status.HostIP = "3.3.3.3"
	pPodFakeKubelet.Status.HostIPs = []corev1.HostIP{
		{IP: "3.3.3.3"},
	}

	vPodWithNodeName := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			NodeName: testNodeName,
		},
	}
	vPodWithHostIP := vPodWithNodeName.DeepCopy()
	vPodWithHostIP.Status.HostIP = pVclusterService.Spec.ClusterIP
	vPodWithHostIP.Status.HostIPs = []corev1.HostIP{
		{IP: pVclusterService.Spec.ClusterIP},
	}

	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
	}

	priorityClassName := "high-priority"
	vPodWithoutPriorityClass := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec:       corev1.PodSpec{},
	}
	vPodWithPriorityClass := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			PriorityClassName: priorityClassName,
		},
	}
	pPodWithPriorityClass := &corev1.Pod{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: ptr.To(false),
			EnableServiceLinks:           ptr.To(false),
			HostAliases: []corev1.HostAlias{{
				IP:        pVclusterService.Spec.ClusterIP,
				Hostnames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc"},
			}},
			Hostname:           vPodWithPriorityClass.Name,
			ServiceAccountName: "vc-workload-vcluster",
			PriorityClassName:  priorityClassName,
		},
	}
	pPodWithTranslatedPriorityClass := pPodWithPriorityClass.DeepCopy()
	pPodWithTranslatedPriorityClass.Spec.PriorityClassName = "vcluster-high-priority-x-test-x-vcluster"

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
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
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(vHostPathPod.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Fake Kubelet enabled with Node sync",
			InitialVirtualState:  []runtime.Object{testNode.DeepCopy(), vPodWithNodeName, vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{testNode.DeepCopy(), pVclusterNodeService.DeepCopy(), pPodFakeKubelet.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {vPodWithHostIP},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.Nodes.Selector.All = true
				ctx.Config.Networking.Advanced.ProxyKubelets.ByIP = true
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).Sync(syncContext, synccontext.NewSyncEventWithOld(pPodFakeKubelet, pPodFakeKubelet, vPodWithNodeName, vPodWithNodeName))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "From Host PriorityClasses sync enabled",
			InitialVirtualState:  []runtime.Object{vPodWithPriorityClass, vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pPodWithPriorityClass},
			},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.FromHost.PriorityClasses.Enabled = true
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(vPodWithPriorityClass))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "To Host PriorityClasses sync enabled",
			InitialVirtualState:  []runtime.Object{vPodWithPriorityClass, vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pPodWithTranslatedPriorityClass},
			},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.PriorityClasses.Enabled = true
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(vPodWithPriorityClass))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "To Host Pods.PriorityClassName set",
			InitialVirtualState:  []runtime.Object{vPodWithoutPriorityClass, vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {pPodWithPriorityClass},
			},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.Pods.PriorityClassName = priorityClassName
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(vPodWithoutPriorityClass))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "SyncToHost error when both HybridScheduling and Virtual Scheduler are enabled",
			InitialVirtualState:  []runtime.Object{vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.Pods.HybridScheduling.Enabled = true
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				vPod := &corev1.Pod{
					ObjectMeta: vObjectMeta,
				}
				_, err := syncer.(*podSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(vPod))
				assert.ErrorContains(t, err, "you are trying to use a vCluster pro feature")
			},
		},
	})
}
