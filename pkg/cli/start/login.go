package start

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netUrl "net/url"
	"strings"

	types "github.com/loft-sh/api/v5/pkg/auth"
	"github.com/loft-sh/api/v5/pkg/product"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
)

const defaultUser = "admin"
const retryCount = 3

// Variables for testing purposes. They are set to the default implementations, but can be overridden in tests.
var (
	// openURL opens a URL in the default browser.
	openURL = open.Run
	// cliLogin logs the CLI into the platform.
	cliLogin = (*LoftStarter).loginViaCLI
)

func (l *LoftStarter) login(url string) error {
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// check if we are already logged in
	if l.isLoggedIn(url) {
		// still open the browser
		l.openBrowser(url, url)
		return nil
	}

	// log into the CLI
	if err := cliLogin(l, url); err != nil {
		return err
	}

	// log into the UI
	l.loginUI(url)

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

	config := l.LoadedConfig(l.Log)
	httpClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.Platform.Insecure},
	}}

	accessKey := &types.AccessKey{}
	for range retryCount {
		resp, err := httpClient.Post(url+"/auth/password/login", "application/json", bytes.NewBuffer(loginRequestBytes))
		if err != nil {
			return err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()

		err = json.Unmarshal(body, accessKey)
		if err != nil {
			return err
		}
		if accessKey.AccessKey == "" {
			continue
		}
		break
	}
	if accessKey.AccessKey == "" {
		return fmt.Errorf("couldn't retrieve access key from platform to login")
	}

	// log into loft
	loginClient := platform.NewLoginClientFromConfig(config)
	url = strings.TrimSuffix(url, "/")
	err = loginClient.LoginWithAccessKey(url, accessKey.AccessKey, config.Platform.Insecure)
	if err != nil {
		return err
	}

	l.Log.WriteString(logrus.InfoLevel, "\n")
	l.Log.Donef(product.Replace("Successfully logged in via CLI into Loft instance %s"), ansi.Color(url, "white+b"))

	return nil
}

func (l *LoftStarter) loginUI(url string) {
	queryString := fmt.Sprintf("username=%s&password=%s", defaultUser, netUrl.QueryEscape(l.Password))
	loginURL := fmt.Sprintf("%s/login#%s", url, queryString)

	l.openBrowser(loginURL, url)
}

// openBrowser tries to open target in the default browser. Failures are logged, not returned.
// displayURL is what gets logged, and it differs from target when target holds secrets such as a password.
func (l *LoftStarter) openBrowser(target, displayURL string) {
	if err := openURL(target); err != nil {
		l.Log.Warnf("Couldn't open %s in a browser: %v", displayURL, err)
	}
	l.Log.Infof("If the browser does not open automatically, please navigate to %s", displayURL)
}
