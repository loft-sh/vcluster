package product

import (
	"fmt"
	"os"
	"sync"

	"k8s.io/klog/v2"
)

// Product is the global variable that
var product ProductType = Loft
var once sync.Once

func Product() ProductType {
	once.Do(loadProductVar)

	return product
}

func IsProduct(products ...ProductType) bool {
	once.Do(loadProductVar)

	for _, p := range products {
		if product == p {
			return true
		}
	}
	return false
}

type ProductType string

const (
	Loft        ProductType = "loft"
	DevPodPro   ProductType = "devpod-pro"
	VClusterPro ProductType = "vcluster-pro"
)

func loadProductVar() {
	productEnv := os.Getenv("PRODUCT")
	if productEnv == string(DevPodPro) {
		product = DevPodPro
	} else if productEnv == string(VClusterPro) {
		product = VClusterPro
	} else if productEnv == string(Loft) {
		product = Loft
	} else if productEnv != "" {
		klog.TODO().Error(fmt.Errorf("unrecognized product %s", productEnv), "error parsing product", "product", productEnv)
	}
}
