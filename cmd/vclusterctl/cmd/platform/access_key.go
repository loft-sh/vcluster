package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

var (
	ErrNoConfigLoaded = errors.New("no config loaded")
	ErrNotLoggedIn    = errors.New("not logged in")
)

// AccessKeyCmd holds the cmd flags
type AccessKeyCmd struct {
	*flags.GlobalFlags

	Project               string
	VirtualCluster        string
	DirectClusterEndpoint bool

	log log.Logger
}

// NewAccessKeyCmd creates a new command
func NewAccessKeyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &AccessKeyCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	accessKeyCmd := &cobra.Command{
		Use:     "access-key",
		Aliases: []string{"token"},
		Short:   "Prints the access token to a vCluster platform instance",
		Long: `########################################################
############# vcluster platform token ##################
########################################################

Prints an access token to a vCluster platform instance. This
can be used as an ExecAuthenticator for kubernetes

Example:
vcluster platform token
########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	accessKeyCmd.Flags().BoolVar(&cmd.DirectClusterEndpoint, "direct-cluster-endpoint", false, "When enabled prints a direct cluster endpoint token")
	accessKeyCmd.Flags().StringVar(&cmd.Project, "project", "", "The project containing the virtual cluster")
	accessKeyCmd.Flags().StringVar(&cmd.VirtualCluster, "virtual-cluster", "", "The virtual cluster")
	return accessKeyCmd
}

// Run executes the command
func (cmd *AccessKeyCmd) Run(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	tokenFunc := getToken

	if cmd.Project != "" && cmd.VirtualCluster != "" {
		cmd.log.Debug("project and virtual cluster set, attempting fetch virtual cluster certificate data")
		tokenFunc = getCertificate
	}

	return tokenFunc(cmd, platformClient)
}

func getToken(_ *AccessKeyCmd, platformClient platform.Client) error {
	// get config
	config := platformClient.Config()
	if config == nil {
		return ErrNoConfigLoaded
	} else if config.Platform.Host == "" || config.Platform.AccessKey == "" {
		return fmt.Errorf("%w: please make sure you have run '%s [%s]'", ErrNotLoggedIn, product.LoginCmd(), product.Url())
	}

	// by default we print the access key as token
	token := config.Platform.AccessKey

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

func getCertificate(cmd *AccessKeyCmd, platformClient platform.Client) error {
	certificateData, keyData, err := platform.VirtualClusterAccessPointCertificate(platformClient, cmd.Project, cmd.VirtualCluster, false)
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
