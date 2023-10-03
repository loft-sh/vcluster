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

// StartCmd returns the start command for the product
func StartCmd() string {
	loginCmd := ""

	switch Product() {
	case Loft:
		return "loft start"
	case DevPodPro:
		return "devpod pro start"
	case VClusterPro:
		return "vcluster pro start"
	}

	return loginCmd
}

// Url returns the url command for the product
func Url() string {
	loginCmd := ""

	switch Product() {
	case Loft:
		return "loft-url"
	case DevPodPro:
		return "devpod-pro-url"
	case VClusterPro:
		return "vcluster-pro-url"
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
		return "devpod pro reset password"
	case VClusterPro:
		return "vcluster pro reset password"
	}

	return resetPassword
}

// Name returns the name of the product
func Name() string {
	name := ""

	switch Product() {
	case Loft:
		return "Loft"
	case DevPodPro:
		return "DevPod Pro"
	case VClusterPro:
		return "vCluster.Pro"
	}

	return name
}
