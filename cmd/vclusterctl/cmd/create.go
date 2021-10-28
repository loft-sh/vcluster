package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/pkg/upgrade"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var VersionMap = map[string]string{
	"1.22": "rancher/k3s:v1.22.2-k3s2",
	"1.21": "rancher/k3s:v1.21.5-k3s2",
	"1.20": "rancher/k3s:v1.20.11-k3s2",
	"1.19": "rancher/k3s:v1.19.13-k3s1",
	"1.18": "rancher/k3s:v1.18.20-k3s1",
	"1.17": "rancher/k3s:v1.17.17-k3s1",
	"1.16": "rancher/k3s:v1.16.15-k3s1",
}

const noDeployValues = `  baseArgs:
    - server
    - --write-kubeconfig=/k3s-config/kube-config.yaml
    - --data-dir=/data
    - --no-deploy=traefik,servicelb,metrics-server,local-storage
    - --disable-network-policy
    - --disable-agent
    - --disable-scheduler
    - --disable-cloud-controller
    - --flannel-backend=none
    - --kube-controller-manager-arg=controllers=*,-nodeipam,-nodelifecycle,-persistentvolume-binder,-attachdetach,-persistentvolume-expander,-cloud-node-lifecycle`

var baseArgsMap = map[string]string{
	"1.17": noDeployValues,
	"1.16": noDeployValues,
}

var errorMessageFind = "provided IP is not in the valid range. The range of valid IPs is "
var replaceRegEx = regexp.MustCompile("[^0-9]+")

// CreateCmd holds the login cmd flags
type CreateCmd struct {
	*flags.GlobalFlags

	ChartVersion  string
	ChartName     string
	ChartRepo     string
	ReleaseValues string
	K3SImage      string
	ExtraValues   []string

	CreateNamespace    bool
	DisableIngressSync bool
	CreateClusterRole  bool
	Expose             bool
	Connect            bool
	Upgrade            bool

	log log.Logger
}

// NewCreateCmd creates a new command
func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new virtual cluster",
		Long: `
#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
	`,
		Args: cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.ChartVersion, "chart-version", upgrade.GetVersion(), "The virtual cluster chart version to use (e.g. v0.4.0)")
	cobraCmd.Flags().StringVar(&cmd.ChartName, "chart-name", "vcluster", "The virtual cluster chart name to use")
	cobraCmd.Flags().StringVar(&cmd.ChartRepo, "chart-repo", "https://charts.loft.sh", "The virtual cluster chart repo to use")
	cobraCmd.Flags().StringVar(&cmd.ReleaseValues, "release-values", "", "Path where to load the virtual cluster helm release values from")
	cobraCmd.Flags().StringVar(&cmd.K3SImage, "k3s-image", "", "If specified, use this k3s image version")
	cobraCmd.Flags().StringSliceVarP(&cmd.ExtraValues, "extra-values", "f", []string{}, "Path where to load extra helm values from")
	cobraCmd.Flags().BoolVar(&cmd.CreateNamespace, "create-namespace", true, "If true the namespace will be created if it does not exist")
	cobraCmd.Flags().BoolVar(&cmd.DisableIngressSync, "disable-ingress-sync", false, "If true the virtual cluster will not sync any ingresses")
	cobraCmd.Flags().BoolVar(&cmd.CreateClusterRole, "create-cluster-role", false, "If true a cluster role will be created to access nodes, storageclasses and priorityclasses")
	cobraCmd.Flags().BoolVar(&cmd.Expose, "expose", false, "If true will create a load balancer service to expose the vcluster endpoint")
	cobraCmd.Flags().BoolVar(&cmd.Connect, "connect", false, "If true will run vcluster connect directly after the vcluster was created")
	cobraCmd.Flags().BoolVar(&cmd.Upgrade, "upgrade", true, "If true will try to upgrade the vcluster instead of failing if it already exists")
	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(cobraCmd *cobra.Command, args []string) error {
	// test for helm
	helmExecutablePath, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("seems like helm is not installed. Helm is required for the creation of a virtual cluster. Please visit https://helm.sh/docs/intro/install/ for install instructions")
	}

	output, err := exec.Command(helmExecutablePath, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Seems like there are issues with your helm client: \n\n%s", output)
	}

	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: cmd.Context,
	})

	// load the raw config
	rawConfig, err := kubeClientConfig.RawConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}
	if cmd.Context != "" {
		rawConfig.CurrentContext = cmd.Context
	}

	// load the rest config
	kubeConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("there is an error loading your current kube config (%v), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	client, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	namespace, _, err := kubeClientConfig.Namespace()
	if err != nil {
		return err
	} else if namespace == "" {
		namespace = "default"
	}
	if cmd.Namespace != "" {
		namespace = cmd.Namespace
	}

	// make sure namespace exists
	_, err = client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			// try to create the namespace
			cmd.log.Infof("Creating namespace %s", namespace)
			_, err = client.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return errors.Wrap(err, "create namespace")
			}
		} else if kerrors.IsForbidden(err) == false {
			return err
		}
	}

	// load the default values
	values := ""
	if cmd.ReleaseValues == "" {
		values, err = cmd.getDefaultReleaseValues(client, namespace, cmd.log)
		if err != nil {
			return err
		}
	} else {
		byteValues, err := ioutil.ReadFile(cmd.ReleaseValues)
		if err != nil {
			return errors.Wrap(err, "read release values")
		}

		values = string(byteValues)
	}

	// check if vcluster already exists
	if cmd.Upgrade == false {
		_, err = client.AppsV1().StatefulSets(namespace).Get(context.TODO(), args[0], metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("vcluster %s already exists in namespace %s", args[0], namespace)
		}
	}

	// we have to upgrade / install the chart
	err = helm.NewClient(&rawConfig, cmd.log).Upgrade(args[0], namespace, helm.UpgradeOptions{
		Chart:       cmd.ChartName,
		Repo:        cmd.ChartRepo,
		Version:     cmd.ChartVersion,
		Values:      values,
		ValuesFiles: cmd.ExtraValues,
	})
	if err != nil {
		return err
	}

	cmd.log.Donef("Successfully created virtual cluster %s in namespace %s. Use 'vcluster connect %s --namespace %s' to access the virtual cluster", args[0], namespace, args[0], namespace)

	// check if we should connect to the vcluster
	if cmd.Connect {
		connectCmd := &ConnectCmd{
			GlobalFlags: cmd.GlobalFlags,
			KubeConfig:  "./kubeconfig.yaml",
			LocalPort:   8443,
			Log:         cmd.log,
		}

		return connectCmd.Connect(args[0])
	}

	return nil
}

func (cmd *CreateCmd) getDefaultReleaseValues(client kubernetes.Interface, namespace string, log log.Logger) (string, error) {
	image := cmd.K3SImage
	serverVersionString := ""
	if image == "" {
		serverVersion, err := client.Discovery().ServerVersion()
		if err != nil {
			return "", err
		}

		serverVersionString = replaceRegEx.ReplaceAllString(serverVersion.Major, "") + "." + replaceRegEx.ReplaceAllString(serverVersion.Minor, "")
		serverMinorInt, err := strconv.Atoi(replaceRegEx.ReplaceAllString(serverVersion.Minor, ""))
		if err != nil {
			return "", err
		}

		var ok bool
		image, ok = VersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 22 {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.22", serverVersionString)
				image = VersionMap["1.22"]
				serverVersionString = "1.22"
			} else {
				log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.16", serverVersionString)
				image = VersionMap["1.16"]
				serverVersionString = "1.16"
			}
		}
	}

	cidr := ""
	_, err := client.CoreV1().Services(namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-service-",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			ClusterIP: "4.4.4.4",
		},
	}, metav1.CreateOptions{})
	if err == nil {
		log.Warnf("couldn't find cluster service cidr, will fallback to 10.96.0.0/12, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match")
		cidr = "10.96.0.0/12"
	} else {
		errorMessage := err.Error()
		idx := strings.Index(errorMessage, errorMessageFind)
		if idx == -1 {
			log.Warnf("couldn't find cluster service cidr (" + errorMessage + "), will fallback to 10.96.0.0/12, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match")
			cidr = "10.96.0.0/12"
		} else {
			cidr = strings.TrimSpace(errorMessage[idx+len(errorMessageFind):])
		}
	}

	// build values
	values := `vcluster:
  image: ##IMAGE##
  extraArgs:
    - --service-cidr=##CIDR##
##BASEARGS##
storage:
  size: 5Gi
`
	if cmd.DisableIngressSync {
		values += `
syncer:
  extraArgs: ["--disable-sync-resources=ingresses"]`
	}
	if cmd.CreateClusterRole {
		values += `
rbac:
  clusterRole:
    create: true`
	}

	if cmd.Expose {
		values += `
service:
  type: LoadBalancer`
	}

	values = strings.ReplaceAll(values, "##IMAGE##", image)
	values = strings.ReplaceAll(values, "##CIDR##", cidr)
	if cmd.K3SImage == "" {
		baseArgs := baseArgsMap[serverVersionString]
		values = strings.ReplaceAll(values, "##BASEARGS##", baseArgs)
	}
	values = strings.TrimSpace(values)
	return values, nil
}
