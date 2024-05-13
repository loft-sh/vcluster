package printhelper

import (
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

const passwordChangedHint = "(has been changed)"

func PrintDNSConfiguration(host string, log log.Logger) {
	log.WriteString(logrus.InfoLevel, `

###################################     DNS CONFIGURATION REQUIRED     ##################################

Create a DNS A-record for `+host+` with the EXTERNAL-IP of your nginx-ingress controller.
To find this EXTERNAL-IP, run the following command and look at the output:

> kubectl get services -n ingress-nginx
                                                     |---------------|
NAME                       TYPE           CLUSTER-IP | EXTERNAL-IP   |  PORT(S)                      AGE
ingress-nginx-controller   LoadBalancer   10.0.0.244 | XX.XXX.XXX.XX |  80:30984/TCP,443:31758/TCP   19m
                                                     |^^^^^^^^^^^^^^^|

EXTERNAL-IP may be 'pending' for a while until your cloud provider has created a new load balancer.

#########################################################################################################

The command will wait until loft is reachable under the host. You can also abort and use port-forwarding instead
by running 'loft start' again.

`)
}

func PrintSuccessMessageLocalInstall(password, url string, log log.Logger) {
	if password == "" {
		password = passwordChangedHint
	}

	log.WriteString(logrus.InfoLevel, fmt.Sprintf(product.Replace(`

##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color(product.ResetPassword(), "green+b")+`

Login via UI:  %s
Login via CLI: %s

!!! You must accept the untrusted certificate in your browser !!!

#################################################################

Loft was successfully installed and port-forwarding has been started.
If you stop this command, run 'loft start' again to restart port-forwarding.

Thanks for using Loft!
`), ansi.Color(url, "green+b"), ansi.Color(product.LoginCmd()+" --insecure "+url, "green+b")))
}

func PrintSuccessMessageRemoteInstall(host, password string, log log.Logger) {
	url := "https://" + host

	if password == "" {
		password = passwordChangedHint
	}

	log.WriteString(logrus.InfoLevel, fmt.Sprintf(product.Replace(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color(product.ResetPassword(), "green+b")+`

Login via UI:  %s
Login via CLI: %s

!!! You must accept the untrusted certificate in your browser !!!

Follow this guide to add a valid certificate: %s

#################################################################

Loft was successfully installed and can now be reached at: %s

Thanks for using Loft!
`),
		ansi.Color(url, "green+b"),
		ansi.Color(product.LoginCmd()+" --insecure "+url, "green+b"),
		"https://loft.sh/docs/administration/ssl",
		url,
	))
}

func PrintSuccessMessageLoftRouterInstall(host, password string, log log.Logger) {
	url := "https://" + host

	if password == "" {
		password = passwordChangedHint
	}

	log.WriteString(logrus.InfoLevel, fmt.Sprintf(product.Replace(`


##########################   LOGIN   ############################

Username: `+ansi.Color("admin", "green+b")+`
Password: `+ansi.Color(password, "green+b")+`  # Change via UI or via: `+ansi.Color(product.ResetPassword(), "green+b")+`

Login via UI:  %s
Login via CLI: %s

#################################################################

Loft was successfully installed and can now be reached at: %s

Thanks for using Loft!
`),
		ansi.Color(url, "green+b"),
		ansi.Color(product.LoginCmd()+" "+url, "green+b"),
		url,
	))
}
