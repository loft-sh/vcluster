package main

import (
	"fmt"
	"os"
	"strings"

	syncercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"

	"github.com/loft-sh/vcluster/pkg/constants"

	"github.com/loft-sh/utils/pkg/helm/values"
)

func main() {
	images := []string{}

	// loft
	images = append(images, "loftsh/vcluster:"+cleanTag(os.Args[1]))
	images = append(images, syncercontext.DefaultHostsRewriteImage)

	// loop over k3s versions
	for _, image := range values.K3SVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}

	// loop over k0s versions
	for _, image := range values.K0SVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}

	// loop over k8s versions
	for _, image := range values.K8SAPIVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}
	for _, image := range values.K8SControllerVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}
	for _, image := range values.K8SEtcdVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}

	// loop over core-dns versions
	for _, image := range constants.CoreDNSVersionMap {
		if contains(images, image) {
			continue
		}

		images = append(images, image)
	}

	fmt.Print(strings.Join(images, "\n") + "\n")
}

func contains(a []string, str string) bool {
	for _, s := range a {
		if s == str {
			return true
		}
	}
	return false
}

func cleanTag(tag string) string {
	if len(tag) > 0 && tag[0] == 'v' {
		return tag[1:]
	}

	return tag
}
