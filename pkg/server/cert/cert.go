package cert

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenAPIServerServingCerts(
	ctx context.Context,
	workloadNamespaceClient,
	vClient client.Client,
	vConfig *config.VirtualClusterConfig,
	caCertFile,
	caKeyFile string,
	currentCert,
	currentKey []byte,
) ([]byte, []byte, []string, error) {
	sans, err := getExtraSANs(ctx, workloadNamespaceClient, vClient, vConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting extra sans: %w", err)
	}

	sans = lo.UniqBy(sans, func(s string) string {
		return strings.ToLower(s)
	})

	regen := false
	commonName := "kube-apiserver"
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	dnsNames := []string{
		"kubernetes.default.svc." + vConfig.Networking.Advanced.ClusterDomain,
		"kubernetes.default.svc",
		"kubernetes.default",
		"kubernetes",
		"localhost",
	}

	// if konnectivity is enabled, we need to add the konnectivity service to the sans
	if vConfig.PrivateNodes.Enabled && vConfig.ControlPlane.Advanced.Konnectivity.Server.Enabled {
		dnsNames = append(dnsNames, "konnectivity")
		dnsNames = append(dnsNames, "konnectivity.kube-system")
		dnsNames = append(dnsNames, "konnectivity.kube-system.svc")
		dnsNames = append(dnsNames, "konnectivity.kube-system.svc."+vConfig.Networking.Advanced.ClusterDomain)
	}

	altNames := &certhelper.AltNames{
		DNSNames: dnsNames,
		IPs:      []net.IP{net.ParseIP("127.0.0.1")},
	}

	addSANs(altNames, sans)

	altNamesSlice := []string{}
	for _, ip := range altNames.IPs {
		altNamesSlice = append(altNamesSlice, ip.String())
	}
	altNamesSlice = append(altNamesSlice, altNames.DNSNames...)

	caBytes, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, nil, nil, err
	}

	caCert, err := certhelper.ParseCertsPEM(caBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	// check if caCert is expired.
	if time.Now().After(caCert[0].NotAfter) {
		return nil, nil, nil, fmt.Errorf("expired CA certificate: %s", caCertFile)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caBytes)

	// check for certificate expiration
	if !regen {
		regen = expired(&currentCert, pool)
	}

	if !regen {
		regen = sansChanged(&currentCert, altNames)
	}

	if !regen {
		if len(currentCert) > 0 && len(currentKey) > 0 {
			return currentCert, currentKey, altNamesSlice, nil
		}
	}

	caKeyBytes, err := os.ReadFile(caKeyFile)
	if err != nil {
		return nil, nil, nil, err
	}

	caKey, err := certhelper.ParsePrivateKeyPEM(caKeyBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	privateKey := currentKey
	if regen || len(currentKey) == 0 {
		privateKey, err = certhelper.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error generating key: %w", err)
		}
	}
	key, err := certhelper.ParsePrivateKeyPEM(privateKey)
	if err != nil {
		return nil, nil, nil, err
	}

	cfg := certhelper.Config{
		CommonName: commonName,
		AltNames:   *altNames,
		Usages:     extKeyUsage,
	}
	cert, err := certhelper.NewSignedCert(cfg, key.(crypto.Signer), caCert[0], caKey.(crypto.Signer))
	if err != nil {
		return nil, nil, nil, err
	}
	certificate := append(certhelper.EncodeCertPEM(cert), certhelper.EncodeCertPEM(caCert[0])...)
	return certificate, privateKey, altNamesSlice, nil
}

func getExtraSANs(ctx context.Context, workloadNamespaceClient, vClient client.Client, vConfig *config.VirtualClusterConfig) ([]string, error) {
	retSANs := []string{}

	// ingress host
	if vConfig.ControlPlane.Ingress.Host != "" {
		retSANs = append(retSANs, vConfig.ControlPlane.Ingress.Host)
	}

	// make sure other sans are there as well
	retSANs = append(retSANs, vConfig.ControlPlane.Proxy.ExtraSANs...)

	// if we have a custom endpoint, we need to add the host to the sans
	if vConfig.ControlPlane.Endpoint != "" {
		host, _, err := net.SplitHostPort(vConfig.ControlPlane.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint %s: %w", vConfig.ControlPlane.Endpoint, err)
		} else if host != "" {
			retSANs = append(retSANs, host)
		}
	}

	// if dedicated mode is enabled, we need to get the service ip within the virtual cluster
	if vConfig.PrivateNodes.Enabled {
		// add the cluster ip of the kubernetes service
		svc := &corev1.Service{}
		err := vClient.Get(ctx, types.NamespacedName{
			Namespace: "default",
			Name:      "kubernetes",
		}, svc)
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster kubernetes service: %w", err)
		}
		retSANs = append(retSANs, svc.Spec.ClusterIP)

		// get standalone endpoints via annotation
		if svc.Annotations[constants.VClusterStandaloneEndpointsAnnotation] != "" {
			retSANs = append(retSANs, strings.Split(svc.Annotations[constants.VClusterStandaloneEndpointsAnnotation], ",")...)
		}

		// get endpoint
		clusterInfo := &corev1.ConfigMap{}
		err = vClient.Get(ctx, types.NamespacedName{
			Namespace: "kube-public",
			Name:      "cluster-info",
		}, clusterInfo)
		if err != nil && !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error getting vcluster cluster-info configmap: %w", err)
		}
		if clusterInfo.Data["kubeconfig"] != "" {
			clusterInfo, err := clientcmd.Load([]byte(clusterInfo.Data["kubeconfig"]))
			if err != nil {
				klog.FromContext(ctx).Error(err, "error loading kubeconfig")
			} else {
				for _, cluster := range clusterInfo.Clusters {
					url, err := url.Parse(cluster.Server)
					if err != nil {
						continue
					}

					retSANs = append(retSANs, url.Hostname())
				}
			}
		}
	}

	// if standalone mode is enabled, we don't need to add any other sans
	if vConfig.ControlPlane.Standalone.Enabled {
		sort.Strings(retSANs)
		return retSANs, nil
	}

	// add default sans
	retSANs = append(retSANs, vConfig.Name, vConfig.Name+"."+vConfig.HostNamespace, "*."+constants.NodeSuffix)

	// get cluster ip of target service
	svc := &corev1.Service{}
	err := workloadNamespaceClient.Get(ctx, types.NamespacedName{
		Namespace: vConfig.HostNamespace,
		Name:      vConfig.Name,
	}, svc)
	if err != nil {
		return nil, fmt.Errorf("error getting vcluster service %s/%s: %w", vConfig.HostNamespace, vConfig.Name, err)
	} else if svc.Spec.ClusterIP == "" {
		return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", vConfig.HostNamespace, vConfig.Name)
	}

	// append general hostnames
	retSANs = append(
		retSANs,
		vConfig.Name,
		vConfig.Name+"."+vConfig.HostNamespace,
		"*."+translate.VClusterName+"."+vConfig.HostNamespace+"."+constants.NodeSuffix,
	)

	// if the service is a node port, we need to add the node ips to the sans
	if svc.Spec.Type == corev1.ServiceTypeNodePort {
		pods := &corev1.PodList{}
		err = workloadNamespaceClient.List(ctx, pods, client.InNamespace(vConfig.HostNamespace), client.MatchingLabels{"app": "vcluster", "release": vConfig.Name})
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster control plane pods: %w", err)
		}
		for _, pod := range pods.Items {
			if len(pod.Status.HostIPs) > 0 {
				for _, hostIP := range pod.Status.HostIPs {
					if hostIP.IP == "" {
						continue
					}

					retSANs = append(retSANs, hostIP.IP)
				}
			}
		}
	}

	// add cluster ip
	retSANs = append(retSANs, svc.Spec.ClusterIP)

	// get load balancer ip
	// currently, the load balancer service is named <serviceName>, but the syncer image might run in legacy environments
	// where the load balancer service is the same service, the service is only updated if the helm template is rerun,
	// so we are leaving this snippet in, but the load balancer ip will be read via the lbSVC var below
	for _, ing := range svc.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			retSANs = append(retSANs, ing.IP)
		}
		if ing.Hostname != "" {
			retSANs = append(retSANs, ing.Hostname)
		}
	}

	// add extra sans
	for _, extraSans := range ExtraSANs {
		extraSansValues, err := extraSans(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting extra sans: %w", err)
		}

		retSANs = append(retSANs, extraSansValues...)
	}

	// add pod IP
	podIP := os.Getenv("POD_IP")
	if podIP != "" {
		retSANs = append(retSANs, podIP)
	}

	if vConfig.Networking.Advanced.ProxyKubelets.ByIP {
		// get cluster ips of node services
		svcs := &corev1.ServiceList{}
		err = workloadNamespaceClient.List(ctx, svcs, client.InNamespace(vConfig.HostNamespace), client.MatchingLabels{nodeservice.ServiceClusterLabel: translate.VClusterName})
		if err != nil {
			return nil, err
		}
		for _, svc := range svcs.Items {
			if svc.Spec.ClusterIP == "" {
				continue
			}

			retSANs = append(retSANs, svc.Spec.ClusterIP)
		}
	}

	sort.Strings(retSANs)
	return retSANs, nil
}

func addSANs(altNames *certhelper.AltNames, sans []string) {
	for _, san := range sans {
		ip := net.ParseIP(san)
		if ip == nil {
			altNames.DNSNames = append(altNames.DNSNames, san)
		} else {
			altNames.IPs = append(altNames.IPs, ip)
		}
	}
}

func expired(certBytes *[]byte, pool *x509.CertPool) bool {
	certificates, err := certhelper.ParseCertsPEM(*certBytes)
	if err != nil {
		return false
	}
	_, err = certificates[0].Verify(x509.VerifyOptions{
		Roots: pool,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageAny,
		},
	})
	if err != nil {
		return true
	}
	return certhelper.IsCertExpired(certificates[0])
}

func sansChanged(certBytes *[]byte, sans *certhelper.AltNames) bool {
	if sans == nil {
		return false
	}

	certificates, err := certhelper.ParseCertsPEM(*certBytes)
	if err != nil {
		return false
	}

	if len(certificates) == 0 {
		return false
	}

	if !sets.NewString(certificates[0].DNSNames...).HasAll(sans.DNSNames...) {
		return true
	}

	ips := sets.NewString()
	for _, ip := range certificates[0].IPAddresses {
		ips.Insert(ip.String())
	}

	for _, ip := range sans.IPs {
		if !ips.Has(ip.String()) {
			return true
		}
	}

	return false
}
