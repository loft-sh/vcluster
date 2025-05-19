package cert

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/certhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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
	SANs, err := getExtraSANs(ctx, workloadNamespaceClient, vClient, vConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error getting extra sans: %w", err)
	}

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

	altNames := &certhelper.AltNames{
		DNSNames: dnsNames,
		IPs:      []net.IP{net.ParseIP("127.0.0.1")},
	}

	addSANs(altNames, SANs)

	altNamesSlice := []string{}
	for _, ip := range altNames.IPs {
		altNamesSlice = append(altNamesSlice, ip.String())
	}
	altNamesSlice = append(altNamesSlice, altNames.DNSNames...)

	caBytes, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, nil, nil, err
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

	caCert, err := certhelper.ParseCertsPEM(caBytes)
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
	retSANs := []string{
		vConfig.WorkloadService,
		vConfig.WorkloadService + "." + vConfig.WorkloadNamespace, "*." + constants.NodeSuffix,
	}

	// get cluster ip of target service
	svc := &corev1.Service{}
	err := workloadNamespaceClient.Get(ctx, types.NamespacedName{
		Namespace: vConfig.WorkloadNamespace,
		Name:      vConfig.WorkloadService,
	}, svc)
	if err != nil {
		return nil, fmt.Errorf("error getting vcluster service %s/%s: %w", vConfig.WorkloadNamespace, vConfig.WorkloadService, err)
	} else if svc.Spec.ClusterIP == "" {
		return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", vConfig.WorkloadNamespace, vConfig.WorkloadService)
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

	// if dedicated mode is enabled, we need to get the service ip within the virtual cluster
	if vConfig.PrivateNodes.Enabled {
		svc := &corev1.Service{}
		err = vClient.Get(ctx, types.NamespacedName{
			Namespace: "default",
			Name:      "kubernetes",
		}, svc)
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster kubernetes service: %w", err)
		}

		retSANs = append(retSANs, svc.Spec.ClusterIP)
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

	// get cluster ip of load balancer service
	lbSVC := &corev1.Service{}
	err = workloadNamespaceClient.Get(ctx, types.NamespacedName{
		Namespace: vConfig.WorkloadNamespace,
		Name:      vConfig.WorkloadService,
	}, lbSVC)
	// proceed only if load balancer service exists
	if !kerrors.IsNotFound(err) {
		if err != nil {
			return nil, fmt.Errorf("error getting vcluster load balancer service %s/%s: %w", vConfig.WorkloadNamespace, vConfig.WorkloadService, err)
		} else if lbSVC.Spec.ClusterIP == "" {
			return nil, fmt.Errorf("target service %s/%s is missing a clusterIP", vConfig.WorkloadNamespace, vConfig.WorkloadService)
		}

		for _, ing := range lbSVC.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				retSANs = append(retSANs, ing.IP)
			}
			if ing.Hostname != "" {
				retSANs = append(retSANs, ing.Hostname)
			}
		}
		// append hostnames for load balancer service
		retSANs = append(
			retSANs,
			vConfig.WorkloadService,
			vConfig.WorkloadService+"."+vConfig.WorkloadNamespace, "*."+translate.VClusterName+"."+vConfig.WorkloadNamespace+"."+constants.NodeSuffix,
		)
	}

	if vConfig.Networking.Advanced.ProxyKubelets.ByIP {
		// get cluster ips of node services
		svcs := &corev1.ServiceList{}
		err = workloadNamespaceClient.List(ctx, svcs, client.InNamespace(vConfig.WorkloadNamespace), client.MatchingLabels{nodeservice.ServiceClusterLabel: translate.VClusterName})
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

	// ingress host
	if vConfig.ControlPlane.Ingress.Host != "" {
		retSANs = append(retSANs, vConfig.ControlPlane.Ingress.Host)
	}

	// make sure other sans are there as well
	retSANs = append(retSANs, vConfig.ControlPlane.Proxy.ExtraSANs...)
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
