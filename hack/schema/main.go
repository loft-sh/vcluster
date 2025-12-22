package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/vcluster/config"
	"gopkg.in/yaml.v3"
)

const (
	OutFile       = "chart/values.schema.json"
	ValuesOutFile = "chart/values.yaml"
)
const (
	defsPrefix = "#/$defs/"
)

var SkipProperties = map[string]string{
	"EnableSwitch":              "*",
	"EnableSwitchWithTranslate": "enabled",
	"SyncAllResource":           "enabled",
	"DistroContainerEnabled":    "enabled",
	"EtcdDeployService":         "*",
	"EtcdDeployHeadlessService": "*",
	"LabelsAndAnnotations":      "*",
}

var SkipKeys = map[string]bool{
	"annotations": true,
	"labels":      true,
}

// Run executes the command logic
func main() {
	reflector, err := getReflector()
	if err != nil {
		panic(err)
	}

	generatedSchema := reflector.Reflect(&config.Config{})
	transformMapProperties(generatedSchema)
	modifySchema(generatedSchema, cleanUp)
	err = writeSchema(generatedSchema, OutFile)
	if err != nil {
		panic(err)
	}

	err = writeValues(generatedSchema)
	if err != nil {
		panic(err)
	}
}

func writeValues(schema *jsonschema.Schema) error {
	yamlNode := &yaml.Node{}
	err := yaml.Unmarshal([]byte(config.Values), yamlNode)
	if err != nil {
		return err
	}

	// traverse yaml nodes
	err = traverseNode(yamlNode, schema, schema.Definitions, 0)
	if err != nil {
		return fmt.Errorf("traverse node: %w", err)
	}

	b := &bytes.Buffer{}
	enc := yaml.NewEncoder(b)
	enc.SetIndent(2)
	err = enc.Encode(yamlNode)
	if err != nil {
		return err
	}

	err = os.WriteFile(ValuesOutFile, b.Bytes(), 0666)
	if err != nil {
		return err
	}

	return nil
}

func traverseNode(node *yaml.Node, schema *jsonschema.Schema, definitions jsonschema.Definitions, depth int) error {
	if node.Kind == yaml.MappingNode {
		// next nodes are key: value, key: value
		if len(node.Content)%2 != 0 {
			return fmt.Errorf("unexpected amount of children: %d", len(node.Content))
		}

		// loop over content
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1]

			// find properties
			properties := schema.Properties
			ref := strings.TrimPrefix(schema.Ref, defsPrefix)
			if ref != "" {
				refSchema, ok := definitions[ref]
				if ok {
					properties = refSchema.Properties
				}
			}
			if properties == nil || SkipProperties[ref] == key || SkipProperties[ref] == "*" || SkipKeys[key] {
				continue
			}

			// get property
			valueSchema, ok := properties.Get(key)
			if ok {
				// set comment
				node.Content[i].HeadComment = valueSchema.Description

				// add new line if property on level 0
				if i > 0 && depth < 2 {
					node.Content[i].HeadComment = "\n" + node.Content[i].HeadComment
				}

				// next node
				err := traverseNode(value, valueSchema, definitions, depth+1)
				if err != nil {
					return err
				}
			}
		}
	} else {
		for _, child := range node.Content {
			err := traverseNode(child, schema, definitions, depth)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getReflector() (*jsonschema.Reflector, error) {
	r := new(jsonschema.Reflector)
	r.RequiredFromJSONSchemaTags = true
	r.BaseSchemaID = "https://vcluster.com/schemas"
	r.ExpandedStruct = true

	commentMap := map[string]string{}
	err := jsonschema.ExtractGoComments("github.com/loft-sh/vcluster", "config", commentMap)
	if err != nil {
		return nil, err
	}

	err = jsonschema.ExtractGoComments("./", "vendor/github.com/loft-sh/api/v4/pkg/vclusterconfig", commentMap)
	if err != nil {
		return nil, err
	}

	for k, comment := range commentMap {
		if strings.Contains(comment, "<") || strings.Contains(comment, ">") {
			return nil, fmt.Errorf("comment for %s (%s) contains '<' or '>', please remove it because it will break docs generation", k, comment)
		}
	}

	renamedMap := make(map[string]string, len(commentMap))
	for k, v := range commentMap {
		newKey := strings.Replace(k, "vendor/github.com/loft-sh/api/v4/pkg/vclusterconfig.", "github.com/loft-sh/api/v4/pkg/vclusterconfig.", 1)
		renamedMap[newKey] = v
	}
	r.CommentMap = renamedMap

	return r, nil
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
