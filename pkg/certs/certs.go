package certs

import "time"

type Info struct {
	Filename   string    `json:"filename,omitempty"`
	Subject    string    `json:"subject,omitempty"`
	Issuer     string    `json:"issuer,omitempty"`
	ExpiryTime time.Time `json:"expiryTime"`
	Status     string    `json:"status,omitempty"` // "OK", "EXPIRED"
}
