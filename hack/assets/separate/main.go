package main

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/coredns"
)

var (
	kubernetesDistro  = flag.String("kubernetes-distro", "", "The Kubernetes distro to include in the list")
	kubernetesVersion = flag.String("kubernetes-version", "", "The Kubernetes version to include in the image list")
	vClusterVersion   = flag.String("vcluster-version", "", "The vCluster version to include in the image list")
)

func main() {
	flag.Parse()

	assert(*kubernetesDistro != "", "-kubernetes-distro flag has to be set")
	assert(*kubernetesVersion != "", "-kubernetes-version flag has to be set")
	assert(*vClusterVersion != "", "-vcluster-version flag has to be set")

	images := []string{}

	// vCluster
	assert(cleanTag(*vClusterVersion) != "", "vCluster version does not contain a numnber")
	images = append(images, "ghcr.io/loft-sh/vcluster-pro:"+cleanTag(*vClusterVersion))
	images = append(images, "ghcr.io/loft-sh/vcluster-pro-fips:"+cleanTag(*vClusterVersion))
	images = append(images, config.DefaultHostsRewriteImage)

	var versionMap map[string]string
	switch *kubernetesDistro {
	case "k8s":
		versionMap = vclusterconfig.K8SAPIVersionMap
	case "k3s":
		versionMap = vclusterconfig.K3SVersionMap
	case "k0s":
		versionMap = vclusterconfig.K0SVersionMap
	}
	assert(versionMap != nil, "no version map found for kubernetes distro", *kubernetesDistro)

	// loop over version map
	kubernetesImage := versionMap[*kubernetesVersion]
	assert(len(kubernetesImage) > 0, "could not find kubernetes version for distro", *kubernetesDistro, *kubernetesVersion)
	images = append(images, kubernetesImage)

	if *kubernetesDistro == "k8s" {
		controllerManagerImage := vclusterconfig.K8SControllerVersionMap[*kubernetesVersion]
		assert(len(controllerManagerImage) > 0, "could not find controller manager image", *kubernetesVersion)
		images = append(images, controllerManagerImage)

		etcdImage := vclusterconfig.K8SEtcdVersionMap[*kubernetesVersion]
		assert(len(etcdImage) > 0, "could not find etcd image", *kubernetesVersion)
		images = append(images, etcdImage)
	}

	// loop over core-dns versions
	coreDNSImage := constants.CoreDNSVersionMap[*kubernetesVersion]
	assert(len(coreDNSImage) > 0, "could not find CoreDNS image", *kubernetesVersion)
	images = append(images, coreDNSImage)

	if !slices.Contains(images, coredns.DefaultImage) {
		images = append(images, coredns.DefaultImage)
	}

	fmt.Print(strings.Join(images, "\n") + "\n")
}

func cleanTag(tag string) string {
	if len(tag) > 0 && tag[0] == 'v' {
		return tag[1:]
	}

	return tag
}

func assert(condition bool, message ...string) {
	if !condition {
		panic(fmt.Sprintf("assert failed: %v", strings.Join(message, ", ")))
	}
}
