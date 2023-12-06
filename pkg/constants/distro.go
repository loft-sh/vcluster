package constants

import "os"

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	K0SDistro = "k0s"
)

func GetVClusterDistro() string {
	_, err := os.Stat("/k0s-binary/k0s")
	if err == nil {
		return K0SDistro
	}

	_, err = os.Stat("/k3s-binary/k3s")
	if err == nil {
		return K3SDistro
	}

	return K8SDistro
}
