package servicecidr

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/config"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	ServiceCIDRAnnotation = "vcluster.loft.sh/service-cidr"
)

func GetServiceCIDR(ctx context.Context, vConfig *config.Config, client kubernetes.Interface, vClusterServiceName, vClusterNamespace string) (string, error) {
	if vConfig.ServiceCIDR != "" {
		return vConfig.ServiceCIDR, nil
	} else if vConfig.PrivateNodes.Enabled {
		if vConfig.Networking.ServiceCIDR != "" {
			return vConfig.Networking.ServiceCIDR, nil
		}

		// fallback to the default service cidr
		return "10.96.0.0/12", nil
	}

	// check if we need to use the new service cidr detection
	vClusterService, err := client.CoreV1().Services(vClusterNamespace).Get(ctx, vClusterServiceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get vCluster service %s: %w", vClusterServiceName, err)
	}

	// check if we already found the service cidr
	if vClusterService.Annotations[ServiceCIDRAnnotation] != "" {
		klog.Infof("using cached service cidr from annotation: %s", vClusterService.Annotations[ServiceCIDRAnnotation])
		return vClusterService.Annotations[ServiceCIDRAnnotation], nil
	}

	// Create a service to determine the supported IP Families for the cluster and use it's assigned IPs
	// to determine the service CIDR
	ipFamilyService, err := getIPFamilyService(ctx, client, vClusterNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to get ip family service: %w", err)
	}

	// create function to check if the ip is in the service cidr
	isInRange := func(ip net.IP) bool {
		policy := corev1.IPFamilyPolicySingleStack
		ipFamily := corev1.IPv4Protocol
		if ip.To4() == nil {
			ipFamily = corev1.IPv6Protocol
		}

		// create an actual service to check this
		testService, err := client.CoreV1().Services(vClusterNamespace).Create(ctx, &corev1.Service{
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
				IPFamilies:     []corev1.IPFamily{ipFamily},
				ClusterIP:      ip.String(),
			},
		}, metav1.CreateOptions{})
		if err == nil {
			err := client.CoreV1().Services(vClusterNamespace).Delete(ctx, testService.GetName(), metav1.DeleteOptions{})
			if err != nil && !kerrors.IsNotFound(err) {
				klog.Warningf("failed to delete test service %s: %v", testService.GetName(), err)
			}

			return true
		}

		// see https://github.com/kubernetes/kubernetes/blob/4c1e535e39abf2acba78ac3408ab5eed77364e68/pkg/registry/core/service/ipallocator/interfaces.go#L44 for the error messages
		if !strings.Contains(err.Error(), "is already allocated") &&
			!strings.Contains(err.Error(), "network does not match") &&
			!strings.Contains(err.Error(), "not in the valid range") {
			klog.Warningf("failed to create test service %s: %v", testService.GetName(), err)
		}

		// there is a special case here where if a service with the given ip already exists or the allocator is not ready, it is part of the service cidr but returns an error
		return strings.Contains(err.Error(), "is already allocated") || strings.Contains(err.Error(), "allocator not ready")
	}

	// check if dual stack
	serviceCIDRs := []string{}
	for _, ip := range ipFamilyService.Spec.ClusterIPs {
		// get the vCluster service ip
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return "", fmt.Errorf("failed to parse service %s cluster IP: %v", ipFamilyService.Name, ip)
		}

		// get the prefix
		serviceCIDRs = append(serviceCIDRs, DetectPrefix(parsedIP, isInRange))
	}
	if len(serviceCIDRs) == 0 {
		return "", fmt.Errorf("no service cidrs found")
	}

	// join the service cidrs
	serviceCIDR := strings.Join(serviceCIDRs, ",")

	// set annotation on the vCluster service
	vClusterService.Annotations[ServiceCIDRAnnotation] = serviceCIDR
	_, err = client.CoreV1().Services(vClusterNamespace).Update(ctx, vClusterService, metav1.UpdateOptions{})
	if err != nil {
		// retry on conflict
		if kerrors.IsConflict(err) {
			time.Sleep(time.Second)
			klog.Warningf("failed to update vCluster service %s: %v, will retry", vClusterServiceName, err)
			return GetServiceCIDR(ctx, vConfig, client, vClusterServiceName, vClusterNamespace)
		}

		return "", fmt.Errorf("failed to update vCluster service %s: %w", vClusterServiceName, err)
	}

	return serviceCIDR, nil
}

// DetectPrefix inspects A to pick v4 vs v6,
// then runs the corresponding binary search.
func DetectPrefix(A net.IP, isInRange func(net.IP) bool) string {
	// ipv4?
	if A.To4() != nil {
		k := detectIPv4Prefix(A, isInRange)
		base := A.Mask(net.CIDRMask(k, 32))
		return fmt.Sprintf("%s/%d", base.String(), k)
	}

	// ipv6?
	k := detectIPv6Prefix(A, isInRange)
	base := A.Mask(net.CIDRMask(k, 128))
	return fmt.Sprintf("%s/%d", base.String(), k)
}

// IPv4 helpers --------------------------------------------------

func ipToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

// detectIPv4Prefix finds the minimal prefix k ∈ [1..32] around A
// using ≤5 queries, skipping the reserved network and broadcast addresses.
func detectIPv4Prefix(A net.IP, isInRange func(net.IP) bool) int {
	a := ipToUint32(A)
	low, high := 1, 32

	for low < high {
		mid := (low + high) / 2
		mask := uint32(0xFFFFFFFF) << (32 - mid)
		netBase := a & mask
		bcast := netBase | ^mask

		var testLow, testHigh uint32
		if mid == 32 {
			// /32: only one address
			testLow, testHigh = netBase, netBase
		} else {
			// skip network (netBase) and broadcast (bcast)
			testLow, testHigh = netBase+1, bcast-1
		}

		if isInRange(uint32ToIP(testLow)) && isInRange(uint32ToIP(testHigh)) {
			high = mid
		} else {
			low = mid + 1
		}
	}

	return low
}

// IPv6 helpers --------------------------------------------------

func toBigInt(ip net.IP) *big.Int {
	ip16 := ip.To16()
	if ip16 == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(ip16)
}

func toIP(i *big.Int) net.IP {
	b := i.Bytes()
	if len(b) < 16 {
		b = bytes.Join([][]byte{make([]byte, 16-len(b)), b}, nil)
	}
	return net.IP(b)
}

// detectIPv6Prefix finds the minimal prefix k ∈ [0..128], skipping
// the reserved network and (IPv4‐style) broadcast edges.
func detectIPv6Prefix(A net.IP, isInRange func(net.IP) bool) int {
	a := toBigInt(A)
	one := big.NewInt(1)
	allOnes := new(big.Int).Sub(new(big.Int).Lsh(one, 128), one)

	low, high := 0, 128
	for low < high {
		mid := (low + high) / 2

		// build mask = allOnes << (128-mid)
		mask := new(big.Int).Lsh(allOnes, uint(128-mid))
		netBase := new(big.Int).And(a, mask)

		// hostMask = ~mask & allOnes
		hostMask := new(big.Int).And(new(big.Int).Xor(mask, allOnes), allOnes)
		bcast := new(big.Int).Or(netBase, hostMask)

		var firstHost, lastHost *big.Int
		if mid == 128 {
			firstHost, lastHost = new(big.Int).Set(netBase), new(big.Int).Set(netBase)
		} else {
			firstHost = new(big.Int).Add(netBase, one)
			lastHost = new(big.Int).Sub(bcast, one)
		}

		if isInRange(toIP(firstHost)) && isInRange(toIP(lastHost)) {
			high = mid
		} else {
			low = mid + 1
		}
	}
	return low
}

// getIPFamilyService tries to create a dual stack service to use when determining the service CIDR. Currently, this
// tries a service with ipFamilyPolicy == PreferDualStack first, and falls back to SingleStack. It may not be necessary
// to try and fallback, but keeping in case previous versions of Kubernetes fail if dual stack is not configured.
func getIPFamilyService(ctx context.Context, client kubernetes.Interface, vClusterNamespace string) (*corev1.Service, error) {
	var errs []error

	for _, policy := range []corev1.IPFamilyPolicy{
		corev1.IPFamilyPolicyPreferDualStack,
		corev1.IPFamilyPolicySingleStack,
	} {
		if service, err := tryIPFamilyService(ctx, client, vClusterNamespace, policy); err != nil {
			errs = append(errs, err)
		} else if service != nil {
			return service, nil
		}
	}

	return nil, fmt.Errorf("failed to create service: %w", errors.Join(errs...))
}

func tryIPFamilyService(ctx context.Context, client kubernetes.Interface, vClusterNamespace string, ipFamilyPolicy corev1.IPFamilyPolicy) (*corev1.Service, error) {
	testService, err := client.CoreV1().Services(vClusterNamespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-service-delete-me-",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			IPFamilyPolicy: &ipFamilyPolicy,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		klog.Warningf("failed to create service with ipFamilyPolicy %s: %v", ipFamilyPolicy, err)
		return nil, fmt.Errorf("create service: %w", err)
	}

	defer func() {
		err := client.CoreV1().Services(vClusterNamespace).Delete(ctx, testService.GetName(), metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			klog.Warningf("failed to delete ipFamilyPolicy service %s: %v", testService.GetName(), err)
		}
	}()

	// Return immediately if cluster ips are already assigned.
	if len(testService.Spec.ClusterIPs) > 0 {
		return testService, nil
	}

	// Wait for cluster IPs if not assigned
	var ipAssignedService *corev1.Service
	if err := wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, time.Minute, true, func(ctx context.Context) (bool, error) {
		var err error
		ipAssignedService, err = client.CoreV1().Services(vClusterNamespace).Get(ctx, testService.GetName(), metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if len(ipAssignedService.Spec.ClusterIPs) > 0 {
			return true, nil
		}

		return false, nil
	}); err != nil {
		return nil, fmt.Errorf("wait for clusterIPs: %w", err)
	}
	return ipAssignedService, nil
}
