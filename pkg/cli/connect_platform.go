package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/util/serviceaccount"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type connectPlatform struct {
	*flags.GlobalFlags
	*ConnectOptions

	log log.Logger
}

func ConnectPlatform(ctx context.Context, options *ConnectOptions, globalFlags *flags.GlobalFlags, vClusterName string, command []string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, globalFlags.LoadedConfig(log))
	if err != nil {
		return err
	}

	// retrieve the vcluster
	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return fmt.Errorf("get platform vcluster %s: %w", vClusterName, err)
	}
	if vCluster == nil {
		return errors.New("empty vCluster")
	}

	// create connect platform command
	cmd := connectPlatform{
		GlobalFlags:    globalFlags,
		ConnectOptions: options,

		log: log,
	}

	err = cmd.validateProFlags()
	if err != nil {
		return err
	}

	// create management client
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// wait for vCluster to become ready
	if vCluster.VirtualCluster == nil {
		return errors.New("nil virtual cluster object")
	}

	vc, err := platform.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, log)
	if err != nil {
		return err
	}
	vCluster.VirtualCluster = vc
	if vCluster.VirtualCluster == nil {
		return errors.New("platform returned empty virtual cluster")
	}

	// retrieve vCluster kube config
	kubeConfig, err := cmd.getVClusterKubeConfig(ctx, platformClient, globalFlags, vCluster)
	if err != nil {
		return err
	}

	// check if we should execute command
	if len(command) > 0 {
		return executeCommand(*kubeConfig, command, nil, cmd.log)
	}

	return writeKubeConfig(kubeConfig, vCluster.VirtualCluster.Name, options, globalFlags, false, log)
}

func (cmd *connectPlatform) validateProFlags() error {
	if cmd.PodName != "" {
		return fmt.Errorf("cannot use --pod with a pro vCluster")
	}
	if cmd.Server != "" {
		return fmt.Errorf("cannot use --server with a pro vCluster")
	}
	if cmd.LocalPort != 0 {
		return fmt.Errorf("cannot use --local-port with a pro vCluster")
	}
	if cmd.Address != "" {
		return fmt.Errorf("cannot use --address with a pro vCluster")
	}

	return nil
}

func (cmd *connectPlatform) getVClusterKubeConfig(ctx context.Context, platformClient platform.Client, globalFlags *flags.GlobalFlags, vCluster *platform.VirtualClusterInstanceProject) (*clientcmdapi.Config, error) {
	if vCluster == nil || vCluster.Project == nil || vCluster.VirtualCluster == nil {
		return nil, errors.New("invalid vcluster VirtualClusterInstanceProject object")
	}

	contextOptions, err := platform.CreateVirtualClusterInstanceOptions(ctx, platformClient, globalFlags.Config, vCluster.Project.Name, vCluster.VirtualCluster, false)
	if err != nil {
		return nil, fmt.Errorf("prepare vCluster kube config: %w", err)
	}

	// make sure access key is set
	if contextOptions.Token == "" {
		contextOptions.Token = platformClient.Config().Platform.AccessKey
	}

	// get current context
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	}).RawConfig()
	if err != nil {
		return nil, err
	}

	// make sure kube context name is set
	if cmd.KubeConfigContextName == "" {
		// use parent context if this is a vcluster context
		kubeContext := rawConfig.CurrentContext
		_, _, parentContext := find.VClusterPlatformFromContext(kubeContext)
		if parentContext == "" {
			_, _, parentContext = find.VClusterFromContext(kubeContext)
		}
		if parentContext != "" {
			kubeContext = parentContext
		}
		cmd.KubeConfigContextName = find.VClusterPlatformContextName(vCluster.VirtualCluster.Name, vCluster.Project.Name, kubeContext)
	}

	// set insecure true?
	if cmd.Insecure {
		contextOptions.InsecureSkipTLSVerify = true
	}

	// build kube config
	kubeConfig, err := clihelper.GetProKubeConfig(contextOptions)
	if err != nil {
		return nil, err
	}

	// we want to use a service account token in the kube config
	if cmd.ServiceAccount != "" {
		// check if its enabled on the pro vcluster
		if !vCluster.VirtualCluster.Status.VirtualCluster.ForwardToken {
			return nil, fmt.Errorf("forward token is not enabled on the virtual cluster and hence you cannot authenticate with a service account token")
		}

		// init client
		vKubeClient, serviceAccount, serviceAccountNamespace, err := getServiceAccountClientAndName(*kubeConfig, cmd.ConnectOptions)
		if err != nil {
			return nil, err
		}

		// create service account token
		token, err := serviceaccount.CreateServiceAccountToken(ctx, vKubeClient, serviceAccount, serviceAccountNamespace, cmd.ServiceAccountClusterRole, int64(cmd.ServiceAccountExpiration), cmd.log)
		if err != nil {
			return nil, err
		}

		// set service account token
		for k := range kubeConfig.AuthInfos {
			kubeConfig.AuthInfos[k] = &clientcmdapi.AuthInfo{
				Token:                token,
				Extensions:           make(map[string]runtime.Object),
				ImpersonateUserExtra: make(map[string][]string),
			}
		}
	}

	return kubeConfig, nil
}
