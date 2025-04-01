package list

import (
	"encoding/json"

	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

func printJson(logger log.Logger, value []map[string]string) error {
	bytes, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}
	logger.WriteString(logrus.InfoLevel, string(bytes))
	return nil
}
