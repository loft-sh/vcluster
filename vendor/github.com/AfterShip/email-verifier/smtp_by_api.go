package emailverifier

const (
	YAHOO = "yahoo"
)

type smtpAPIVerifier interface {
	// isSupported the specific host supports the check by api.
	isSupported(host string) bool
	// check must be called before isSupported == true
	check(domain, username string) (*SMTP, error)
}
