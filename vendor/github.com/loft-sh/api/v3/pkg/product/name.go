package product

import (
	"fmt"
	"os"
	"sync"

	"github.com/loft-sh/admin-apis/pkg/features"
	"k8s.io/klog/v2"
)

// Product is the global variable to be set at build time
var productName string = string(features.Loft)
var once sync.Once

func loadProductVar() {
	productEnv := os.Getenv("PRODUCT")
	if productEnv == string(features.DevPodPro) {
		productName = string(features.DevPodPro)
	} else if productEnv == string(features.VClusterPro) {
		productName = string(features.VClusterPro)
	} else if productEnv == string(features.Loft) {
		productName = string(features.Loft)
	} else if productEnv != "" {
		klog.TODO().Error(fmt.Errorf("unrecognized product %s", productEnv), "error parsing product", "product", productEnv)
	}
}

func Name() features.ProductName {
	once.Do(loadProductVar)
	return features.ProductName(productName)
}

// Name returns the name of the product
func DisplayName() string {
	loftDisplayName := "Loft"

	switch Name() {
	case features.DevPodPro:
		return "DevPod Pro"
	case features.VClusterPro:
		return "vCluster.Pro"
	case features.Loft:
	}

	return loftDisplayName
}
