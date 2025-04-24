package dedicated

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

const (
	KonnectivityManifests = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:konnectivity-server
  labels:
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: system:konnectivity-server
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: konnectivity-agent
  namespace: kube-system
  labels:
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    addonmanager.kubernetes.io/mode: Reconcile
    k8s-app: konnectivity-agent
  namespace: kube-system
  name: konnectivity-agent
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      k8s-app: konnectivity-agent
  template:
    metadata:
      labels:
        k8s-app: konnectivity-agent
    spec:
      priorityClassName: system-cluster-critical
      tolerations:
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
        - image: {{ .Image }}
          name: konnectivity-agent
          command: ["/proxy-agent"]
          args: [
                  "--logtostderr=true",
                  "--ca-cert=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
                  "--proxy-server-host={{ .Host }}",
                  "--proxy-server-port=8132",
                  "--admin-server-port=8133",
                  "--health-server-port=8134",
                  "--service-account-token-path=/var/run/secrets/tokens/konnectivity-agent-token"
                  ]
          volumeMounts:
            - mountPath: /var/run/secrets/tokens
              name: konnectivity-agent-token
          livenessProbe:
            httpGet:
              port: 8134
              path: /healthz
            initialDelaySeconds: 15
            timeoutSeconds: 15
      serviceAccountName: konnectivity-agent
      volumes:
        - name: konnectivity-agent-token
          projected:
            sources:
              - serviceAccountToken:
                  path: konnectivity-agent-token
                  audience: system:konnectivity-server
`
)

type KonnectivityConfig struct {
	Replicas int32
	Image    string
	Host     string
}

func ApplyKonnectivityManifests(ctx context.Context, vConfig *rest.Config, kubeadmConfig *kubeadmapi.InitConfiguration) error {
	var err error

	// get the host from the control plane endpoint
	host := kubeadmConfig.ClusterConfiguration.ControlPlaneEndpoint
	if strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return fmt.Errorf("error splitting host and port: %w", err)
		}
	}

	// create the vars
	vars := map[string]interface{}{
		"Replicas": 1,
		"Host":     host,
		"Image":    "registry.k8s.io/kas-network-proxy/proxy-agent:v0.32.0",
	}

	// process the template
	manifest, err := processKonnectivityTemplate(vars)
	if err != nil {
		return fmt.Errorf("error processing konnectivity template: %w", err)
	}

	// apply the manifests
	klog.Infof("Applying kube proxy manifests...")
	return applier.ApplyManifest(ctx, vConfig, manifest)
}

func processKonnectivityTemplate(vars map[string]interface{}) ([]byte, error) {
	manifestTemplate, err := template.New("konnectivity").Parse(KonnectivityManifests)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %w", KonnectivityManifests, err)
	}

	buf := new(bytes.Buffer)
	err = manifestTemplate.Execute(buf, vars)
	if err != nil {
		return nil, fmt.Errorf("manifestTemplate.Execute failed for manifest %s: %w", KonnectivityManifests, err)
	}

	return buf.Bytes(), nil
}
