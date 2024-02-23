package main

import (
	"github.com/loft-sh/vcluster/hack/docs/util"
	"github.com/loft-sh/vcluster/pkg/config"
)

const configPartialBasePath = "docs/pages/configuration/_partials/"

func main() {
	repository := "github.com/loft-sh/vcluster"
	configGoPackage := "./pkg/config"
	versionedConfigBasePath := configPartialBasePath

	schema := util.GenerateSchema(config.Config{}, map[string]string{
		repository:                             configGoPackage,
		"k8s.io/api/admissionregistration/v1":  "./vendor/k8s.io/api/admissionregistration/v1",
		"k8s.io/api/app/v1":                    "./vendor/k8s.io/api/apps/v1",
		"k8s.io/api/core/v1":                   "./vendor/k8s.io/api/core/v1",
		"k8s.io/api/networking/v1":             "./vendor/k8s.io/api/networking/v1",
		"k8s.io/api/rbac/v1":                   "./vendor/k8s.io/api/rbac/v1",
		"k8s.io/apimachinery/pkg/api/resource": "./vendor/k8s.io/apimachinery/pkg/api/resource",
		"k8s.io/apimachinery/pkg/apis/meta/v1": "./vendor/k8s.io/apimachinery/pkg/apis/meta/v1",
		"k8s.io/apiserver/pkg/apis/audit/v1":   "./vendor/k8s.io/apiserver/pkg/apis/audit/v1",
	})
	util.GenerateReference(schema, versionedConfigBasePath)
}
