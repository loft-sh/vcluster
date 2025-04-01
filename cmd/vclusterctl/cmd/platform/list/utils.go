package list

import (
	"encoding/json"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

func getJsonMap(keys []string, values [][]string) (results []map[string]string) {
	for _, value := range values {
		jsonMap := make(map[string]string)

		for i, key := range keys {
			jsonMap[key] = value[i]
		}
		results = append(results, jsonMap)
	}
	return results
}

func printJson(logger log.Logger, keys []string, values [][]string) error {
	result := getJsonMap(keys, values)
	bytes, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}
	logger.WriteString(logrus.InfoLevel, string(bytes))
	return nil
}
