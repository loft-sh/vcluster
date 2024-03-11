package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/vcluster/config"
)

const OutFile = "chart/values.schema.json"

// Run executes the command logic
func main() {
	generatedSchema, err := generateSchema(&config.Config{})
	if err != nil {
		panic(err)
	}

	transformMapProperties(generatedSchema)
	modifySchema(generatedSchema, cleanUp)
	err = writeSchema(generatedSchema, OutFile)
	if err != nil {
		panic(err)
	}
}

func generateSchema(configInstance interface{}) (*jsonschema.Schema, error) {
	r := new(jsonschema.Reflector)
	r.RequiredFromJSONSchemaTags = true
	r.BaseSchemaID = "https://vcluster.com/schemas"
	r.ExpandedStruct = true

	commentMap := map[string]string{}

	err := jsonschema.ExtractGoComments("github.com/loft-sh/vcluster", "config", commentMap)
	if err != nil {
		return nil, err
	}

	r.CommentMap = commentMap

	return r.Reflect(configInstance), nil
}

func writeSchema(schema *jsonschema.Schema, schemaFile string) error {
	prefix := ""
	schemaString, err := json.MarshalIndent(schema, prefix, "  ")
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(schemaFile), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(schemaFile, schemaString, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func modifySchema(schema *jsonschema.Schema, visitors ...func(s *jsonschema.Schema)) {
	// Apply visitors
	if len(visitors) > 0 {
		for _, visitor := range visitors {
			walk(schema, visitor)
		}
	}
}

func transformMapProperties(s *jsonschema.Schema) {
	plugins, ok := s.Properties.Get("plugins")
	if ok {
		plugins.AnyOf = modifyAnyOf(plugins)
		plugins.PatternProperties = nil
	}

	plugin, ok := s.Properties.Get("plugin")
	if ok {
		plugin.AnyOf = modifyAnyOf(plugin)
		plugin.PatternProperties = nil
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

func cleanUp(s *jsonschema.Schema) {
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		s.Ref = ""
		s.Type = ""
		s.Items = nil
		s.PatternProperties = nil
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
