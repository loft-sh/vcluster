package assets

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/util/version"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
)

// Enumeration of supported kubernetes distros
const (
	k8s = "k8s"
	k3s = "k3s"
	k0s = "k0s"
)

var usage = fmt.Sprintf(`Usage:
  go run -mod vendor ./hack/assets/cmd/main.go [v]X.Y.Z [--latest] [--optional]
  go run -mod vendor ./hack/assets/cmd/main.go [v]X.Y.Z [--kubernetes-distro <%s>] [--kubernetes-version X.Y.Z]
  go run -mod vendor ./hack/assets/cmd/main.go --list-distros
  go run -mod vendor ./hack/assets/cmd/main.go --list-versions`,
	strings.Join(GetSupportedDistros(), " | "))

// Main is the entrypoint for the assets command
func Main() {
	listDistros := pflag.Bool("list-distros", false, "Only the list of supported Kubernetes distros is returned")
	listVersions := pflag.Bool("list-versions", false, "Only the list of supported Kubernetes versions is returned")
	latest := pflag.Bool("latest", false, "Only the latest image of each group is returned")
	optional := pflag.Bool("optional", false, "Include all images except the latest")

	k8sSupportedVersions := GetSupportedKubernetesVersions()
	kubernetesDistro := pflag.String("kubernetes-distro", "", fmt.Sprintf("The Kubernetes distro to include in the list (accepted values: %s)", strings.Join(GetSupportedDistros(), ", ")))
	kubernetesVersion := pflag.String("kubernetes-version", "", fmt.Sprintf("The Kubernetes version to include in the list (accepted values: %s)", strings.Join(k8sSupportedVersions, ", ")))
	pflag.Parse()

	if *listDistros && *listVersions {
		fmt.Println("Flags --list-distros and --list-versions are not compatible")
		os.Exit(1)
	}

	if *listDistros {
		for _, distro := range GetSupportedDistros() {
			fmt.Println(distro)
		}
		os.Exit(0)
	}

	if *listVersions {
		for _, v := range k8sSupportedVersions {
			fmt.Println(v)
		}
		os.Exit(0)
	}

	if pflag.NArg() < 1 {
		fmt.Println(usage)
		os.Exit(1)
	}

	if *kubernetesDistro != "" && !slices.Contains(GetSupportedDistros(), *kubernetesDistro) {
		fmt.Printf("Invalid value for --kubernetes-distro. Accepted values are: %s", strings.Join(GetSupportedDistros(), ", "))
		os.Exit(1)
	}

	if *kubernetesVersion != "" && !slices.Contains(k8sSupportedVersions, *kubernetesVersion) {
		fmt.Printf("Invalid value for --kubernetes-version. Accepted values are: %s", strings.Join(k8sSupportedVersions, ", "))
		os.Exit(1)
	}

	if *latest && *kubernetesVersion != "" {
		fmt.Println("Flags --latest and --kubernetes-version are not compatible")
		os.Exit(1)
	}

	if *latest && *optional {
		fmt.Println("Flags --latest and --optional are not compatible")
		os.Exit(1)
	}

	cleanTag := strings.TrimLeft(pflag.Arg(0), "v")

	images := GetImages(cleanTag, *latest, *optional, *kubernetesVersion, *kubernetesDistro)
	for _, img := range images {
		fmt.Println(img)
	}
}

// GetSupportedDistros returns a list of supported Kubernetes distros
func GetSupportedDistros() []string {
	return []string{k8s, k3s, k0s}
}

// GetImages returns a list of images based on the given parameters
func GetImages(cleanTag string, latest bool, optional bool, kubernetesVersion string, kubernetesDistro string) []string {
	images := GetVclusterImages(latest, optional, cleanTag)
	images = UniqueAppend(images,
		GetImageList(latest, optional, kubernetesVersion, GetVclusterDependencyImageMaps(kubernetesDistro))...,
	)
	return images
}

// GetSupportedKubernetesVersions returns a list of supported Kubernetes versions
func GetSupportedKubernetesVersions() []string {
	k8sSupportedVersions := maps.Keys(vclusterconfig.K8SVersionMap)
	slices.SortFunc(k8sSupportedVersions, versionsDescCmp)
	return k8sSupportedVersions
}

// GetImageList returns a list of images based on the given groups
// If latest is true, only the latest image of each group is returned
// If kubernetesVersion is specified, only the images matching the version are returned
func GetImageList(latest bool, optional bool, kubernetesVersion string, groups []map[string]string) []string {
	selectedImages := make([]string, 0, len(groups))
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		if kubernetesVersion != "" {
			if img, ok := g[kubernetesVersion]; ok {
				selectedImages = append(selectedImages, img)
			}
			continue
		}
		sortedImages := slices.Compact(getSortedDescValues(g))
		if latest {
			selectedImages = append(selectedImages, sortedImages[0])
			continue
		}
		if optional {
			selectedImages = append(selectedImages, sortedImages[1:]...)
			continue
		}
		selectedImages = append(selectedImages, sortedImages[:]...)
	}
	return selectedImages
}

// GetVclusterImages returns a list of vcluster images
func GetVclusterImages(latest, optional bool, cleanTag string) []string {
	images := []string{"ghcr.io/loft-sh/vcluster-oss:" + cleanTag}
	if !optional {
		if latest {
			images = nil
		}
		images = append(images,
			"ghcr.io/loft-sh/vcluster-pro:"+cleanTag,
			config.DefaultHostsRewriteImage,
		)
	}
	return images
}

// GetVclusterDependencyImageMaps returns a list of maps containing vcluster image versions
func GetVclusterDependencyImageMaps(distro string) []map[string]string {
	var ret []map[string]string
	switch distro {
	case k8s:
		ret = append(ret,
			vclusterconfig.K8SVersionMap,
			vclusterconfig.K8SEtcdVersionMap)
	case k3s:
		ret = append(ret, vclusterconfig.K3SVersionMap)
	case k0s:
		ret = append(ret, vclusterconfig.K0SVersionMap)
	default: // All distros
		ret = append(ret,
			vclusterconfig.K8SVersionMap,
			vclusterconfig.K8SEtcdVersionMap,
			vclusterconfig.K3SVersionMap,
		)
	}
	ret = append(ret, constants.CoreDNSVersionMap)
	return ret
}

// UniqueAppend Appends unique elements to the slice
func UniqueAppend(slice []string, elem ...string) []string {
	for _, e := range elem {
		if !slices.Contains(slice, e) {
			slice = append(slice, e)
		}
	}
	return slice
}

// Gets sorted slice of images in descending order
func getSortedDescValues(versionImageMap map[string]string) []string {
	versions := maps.Keys(versionImageMap)
	slices.SortFunc(versions, versionsDescCmp)
	images := make([]string, len(versions))
	for i, v := range versions {
		images[i] = versionImageMap[v]
	}
	return images
}

// Comparison function for versions in descending order
func versionsDescCmp(X, Y string) int {
	if version.MustParse(X).GreaterThan(version.MustParse(Y)) {
		return -1
	}
	return 1
}
