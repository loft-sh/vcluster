package config

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"

	"github.com/ghodss/yaml"
)

var (
	// ErrUnsupportedType is returned if the type is not implemented
	ErrUnsupportedType = errors.New("unsupported type")
)

func Diff(fromConfig *Config, toConfig *Config) (string, error) {
	// convert to map[string]interface{}
	fromRaw := map[string]interface{}{}
	err := convert(fromConfig, &fromRaw)
	if err != nil {
		return "", err
	}

	toRaw := map[string]interface{}{}
	err = convert(toConfig, &toRaw)
	if err != nil {
		return "", err
	}

	diffRaw := diff(fromRaw, toRaw)
	if diffRaw == nil {
		diffRaw = map[string]interface{}{}
	}

	out, err := yaml.Marshal(diffRaw)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func diff(from, to any) any {
	if reflect.DeepEqual(from, to) {
		return nil
	}

	switch fromType := from.(type) {
	case map[string]interface{}:
		toMap, ok := to.(map[string]interface{})
		if !ok {
			return to
		}

		retMap := map[string]interface{}{}

		// from -> to
		for k, fromValue := range fromType {
			toValue, ok := toMap[k]
			if !ok {
				switch fromValue.(type) {
				// if its a boolean, its true -> false
				case bool:
					retMap[k] = false
				// if its a string, its "something" -> ""
				case string:
					retMap[k] = ""
				// if its an int, its 3 -> 0
				case int:
					retMap[k] = 0
				}
			} else if !reflect.DeepEqual(fromValue, toValue) {
				switch fromValue.(type) {
				case map[string]interface{}:
					retMap[k] = diff(fromValue, toValue)
				default:
					retMap[k] = toValue
				}
			}
		}

		// to -> from
		for k, toValue := range toMap {
			_, ok := fromType[k]
			if !ok {
				retMap[k] = toValue
			}
		}

		return retMap
	default:
		return to
	}
}

func convert(from, to any) error {
	rawFrom, err := json.Marshal(from)
	if err != nil {
		return err
	}

	return json.Unmarshal(rawFrom, to)
}

type StrBool string

func (f *StrBool) UnmarshalJSON(data []byte) error {
	var jsonObj interface{}
	err := json.Unmarshal(data, &jsonObj)
	if err != nil {
		return err
	}
	switch obj := jsonObj.(type) {
	case string:
		*f = StrBool(obj)
		return nil
	case bool:
		*f = StrBool(strconv.FormatBool(obj))
		return nil
	}
	return ErrUnsupportedType
}

func (f *StrBool) MarshalJSON() ([]byte, error) {
	if *f == "true" {
		return []byte("true"), nil
	} else if *f == "false" {
		return []byte("false"), nil
	}

	return []byte("\"" + *f + "\""), nil
}
