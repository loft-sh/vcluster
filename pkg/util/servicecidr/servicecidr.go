package servicecidr

import (
	"context"
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CIDRConfigMapPrefix = "vc-cidr-"
	CIDRConfigMapKey    = "cidr"
	K0sConfigKey        = "config.yaml"
	K0sCIDRPlaceHolder  = "CIDR_PLACEHOLDER"
	K0sConfigReadyFlag  = "CONFIG_READY"

	ErrorMessageFind = "The range of valid IPs is "
	FallbackCIDR     = "10.96.0.0/12"
)

func GetCIDRConfigMapName(vclusterName string) string {
	return fmt.Sprintf("%s%s", CIDRConfigMapPrefix, vclusterName)
}

func GetK0sSecretName(vclusterName string) string {
	return fmt.Sprintf("vc-%s-config", vclusterName)
}

func EnsureServiceCIDRConfigmap(ctx context.Context, c kubernetes.Interface, currentNamespace string, vclusterName string) (string, error) {
	cm, err := c.CoreV1().ConfigMaps(currentNamespace).Get(ctx, GetCIDRConfigMapName(vclusterName), metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return "", err
	}
	exists := !kerrors.IsNotFound(err)
	if !exists {
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetCIDRConfigMapName(vclusterName),
				Namespace: currentNamespace,
			},
		}
	}
	cidrData, ok := cm.Data[CIDRConfigMapKey]
	// do nothing if a valid CIDR is already present in the expected Configmap data key
	if exists && ok {
		_, _, err = net.ParseCIDR(cidrData)
		if err == nil {
			return cidrData, err
		}
	}

	// find out correct cidr
	cidr, warning := GetServiceCIDR(c, currentNamespace)
	if warning != "" {
		klog.Warning(warning)
	}

	if !exists {
		cm.Data = map[string]string{
			CIDRConfigMapKey: cidr,
		}
		_, err = c.CoreV1().ConfigMaps(currentNamespace).Create(ctx, cm, metav1.CreateOptions{})
		return cidr, err
	}

	// create and execute a Patch call for the ConfigMap
	originalObject := cm.DeepCopy()
	patch := client.MergeFrom(originalObject)
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[CIDRConfigMapKey] = cidr
	data, err := patch.Data(cm)
	if err != nil {
		return "", fmt.Errorf("failed to create patch for the %s/%s Configmap: %v", cm.Namespace, cm.Name, err)
	}
	_, err = c.CoreV1().ConfigMaps(currentNamespace).Patch(ctx, cm.Name, patch.Type(), data, metav1.PatchOptions{})
	return cidr, err
}

func EnsureServiceCIDRInK0sSecret(ctx context.Context, c kubernetes.Interface, currentNamespace string, vclusterName string) error {
	secret, err := c.CoreV1().Secrets(currentNamespace).Get(context.Background(), GetK0sSecretName(vclusterName), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not read k0s configuration secret %s/%s: %v", currentNamespace, GetK0sSecretName(vclusterName), err)
	}
	configData, ok := secret.Data[K0sConfigKey]
	if !ok {
		return fmt.Errorf("k0s configuration secret %s/%s does not contain the expected key - %s", secret.Namespace, secret.Name, K0sConfigKey)
	}

	// find out correct cidr
	cidr, warning := GetServiceCIDR(c, currentNamespace)
	if warning != "" {
		klog.Warning(warning)
	}
	newData := strings.ReplaceAll(string(configData), K0sCIDRPlaceHolder, cidr)

	originalObject := secret.DeepCopy()
	secret.Data[K0sConfigKey] = []byte(newData)
	secret.Data[K0sConfigReadyFlag] = []byte("true")
	patch := client.MergeFrom(originalObject)
	data, err := patch.Data(secret)
	if err != nil {
		return fmt.Errorf("failed to create patch for the %s/%s Secret: %v", secret.Namespace, secret.Name, err)
	}
	_, err = c.CoreV1().Secrets(secret.Namespace).Patch(ctx, secret.Name, patch.Type(), data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch k0s configuration secret %s/%s: %v", secret.Namespace, secret.Name, err)
	}
	return nil
}

func GetServiceCIDR(client kubernetes.Interface, namespace string) (string, string) {
	ipv4CIDR, ipv4Err := getServiceCIDR(client, namespace, false)
	ipv6CIDR, ipv6Err := getServiceCIDR(client, namespace, true)
	if ipv4Err != nil && ipv6Err != nil {
		return FallbackCIDR, fmt.Sprintf("failed to detect service CIDR, will fallback to %s, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match. Error details: failed to find IPv4 service CIDR: %v ; or IPv6 service CIDR: %v", FallbackCIDR, ipv4Err, ipv6Err)
	}
	if ipv4Err != nil {
		return ipv6CIDR, fmt.Sprintf("failed to find IPv4 service CIDR: %v", ipv4Err)
	}
	if ipv6Err != nil {
		return ipv4CIDR, fmt.Sprintf("failed to find IPv6 service CIDR: %v", ipv6Err)
	}

	// Both IPv4 and IPv6 are configured, we need to find out which one is the default
	policy := corev1.IPFamilyPolicyPreferDualStack
	testService, err := client.CoreV1().Services(namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-service-delete-me-",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			IPFamilyPolicy: &policy,
		},
	}, metav1.CreateOptions{})

	if err == nil {
		defer func() {
			_ = client.CoreV1().Services(namespace).Delete(context.Background(), testService.GetName(), metav1.DeleteOptions{})
		}()

		// check if this is dual stack, and which family is default
		if len(testService.Spec.IPFamilies) > 0 {
			if testService.Spec.IPFamilies[0] == corev1.IPv4Protocol {
				// IPv4 is the default
				return fmt.Sprintf("%s,%s", ipv4CIDR, ipv6CIDR), ""
			} else {
				// IPv6 is the default
				return fmt.Sprintf("%s,%s", ipv6CIDR, ipv4CIDR), ""
			}
		} else {
			return ipv4CIDR, fmt.Sprintf("unexpected number of entries in .Spec.IPFamilies - %d, defaulting to IPv4 CIDR only", len(testService.Spec.IPFamilies))
		}
	}

	return fmt.Sprintf("%s,%s", ipv4CIDR, ipv6CIDR), "failed to find host cluster default Service IP family, defaulting to IPv4 family"
}

func getServiceCIDR(client kubernetes.Interface, namespace string, ipv6 bool) (string, error) {
	clusterIP := "4.4.4.4"
	if ipv6 {
		// https://www.ietf.org/rfc/rfc3849.txt
		clusterIP = "2001:DB8::1"
	}
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
			ClusterIP: clusterIP,
		},
	}, metav1.CreateOptions{})
	if err == nil {
		return "", fmt.Errorf("couldn't find host cluster Service CIDR")
	}

	errorMessage := err.Error()
	idx := strings.Index(errorMessage, ErrorMessageFind)
	if idx == -1 {
		return "", fmt.Errorf("couldn't find host cluster Service CIDR (\"%s\")", errorMessage)
	}

	cidr := strings.TrimSpace(errorMessage[idx+len(ErrorMessageFind):])
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	isIPv4 := ip.To4() != nil

	if isIPv4 && ipv6 {
		return "", fmt.Errorf("invalid IP family, got IPv4 when trying to determine IPv6 Service CIDR")
	}
	if !isIPv4 && !ipv6 {
		return "", fmt.Errorf("invalid IP family, got invalid IPv4 address when trying to determine IPv4 Service CIDR")
	}
	return cidr, nil
}
