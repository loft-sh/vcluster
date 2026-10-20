package token

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcertutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/retry"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	kubeadmconfigv1beta4 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta4"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util"
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
	Profile      string
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
	createCmd.Flags().BoolVar(&cmd.ControlPlane, "control-plane", false, "If set the created token will be used to join the control plane node. Mutually exclusive with --kubeadm and --profile")
	createCmd.Flags().StringVar(&cmd.Profile, "profile", "", "The node profile to attach to the token. Validated against the project's allowedNodeProfiles. Mutually exclusive with --kubeadm and --control-plane")
	return createCmd
}

func (cmd *CreateCmd) Run(ctx context.Context) error {
	if cmd.Kubeadm {
		if cmd.ControlPlane {
			return fmt.Errorf("--kubeadm and --control-plane are mutually exclusive")
		} else if cmd.Profile != "" {
			return fmt.Errorf("--kubeadm and --profile are mutually exclusive")
		}
	}
	if cmd.ControlPlane && cmd.Profile != "" {
		return fmt.Errorf("--control-plane and --profile are mutually exclusive")
	}

	var profileSpec *storagev1.NodeProfileSpec
	if cmd.Profile != "" {
		cfg := cmd.GlobalFlags.LoadedConfig(cmd.Log)
		platformClient, err := platform.InitClientFromConfig(ctx, cfg)
		if err != nil {
			return fmt.Errorf("connect to platform: %w", err)
		}

		project, err := find.ResolveConnectedProject(ctx, platformClient, cmd.GlobalFlags, cmd.Log)
		if err != nil {
			return err
		}

		profileSpec, err = platform.ResolveNodeProfile(ctx, platformClient, project, cmd.Profile)
		if err != nil {
			return err
		}
	}

	// get the client
	vClient, err := getClient(cmd.GlobalFlags)
	if err != nil {
		return err
	}

	// create the token
	platformEndpoint, apiEndpoint, token, tokenSecretName, caHash, err := CreateBootstrapToken(ctx, vClient, cmd.Expires, cmd.ControlPlane)
	if err != nil {
		return err
	}

	if profileSpec != nil {
		if err := patchTokenWithProfile(ctx, vClient, tokenSecretName, cmd.Profile, *profileSpec); err != nil {
			// cleanup the token
			if err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).Delete(ctx, tokenSecretName, metav1.DeleteOptions{}); err != nil {
				cmd.Log.Errorf("failed to cleanup bootstrap token: %v", err)
			}
			return err
		}
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

// patchTokenWithProfile embeds the resolved NodeProfileSpec into the bootstrap token Secret
// so that the join script renderer can apply labels, taints, and annotations to the node.
// It also sets NodeProfileReferenceLabel to record which catalog profile was used.
func patchTokenWithProfile(ctx context.Context, vClient *kubernetes.Clientset, secretName, profileName string, spec storagev1.NodeProfileSpec) error {
	payload, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal profile spec: %w", err)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret, err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to load bootstrap token secret %s: %w", secretName, err)
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[storagev1.NodeProfileConfigSecretKey] = payload
		if secret.Labels == nil {
			secret.Labels = map[string]string{}
		}
		secret.Labels[storagev1.NodeProfileReferenceLabel] = profileName

		if _, err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to patch bootstrap token secret %s with profile %s: %w", secretName, profileName, err)
		}
		return nil
	})
}

// CreateBootstrapToken attempts to create a token with the given ID. Its public because it's used in e2e tests.
func CreateBootstrapToken(ctx context.Context, vClient *kubernetes.Clientset, expires string, controlPlane bool) (platformEndpoint, apiEndpoint, token, tokenSecretName, caHash string, err error) {
	// get api server endpoint
	kubeadmConfig, err := vClient.CoreV1().ConfigMaps("kube-system").Get(ctx, kubeadmconstants.KubeadmConfigConfigMap, metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("getting kubeadm config: %w. Are you connected to a vCluster with private nodes enabled?", err)
	}

	// parse kubeadm config
	clusterConfig := &kubeadmconfigv1beta4.ClusterConfiguration{}
	if err := yaml.Unmarshal([]byte(kubeadmConfig.Data["ClusterConfiguration"]), clusterConfig); err != nil {
		return "", "", "", "", "", fmt.Errorf("unmarshalling kubeadm config: %w", err)
	}

	platformEndpoint = kubeadmConfig.Annotations[JoinScriptEndpointAnnotation]
	if err := validateJoinScriptEndpoint(platformEndpoint); err != nil {
		return "", "", "", "", "", err
	}

	apiEndpoint = clusterConfig.ControlPlaneEndpoint
	if _, _, err := util.ParseHostPort(apiEndpoint); err != nil {
		return "", "", "", "", "", err
	}

	// basically copied from https://github.com/kubernetes-sigs/cluster-api/blob/9c1392dcc6b921570161c3e3ce7c859d7dab3a4d/bootstrap/kubeadm/internal/controllers/token.go#L33
	token, err = bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("unable to generate bootstrap token: %w", err)
	}

	// generate the token id and secret
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(token)
	if len(substrs) != 3 {
		return "", "", "", "", "", fmt.Errorf("the bootstrap token %q was not of the form %q", token, bootstrapapi.BootstrapTokenPattern)
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
			return "", "", "", "", "", fmt.Errorf("invalid duration: %w", err)
		}

		secretToken.Data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(time.Now().UTC().Add(ttl).Format(time.RFC3339))
	}

	// create the secret
	if _, err := vClient.CoreV1().Secrets(metav1.NamespaceSystem).Create(ctx, secretToken, metav1.CreateOptions{}); err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to create bootstrap token secret: %w", err)
	}

	// get the ca cert from configmap
	configMap, err := vClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, "kube-root-ca.crt", metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to get ca cert: %w", err)
	}

	// now calculate the ca cert hash we will need for the join command
	caCerts, err := clientcertutil.ParseCertsPEM([]byte(configMap.Data["ca.crt"]))
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("failed to load CA certificate referenced by kubeconfig: %w", err)
	} else if len(caCerts) == 0 {
		return "", "", "", "", "", fmt.Errorf("no CA certificate found in configmap %s", configMap.Name)
	} else if len(caCerts) > 1 {
		return "", "", "", "", "", fmt.Errorf("multiple CA certificates found in configmap %s", configMap.Name)
	}

	return platformEndpoint, apiEndpoint, token, secretName, pubkeypin.Hash(caCerts[0]), nil
}

func validateJoinScriptEndpoint(endpoint string) error {
	if endpoint == "" {
		return nil
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid join-script-endpoint URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("join-script-endpoint must use https scheme, got %q", u.Scheme)
	}

	hostPort := net.JoinHostPort(u.Hostname(), lo.CoalesceOrEmpty(u.Port(), "6443"))
	if _, _, err := util.ParseHostPort(hostPort); err != nil {
		return fmt.Errorf("invalid join-script-endpoint: %s", endpoint)
	}

	return nil
}

func newKubeClientConfig(flags *flags.GlobalFlags) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{
		CurrentContext: flags.Context,
	})
}

func getClient(flags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	restConfig, err := newKubeClientConfig(flags).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	return kubernetes.NewForConfig(restConfig)
}
