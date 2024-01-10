package constants

import "os"

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	K0SDistro = "k0s"
	EKSDistro = "eks"
	Unknown   = "unknown"
)

func GetVClusterDistro() string {
	distro := os.Getenv("VCLUSTER_DISTRO")
	switch distro {
	case K3SDistro, K8SDistro, K0SDistro, EKSDistro:
		return distro
	default:
		return Unknown
	}
}
