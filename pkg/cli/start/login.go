package start

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"os/exec"
	"strings"

	types "github.com/loft-sh/api/v4/pkg/auth"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

const defaultUser = "admin"

func (l *LoftStarter) login(url string) error {
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// check if we are already logged in
	if l.isLoggedIn(url) {
		// still open the UI
		err := open.Run(url)
		if errors.Is(err, exec.ErrNotFound) {
			l.Log.Warnf("Couldn't open the login page in a browser. No browser found: %v", err)
		} else if err != nil {
			return fmt.Errorf("couldn't open the login page in a browser: %w", err)
		}

		l.Log.Infof("If the browser does not open automatically, please navigate to %s", url)

		return nil
	}

	// log into the CLI
	err := l.loginViaCLI(url)
	if err != nil {
		return err
	}

	// log into the UI
	err = l.loginUI(url)
	if err != nil {
		return err
	}

	return nil
}

func (l *LoftStarter) loginViaCLI(url string) error {
	loginRequestBytes, err := json.Marshal(types.PasswordLoginRequest{
		Username: defaultUser,
		Password: l.Password,
	})
	if err != nil {
		return err
	}

	// Try a secure connection first. During bootstrap the platform typically
	// serves a self-signed certificate, so fall back to insecure if the secure
	// attempt fails with a TLS error.
	accessKey, insecure, err := l.passwordLogin(url, loginRequestBytes, false)
	if err != nil {
		if !isTLSError(err) {
			return err
		}
		l.Log.Infof("TLS verification failed, retrying without verification (self-signed certificate expected during bootstrap)")
		accessKey, insecure, err = l.passwordLogin(url, loginRequestBytes, true)
		if err != nil {
			return err
		}
	}

	// log into loft
	config := l.LoadedConfig(l.Log)
	loginClient := platform.NewLoginClientFromConfig(config)
	url = strings.TrimSuffix(url, "/")
	err = loginClient.LoginWithAccessKey(url, accessKey, insecure)
	if err != nil {
		return err
	}

	l.Log.WriteString(logrus.InfoLevel, "\n")
	l.Log.Donef(product.Replace("Successfully logged in via CLI into Loft instance %s"), ansi.Color(url, "white+b"))

	return nil
}

// passwordLogin posts the admin credentials to the platform's password login
// endpoint and returns the access key. The insecure parameter controls whether
// TLS certificate verification is skipped.
func (l *LoftStarter) passwordLogin(url string, loginRequestBytes []byte, insecure bool) (string, bool, error) {
	httpClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}}

	accessKey := &types.AccessKey{}
	for i := 0; i < 3; i++ {
		resp, err := httpClient.Post(url+"/auth/password/login", "application/json", bytes.NewBuffer(loginRequestBytes))
		if err != nil {
			return "", insecure, err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return "", insecure, err
		}
		_ = resp.Body.Close()

		err = json.Unmarshal(body, accessKey)
		if err != nil {
			return "", insecure, err
		}
		if accessKey.AccessKey == "" {
			continue
		}
		break
	}
	if accessKey.AccessKey == "" {
		return "", insecure, fmt.Errorf("couldn't retrieve access key from platform to login")
	}

	return accessKey.AccessKey, insecure, nil
}

// isTLSError returns true if the error is caused by a TLS certificate
// verification failure.
func isTLSError(err error) bool {
	var urlErr *netUrl.Error
	if !errors.As(err, &urlErr) {
		return false
	}
	var certErr *tls.CertificateVerificationError
	if errors.As(urlErr.Err, &certErr) {
		return true
	}
	var unknownAuthErr x509.UnknownAuthorityError
	return errors.As(urlErr.Err, &unknownAuthErr)
}

func (l *LoftStarter) loginUI(url string) error {
	queryString := fmt.Sprintf("username=%s&password=%s", defaultUser, netUrl.QueryEscape(l.Password))
	loginURL := fmt.Sprintf("%s/login#%s", url, queryString)

	err := open.Run(loginURL)
	if errors.Is(err, exec.ErrNotFound) {
		l.Log.Warnf("Couldn't open the login page in a browser. No browser found: %v", err)
	} else if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w", err)
	}

	l.Log.Infof("If the browser does not open automatically, please navigate to %s", url)

	return nil
}
