package constants

import "os"

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	K0SDistro = "k0s"
	EKSDistro = "eks"
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

	_, err = os.Stat("/k8s-binaries")
	if err == nil {
		return K8SDistro
	}

	_, err = os.Stat("/eks-binaries")
	if err == nil {
		return EKSDistro
	}

	return "unknown"
}
