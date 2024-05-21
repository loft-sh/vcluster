package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

var (
	ErrNoConfigLoaded = errors.New("no config loaded")
	ErrNotLoggedIn    = errors.New("not logged in")
)

// TokenCmd holds the cmd flags
type TokenCmd struct {
	*flags.GlobalFlags
	log            log.Logger
	Project        string
	VirtualCluster string
	// Deprecated please use access keys instead
	DirectClusterEndpoint bool
}

// NewTokenCmd creates a new command
func NewTokenCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TokenCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("token", `
Prints an access token to a loft instance. This can
be used as an ExecAuthenticator for kubernetes

Example:
loft token
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
#################### devspace token ####################
########################################################
Prints an access token to a loft instance. This can
be used as an ExecAuthenticator for kubernetes

Example:
devspace token
########################################################
	`
	}

	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: product.Replace("Token prints the access token to a loft instance"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	tokenCmd.Flags().BoolVar(&cmd.DirectClusterEndpoint, "direct-cluster-endpoint", false, "When enabled prints a direct cluster endpoint token")
	tokenCmd.Flags().StringVar(&cmd.Project, "project", "", "The project containing the virtual cluster")
	tokenCmd.Flags().StringVar(&cmd.VirtualCluster, "virtual-cluster", "", "The virtual cluster")
	return tokenCmd
}

// Run executes the command
func (cmd *TokenCmd) Run(ctx context.Context) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}
	self, err := baseClient.GetSelf(ctx)
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)

	tokenFunc := getToken

	if cmd.Project != "" && cmd.VirtualCluster != "" {
		cmd.log.Debug("project and virtual cluster set, attempting fetch virtual cluster certificate data")
		tokenFunc = getCertificate
	}

	return tokenFunc(cmd, baseClient)
}

func getToken(cmd *TokenCmd, baseClient client.Client) error {
	// get config
	config := baseClient.Config()
	if config == nil {
		return ErrNoConfigLoaded
	} else if config.Host == "" || config.AccessKey == "" {
		return fmt.Errorf("%w: please make sure you have run '%s [%s]'", ErrNotLoggedIn, product.LoginCmd(), product.Url())
	}

	// by default we print the access key as token
	token := config.AccessKey

	// check if we should print a cluster gateway token instead
	if cmd.DirectClusterEndpoint {
		var err error
		token, err = baseClient.DirectClusterEndpointToken(false)
		if err != nil {
			return err
		}
	}

	return printToken(token)
}

func printToken(token string) error {
	// Print token to stdout
	response := &v1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		Status: &v1beta1.ExecCredentialStatus{
			Token: token,
		},
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	_, err = os.Stdout.Write(bytes)
	return err
}

func getCertificate(cmd *TokenCmd, baseClient client.Client) error {
	certificateData, keyData, err := baseClient.VirtualClusterAccessPointCertificate(cmd.Project, cmd.VirtualCluster, false)
	if err != nil {
		return err
	}

	return printCertificate(certificateData, keyData)
}

func printCertificate(certificateData, keyData string) error {
	// Print certificate-based exec credential to stdout
	response := &v1beta1.ExecCredential{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExecCredential",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		Status: &v1beta1.ExecCredentialStatus{
			ClientCertificateData: certificateData,
			ClientKeyData:         keyData,
		},
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	_, err = os.Stdout.Write(bytes)
	return err
}
