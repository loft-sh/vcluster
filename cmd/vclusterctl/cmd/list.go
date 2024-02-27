package cmd

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VCluster holds information about a cluster
type VCluster struct {
	Created    time.Time
	Name       string
	Namespace  string
	Context    string
	Cluster    string
	Version    string
	Status     string
	AgeSeconds int
	Connected  bool
	Pro        bool
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
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, _ []string) error {
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

	proClient, err := procli.CreateProClient()
	if err != nil {
		cmd.log.Debugf("Error creating pro client: %v", err)
	}

	vClusters, proVClusters, err := find.ListVClusters(cobraCmd.Context(), proClient, cmd.Context, "", namespace, "", cmd.log.ErrorStreamOnly())
	if err != nil {
		return err
	}

	var output []VCluster
	output = append(output, ossToVClusters(vClusters, currentContext)...)
	output = append(output, proToVClusters(proVClusters, currentContext)...)

	if cmd.output == "json" {
		bytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return errors.Wrap(err, "json marshal vclusters")
		}
		cmd.log.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	} else {
		header := []string{"NAME", "CLUSTER", "NAMESPACE", "STATUS", "VERSION", "CONNECTED", "CREATED", "AGE", "DISTRO"}
		values := toValues(output)
		table.PrintTable(cmd.log, header, values)
		if strings.HasPrefix(cmd.Context, "vcluster_") || strings.HasPrefix(cmd.Context, "vcluster-pro_") {
			cmd.log.Infof("Run `vcluster disconnect` to switch back to the parent context")
		}
	}

	return nil
}

func ossToVClusters(vClusters []find.VCluster, currentContext string) []VCluster {
	var output []VCluster
	for _, vCluster := range vClusters {
		vClusterOutput := VCluster{
			Name:       vCluster.Name,
			Namespace:  vCluster.Namespace,
			Created:    vCluster.Created.Time,
			Version:    vCluster.Version,
			AgeSeconds: int(time.Since(vCluster.Created.Time).Round(time.Second).Seconds()),
			Cluster:    vCluster.Context,
			Status:     string(vCluster.Status),
			Pro:        false,
		}
		vClusterOutput.Connected = currentContext == find.VClusterContextName(
			vCluster.Name,
			vCluster.Namespace,
			vCluster.Context,
		)
		output = append(output, vClusterOutput)
	}
	return output
}

func proToVClusters(vClusters []procli.VirtualClusterInstanceProject, currentContext string) []VCluster {
	var output []VCluster
	for _, vCluster := range vClusters {
		status := string(vCluster.VirtualCluster.Status.Phase)
		if vCluster.VirtualCluster.DeletionTimestamp != nil {
			status = "Terminating"
		} else if status == "" {
			status = "Pending"
		}

		connected := strings.HasPrefix(currentContext, "vcluster-pro_"+vCluster.VirtualCluster.Name+"_"+vCluster.Project.Name)
		vClusterOutput := VCluster{
			Name:       vCluster.VirtualCluster.Spec.ClusterRef.VirtualCluster,
			Namespace:  vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
			Cluster:    vCluster.VirtualCluster.Spec.ClusterRef.Cluster,
			Connected:  connected,
			Created:    vCluster.VirtualCluster.CreationTimestamp.Time,
			AgeSeconds: int(time.Since(vCluster.VirtualCluster.CreationTimestamp.Time).Round(time.Second).Seconds()),
			Status:     status,
			Pro:        true,
			Version:    vCluster.VirtualCluster.Status.VirtualCluster.HelmRelease.Chart.Version,
		}
		output = append(output, vClusterOutput)
	}
	return output
}

func toValues(vClusters []VCluster) [][]string {
	var values [][]string
	for _, vCluster := range vClusters {
		isConnected := ""
		if vCluster.Connected {
			isConnected = "True"
		}

		distro := "OSS"
		if vCluster.Pro {
			distro = "Pro"
		}

		values = append(values, []string{
			vCluster.Name,
			vCluster.Cluster,
			vCluster.Namespace,
			vCluster.Status,
			vCluster.Version,
			isConnected,
			vCluster.Created.String(),
			time.Since(vCluster.Created).Round(1 * time.Second).String(),
			distro,
		})
	}
	return values
}
