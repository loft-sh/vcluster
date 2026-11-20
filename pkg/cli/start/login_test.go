package start

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

// stubOpenURL replaces openURL for the duration of the test and records the URL it was called with.
func stubOpenURL(t *testing.T, err error) *string {
	t.Helper()

	var opened string
	original := openURL
	openURL = func(input string) error {
		opened = input
		return err
	}
	t.Cleanup(func() { openURL = original })

	return &opened
}

func newTestStarter(password string) (*LoftStarter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	l := &LoftStarter{}
	l.Log = log.NewStreamLogger(out, out, logrus.InfoLevel)
	l.Password = password
	return l, out
}

func TestOpenBrowser(t *testing.T) {
	const (
		url        = "https://localhost:9898"
		infoLog    = "If the browser does not open automatically, please navigate to " + url
		warningLog = "Couldn't open " + url + " in a browser: open failed"
	)

	testCases := []struct {
		name        string
		openErr     error
		wantWarning bool
	}{
		{
			name:        "browser opens",
			openErr:     nil,
			wantWarning: false,
		},
		{
			name:        "browser fails to open",
			openErr:     errors.New("open failed"),
			wantWarning: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opened := stubOpenURL(t, tc.openErr)
			l, out := newTestStarter("")

			l.openBrowser(url, url)

			assert.Equal(t, *opened, url)
			assert.Assert(t, bytes.Contains(out.Bytes(), []byte(infoLog)), "log output: %q", out.String())
			assert.Equal(t, bytes.Contains(out.Bytes(), []byte(warningLog)), tc.wantWarning, "log output: %q", out.String())
		})
	}
}

// stubCLILogin replaces cliLogin with a no-op for the duration of the test and reports whether it was called.
func stubCLILogin(t *testing.T) *bool {
	t.Helper()

	called := false
	original := cliLogin
	cliLogin = func(*LoftStarter, string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { cliLogin = original })

	return &called
}

// setLoggedInHost sets the platform host in the CLI config, which is what isLoggedIn compares against.
// config.Read caches a single config for the whole process, so this edits the loaded config instead of writing a file.
func setLoggedInHost(t *testing.T, l *LoftStarter, host string) {
	t.Helper()

	l.GlobalFlags = &flags.GlobalFlags{Config: filepath.Join(t.TempDir(), "config.json")}
	cfg := l.LoadedConfig(l.Log)
	original := cfg.Platform.Host
	cfg.Platform.Host = host
	t.Cleanup(func() { cfg.Platform.Host = original })
}

func TestLoginSucceedsWhenBrowserFailsToOpen(t *testing.T) {
	const url = "https://localhost:9898"

	testCases := []struct {
		name         string
		loggedInHost string
		wantCLILogin bool
		wantOpened   string
	}{
		{
			name:         "not logged in yet",
			loggedInHost: "",
			wantCLILogin: true,
			wantOpened:   url + "/login#username=admin&password=password",
		},
		{
			name:         "already logged in",
			loggedInHost: url,
			wantCLILogin: false,
			wantOpened:   url,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opened := stubOpenURL(t, errors.New("open failed"))
			cliLoginCalled := stubCLILogin(t)
			l, out := newTestStarter("password")
			setLoggedInHost(t, l, tc.loggedInHost)

			err := l.login(url)

			assert.NilError(t, err)
			assert.Equal(t, *cliLoginCalled, tc.wantCLILogin)
			assert.Equal(t, *opened, tc.wantOpened)
			assert.Assert(t, bytes.Contains(out.Bytes(), []byte("Couldn't open "+url+" in a browser")), "log output: %q", out.String())
		})
	}
}

func TestLoginUIDoesNotLogPassword(t *testing.T) {
	const (
		url      = "https://localhost:9898"
		password = "s3cr&t pass"
	)

	testCases := []struct {
		name    string
		openErr error
	}{
		{
			name:    "browser opens",
			openErr: nil,
		},
		{
			name:    "browser fails to open",
			openErr: errors.New("open failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opened := stubOpenURL(t, tc.openErr)
			l, out := newTestStarter(password)

			l.loginUI(url)

			assert.Equal(t, *opened, url+"/login#username=admin&password=s3cr%26t+pass")
			assert.Assert(t, bytes.Contains(out.Bytes(), []byte(url)), "log output: %q", out.String())
			assert.Assert(t, !bytes.Contains(out.Bytes(), []byte("s3cr")), "password leaked into log output: %q", out.String())
		})
	}
}
