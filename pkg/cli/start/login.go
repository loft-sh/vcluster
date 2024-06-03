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
		if err != nil {
			return fmt.Errorf("couldn't open the login page in a browser: %w", err)
		}

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
	loginPath := "%s/auth/password/login"

	loginRequest := types.PasswordLoginRequest{
		Username: defaultUser,
		Password: l.Password,
	}

	loginRequestBytes, err := json.Marshal(loginRequest)
	if err != nil {
		return err
	}

	loginRequestBuf := bytes.NewBuffer(loginRequestBytes)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	resp, err := httpClient.Post(fmt.Sprintf(loginPath, url), "application/json", loginRequestBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	accessKey := &types.AccessKey{}
	err = json.Unmarshal(body, accessKey)
	if err != nil {
		return err
	}

	// log into loft
	loginClient := platform.NewLoginClientFromConfig(l.LoadedConfig(l.Log))
	url = strings.TrimSuffix(url, "/")
	err = loginClient.LoginWithAccessKey(url, accessKey.AccessKey, true)
	if err != nil {
		return err
	}

	l.Log.WriteString(logrus.InfoLevel, "\n")
	l.Log.Donef(product.Replace("Successfully logged in via CLI into Loft instance %s"), ansi.Color(url, "white+b"))

	return nil
}

func (l *LoftStarter) loginUI(url string) error {
	queryString := fmt.Sprintf("username=%s&password=%s", defaultUser, netUrl.QueryEscape(l.Password))
	loginURL := fmt.Sprintf("%s/login#%s", url, queryString)

	err := open.Run(loginURL)
	if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w", err)
	}

	l.Log.Infof("If the browser does not open automatically, please navigate to %s", loginURL)

	return nil
}
