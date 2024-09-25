package webhook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

var (
	secretName      = "sample-webhook-secret"
	deploymentName  = "sample-webhook-deployment"
	serviceName     = "e2e-test-webhook"
	roleBindingName = "webhook-auth-reader"

	image                   = "registry.k8s.io/e2e-test-images/agnhost:2.33"
	pauseImage              = "registry.k8s.io/pause:3.6"
	uniqueName              = "webhook-test-" + string(uuid.NewUUID())
	skipNamespaceLabelKey   = "skip-webhook-admission"
	skipNamespaceLabelValue = "yes"
	skippedNamespaceName    = "exempted-namesapce"
	disallowedPodName       = "disallowed-pod"
	toBeAttachedPodName     = "to-be-attached-pod"
	hangingPodName          = "hanging-pod"
	disallowedConfigMapName = "disallowed-configmap"
	allowedConfigMapName    = "allowed-configmap"
)

var _ = ginkgo.Describe("AdmissionWebhook", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	var certCtx *certContext
	servicePort := int32(8443)
	containerPort := int32(8444)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-webhook-%d-%s", iteration, random.String(5))

		// create test namespace
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns, Labels: map[string]string{uniqueName: "true"}}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		createWebhookConfigurationReadyNamespace(f, ns)

		ginkgo.By("Setting up server cert")
		certCtx = setupServerCert(f, ns, serviceName)
		createAuthReaderRoleBinding(f, ns)

		deployWebhookAndService(f, image, certCtx, servicePort, containerPort, ns)
	})

	ginkgo.AfterEach(func() {
		cleanWebhookTest(f, ns)
	})

	/*
		Release: v1.16
		Testname: Admission webhook, deny create
		Description: Register an admission webhook configuration that admits pod and configmap. Attempts to create
		non-compliant pods and configmaps, or update/patch compliant pods and configmaps to be non-compliant MUST
		be denied. An attempt to create a pod that causes a webhook to hang MUST result in a webhook timeout error,
		and the pod creation MUST be denied. An attempt to create a non-compliant configmap in a whitelisted
		namespace based on the webhook namespace selector MUST be allowed.
	*/
	ginkgo.It("should be able to deny pod and configmap creation", func() {
		webhookCleanup := registerWebhook(f, uniqueName, certCtx, servicePort, ns)
		defer webhookCleanup()
		testWebhook(f, ns)
	})

	/*
		Release: v1.16
		Testname: Admission webhook, deny attach
		Description: Register an admission webhook configuration that denies connecting to a pod's attach sub-resource.
		Attempts to attach MUST be denied.
	*/
	ginkgo.It("should be able to deny attaching pod", func() {
		webhookCleanup := registerWebhookForAttachingPod(f, uniqueName, certCtx, servicePort, ns)
		defer webhookCleanup()
		testAttachingPodWebhook(f, ns)
	})
})

// createWebhookConfigurationReadyNamespace creates a separate namespace for webhook configuration ready markers to
// prevent cross-talk with webhook configurations being tested.
func createWebhookConfigurationReadyNamespace(f *framework.Framework, namespace string) {
	ctx := f.Context
	_, err := f.VClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace + "-markers",
			Labels: map[string]string{uniqueName + "-markers": "true"},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err, "creating namespace for webhook configuration ready markers")
}

func registerWebhook(f *framework.Framework, configName string, certCtx *certContext, servicePort int32, namespace string) func() {
	client := f.VClusterClient
	ctx := f.Context
	ginkgo.By("Registering the webhook via the AdmissionRegistration API")

	// A webhook that cannot talk to server, with fail-open policy
	failOpenHook := failingWebhook(namespace, "fail-open.k8s.io", servicePort)
	policyIgnore := admissionregistrationv1.Ignore
	failOpenHook.FailurePolicy = &policyIgnore
	failOpenHook.NamespaceSelector = &metav1.LabelSelector{
		MatchLabels: map[string]string{uniqueName: "true"},
	}

	_, err := createValidatingWebhookConfiguration(f, &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			newDenyPodWebhookFixture(certCtx, servicePort, namespace),
			newDenyConfigMapWebhookFixture(certCtx, servicePort, namespace),
			// Server cannot talk to this webhook, so it always fails.
			// Because this webhook is configured fail-open, request should be admitted after the call fails.
			failOpenHook,

			// Register a webhook that can be probed by marker requests to detect when the configuration is ready.
			newValidatingIsReadyWebhookFixture(certCtx, servicePort, namespace),
		},
	})
	framework.ExpectNoError(err, "registering webhook config %s with namespace %s", configName, namespace)

	err = waitWebhookConfigurationReady(f, namespace)
	framework.ExpectNoError(err, "waiting for webhook configuration to be ready")

	return func() {
		err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, configName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	}
}

// failingWebhook returns a webhook with rule of create configmaps,
// but with an invalid client config so that server cannot communicate with it
func failingWebhook(namespace, name string, servicePort int32) admissionregistrationv1.ValidatingWebhook {
	sideEffectsNone := admissionregistrationv1.SideEffectClassNone

	return admissionregistrationv1.ValidatingWebhook{
		Name: name,
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"configmaps"},
			},
		}},
		ClientConfig: admissionregistrationv1.WebhookClientConfig{
			Service: &admissionregistrationv1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      strPtr("/configmaps"),
				Port:      ptr.To(servicePort),
			},
			// Without CA bundle, the call to webhook always fails
			CABundle: nil,
		},
		SideEffects:             &sideEffectsNone,
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
	}
}

func strPtr(s string) *string { return &s }

// createValidatingWebhookConfiguration ensures the webhook config scopes object or namespace selection
// to avoid interfering with other tests, then creates the config.
func createValidatingWebhookConfiguration(f *framework.Framework, config *admissionregistrationv1.ValidatingWebhookConfiguration) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	ctx := f.Context
	for _, webhook := range config.Webhooks {
		if webhook.NamespaceSelector != nil && webhook.NamespaceSelector.MatchLabels[uniqueName] == "true" {
			continue
		}
		if webhook.ObjectSelector != nil && webhook.ObjectSelector.MatchLabels[uniqueName] == "true" {
			continue
		}
		f.Log.Fatalf(`webhook %s in config %s has no namespace or object selector with %s="true", and can interfere with other tests`, webhook.Name, config.Name, uniqueName)
	}
	return f.VClusterClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, config, metav1.CreateOptions{})
}

func newDenyPodWebhookFixture(certCtx *certContext, servicePort int32, namespace string) admissionregistrationv1.ValidatingWebhook {
	sideEffectsNone := admissionregistrationv1.SideEffectClassNone
	return admissionregistrationv1.ValidatingWebhook{
		Name: "deny-unwanted-pod-container-name-and-label.k8s.io",
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods"},
			},
		}},
		ClientConfig: admissionregistrationv1.WebhookClientConfig{
			Service: &admissionregistrationv1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      strPtr("/pods"),
				Port:      ptr.To(servicePort),
			},
			CABundle: certCtx.signingCert,
		},
		SideEffects:             &sideEffectsNone,
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		// Scope the webhook to just this namespace
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{uniqueName: "true"},
		},
	}
}

func newDenyConfigMapWebhookFixture(certCtx *certContext, servicePort int32, namespace string) admissionregistrationv1.ValidatingWebhook {
	sideEffectsNone := admissionregistrationv1.SideEffectClassNone
	return admissionregistrationv1.ValidatingWebhook{
		Name: "deny-unwanted-configmap-data.k8s.io",
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"configmaps"},
			},
		}},
		// The webhook skips the namespace that has label "skip-webhook-admission":"yes"
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{uniqueName: "true"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      skipNamespaceLabelKey,
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{skipNamespaceLabelValue},
				},
			},
		},
		ClientConfig: admissionregistrationv1.WebhookClientConfig{
			Service: &admissionregistrationv1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      strPtr("/configmaps"),
				Port:      ptr.To(servicePort),
			},
			CABundle: certCtx.signingCert,
		},
		SideEffects:             &sideEffectsNone,
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
	}
}

// newValidatingIsReadyWebhookFixture creates a validating webhook that can be added to a webhook configuration and then probed
// with "marker" requests via waitWebhookConfigurationReady to wait for a webhook configuration to be ready.
func newValidatingIsReadyWebhookFixture(certCtx *certContext, servicePort int32, namespace string) admissionregistrationv1.ValidatingWebhook {
	sideEffectsNone := admissionregistrationv1.SideEffectClassNone
	failOpen := admissionregistrationv1.Ignore
	return admissionregistrationv1.ValidatingWebhook{
		Name: "validating-is-webhook-configuration-ready.k8s.io",
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"configmaps"},
			},
		}},
		ClientConfig: admissionregistrationv1.WebhookClientConfig{
			Service: &admissionregistrationv1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      strPtr("/always-deny"),
				Port:      ptr.To(servicePort),
			},
			CABundle: certCtx.signingCert,
		},
		// network failures while the service network routing is being set up should be ignored by the marker
		FailurePolicy:           &failOpen,
		SideEffects:             &sideEffectsNone,
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		// Scope the webhook to just the markers namespace
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{uniqueName + "-markers": "true"},
		},
		// appease createValidatingWebhookConfiguration isolation requirements
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{uniqueName: "true"},
		},
	}
}

// waitWebhookConfigurationReady sends "marker" requests until a webhook configuration is ready.
// A webhook created with newValidatingIsReadyWebhookFixture or newMutatingIsReadyWebhookFixture should first be added to
// the webhook configuration.
func waitWebhookConfigurationReady(f *framework.Framework, namespace string) error {
	cmClient := f.VClusterClient.CoreV1().ConfigMaps(namespace + "-markers")
	return wait.PollUntilContextTimeout(f.Context, 100*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		marker := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: string(uuid.NewUUID()),
				Labels: map[string]string{
					uniqueName: "true",
				},
			},
		}
		_, err := cmClient.Create(ctx, marker, metav1.CreateOptions{})
		if err != nil {
			// The always-deny webhook does not provide a reason, so check for the error string we expect
			if strings.Contains(err.Error(), "denied") {
				return true, nil
			}
			return false, err
		}
		// best effort cleanup of markers that are no longer needed
		_ = cmClient.Delete(ctx, marker.GetName(), metav1.DeleteOptions{})
		f.Log.Infof("Waiting for webhook configuration to be ready...")
		return false, nil
	})
}

func testWebhook(f *framework.Framework, namespace string) {
	ginkgo.By("create a pod that should be denied by the webhook")
	client := f.VClusterClient
	ctx := f.Context
	// Creating the pod, the request should be rejected
	pod := nonCompliantPod()
	_, err := client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	framework.ExpectError(err, "create pod %s in namespace %s should have been denied by webhook", pod.Name, namespace)
	expectedErrMsg1 := "the pod contains unwanted container name"
	if !strings.Contains(err.Error(), expectedErrMsg1) {
		f.Log.Fatalf("expect error contains %q, got %q", expectedErrMsg1, err.Error())
	}
	expectedErrMsg2 := "the pod contains unwanted label"
	if !strings.Contains(err.Error(), expectedErrMsg2) {
		f.Log.Fatalf("expect error contains %q, got %q", expectedErrMsg2, err.Error())
	}

	ginkgo.By("create a pod that causes the webhook to hang")
	client = f.VClusterClient
	// Creating the pod, the request should be rejected
	pod = hangingPod()
	_, err = client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	framework.ExpectError(err, "create pod %s in namespace %s should have caused webhook to hang", pod.Name, namespace)
	// ensure the error is webhook-related, not client-side
	if !strings.Contains(err.Error(), "webhook") {
		f.Log.Fatalf("expect error %q, got %q", "webhook", err.Error())
	}
	// ensure the error is a timeout
	if !strings.Contains(err.Error(), "deadline") {
		f.Log.Fatalf("expect error %q, got %q", "deadline", err.Error())
	}
	// ensure the pod was not actually created
	if _, err := client.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{}); !kerrors.IsNotFound(err) {
		f.Log.Fatalf("expect notfound error looking for rejected pod, got %v", err)
	}

	ginkgo.By("create a configmap that should be denied by the webhook")
	// Creating the configmap, the request should be rejected
	configmap := nonCompliantConfigMap()
	_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, configmap, metav1.CreateOptions{})
	framework.ExpectError(err, "create configmap %s in namespace %s should have been denied by the webhook", configmap.Name, namespace)
	expectedErrMsg := "the configmap contains unwanted key and value"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		f.Log.Fatalf("expect error contains %q, got %q", expectedErrMsg, err.Error())
	}

	ginkgo.By("create a configmap that should be admitted by the webhook")
	// Creating the configmap, the request should be admitted
	configmap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: allowedConfigMapName,
		},
		Data: map[string]string{
			"admit": "this",
		},
	}
	_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, configmap, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create configmap %s in namespace: %s", configmap.Name, namespace)

	ginkgo.By("update (PUT) the admitted configmap to a non-compliant one should be rejected by the webhook")
	toNonCompliantFn := func(cm *corev1.ConfigMap) {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data["webhook-e2e-test"] = "webhook-disallow"
	}
	_, err = updateConfigMap(ctx, client, namespace, allowedConfigMapName, toNonCompliantFn)
	framework.ExpectError(err, "update (PUT) admitted configmap %s in namespace %s to a non-compliant one should be rejected by webhook", allowedConfigMapName, namespace)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		f.Log.Fatalf("expect error contains %q, got %q", expectedErrMsg, err.Error())
	}

	ginkgo.By("update (PATCH) the admitted configmap to a non-compliant one should be rejected by the webhook")
	patch := nonCompliantConfigMapPatch()
	_, err = client.CoreV1().ConfigMaps(namespace).Patch(ctx, allowedConfigMapName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	framework.ExpectError(err, "update admitted configmap %s in namespace %s by strategic merge patch to a non-compliant one should be rejected by webhook. Patch: %+v", allowedConfigMapName, namespace, patch)
	if !strings.Contains(err.Error(), expectedErrMsg) {
		f.Log.Fatalf("expect error contains %q, got %q", expectedErrMsg, err.Error())
	}

	ginkgo.By("create a namespace that bypass the webhook")
	err = createNamespace(f, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: skippedNamespaceName,
		Labels: map[string]string{
			skipNamespaceLabelKey: skipNamespaceLabelValue,
			uniqueName:            "true",
		},
	}})
	framework.ExpectNoError(err, "creating namespace %q", skippedNamespaceName)
	// clean up the namespace
	defer func() {
		_ = client.CoreV1().Namespaces().Delete(ctx, skippedNamespaceName, metav1.DeleteOptions{})
	}()
	ginkgo.By("create a configmap that violates the webhook policy but is in a whitelisted namespace")
	configmap = nonCompliantConfigMap()
	_, err = client.CoreV1().ConfigMaps(skippedNamespaceName).Create(ctx, configmap, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create configmap %s in namespace: %s", configmap.Name, skippedNamespaceName)
}

func createNamespace(f *framework.Framework, ns *corev1.Namespace) error {
	return wait.PollUntilContextTimeout(f.Context, 100*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := f.VClusterClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			if strings.HasPrefix(err.Error(), "object is being deleted:") {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func nonCompliantPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: disallowedPodName,
			Labels: map[string]string{
				"webhook-e2e-test": "webhook-disallow",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "webhook-disallow",
					Image: pauseImage,
				},
			},
		},
	}
}

func hangingPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: hangingPodName,
			Labels: map[string]string{
				"webhook-e2e-test": "wait-forever",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "wait-forever",
					Image: pauseImage,
				},
			},
		},
	}
}

func nonCompliantConfigMap() *corev1.ConfigMap {
	return namedNonCompliantConfigMap(disallowedConfigMapName)
}

func namedNonCompliantConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string]string{
			"webhook-e2e-test": "webhook-disallow",
		},
	}
}

func nonCompliantConfigMapPatch() string {
	return `{"data":{"webhook-e2e-test":"webhook-disallow"}}`
}

type updateConfigMapFn func(cm *corev1.ConfigMap)

func updateConfigMap(ctx context.Context, c *kubernetes.Clientset, ns, name string, update updateConfigMapFn) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	pollErr := wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		var err error
		if cm, err = c.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{}); err != nil {
			return false, err
		}
		update(cm)
		if cm, err = c.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err == nil {
			return true, nil
		}
		// Only retry update on conflict
		if !kerrors.IsConflict(err) {
			return false, err
		}
		return false, nil
	})
	return cm, pollErr
}

func cleanWebhookTest(f *framework.Framework, namespace string) {
	ctx := f.Context
	client := f.VClusterClient
	_ = client.CoreV1().Services(namespace).Delete(ctx, serviceName, metav1.DeleteOptions{})
	_ = client.AppsV1().Deployments(namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{})
	_ = client.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	_ = client.RbacV1().RoleBindings("kube-system").Delete(ctx, roleBindingName, metav1.DeleteOptions{})

	err := f.DeleteTestNamespace(namespace, false)
	framework.ExpectNoError(err)

	err = f.DeleteTestNamespace(namespace+"-markers", false)
	framework.ExpectNoError(err)
}

func createAuthReaderRoleBinding(f *framework.Framework, namespace string) {
	ginkgo.By("Create role binding to let webhook read extension-apiserver-authentication")
	client := f.VClusterClient
	ctx := f.Context
	// Create the role binding to allow the webhook read the extension-apiserver-authentication configmap
	_, err := client.RbacV1().RoleBindings("kube-system").Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
			Annotations: map[string]string{
				rbacv1.AutoUpdateAnnotationKey: "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "",
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},

		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil && kerrors.IsAlreadyExists(err) {
		f.Log.Infof("role binding %s already exists", roleBindingName)
	} else {
		framework.ExpectNoError(err, "creating role binding %s:webhook to access configMap", namespace)
	}
}

func deployWebhookAndService(f *framework.Framework, image string, certCtx *certContext, servicePort int32, containerPort int32, namespace string) {
	ginkgo.By("Deploying the webhook pod")
	client := f.VClusterClient
	ctx := f.Context

	// Creating the secret that contains the webhook's cert.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": certCtx.cert,
			"tls.key": certCtx.key,
		},
	}

	_, err := client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	framework.ExpectNoError(err, "creating secret %q in namespace %q", secretName, namespace)

	// Create the deployment of the webhook
	podLabels := map[string]string{"app": "sample-webhook", "webhook": "true"}
	replicas := int32(1)
	mounts := []corev1.VolumeMount{
		{
			Name:      "webhook-certs",
			ReadOnly:  true,
			MountPath: "/webhook.local.config/certificates",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "webhook-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: secretName},
			},
		},
	}
	containers := []corev1.Container{
		{
			Name:         "sample-webhook",
			VolumeMounts: mounts,
			Args: []string{
				"webhook",
				"--tls-cert-file=/webhook.local.config/certificates/tls.crt",
				"--tls-private-key-file=/webhook.local.config/certificates/tls.key",
				"--alsologtostderr",
				"-v=4",
				// Use a non-default port for containers.
				fmt.Sprintf("--port=%d", containerPort),
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Scheme: corev1.URISchemeHTTPS,
						Port:   intstr.FromInt(int(containerPort)),
						Path:   "/readyz",
					},
				},
				PeriodSeconds:    1,
				SuccessThreshold: 1,
				FailureThreshold: 30,
			},
			Image: image,
			Ports: []corev1.ContainerPort{{ContainerPort: containerPort}},
		},
	}
	d := newDeployment(deploymentName, replicas, podLabels, "", "", appsv1.RollingUpdateDeploymentStrategyType)
	d.Spec.Template.Spec.Containers = containers
	d.Spec.Template.Spec.Volumes = volumes

	// deployment, err := client.AppsV1().Deployments(namespace).Create(ctx, d, metav1.CreateOptions{})
	_, err = client.AppsV1().Deployments(namespace).Create(ctx, d, metav1.CreateOptions{})
	framework.ExpectNoError(err, "creating deployment %s in namespace %s", deploymentName, namespace)
	ginkgo.By("Wait for the deployment to be ready")
	time.Sleep(1 * time.Minute) //TODO: replace with polling

	ginkgo.By("Deploying the webhook service")

	serviceLabels := map[string]string{"webhook": "true"}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName,
			Labels:    map[string]string{"test": "webhook"},
		},
		Spec: corev1.ServiceSpec{
			Selector: serviceLabels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       servicePort,
					TargetPort: intstr.FromInt(int(containerPort)),
				},
			},
		},
	}
	_, err = client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	framework.ExpectNoError(err, "creating service %s in namespace %s", serviceName, namespace)

	ginkgo.By("Wait for the service to be paired with the endpoint")
	time.Sleep(1 * time.Minute) //TODO: replace with polling
}

// NewDeployment returns a deployment spec with the specified argument.
func newDeployment(deploymentName string, replicas int32, podLabels map[string]string, containerName, image string, strategyType appsv1.DeploymentStrategyType) *appsv1.Deployment {
	zero := int64(0)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: podLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Strategy: appsv1.DeploymentStrategy{
				Type: strategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &zero,
					Containers: []corev1.Container{
						{
							Name:            containerName,
							Image:           image,
							SecurityContext: &corev1.SecurityContext{},
						},
					},
				},
			},
		},
	}
}

func registerWebhookForAttachingPod(f *framework.Framework, configName string, certCtx *certContext, servicePort int32, ns string) func() {
	client := f.VClusterClient
	ctx := f.Context
	ginkgo.By("Registering the webhook via the AdmissionRegistration API")

	namespace := ns
	sideEffectsNone := admissionregistrationv1.SideEffectClassNone

	_, err := createValidatingWebhookConfiguration(f, &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: configName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "deny-attaching-pod.k8s.io",
				Rules: []admissionregistrationv1.RuleWithOperations{{
					Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Connect},
					Rule: admissionregistrationv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods/attach"},
					},
				}},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: namespace,
						Name:      serviceName,
						Path:      strPtr("/pods/attach"),
						Port:      ptr.To(servicePort),
					},
					CABundle: certCtx.signingCert,
				},
				SideEffects:             &sideEffectsNone,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
				// Scope the webhook to just this namespace
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{uniqueName: "true"},
				},
			},
			// Register a webhook that can be probed by marker requests to detect when the configuration is ready.
			newValidatingIsReadyWebhookFixture(certCtx, servicePort, ns),
		},
	})
	framework.ExpectNoError(err, "registering webhook config %s with namespace %s", configName, namespace)

	err = waitWebhookConfigurationReady(f, ns)
	framework.ExpectNoError(err, "waiting for webhook configuration to be ready")

	return func() {
		_ = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, configName, metav1.DeleteOptions{})
	}
}

func toBeAttachedPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: toBeAttachedPodName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container1",
					Image: pauseImage,
				},
			},
		},
	}
}

func testAttachingPodWebhook(f *framework.Framework, ns string) {
	ginkgo.By("create a pod")
	client := f.VClusterClient
	ctx := f.Context
	pod := toBeAttachedPod()
	_, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create pod %s in namespace: %s", pod.Name, ns)
	err = f.WaitForPodRunning(pod.Name, ns)
	framework.ExpectNoError(err, "error while waiting for pod %s to go to Running phase in namespace: %s", pod.Name, ns)

	ginkgo.By("'kubectl attach' the pod, should be denied by the webhook")
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	_, err = framework.NewKubectlCommand(f.VClusterKubeConfigFile.Name(), ns, "attach", fmt.Sprintf("--namespace=%v", ns), pod.Name, "-i", "-c=container1").WithTimeout(timer.C).Exec()
	framework.ExpectError(err, "'kubectl attach' the pod, should be denied by the webhook")
	if e, a := "attaching to pod 'to-be-attached-pod' is not allowed", err.Error(); !strings.Contains(a, e) {
		framework.Failf("unexpected 'kubectl attach' error message. expected to contain %q, got %q", e, a)
	}
}
