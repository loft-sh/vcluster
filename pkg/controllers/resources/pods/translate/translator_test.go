package translate

import (
	"fmt"
	"path/filepath"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

func TestPodAffinityTermsTranslation(t *testing.T) {
	longKey := "pretty-loooooooooooong-test-key"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-little-pody",
			Namespace: "equestria",
		},
	}
	basicSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"random-key": "value"},
	}
	basicSelectorTranslatedWithMarker := &metav1.LabelSelector{MatchLabels: map[string]string{}}
	for k, v := range basicSelector.MatchLabels {
		basicSelectorTranslatedWithMarker.MatchLabels[translate.HostLabel(k)] = v
	}
	basicSelectorTranslatedWithMarker.MatchLabels[translate.MarkerLabel] = translate.VClusterName

	testCases := []translatePodAffinityTermTestCase{
		{
			name: "no selector",
			term: corev1.PodAffinityTerm{
				Namespaces: []string{"blabla"},
			},
			expectedTerm: corev1.PodAffinityTerm{},
		},
		{
			name: "empty namespaces array and null namespaceSelector",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				Namespaces:    []string{},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{MatchLabels: map[string]string{translate.NamespaceLabel: pod.GetNamespace()}},
				),
			},
		},
		{
			name: "null namespaceSelector and defined namespaces array",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				Namespaces:    []string{pod.GetNamespace(), "dummy namespace"},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendNamespacesToMatchExpressions(
					basicSelectorTranslatedWithMarker,
					pod.GetNamespace(), "dummy namespace"),
			},
		},
		{
			name: "empty namespaceSelector and defined namespaces array",
			term: corev1.PodAffinityTerm{
				LabelSelector:     basicSelector,
				Namespaces:        []string{pod.GetNamespace()},
				NamespaceSelector: &metav1.LabelSelector{},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: basicSelectorTranslatedWithMarker,
			},
		},
		{ // TODO: remove once we implement the support for using .namespaces and .namespaceSelector together
			name: "namespaces array specified together with the namespaceSelector",
			term: corev1.PodAffinityTerm{
				LabelSelector:     basicSelector,
				Namespaces:        []string{pod.GetNamespace()},
				NamespaceSelector: basicSelector,
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: appendNamespacesToMatchExpressions(
					basicSelectorTranslatedWithMarker,
					pod.GetNamespace(),
				),
			},
			expectedEvents: []string{"Warning SyncWarning Inter-pod affinity rule(s) that use both .namespaces and .namespaceSelector fields in the same term are not supported by vcluster yet. The .namespaceSelector fields of the unsupported affinity entries will be ignored."},
		},
		{
			name: "defined namespaceSelector and no namespaces",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						longKey: "good-value",
					},
				},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{MatchLabels: map[string]string{
						translate.HostLabelNamespace(longKey): "good-value",
					}},
				),
			},
		},
		{
			name: "namespaceSelector with MatchExpressions",
			term: corev1.PodAffinityTerm{
				LabelSelector: basicSelector,
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      longKey,
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"bad-value"},
						},
					},
				},
			},
			expectedTerm: corev1.PodAffinityTerm{
				LabelSelector: translate.MergeLabelSelectors(
					basicSelectorTranslatedWithMarker,
					&metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      translate.HostLabelNamespace(longKey),
								Operator: metav1.LabelSelectorOpNotIn,
								Values:   []string{"bad-value"},
							},
						},
					},
				),
			},
		},
	}

	for _, testCase := range testCases {
		fakeRecorder := record.NewFakeRecorder(10)
		tr := &translator{
			eventRecorder: fakeRecorder,
			log:           loghelper.New("pods-syncer-translator-test"),
		}

		result := tr.translatePodAffinityTerm(pod, testCase.term)
		assert.Assert(t, cmp.DeepEqual(result, testCase.expectedTerm), "Unexpected translation of the PodAffinityTerm in the '%s' test case", testCase.name)

		// read the events from the recorder mock
		close(fakeRecorder.Events)
		events := make([]string, 0)
		for v := range fakeRecorder.Events {
			events = append(events, v)
		}
		if len(events) == 0 {
			events = nil
		}

		assert.Assert(t, cmp.DeepEqual(events, testCase.expectedEvents), "Unexpected Event in the '%s' test case", testCase.name)
	}
}

type translatePodAffinityTermTestCase struct {
	name           string
	term           corev1.PodAffinityTerm
	expectedTerm   corev1.PodAffinityTerm
	expectedEvents []string
}

func TestVolumeTranslation(t *testing.T) {
	virtualPath := fmt.Sprintf(VirtualPathTemplate, testingutil.DefaultTestCurrentNamespace, testingutil.DefaultTestVClusterName)
	hostToContainer := corev1.MountPropagationHostToContainer

	testCases := []translatePodVolumesTestCase{
		{
			name: "ephemeral volume",
			vPod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "eph-vol",
							VolumeSource: corev1.VolumeSource{
								Ephemeral: &corev1.EphemeralVolumeSource{
									VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{},
								},
							},
						},
					},
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: "eph-vol",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: translate.Default.HostName(nil, "pod-name-eph-vol", "test-ns").Name,
						},
						Ephemeral: nil,
					},
				},
			},
		},
		{
			name:                   "hostpath mapper",
			mountPhysicalHostPaths: true,
			vPod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "varlog",
									MountPath: "/var/log/pods/",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log/pods/",
								},
							},
						},
					},
				},
			},
			expectedVolumes: []corev1.Volume{
				{
					Name: "varlog",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: virtualPath + "/log/pods",
						},
					},
				},
				{
					Name: fmt.Sprintf("%s-%s", "varlog", PhysicalVolumeNameSuffix),
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: PodLoggingHostPath,
						},
					},
				},
			},
			expectedVolumeMounts: []corev1.VolumeMount{
				{
					Name:             "varlog",
					ReadOnly:         true,
					MountPath:        "/var/log/pods/",
					MountPropagation: &hostToContainer,
				},
				{
					Name:      fmt.Sprintf("%s-%s", "varlog", PhysicalVolumeNameSuffix),
					ReadOnly:  true,
					MountPath: PhysicalPodLogVolumeMountPath,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeRecorder := record.NewFakeRecorder(10)
			pClient := testingutil.NewFakeClient(scheme.Scheme)
			vClient := testingutil.NewFakeClient(scheme.Scheme)
			registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
			tr := &translator{
				eventRecorder:          fakeRecorder,
				log:                    loghelper.New("pods-syncer-translator-test"),
				pClient:                pClient,
				mountPhysicalHostPaths: testCase.mountPhysicalHostPaths,
				virtualPodLogsPath:     filepath.Join(virtualPath, "log", "pods"),
			}

			pPod := testCase.vPod.DeepCopy()
			err := tr.translateVolumes(registerCtx.ToSyncContext("pods-syncer-translator-test"), pPod, &testCase.vPod)
			assert.NilError(t, err)
			assert.Assert(t, cmp.DeepEqual(pPod.Spec.Volumes, testCase.expectedVolumes), "Unexpected translation of the Volumes in the '%s' test case", testCase.name)

			if len(testCase.expectedVolumeMounts) > 0 {
				assert.Assert(t, cmp.DeepEqual(pPod.Spec.Containers[0].VolumeMounts, testCase.expectedVolumeMounts), "Unexpected translation of the Volume Mounts in the '%s' test case", testCase.name)
			}
		})
	}
}

func TestRewriteHostsTranslation(t *testing.T) {
	testCases := []struct {
		name      string
		configMod func(*config.VirtualClusterConfig)
		pod       corev1.Pod
		test      func(*testing.T, *corev1.Pod)
	}{
		{
			name: "Add init container that rewrites /etc/hosts",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Subdomain: "subdomain",
					Containers: []corev1.Container{
						{},
					},
				},
			},
			test: func(t *testing.T, p *corev1.Pod) {
				assert.Equal(t, len(p.Spec.InitContainers), 1)
				initC := p.Spec.InitContainers[0]
				assert.Equal(t, initC.Args[1], `sed -E -e 's/^(\d+.\d+.\d+.\d+\s+)pod-name$/\1 pod-name.subdomain.test-ns.cluster.local pod-name/' /etc/hosts > /hosts/hosts`)
				cont := p.Spec.Containers[0]
				assert.Equal(t, len(cont.VolumeMounts), 1)
				assert.Equal(t, cont.VolumeMounts[0].MountPath, "/etc/hosts")
			},
		},
		{
			name: "Use default registry if specified",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Subdomain: "subdomain",
				},
			},
			configMod: func(c *config.VirtualClusterConfig) {
				c.ControlPlane.Advanced.DefaultImageRegistry = "my-registry"
			},
			test: func(t *testing.T, p *corev1.Pod) {
				assert.Equal(t, len(p.Spec.InitContainers), 1)
				initC := p.Spec.InitContainers[0]
				assert.Equal(t, initC.Image, "my-registry/library/alpine:3.20")
			},
		},
		{
			name: "Use image overrides if specified",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "test-ns",
				},
				Spec: corev1.PodSpec{
					Subdomain: "subdomain",
				},
			},
			configMod: func(c *config.VirtualClusterConfig) {
				c.ControlPlane.Advanced.DefaultImageRegistry = "my-registry"
				c.Sync.ToHost.Pods.RewriteHosts.InitContainer.Image = vclusterconfig.Image{
					Registry:   "private",
					Repository: "minimal-sed",
					Tag:        "very-specific-tag",
				}
			},
			test: func(t *testing.T, p *corev1.Pod) {
				assert.Equal(t, len(p.Spec.InitContainers), 1)
				initC := p.Spec.InitContainers[0]
				assert.Equal(t, initC.Image, "private/minimal-sed:very-specific-tag")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := testingutil.NewFakeConfig()
			if testCase.configMod != nil {
				testCase.configMod(cfg)
			}
			pClient := testingutil.NewFakeClient(scheme.Scheme)
			vClient := testingutil.NewFakeClient(scheme.Scheme)
			registerCtx := generictesting.NewFakeRegisterContext(cfg, pClient, vClient)
			fakeRecorder := record.NewFakeRecorder(10)

			tr, err := NewTranslator(registerCtx, fakeRecorder)
			assert.NilError(t, err)

			pod := testCase.pod.DeepCopy()
			tr.(*translator).rewritePodHostnameFQDN(pod, pod.Name, pod.Name, fmt.Sprintf("%s.%s.%s.cluster.local", pod.Name, pod.Spec.Subdomain, pod.Namespace))

			testCase.test(t, pod)
		})
	}
}

type translatePodVolumesTestCase struct {
	name                   string
	vPod                   corev1.Pod
	expectedVolumes        []corev1.Volume
	expectedVolumeMounts   []corev1.VolumeMount
	mountPhysicalHostPaths bool
}

func appendNamespacesToMatchExpressions(source *metav1.LabelSelector, namespaces ...string) *metav1.LabelSelector {
	ls := source.DeepCopy()
	ls.MatchExpressions = append(ls.MatchExpressions, metav1.LabelSelectorRequirement{
		Key:      translate.NamespaceLabel,
		Operator: metav1.LabelSelectorOpIn,
		Values:   append([]string{}, namespaces...),
	})
	return ls
}
