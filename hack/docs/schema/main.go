package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/vcluster/hack/docs/util"
	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

const jsonschemaFile = "vcluster-schema.json"
const openapiSchemaFile = "docs/schemas/config-openapi.json"

// Run executes the command logic
func main() {
	r := new(jsonschema.Reflector)
	r.AllowAdditionalProperties = true
	//r.PreferYAMLSchema = true
	r.RequiredFromJSONSchemaTags = false
	//r.YAMLEmbeddedStructs = false
	r.ExpandedStruct = true

	for base, pkgpath := range map[string]string{
		"github.com/loft-sh/vcluster":          "./pkg/config",
		"k8s.io/api/admissionregistration/v1":  "./vendor/k8s.io/api/admissionregistration/v1",
		"k8s.io/api/app/v1":                    "./vendor/k8s.io/api/apps/v1",
		"k8s.io/api/core/v1":                   "./vendor/k8s.io/api/core/v1",
		"k8s.io/api/networking/v1":             "./vendor/k8s.io/api/networking/v1",
		"k8s.io/api/rbac/v1":                   "./vendor/k8s.io/api/rbac/v1",
		"k8s.io/apimachinery/pkg/api/resource": "./vendor/k8s.io/apimachinery/pkg/api/resource",
		"k8s.io/apimachinery/pkg/apis/meta/v1": "./vendor/k8s.io/apimachinery/pkg/apis/meta/v1",
		"k8s.io/apiserver/pkg/apis/audit/v1":   "./vendor/k8s.io/apiserver/pkg/apis/audit/v1",
	} {
		err := r.AddGoComments(base, pkgpath)
		if err != nil {
			panic(err)
		}
	}

	util.PostProcessVendoredComments(r.CommentMap)

	//for name, _ := range r.CommentMap {
	//	if strings.HasPrefix(name, "github.com/loft-sh/vcluster") {
	//		continue
	//	}
	//
	//	parts := strings.Split(name, "/vendor/")
	//	if len(parts) > 1 {
	//		r.CommentMap[parts[1]] = r.CommentMap[name]
	//		delete(r.CommentMap, name)
	//	}
	//}

	//for name, value := range r.CommentMap {
	//	fmt.Println(name, ":", value)
	//}

	openapiSchema := r.Reflect(&config.Config{})
	genSchema(openapiSchema, openapiSchemaFile)

	jsonSchema := r.Reflect(&config.Config{})
	genSchema(jsonSchema, jsonschemaFile, CleanUp)
}

func genSchema(schema *jsonschema.Schema, schemaFile string, visitors ...func(s *jsonschema.Schema)) {
	isOpenAPISpec := schemaFile == openapiSchemaFile
	prefix := ""
	if isOpenAPISpec {
		prefix = "      "
	}

	//// vars
	//vars, ok := schema.Properties.Get("vars")
	//if ok {
	//	vars.(*jsonschema.Schema).AnyOf = modifyAnyOf(vars)
	//	vars.(*jsonschema.Schema).PatternProperties = nil
	//}
	//// pipelines
	//pipelines, ok := schema.Properties.Get("pipelines")
	//if ok {
	//	pipelines.(*jsonschema.Schema).AnyOf = modifyAnyOf(pipelines)
	//	pipelines.(*jsonschema.Schema).PatternProperties = nil
	//}
	//// commands
	//commands, ok := schema.Properties.Get("commands")
	//if ok {
	//	commands.(*jsonschema.Schema).AnyOf = modifyAnyOf(commands)
	//	commands.(*jsonschema.Schema).PatternProperties = nil
	//}

	// Apply visitors
	if len(visitors) > 0 {
		for _, visitor := range visitors {
			walk(schema, visitor)
		}
	}

	schemaJSON, err := json.MarshalIndent(schema, prefix, "  ")
	if err != nil {
		panic(err)
	}

	schemaString := string(schemaJSON)

	if isOpenAPISpec {
		schemaString = strings.ReplaceAll(schemaString, "#/$defs/", "#/definitions/Config/$defs/")

		schemaString = fmt.Sprintf(`{
	"swagger": "2.0",
	"info": {
		"version": "%s",
		"title": "vcluster.yaml"
	},
	"paths": {},
	"definitions": {
		"Config": %s
	}
}
`, latest.Version, schemaString)
	}

	err = os.MkdirAll(filepath.Dir(schemaFile), os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(schemaFile, []byte(schemaString), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func modifyAnyOf(field interface{}) []*jsonschema.Schema {
	return []*jsonschema.Schema{
		{
			Type: "object",
			PatternProperties: map[string]*jsonschema.Schema{
				".*": {
					Type: "string",
				},
			},
		},
		{
			Type:              "object",
			PatternProperties: field.(*jsonschema.Schema).PatternProperties,
		},
		{
			Type: "object",
		},
	}
}

func walk(schema *jsonschema.Schema, visit func(s *jsonschema.Schema)) {
	for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
		visit(pair.Value)
	}

	for _, definition := range schema.Definitions {
		for pair := definition.Properties.Oldest(); pair != nil; pair = pair.Next() {
			visit(pair.Value)
		}
	}
}

func CleanUp(s *jsonschema.Schema) {
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		s.Ref = ""
		s.Type = ""
		s.Items = nil
		s.PatternProperties = nil
	}
}
