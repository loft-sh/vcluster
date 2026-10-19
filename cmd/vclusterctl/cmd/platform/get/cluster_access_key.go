package get

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterTokenCmd  holds the lags
type ClusterTokenCmd struct {
	*flags.GlobalFlags

	log    log.Logger
	Output string
}

func newClusterAccessKeyCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterTokenCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("get cluster-access-key", `
Returns the Network Peer Cluster Token

Example:
vcluster platform get cluster-access-key [CLUSTER_NAME]
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

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return fmt.Errorf("new client from path: %w", err)
	}

	managementClient, err := platformClient.Management()
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
		cmd.log.Infof("vCluster platform host: %v", accessKey.LoftHost)
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
