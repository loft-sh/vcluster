package certs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckCertsExpiring_DiskValid(t *testing.T) {
	dir := t.TempDir()

	validPEM := newSelfSignedCertPEM(t, "valid-cert", time.Now().Add(365*24*time.Hour))

	// Write valid certs for all cert map entries
	for fromName, secretKey := range certMap {
		path := filepath.Join(dir, fromName)
		err := os.MkdirAll(filepath.Dir(path), 0777)
		assert.NilError(t, err)

		var data []byte
		switch {
		case filepath.Ext(secretKey) == ".crt":
			data = validPEM
		default:
			data = []byte("placeholder")
		}
		err = os.WriteFile(path, data, 0666)
		assert.NilError(t, err)
	}

	expiring, err := checkCertsExpiring(dir)
	assert.NilError(t, err)
	assert.Assert(t, !expiring, "valid certs should not be flagged as expiring")
}

func TestCheckCertsExpiring_DiskExpiring(t *testing.T) {
	dir := t.TempDir()

	validPEM := newSelfSignedCertPEM(t, "valid-cert", time.Now().Add(365*24*time.Hour))
	expiringPEM := newSelfSignedCertPEM(t, "expiring-cert", time.Now().Add(60*24*time.Hour))

	for fromName, secretKey := range certMap {
		path := filepath.Join(dir, fromName)
		err := os.MkdirAll(filepath.Dir(path), 0777)
		assert.NilError(t, err)

		var data []byte
		switch {
		case filepath.Ext(secretKey) == ".crt" && secretKey == APIServerCertName:
			// Make the apiserver cert expiring soon
			data = expiringPEM
		case filepath.Ext(secretKey) == ".crt":
			data = validPEM
		default:
			data = []byte("placeholder")
		}
		err = os.WriteFile(path, data, 0666)
		assert.NilError(t, err)
	}

	expiring, err := checkCertsExpiring(dir)
	assert.NilError(t, err)
	assert.Assert(t, expiring, "expiring apiserver cert should be detected")
}

func TestCheckCertsExpiring_DiskCAOnlyExpiring(t *testing.T) {
	dir := t.TempDir()

	validPEM := newSelfSignedCertPEM(t, "valid-cert", time.Now().Add(365*24*time.Hour))
	expiringPEM := newSelfSignedCertPEM(t, "expiring-ca", time.Now().Add(60*24*time.Hour))

	for fromName, secretKey := range certMap {
		path := filepath.Join(dir, fromName)
		err := os.MkdirAll(filepath.Dir(path), 0777)
		assert.NilError(t, err)

		var data []byte
		switch {
		case secretKey == CACertName:
			// Only the CA is expiring — should NOT trigger leaf rotation
			data = expiringPEM
		case filepath.Ext(secretKey) == ".crt":
			data = validPEM
		default:
			data = []byte("placeholder")
		}
		err = os.WriteFile(path, data, 0666)
		assert.NilError(t, err)
	}

	expiring, err := checkCertsExpiring(dir)
	assert.NilError(t, err)
	assert.Assert(t, !expiring, "only CA expiring should not trigger leaf rotation")
}

func TestRolloutControlPlaneWithRetry_StatefulSet(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
	})

	err := rolloutControlPlaneWithRetry(context.Background(), client, "ns", "test", "ts-rollout")
	assert.NilError(t, err)

	sts, err := client.AppsV1().StatefulSets("ns").Get(context.Background(), "test", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, sts.Spec.Template.Annotations[certRotationAnnotation], "ts-rollout")
}

func TestRolloutControlPlaneWithRetry_DeploymentFallback(t *testing.T) {
	client := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
	})

	err := rolloutControlPlaneWithRetry(context.Background(), client, "ns", "test", "deploy-rollout")
	assert.NilError(t, err)

	deploy, err := client.AppsV1().Deployments("ns").Get(context.Background(), "test", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, deploy.Spec.Template.Annotations[certRotationAnnotation], "deploy-rollout")
}

func TestRolloutDeployedEtcdWithRetry_NotFoundIsIgnored(t *testing.T) {
	client := fake.NewSimpleClientset()

	err := rolloutDeployedEtcdWithRetry(context.Background(), client, "ns", "test", "etcd-rollout")
	assert.NilError(t, err)
}
