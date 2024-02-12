package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/vcluster-values/helmvalues"
)

func main() {
	beforeFiles := map[string][]byte{}
	validate := len(os.Args) > 2 && os.Args[2] == "validate"
	if validate {
		for _, v := range []string{"k0s", "k3s", "k8s", "eks"} {
			helmSchemaBytes, err := os.ReadFile("charts/" + v + "/values.schema.json")
			if err != nil {
				fmt.Println(err)
				errorOut()
			}
			beforeFiles[v] = helmSchemaBytes
		}
	}
	for _, v := range []string{"k0s", "k3s", "k8s", "eks"} {
		helmSchemaFile, err := os.Create("charts/" + v + "/values.schema.json")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		enc := json.NewEncoder(helmSchemaFile)
		enc.SetIndent("", "    ")
		switch v {
		case "k0s":
			err = enc.Encode(jsonschema.Reflect(&helmvalues.K0s{}))
		case "k3s":
			err = enc.Encode(jsonschema.Reflect(&helmvalues.K3s{}))
		case "k8s":
			err = enc.Encode(jsonschema.Reflect(&helmvalues.K8s{}))
		case "eks":
			err = enc.Encode(jsonschema.Reflect(&helmvalues.K8s{}))
		}
		if err != nil {
			fmt.Println(err)
			errorOut()
		}
	}
	if validate {
		for _, v := range []string{"k0s", "k3s", "k8s", "eks"} {
			helmSchemaBytes, err := os.ReadFile("charts/" + v + "/values.schema.json")
			if err != nil {
				fmt.Println(err)
				errorOut()
			}
			if !slices.Equal(helmSchemaBytes, beforeFiles[v]) {
				errorOut()
			}
		}
	}
}

func errorOut() {
	fmt.Println("json schema is not up to date, please run 'just generate-json-schema'")
	os.Exit(1)
}
