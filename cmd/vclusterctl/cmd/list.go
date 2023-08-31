package cmd

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VCluster holds information about a cluster
type VCluster struct {
	Name       string
	Namespace  string
	Created    time.Time
	AgeSeconds int
	Context    string
	Status     string
	Connected  bool
}

// ListCmd holds the login cmd flags
type ListCmd struct {
	*flags.GlobalFlags

	log    log.Logger
	output string
}

// NewListCmd creates a new command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all virtual clusters",
		Long: `
#######################################################
#################### vcluster list ####################
#######################################################
Lists all virtual clusters

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, args []string) error {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}
	currentContext := rawConfig.CurrentContext

	if cmd.Context == "" {
		cmd.Context = currentContext
	}

	namespace := metav1.NamespaceAll
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	vClusters, err := find.ListVClusters(cobraCmd.Context(), cmd.Context, "", namespace)
	if err != nil {
		return err
	}

	if cmd.output == "json" {
		var output []VCluster
		for _, vcluster := range vClusters {
			vclusterOutput := VCluster{
				Name:       vcluster.Name,
				Namespace:  vcluster.Namespace,
				Created:    vcluster.Created.Time,
				AgeSeconds: int(time.Since(vcluster.Created.Time).Round(time.Second).Seconds()),
				Context:    vcluster.Context,
				Status:     string(vcluster.Status),
			}
			vclusterOutput.Connected = currentContext == find.VClusterContextName(
				vcluster.Name,
				vcluster.Namespace,
				vcluster.Context,
			)
			output = append(output, vclusterOutput)
		}
		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return errors.Wrap(err, "json marshal vclusters")
		}
		cmd.log.WriteString(string(bytes) + "\n")
	} else {
		header := []string{"NAME", "NAMESPACE", "STATUS", "CONNECTED", "CREATED", "AGE", "CONTEXT"}
		values := [][]string{}
		for _, vcluster := range vClusters {
			connected := ""
			if currentContext == find.VClusterContextName(vcluster.Name, vcluster.Namespace, vcluster.Context) {
				connected = "True"
			}

			values = append(values, []string{
				vcluster.Name,
				vcluster.Namespace,
				string(vcluster.Status),
				connected,
				vcluster.Created.String(),
				time.Since(vcluster.Created.Time).Round(1 * time.Second).String(),
				vcluster.Context,
			})
		}

		log.PrintTable(cmd.log, header, values)
		if strings.HasPrefix(cmd.Context, "vcluster_") {
			cmd.log.Infof("Run `vcluster disconnect` to switch back to the parent context")
		}
	}

	return nil
}
