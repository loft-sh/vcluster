package start

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
)

func (l *LoftStarter) upgradeLoft() error {
	extraArgs := []string{}
	if l.NoTunnel {
		extraArgs = append(extraArgs, "--set-string", "env.DISABLE_LOFT_ROUTER=true")
	}
	if l.Password != "" {
		extraArgs = append(extraArgs, "--set", "admin.password="+l.Password)
	}
	if l.Host != "" {
		extraArgs = append(extraArgs, "--set", "ingress.enabled=true", "--set", "ingress.host="+l.Host)
	}
	if l.Version != "" {
		extraArgs = append(extraArgs, "--version", l.Version)
	}
	if l.Product != "" {
		extraArgs = append(extraArgs, "--set", "product="+l.Product)
	}

	if l.Email != "" {
		extraArgs = append(extraArgs, "--set", "admin.email="+l.Email)
	}

	// Do not use --reuse-values if --reset flag is provided because this should be a new install and it will cause issues with `helm template`
	if !l.Reset && l.ReuseValues {
		extraArgs = append(extraArgs, "--reuse-values")
	}

	if l.Values != "" {
		absValuesPath, err := filepath.Abs(l.Values)
		if err != nil {
			return err
		}
		extraArgs = append(extraArgs, "--values", absValuesPath)
	}

	chartName := l.ChartPath
	chartRepo := ""
	if chartName == "" {
		chartName = l.ChartName
		chartRepo = l.ChartRepo
	}

	err := clihelper.UpgradeLoft(chartName, chartRepo, l.Context, l.Namespace, extraArgs, l.Log)
	if err != nil {
		if !l.Reset {
			return errors.New(err.Error() + product.Replace(fmt.Sprintf("\n\nIf want to purge and reinstall Loft, run: %s\n", ansi.Color("loft start --reset", "green+b"))))
		}

		// Try to purge Loft and retry install
		l.Log.Info(product.Replace("Trying to delete objects blocking Loft installation"))

		manifests, err := clihelper.GetLoftManifests(chartName, chartRepo, l.Context, l.Namespace, extraArgs, l.Log)
		if err != nil {
			return err
		}

		kubectlDelete := exec.Command("kubectl", "delete", "-f", "-", "--ignore-not-found=true", "--grace-period=0", "--force")

		buffer := bytes.Buffer{}
		buffer.Write([]byte(manifests))

		kubectlDelete.Stdin = &buffer
		kubectlDelete.Stdout = os.Stdout
		kubectlDelete.Stderr = os.Stderr

		// Ignoring potential errors here
		_ = kubectlDelete.Run()

		// Retry Loft installation
		err = clihelper.UpgradeLoft(chartName, chartRepo, l.Context, l.Namespace, extraArgs, l.Log)
		if err != nil {
			return errors.New(err.Error() + product.Replace(fmt.Sprintf("\n\nLoft installation failed. Reach out to get help:\n- via Slack: %s (fastest option)\n- via Online Chat: %s\n- via Email: %s\n", ansi.Color("https://slack.loft.sh/", "green+b"), ansi.Color("https://loft.sh/", "green+b"), ansi.Color("support@loft.sh", "green+b"))))
		}
	}

	return nil
}
