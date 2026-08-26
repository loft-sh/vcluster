package certhelper

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestIsCertExpired(t *testing.T) {
	for _, tt := range []struct {
		name       string
		expiryDate time.Time
		hasExpired bool
	}{
		{
			name:       "valid cert expiring in 1 year",
			expiryDate: time.Now().Add(365 * 24 * time.Hour),
			hasExpired: false,
		},
		{
			name:       "valid cert expiring in 91 days",
			expiryDate: time.Now().Add(91 * 24 * time.Hour),
			hasExpired: false,
		},
		{
			name:       "valid cert expiring in 90 days",
			expiryDate: time.Now().Add(90 * 24 * time.Hour),
			hasExpired: true,
		},
		{
			name:       "valid cert expiring in 30 days",
			expiryDate: time.Now().Add(30 * 24 * time.Hour),
			hasExpired: true,
		},
		{
			name:       "valid cert expiring in 1 days",
			expiryDate: time.Now().Add(1 * 24 * time.Hour),
			hasExpired: true,
		},
		{
			name:       "cert expiring now",
			expiryDate: time.Now(),
			hasExpired: true,
		},
		{
			name:       "cert expired yesterday",
			expiryDate: time.Now().Add(-1 * 24 * time.Hour),
			hasExpired: true,
		},
		{
			name:       "cert expired a month ago",
			expiryDate: time.Now().Add(-30 * 24 * time.Hour),
			hasExpired: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cert := &x509.Certificate{
				Subject: pkix.Name{
					CommonName: "example.com",
				},
				NotAfter: tt.expiryDate,
			}

			hasExpired := IsCertExpired(cert)
			assert.Equal(t, hasExpired, tt.hasExpired)
		})
	}
}
