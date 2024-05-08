package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/log"
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
################## vcluster pro token ##################
########################################################

Prints an access token to a vCluster platform instance. This
can be used as an ExecAuthenticator for kubernetes

Example:
vcluster pro token
########################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		},
	}

	accessKeyCmd.Flags().BoolVar(&cmd.DirectClusterEndpoint, "direct-cluster-endpoint", false, "When enabled prints a direct cluster endpoint token")
	accessKeyCmd.Flags().StringVar(&cmd.Project, "project", "", "The project containing the virtual cluster")
	accessKeyCmd.Flags().StringVar(&cmd.VirtualCluster, "virtual-cluster", "", "The virtual cluster")
	return accessKeyCmd
}

// Run executes the command
func (cmd *AccessKeyCmd) Run() error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	tokenFunc := getToken

	if cmd.Project != "" && cmd.VirtualCluster != "" {
		cmd.log.Debug("project and virtual cluster set, attempting fetch virtual cluster certificate data")
		tokenFunc = getCertificate
	}

	return tokenFunc(cmd, baseClient)
}

func getToken(cmd *AccessKeyCmd, baseClient client.Client) error {
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

func getCertificate(cmd *AccessKeyCmd, baseClient client.Client) error {
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
