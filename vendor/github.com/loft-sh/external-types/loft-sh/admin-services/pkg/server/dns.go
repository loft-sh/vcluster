package server

type ClaimSubdomainOptions struct {
	*InstanceTokenAuth
	Subdomain         string `form:"subdomain"         json:"subdomain,omitempty"         binding:"required"`
	ReplacedSubdomain string `form:"replacedSubdomain" json:"replacedSubdomain,omitempty" binding:"required"`
	DNSTarget         string `form:"dnsTarget"         json:"dnsTarget,omitempty"         binding:"required"`
}

type ClaimSubdomainResult struct{}
