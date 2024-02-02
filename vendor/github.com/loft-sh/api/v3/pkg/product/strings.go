package product

import (
	"github.com/loft-sh/admin-apis/pkg/features"
)

// LoginCmd returns the login command for the product
func LoginCmd() string {
	loginCmd := "loft login"

	switch Name() {
	case features.DevPodPro:
		return "devpod login"
	case features.VClusterPro:
		return "vcluster login"
	case features.Loft:
	}

	return loginCmd
}

// StartCmd returns the start command for the product
func StartCmd() string {
	loginCmd := "loft start"

	switch Name() {
	case features.DevPodPro:
		loginCmd = "devpod pro start"
	case features.VClusterPro:
		loginCmd = "vcluster pro start"
	case features.Loft:
	}

	return loginCmd
}

// Url returns the url command for the product
func Url() string {
	loginCmd := "loft-url"

	switch Name() {
	case features.DevPodPro:
		loginCmd = "devpod-pro-url"
	case features.VClusterPro:
		loginCmd = "vcluster-pro-url"
	case features.Loft:
	}

	return loginCmd
}

// ResetPassword returns the reset password command for the product
func ResetPassword() string {
	resetPassword := "loft reset password"

	switch Name() {
	case features.DevPodPro:
		return "devpod pro reset password"
	case features.VClusterPro:
		return "vcluster pro reset password"
	case features.Loft:
	}

	return resetPassword
}
