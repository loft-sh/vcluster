/*
Copyright 2017 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certs

import (
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"
)

// SetInitDynamicDefaults checks and sets configuration values for the InitConfiguration object
func SetInitDynamicDefaults() (*InitConfiguration, error) {
	cfg := &InitConfiguration{}

	if err := SetAPIEndpointDynamicDefaults(&cfg.LocalAPIEndpoint); err != nil {
		return nil, err
	}
	err := SetClusterDynamicDefaults(&cfg.ClusterConfiguration, &cfg.LocalAPIEndpoint, &cfg.NodeRegistration)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// SetAPIEndpointDynamicDefaults checks and sets configuration values for the APIEndpoint object
func SetAPIEndpointDynamicDefaults(cfg *APIEndpoint) error {
	// validate cfg.API.AdvertiseAddress.
	addressIP := net.ParseIP(cfg.AdvertiseAddress)
	if addressIP == nil && cfg.AdvertiseAddress != "" {
		return errors.Errorf("couldn't use \"%s\" as \"apiserver-advertise-address\", must be ipv4 or ipv6 address", cfg.AdvertiseAddress)
	}

	// kubeadm allows users to specify address=Loopback as a selector for global unicast IP address that can be found on loopback interface.
	// e.g. This is required for network setups where default routes are present, but network interfaces use only link-local addresses (e.g. as described in RFC5549).
	if addressIP.IsLoopback() {
		loopbackIP, err := netutil.ChooseBindAddressForInterface(netutil.LoopbackInterfaceName)
		if err != nil {
			return err
		}
		if loopbackIP != nil {
			klog.V(4).Infof("Found active IP %v on loopback interface", loopbackIP.String())
			cfg.AdvertiseAddress = loopbackIP.String()
			return nil
		}
		return errors.New("unable to resolve link-local addresses")
	}

	// This is the same logic as the API Server uses, except that if no interface is found the address is set to 0.0.0.0, which is invalid and cannot be used
	// for bootstrapping a cluster.
	ip, err := ChooseAPIServerBindAddress(addressIP)
	if err != nil {
		return err
	}
	cfg.AdvertiseAddress = ip.String()

	return nil
}

// ChooseAPIServerBindAddress is a wrapper for netutil.ResolveBindAddress that also handles
// the case where no default routes were found and an IP for the API server could not be obtained.
func ChooseAPIServerBindAddress(bindAddress net.IP) (net.IP, error) {
	ip, err := netutil.ResolveBindAddress(bindAddress)
	if err != nil {
		if netutil.IsNoRoutesError(err) {
			klog.Warningf("WARNING: could not obtain a bind address for the API Server: %v; using: %s", err, DefaultAPIServerBindAddress)
			defaultIP := net.ParseIP(DefaultAPIServerBindAddress)
			if defaultIP == nil {
				return nil, errors.Errorf("cannot parse default IP address: %s", DefaultAPIServerBindAddress)
			}
			return defaultIP, nil
		}
		return nil, err
	}
	if bindAddress != nil && !bindAddress.IsUnspecified() && !reflect.DeepEqual(ip, bindAddress) {
		klog.Warningf("WARNING: overriding requested API server bind address: requested %q, actual %q", bindAddress, ip)
	}
	return ip, nil
}

// SetClusterDynamicDefaults checks and sets values for the ClusterConfiguration object
func SetClusterDynamicDefaults(cfg *ClusterConfiguration, localAPIEndpoint *APIEndpoint, _ *NodeRegistrationOptions) error {
	// If ControlPlaneEndpoint is specified without a port number defaults it to
	// the bindPort number of the APIEndpoint.
	// This will allow join of additional control plane instances with different bindPort number
	if cfg.ControlPlaneEndpoint != "" {
		host, port, err := ParseHostPort(cfg.ControlPlaneEndpoint)
		if err != nil {
			return err
		}
		if port == "" {
			cfg.ControlPlaneEndpoint = net.JoinHostPort(host, strconv.FormatInt(int64(localAPIEndpoint.BindPort), 10))
		}
	}

	// Downcase SANs. Some domain names (like ELBs) have capitals in them.
	LowercaseSANs(cfg.APIServer.CertSANs)
	return nil
}

// LowercaseSANs can be used to force all SANs to be lowercase so it passes IsDNS1123Subdomain
func LowercaseSANs(sans []string) {
	for i, san := range sans {
		lowercase := strings.ToLower(san)
		if lowercase != san {
			klog.V(1).Infof("lowercasing SAN %q to %q", san, lowercase)
			sans[i] = lowercase
		}
	}
}
