package server

// +genclient
// +genclient:nonNamespaced

// +k8s:openapi-gen=true
type InstanceTokenAuth struct {
	// Token is the jwt token identifying the loft instance.
	Token string `json:"token"       validate:"required"`
	// Certificate is the signing certificate for the token.
	Certificate string `json:"certificate" validate:"required" form:"certificate"`
}
