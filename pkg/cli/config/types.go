package config

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CLI struct {
	Driver            Driver   `json:"driver,omitempty"`
	PreviousContext   string   `json:"previousContext,omitempty"`
	path              string   `json:"-"`
	Platform          Platform `json:"platform,omitempty"`
	TelemetryDisabled bool     `json:"telemetryDisabled,omitempty"`
}

type Driver struct {
	// Type is the current driver type that is used, either helm or platform
	Type DriverType `json:"type,omitempty"`
}

type DriverType string

type Platform struct {
	metav1.TypeMeta `json:",inline"`

	// VirtualClusterAccessPointCertificates is a map of cached certificates for "access point" mode virtual clusters
	VirtualClusterAccessPointCertificates map[string]VirtualClusterCertificatesEntry `json:"virtualClusterAccessPointCertificates,omitempty"`
	// Host is the https endpoint of how to access loft
	Host string `json:"host,omitempty"`
	// LastInstallContext is the last install context
	LastInstallContext string `json:"lastInstallContext,omitempty"`
	// AccessKey is the access key for the given loft host
	AccessKey string `json:"accesskey,omitempty"`
	// VirtualClusterAccessKey is the access key for the given loft host to create virtual clusters
	VirtualClusterAccessKey string `json:"virtualClusterAccessKey,omitempty"`
	// Insecure specifies if the loft instance is insecure
	Insecure bool `json:"insecure,omitempty"`
	// CertificateAuthorityData is passed as certificate-authority-data to the platform config
	CertificateAuthorityData []byte `json:"certificateAuthorityData,omitempty"`
}

type VirtualClusterCertificatesEntry struct {
	LastRequested   metav1.Time `json:"lastRequested,omitempty"`
	ExpirationTime  time.Time   `json:"expirationTime,omitempty"`
	CertificateData string      `json:"certificateData,omitempty"`
	KeyData         string      `json:"keyData,omitempty"`
}
