package token

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcertutil "k8s.io/client-go/util/cert"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmconfigv1beta4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pubkeypin"
	"sigs.k8s.io/yaml"
)

var (
	JoinScriptEndpointAnnotation = "vcluster.loft.sh/join-script-endpoint"
)

type CreateCmd struct {
	*flags.GlobalFlags

	Expires      string
	Kubeadm      bool
	ControlPlane bool
	Log          log.Logger
}

func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################# vcluster token create #################
########################################################
Create a new node bootstrap token for a vCluster with private nodes enabled.
#######################################################
	`

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new node bootstrap token for a vCluster with private nodes enabled.",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	createCmd.Flags().StringVar(&cmd.Expires, "expires", "1h", "The duration the token will be valid for. Format: 1h, 1d, 1w, 1m, 1y. If empty, the token will never expire.")
	createCmd.Flags().BoolVar(&cmd.Kubeadm, "kubeadm", false, "If enabled shows the raw kubeadm join command.")
	createCmd.Flags().BoolVar(&cmd.ControlPlane, "control-plane", false, "If set the created token will be used to join the control plane node. Mutually exclusive with --kubeadm")
	return createCmd
}

func (cmd *CreateCmd) Run(ctx context.Context) error {
	if cmd.Kubeadm && cmd.ControlPlane {
		return fmt.Errorf("--kubeadm and --control-plane are mutually exclusive")
	}
	// get the client
	vClient, err := getClient(cmd.GlobalFlags)
	if err != nil {
		return err
	}

	// create the token
	platformEndpoint, apiEndpoint, token, caHash, err := CreateBootstrapToken(ctx, vClient, cmd.Expires, cmd.ControlPlane)
	if err != nil {
		return err
	}

	// print the join command
	if cmd.Kubeadm {
		fmt.Printf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s\n", apiEndpoint, token, caHash)
	} else {
		if platformEndpoint != "" {
			fmt.Printf("curl -fsSLk \"%s/node/join?token=%s\" | sh -\n", platformEndpoint, url.QueryEscape(token))
		} else {
			fmt.Printf("curl -fsSLk \"https://%s/node/join?token=%s\" | sh -\n", apiEndpoint, url.QueryEscape(token))
		}
	}

	return nil
}

// CreateBootstrapToken attempts to create a token with the given ID. Its public because it's used in e2e tests.
func CreateBootstrapToken(ctx context.Context, vClient *kubernetes.Clientset, expires string, controlPlane bool) (string, string, string, string, error) {
	// get api server endpoint
	kubeadmConfig, err := vClient.CoreV1().ConfigMaps("kube-system").Get(ctx, "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", fmt.Errorf("getting kubeadm config: %w. Are you connected to a vCluster with private nodes enabled?", err)
	}

	// parse kubeadm config
	clusterConfig := &kubeadmconfigv1beta4.ClusterConfiguration{}
	if err := yaml.Unmarshal([]byte(kubeadmConfig.Data["ClusterConfiguration"]), clusterConfig); err != nil {
		return "", "", "", "", fmt.Errorf("unmarshalling kubeadm config: %w", err)
	}

	// basically copied from https://github.com/kubernetes-sigs/cluster-api/blob/9c1392dcc6b921570161c3e3ce7c859d7dab3a4d/bootstrap/kubeadm/internal/controllers/token.go#L33
	token, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", "", "", "", fmt.Errorf("unable to generate bootstrap token: %w", err)
	}

	// generate the token id and secret
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(token)
	if len(substrs) != 3 {
		return "", "", "", "", fmt.Errorf("the bootstrap token %q was not of the form %q", token, bootstrapapi.BootstrapTokenPattern)
	}
	tokenID := substrs[1]
	tokenSecret := substrs[2]

	tokenNodeType := constants.NodeTypeWorker
	if controlPlane {
		tokenNodeType = constants.NodeTypeControlPlane
	}

	// create the secret
	secretName := bootstraputil.BootstrapTokenSecretName(tokenID)
	secretToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				constants.TokenLabelKey:    "true",
				constants.TokenNodeTypeKey: tokenNodeType,
			},
		},
		Type: bootstrapapi.SecretTypeBootstrapToken,
		Data: map[string][]byte{
			bootstrapapi.BootstrapTokenIDKey:               []byte(tokenID),
			bootstrapapi.BootstrapTokenSecretKey:           []byte(tokenSecret),
			bootstrapapi.BootstrapTokenUsageSigningKey:     []byte("true"),
			bootstrapapi.BootstrapTokenUsageAuthentication: []byte("true"),
			bootstrapapi.BootstrapTokenExtraGroupsKey:      []byte("system:bootstrappers:kubeadm:default-node-token"),
			bootstrapapi.BootstrapTokenDescriptionKey:      []byte("token generated by cluster-api-bootstrap-provider-kubeadm"),
		},
	}
	if expires != "" {
		ttl, err := time.ParseDuration(expires)
		if err != nil {
			return "", "", "", "", fmt.Errorf("invalid duration: %w", err)
		}

		secretToken.Data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(time.Now().UTC().Add(ttl).Format(time.RFC3339))
	}

	// create the secret
	if _, err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).Create(ctx, secretToken, metav1.CreateOptions{}); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create bootstrap token secret: %w", err)
	}

	// get the ca cert from configmap
	configMap, err := vClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get ca cert: %w", err)
	}

	// now calculate the ca cert hash we will need for the join command
	caCerts, err := clientcertutil.ParseCertsPEM([]byte(configMap.Data["ca.crt"]))
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load CA certificate referenced by kubeconfig: %w", err)
	} else if len(caCerts) == 0 {
		return "", "", "", "", fmt.Errorf("no CA certificate found in configmap %s", configMap.Name)
	} else if len(caCerts) > 1 {
		return "", "", "", "", fmt.Errorf("multiple CA certificates found in configmap %s", configMap.Name)
	}

	return kubeadmConfig.Annotations[JoinScriptEndpointAnnotation], clusterConfig.ControlPlaneEndpoint, token, pubkeypin.Hash(caCerts[0]), nil
}

func getClient(flags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	// first load the kube config
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})

	// get the client config
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return kubernetes.NewForConfig(restConfig)
}
