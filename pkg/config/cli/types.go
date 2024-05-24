package cli

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	TelemetryDisabled bool           `json:"telemetryDisabled,omitempty"`
	Manager           ManagerConfig  `json:"manager,omitempty"`
	Platform          PlatformConfig `json:"platform,omitempty"`
}

type ManagerConfig struct {
	// Type is the current manager type that is used, either helm or platform
	Type ManagerType `json:"type,omitempty"`
}

type ManagerType string

type PlatformConfig struct {
	metav1.TypeMeta `json:",inline"`

	// host is the http endpoint of how to access loft
	// +optional
	Host string `json:"host,omitempty"`

	// LastInstallContext is the last install context
	// +optional
	LastInstallContext string `json:"lastInstallContext,omitempty"`

	// insecure specifies if the loft instance is insecure
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// access key is the access key for the given loft host
	// +optional
	AccessKey string `json:"accesskey,omitempty"`

	// virtual cluster access key is the access key for the given loft host to create virtual clusters
	// +optional
	VirtualClusterAccessKey string `json:"virtualClusterAccessKey,omitempty"`

	// map of cached certificates for "access point" mode virtual clusters
	// +optional
	VirtualClusterAccessPointCertificates map[string]VirtualClusterCertificatesEntry
}

type VirtualClusterCertificatesEntry struct {
	CertificateData string
	KeyData         string
	LastRequested   metav1.Time
	ExpirationTime  time.Time
}
