package product

import (
	"fmt"
	"os"
	"sync"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"k8s.io/klog/v2"
)

// Product is the global variable to be set at build time
var productName string = string(licenseapi.Loft)
var once sync.Once

func loadProductVar() {
	productEnv := os.Getenv("PRODUCT")
	if productEnv == string(licenseapi.DevPodPro) {
		productName = string(licenseapi.DevPodPro)
	} else if productEnv == string(licenseapi.VClusterPro) {
		productName = string(licenseapi.VClusterPro)
	} else if productEnv == string(licenseapi.Loft) {
		productName = string(licenseapi.Loft)
	} else if productEnv != "" {
		klog.TODO().Error(fmt.Errorf("unrecognized product %s", productEnv), "error parsing product", "product", productEnv)
	}
}

func Name() licenseapi.ProductName {
	once.Do(loadProductVar)
	return licenseapi.ProductName(productName)
}

// Name returns the name of the product
func DisplayName() string {
	loftDisplayName := "Loft"

	switch Name() {
	case licenseapi.DevPodPro:
		return "DevPod Pro"
	case licenseapi.VClusterPro:
		return "vCluster Platform"
	case licenseapi.Loft:
	}

	return loftDisplayName
}
