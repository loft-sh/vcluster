package servicecidr

import (
	"context"
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ErrorMessageFind = "The range of valid IPs is "
	FallbackCIDR     = "10.96.0.0/12"
)

func GetServiceCIDR(ctx context.Context, client kubernetes.Interface, namespace string) (string, string) {
	ipv4CIDR, ipv4Err := getServiceCIDR(ctx, client, namespace, false)
	ipv6CIDR, ipv6Err := getServiceCIDR(ctx, client, namespace, true)
	if ipv4Err != nil && ipv6Err != nil {
		return FallbackCIDR, fmt.Sprintf("failed to detect service CIDR, will fallback to %s, however this is probably wrong, please make sure the host cluster service cidr and virtual cluster service cidr match. Error details: failed to find IPv4 service CIDR: %v ; or IPv6 service CIDR: %v", FallbackCIDR, ipv4Err, ipv6Err)
	}
	if ipv4Err != nil {
		return ipv6CIDR, fmt.Sprintf("failed to find IPv4 service CIDR, will use IPv6 service CIDR. Error details: %v", ipv4Err)
	}
	if ipv6Err != nil {
		return ipv4CIDR, fmt.Sprintf("failed to find IPv6 service CIDR, will use IPv4 service CIDR. Error details: %v", ipv6Err)
	}

	// Both IPv4 and IPv6 are configured, we need to find out which one is the default
	policy := corev1.IPFamilyPolicyPreferDualStack
	testService, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
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
			_ = client.CoreV1().Services(namespace).Delete(ctx, testService.GetName(), metav1.DeleteOptions{})
		}()

		// check if this is dual stack, and which family is default
		if len(testService.Spec.IPFamilies) > 0 {
			if testService.Spec.IPFamilies[0] == corev1.IPv4Protocol {
				// IPv4 is the default
				return fmt.Sprintf("%s,%s", ipv4CIDR, ipv6CIDR), ""
			}

			// IPv6 is the default
			return fmt.Sprintf("%s,%s", ipv6CIDR, ipv4CIDR), ""
		}

		return ipv4CIDR, fmt.Sprintf("unexpected number of entries in .Spec.IPFamilies - %d, defaulting to IPv4 CIDR only", len(testService.Spec.IPFamilies))
	}

	return fmt.Sprintf("%s,%s", ipv4CIDR, ipv6CIDR), "failed to find host cluster default Service IP family, defaulting to IPv4 family"
}

func getServiceCIDR(ctx context.Context, client kubernetes.Interface, namespace string, ipv6 bool) (string, error) {
	clusterIP := "4.4.4.4"
	if ipv6 {
		// https://www.ietf.org/rfc/rfc3849.txt
		clusterIP = "2001:DB8::1"
	}
	_, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
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
