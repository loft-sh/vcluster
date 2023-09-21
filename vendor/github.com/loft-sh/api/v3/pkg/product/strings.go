package product

// LoginCmd returns the login command for the product
func LoginCmd() string {
	loginCmd := ""

	switch Product() {
	case Loft:
		return "loft login"
	case DevPodPro:
		return "devpod login"
	case VClusterPro:
		return "vcluster login"
	}

	return loginCmd
}

// ResetPassword returns the reset password command for the product
func ResetPassword() string {
	resetPassword := ""

	switch Product() {
	case Loft:
		return "loft reset password"
	case DevPodPro:
		return "devpod reset password"
	case VClusterPro:
		return "vcluster reset password"
	}

	return resetPassword
}
