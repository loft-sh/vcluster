package translate

import "fmt"

// GetLoadBalancerSVCName retrieves the service name if service name is set to type LoadBalancer.
// A separate service is created in this case so as to expose only the apiserver and not the kubelet port
func GetLoadBalancerSVCName(serviceName string) string {
	return fmt.Sprintf("%s-lb", serviceName)
}
