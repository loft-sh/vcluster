package cli

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPinKubernetesVersion(t *testing.T) {
	k8sOf := func(values map[string]interface{}) map[string]interface{} {
		return values["controlPlane"].(map[string]interface{})["distro"].(map[string]interface{})["k8s"].(map[string]interface{})
	}

	t.Run("empty values get the version", func(t *testing.T) {
		values := map[string]interface{}{}
		pinKubernetesVersion(values, "v1.36.5")
		assert.Equal(t, k8sOf(values)["version"], "v1.36.5")
	})

	t.Run("sibling keys are kept", func(t *testing.T) {
		values := map[string]interface{}{
			"controlPlane": map[string]interface{}{
				"standalone": map[string]interface{}{"enabled": true},
				"distro": map[string]interface{}{
					"k8s": map[string]interface{}{
						"image": map[string]interface{}{"registry": "my-registry.io"},
					},
				},
			},
		}
		pinKubernetesVersion(values, "v1.36.5")
		assert.DeepEqual(t, values["controlPlane"].(map[string]interface{})["standalone"], map[string]interface{}{"enabled": true})
		assert.Equal(t, k8sOf(values)["version"], "v1.36.5")
		assert.DeepEqual(t, k8sOf(values)["image"], map[string]interface{}{"registry": "my-registry.io"})
	})

	t.Run("user version wins", func(t *testing.T) {
		values := map[string]interface{}{
			"controlPlane": map[string]interface{}{
				"distro": map[string]interface{}{
					"k8s": map[string]interface{}{"version": "v1.35.9"},
				},
			},
		}
		pinKubernetesVersion(values, "v1.36.5")
		assert.Equal(t, k8sOf(values)["version"], "v1.35.9")
	})

	t.Run("user image tag wins", func(t *testing.T) {
		values := map[string]interface{}{
			"controlPlane": map[string]interface{}{
				"distro": map[string]interface{}{
					"k8s": map[string]interface{}{
						"image": map[string]interface{}{"tag": "v1.34.12"},
					},
				},
			},
		}
		pinKubernetesVersion(values, "v1.36.5")
		_, hasVersion := k8sOf(values)["version"]
		assert.Assert(t, !hasVersion)
	})

	t.Run("empty version is a no-op", func(t *testing.T) {
		values := map[string]interface{}{}
		pinKubernetesVersion(values, "")
		assert.Equal(t, len(values), 0)
	})
}
