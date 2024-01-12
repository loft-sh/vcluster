package get

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterTokenCmd  holds the lags
type ClusterTokenCmd struct {
	*flags.GlobalFlags

	log    log.Logger
	Output string
}

func NewClusterAccessKeyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterTokenCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("get cluster-access-key", `
Returns the Network Peer Cluster Token

Example:
loft get cluster-access-key [CLUSTER_NAME]
########################################################
	`)

	useLine, validator := util.NamedPositionalArgsValidator(true, true, "CLUSTER_NAME")

	c := &cobra.Command{
		Use:   "cluster-access-key" + useLine,
		Short: "Retrieve the network peer cluster access key",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVarP(&cmd.Output, "output", "o", "text", "Output format. One of: (json, yaml, text)")

	return c
}

func (cmd *ClusterTokenCmd) Run(ctx context.Context, args []string) error {
	clusterName := args[0]

	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return fmt.Errorf("new client from path: %w", err)
	}

	err = client.VerifyVersion(baseClient)
	if err != nil {
		return fmt.Errorf("verify loft version: %w", err)
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return fmt.Errorf("create management client: %w", err)
	}

	accessKey, err := managementClient.Loft().ManagementV1().Clusters().GetAccessKey(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get cluster access key: %w", err)
	}

	accessKey.ObjectMeta = metav1.ObjectMeta{
		CreationTimestamp: metav1.NewTime(time.Now()),
	}

	switch cmd.Output {
	case OutputJSON:
		serialized, err := json.Marshal(accessKey)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}

		cmd.log.WriteString(logrus.InfoLevel, string(serialized))
	case OutputYAML:
		serialized, err := yaml.Marshal(accessKey)
		if err != nil {
			return fmt.Errorf("yaml marshal: %w", err)
		}

		cmd.log.WriteString(logrus.InfoLevel, string(serialized))
	default:
		cmd.log.Infof("Loft host: %v", accessKey.LoftHost)
		cmd.log.Infof("Access Key: %v", accessKey.AccessKey)

		if accessKey.CaCert != "" {
			cmd.log.Infof("CA Cert: %v", accessKey.CaCert)
		}

		if accessKey.Insecure {
			cmd.log.Infof("Insecure: %v", accessKey.Insecure)
		}
	}

	return nil
}
