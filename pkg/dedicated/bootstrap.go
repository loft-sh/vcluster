package dedicated

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clusterinfophase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/clusterinfo"
	nodebootstraptokenphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/bootstraptoken/node"
)

func PrepareBootstrapToken(client kubernetes.Interface, kubeconfig, controlPlaneEndpoint string) error {
	// load the kubeconfig and make sure the endpoint is correct
	adminConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return errors.Wrap(err, "failed to load admin kubeconfig")
	}
	for key := range adminConfig.Clusters {
		adminConfig.Clusters[key].Server = "https://" + controlPlaneEndpoint
	}

	// write the kubeconfig to a temp file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	_ = tempFile.Close()
	defer os.Remove(tempFile.Name())

	// write the kubeconfig to the temp file
	err = clientcmd.WriteToFile(*adminConfig, tempFile.Name())
	if err != nil {
		return errors.Wrap(err, "failed to write kubeconfig to temp file")
	}

	fmt.Println("[bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles")
	// Create the default node bootstrap token
	if err := nodebootstraptokenphase.UpdateOrCreateTokens(client, false, nil); err != nil {
		return errors.Wrap(err, "error updating or creating token")
	}
	// Create RBAC rules that makes the bootstrap tokens able to get nodes
	if err := nodebootstraptokenphase.AllowBootstrapTokensToGetNodes(client); err != nil {
		return errors.Wrap(err, "error allowing bootstrap tokens to get Nodes")
	}
	// Create RBAC rules that makes the bootstrap tokens able to post CSRs
	if err := nodebootstraptokenphase.AllowBootstrapTokensToPostCSRs(client); err != nil {
		return errors.Wrap(err, "error allowing bootstrap tokens to post CSRs")
	}
	// Create RBAC rules that makes the bootstrap tokens able to get their CSRs approved automatically
	if err := nodebootstraptokenphase.AutoApproveNodeBootstrapTokens(client); err != nil {
		return errors.Wrap(err, "error auto-approving node bootstrap tokens")
	}

	// Create/update RBAC rules that makes the nodes to rotate certificates and get their CSRs approved automatically
	if err := nodebootstraptokenphase.AutoApproveNodeCertificateRotation(client); err != nil {
		return err
	}

	// Create the cluster-info ConfigMap with the associated RBAC rules
	if err := clusterinfophase.CreateBootstrapConfigMapIfNotExists(client, tempFile.Name()); err != nil {
		return errors.Wrap(err, "error creating bootstrap ConfigMap")
	}
	if err := clusterinfophase.CreateClusterInfoRBACRules(client); err != nil {
		return errors.Wrap(err, "error creating clusterinfo RBAC rules")
	}
	return nil
}
