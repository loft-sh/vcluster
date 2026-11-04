package list

import (
	"encoding/json"

	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/sirupsen/logrus"
)

func printJSON(logger log.Logger, value []map[string]string) error {
	bytes, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}
	logger.WriteString(logrus.InfoLevel, string(bytes)+"\n")
	return nil
}

// PrintData is a generic function that prints data in different formats (JSON or table).
// It takes a logger for output, an output type (json/table), a list of headers, a slice of items,
// and a function to extract values from each item.
func PrintData[T any](logger log.Logger, outputType string, headers []string, items []T, getValues func(T) []string) error {
	var err error
	switch outputType {
	case "json":
		// Convert items into a map using headers and value extractor function
		itemsMap := toMap(headers, items, getValues)

		err = printJSON(logger, itemsMap)
	case "table", "default":
		// Convert items into a 2D slice of values
		values := toValues(items, getValues)

		table.PrintTable(logger, headers, values)
	}
	return err
}

func toValues[T any](items []T, getValues func(T) []string) [][]string {
	values := make([][]string, len(items))
	for i, item := range items {
		values[i] = getValues(item)
	}
	return values
}

// toMap converts a slice of items into a slice of maps, where each map represents an item with
// keys from headers and values extracted using the getValues function.
func toMap[T any](headers []string, items []T, getValues func(T) []string) []map[string]string {
	var projectsMap []map[string]string
	for _, item := range items {
		values := getValues(item)
		dataMap := make(map[string]string)
		for i, header := range headers {
			if i < len(values) { // Ensure values exist for the given index
				dataMap[header] = values[i]
			}
		}
		projectsMap = append(projectsMap, dataMap)
	}
	return projectsMap
}
