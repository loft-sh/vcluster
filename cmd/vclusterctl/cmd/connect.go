package cmd

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// ConnectCmd holds the login cmd flags
type ConnectCmd struct {
	*flags.GlobalFlags

	KubeConfig    string
	Namespace     string
	UpdateCurrent bool
	Print         bool
	LocalPort     int

	log log.Logger
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to a virtual cluster",
		Long: `
#######################################################
################## vcluster connect ###################
#######################################################
Connect to a virtual cluster

Example:
vcluster connect test --namespace test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.KubeConfig, "kube-config", "./kubeconfig.yaml", "Writes the created kube config to this file")
	cobraCmd.Flags().BoolVar(&cmd.UpdateCurrent, "update-current", false, "If true updates the current kube config")
	cobraCmd.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	cobraCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "The namespace the vcluster is in")
	cobraCmd.Flags().IntVar(&cmd.LocalPort, "local-port", 8443, "The local port to forward the virtual cluster to")
	return cobraCmd
}

// Run executes the functionality
func (cmd *ConnectCmd) Run(cobraCmd *cobra.Command, args []string) error {
	kubeConfigLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	// set the namespace correctly
	var err error
	if cmd.Namespace == "" {
		cmd.Namespace, _, err = kubeConfigLoader.Namespace()
		if err != nil {
			return err
		}
	}

	podName := args[0] + "-0"

	// get the kube config from the container
	var out []byte
	printedWaiting := false
	err = wait.PollImmediate(time.Second*2, time.Minute*5, func() (done bool, err error) {
		out, err = exec.Command("kubectl", "exec", "--namespace", cmd.Namespace, "-c", "syncer", podName, "--", "cat", "/root/.kube/config").CombinedOutput()
		if err != nil {
			if !printedWaiting {
				cmd.log.Infof("Waiting for vCluster to come up...")
				printedWaiting = true
			}

			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "wait for vCluster")
	}

	kubeConfig, err := clientcmd.Load(out)
	if err != nil {
		return errors.Wrap(err, "parse kube config")
	}

	// find out port we should listen to locally
	if len(kubeConfig.Clusters) != 1 {
		return fmt.Errorf("unexpected kube config")
	}

	port := ""
	for k := range kubeConfig.Clusters {
		splitted := strings.Split(kubeConfig.Clusters[k].Server, ":")
		if len(splitted) != 3 {
			return fmt.Errorf("unexpected server in kubeconfig: %s", kubeConfig.Clusters[k].Server)
		}

		port = splitted[2]
		splitted[2] = strconv.Itoa(cmd.LocalPort)
		kubeConfig.Clusters[k].Server = strings.Join(splitted, ":")
	}

	out, err = clientcmd.Write(*kubeConfig)
	if err != nil {
		return err
	}

	// write kube config to file
	if cmd.UpdateCurrent {
		var clusterConfig *api.Cluster
		for _, c := range kubeConfig.Clusters {
			clusterConfig = c
		}

		var authConfig *api.AuthInfo
		for _, a := range kubeConfig.AuthInfos {
			authConfig = a
		}

		contextName := "vcluster_" + cmd.Namespace + "_" + args[0]
		err = updateKubeConfig(contextName, clusterConfig, authConfig, false)
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully created kube context %s. You can access the vcluster with `kubectl get namespaces --context %s`", contextName, contextName)
	} else if cmd.Print {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return err
		}
	} else {
		err = ioutil.WriteFile(cmd.KubeConfig, out, 0666)
		if err != nil {
			return errors.Wrap(err, "write kube config")
		}

		cmd.log.Donef("Virtual cluster kube config written to: %s. You can access the cluster via `kubectl --kubeconfig %s get namespaces`", cmd.KubeConfig, cmd.KubeConfig)
	}

	forwardPorts := strconv.Itoa(cmd.LocalPort) + ":" + port

	command := []string{"kubectl", "port-forward", "--namespace", cmd.Namespace, podName, forwardPorts}
	cmd.log.Infof("Starting port forwarding: %s", strings.Join(command, " "))
	portforwardCmd := exec.Command(command[0], command[1:]...)
	if !cmd.Print {
		portforwardCmd.Stdout = os.Stdout
	} else {
		portforwardCmd.Stdout = ioutil.Discard
	}

	portforwardCmd.Stderr = os.Stderr
	return portforwardCmd.Run()
}

func updateKubeConfig(contextName string, cluster *api.Cluster, authInfo *api.AuthInfo, setActive bool) error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	config.Clusters[contextName] = cluster
	config.AuthInfos[contextName] = authInfo

	// Update kube context
	context := api.NewContext()
	context.Cluster = contextName
	context.AuthInfo = contextName

	config.Contexts[contextName] = context
	if setActive {
		config.CurrentContext = contextName
	}

	// Save the config
	return clientcmd.ModifyConfig(clientcmd.NewDefaultClientConfigLoadingRules(), config, false)
}
