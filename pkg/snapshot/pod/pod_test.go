package pod

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
)

// TestCreateSnapshotPodStoresCredentialsInSecret: object store credentials
// must be injected from a Secret via secretKeyRef, never written as a
// plaintext value into the Pod spec.
func TestCreateSnapshotPodStoresCredentialsInSecret(t *testing.T) {
	const ns = "vcluster-my-team"
	kubeClient := fake.NewSimpleClientset()

	vCluster := &find.VCluster{
		Name:      "my-vcluster",
		Namespace: ns,
		StatefulSet: &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "syncer", Image: "vcluster:dev"}},
					},
				},
			},
		},
	}
	snapshotOptions := &snapshotapi.Options{
		Type: "s3",
		S3: snapshotapi.S3Options{
			Bucket:          "my-bucket",
			Key:             "snapshot.tar.gz",
			AccessKeyID:     "AKIAEXAMPLE",
			SecretAccessKey: "super-secret-value",
		},
	}

	const secretName, podName = "vcluster-snapshot-options-test", "vcluster-snapshot-test"
	secret, err := createOptionsSecret(context.Background(), kubeClient, vCluster, secretName, snapshotOptions)
	if err != nil {
		t.Fatalf("createOptionsSecret: %v", err)
	}
	if secret.Name != secretName {
		t.Fatalf("secret name = %q, want %q", secret.Name, secretName)
	}

	pod, err := CreateSnapshotPod(context.Background(), kubeClient, []string{"/vcluster", "snapshot"}, vCluster, &Options{}, podName, secret.Name, snapshotOptions, log.NewDiscardLogger(logrus.InfoLevel))
	if err != nil {
		t.Fatalf("CreateSnapshotPod: %v", err)
	}
	if pod.Name != podName {
		t.Fatalf("pod name = %q, want %q", pod.Name, podName)
	}

	// find the storage options env var on the snapshot container
	var found *corev1.EnvVar
	for i, e := range pod.Spec.Containers[0].Env {
		if e.Name == constants.VClusterStorageOptionsEnv {
			found = &pod.Spec.Containers[0].Env[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("%s env var not set on snapshot container", constants.VClusterStorageOptionsEnv)
	}
	if found.Value != "" {
		t.Fatalf("credentials leaked: %s has a plaintext value in the pod spec", constants.VClusterStorageOptionsEnv)
	}
	if found.ValueFrom == nil || found.ValueFrom.SecretKeyRef == nil {
		t.Fatalf("%s must be sourced from a secretKeyRef", constants.VClusterStorageOptionsEnv)
	}
	if found.ValueFrom.SecretKeyRef.Name != secret.Name {
		t.Fatalf("secretKeyRef points at %q, want %q", found.ValueFrom.SecretKeyRef.Name, secret.Name)
	}
	if found.ValueFrom.SecretKeyRef.Key != constants.VClusterStorageOptionsEnv {
		t.Fatalf("secretKeyRef key = %q, want %q", found.ValueFrom.SecretKeyRef.Key, constants.VClusterStorageOptionsEnv)
	}

	// the secret must round-trip back to the original options (credentials included),
	// as base64(json) — exactly what the in-pod reader (ParseOptionsFromEnv) expects.
	stored, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secret.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(stored.Data[constants.VClusterStorageOptionsEnv]))
	if err != nil {
		t.Fatalf("stored options are not base64: %v", err)
	}
	roundTripped := &snapshotapi.Options{}
	if err := json.Unmarshal(decoded, roundTripped); err != nil {
		t.Fatalf("stored options are not valid json: %v", err)
	}
	if !reflect.DeepEqual(roundTripped, snapshotOptions) {
		t.Fatalf("stored options don't round-trip: got %+v, want %+v", roundTripped, snapshotOptions)
	}
}

// TestSetSecretOwner: the options Secret must gain a controller ownerReference
// pointing at the snapshot Pod, so Kubernetes garbage-collects it on a hard
// kill where the deferred cleanup never runs.
func TestSetSecretOwner(t *testing.T) {
	const ns = "vcluster-my-team"
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "vcluster-snapshot-options-abc", Namespace: ns},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "vcluster-snapshot-pod", Namespace: ns, UID: "pod-uid-123"},
	}
	kubeClient := fake.NewSimpleClientset(secret)

	setSecretOwner(ctx, kubeClient, secret, pod)

	stored, err := kubeClient.CoreV1().Secrets(ns).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if len(stored.OwnerReferences) != 1 {
		t.Fatalf("want exactly one ownerReference, got %d", len(stored.OwnerReferences))
	}
	ref := stored.OwnerReferences[0]
	if ref.Kind != "Pod" || ref.APIVersion != corev1.SchemeGroupVersion.String() {
		t.Fatalf("owner GVK = %s/%s, want %s/Pod", ref.APIVersion, ref.Kind, corev1.SchemeGroupVersion.String())
	}
	if ref.Name != pod.Name || ref.UID != pod.UID {
		t.Fatalf("owner = %s/%s, want %s/%s", ref.Name, ref.UID, pod.Name, pod.UID)
	}
	if ref.Controller == nil || !*ref.Controller {
		t.Fatalf("ownerReference must be a controller ref")
	}
	if ref.BlockOwnerDeletion == nil || *ref.BlockOwnerDeletion {
		t.Fatalf("blockOwnerDeletion = %v, want false", ptr.Deref(ref.BlockOwnerDeletion, true))
	}
}

// TestRunSnapshotPodDeletesSecretOnPodCreateFailure: if pod creation fails, the
// deferred cleanup must still delete the credential-bearing options Secret so it
// doesn't leak. Also asserts both names are pinned client-side and share a
// suffix, which is what lets the interrupt path clean up objects it never saw
// created.
func TestRunSnapshotPodDeletesSecretOnPodCreateFailure(t *testing.T) {
	const ns = "vcluster-my-team"
	var createdSecretName, createdPodName string
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("create", "secrets", func(action clienttesting.Action) (bool, runtime.Object, error) {
		createdSecretName = action.(clienttesting.CreateAction).GetObject().(*corev1.Secret).Name
		return false, nil, nil
	})
	kubeClient.PrependReactor("create", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		createdPodName = action.(clienttesting.CreateAction).GetObject().(*corev1.Pod).Name
		return true, nil, errors.New("boom")
	})

	vCluster := &find.VCluster{
		Name:      "my-vcluster",
		Namespace: ns,
		StatefulSet: &appsv1.StatefulSet{
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "syncer", Image: "vcluster:dev"}},
					},
				},
			},
		},
	}
	snapshotOptions := &snapshotapi.Options{
		Type: "s3",
		S3:   snapshotapi.S3Options{Bucket: "my-bucket", SecretAccessKey: "super-secret-value"},
	}

	err := RunSnapshotPod(context.Background(), nil, kubeClient, []string{"/vcluster", "snapshot"}, vCluster, &Options{}, snapshotOptions, log.NewDiscardLogger(logrus.InfoLevel))
	if err == nil {
		t.Fatalf("RunSnapshotPod: expected error from pod creation, got nil")
	}

	secrets, err := kubeClient.CoreV1().Secrets(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if len(secrets.Items) != 0 {
		t.Fatalf("options secret leaked after pod-creation failure: %d secret(s) remain", len(secrets.Items))
	}

	podSuffix := strings.TrimPrefix(createdPodName, "vcluster-snapshot-")
	secretSuffix := strings.TrimPrefix(createdSecretName, "vcluster-snapshot-options-")
	if podSuffix == "" || podSuffix == createdPodName {
		t.Fatalf("pod name %q is not a client-side vcluster-snapshot-<suffix> name", createdPodName)
	}
	if secretSuffix != podSuffix {
		t.Fatalf("secret %q and pod %q don't share a suffix, they can't be correlated in the namespace", createdSecretName, createdPodName)
	}
}

// TestDeleteOptionsSecret: the helper deletes the Secret and tolerates a second
// call on an already-absent Secret (the GC-was-faster case).
func TestDeleteOptionsSecret(t *testing.T) {
	const ns, name = "vcluster-my-team", "vcluster-snapshot-options-abc"
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}})

	deleteOptionsSecret(ctx, kubeClient, ns, name)

	if _, err := kubeClient.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{}); !kerrors.IsNotFound(err) {
		t.Fatalf("secret not deleted, get returned: %v", err)
	}

	// second call on the now-absent secret must not panic or surface an error
	deleteOptionsSecret(ctx, kubeClient, ns, name)
}

// TestDeleteSnapshotPod: the helper deletes the pod and tolerates a name that
// was never created - the interrupt path deletes by name before knowing whether
// the create landed.
func TestDeleteSnapshotPod(t *testing.T) {
	const ns, name = "vcluster-my-team", "vcluster-snapshot-abc"
	ctx := context.Background()
	kubeClient := fake.NewSimpleClientset(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}})

	deleteSnapshotPod(ctx, kubeClient, ns, name)

	if _, err := kubeClient.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{}); !kerrors.IsNotFound(err) {
		t.Fatalf("pod not deleted, get returned: %v", err)
	}

	deleteSnapshotPod(ctx, kubeClient, ns, "vcluster-snapshot-never-created")
}
