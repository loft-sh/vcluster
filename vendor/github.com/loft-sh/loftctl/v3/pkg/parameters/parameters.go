package parameters

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/pkg/clihelper"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/survey"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ParametersFile struct {
	Parameters map[string]interface{} `json:"parameters"`
}

type AppFile struct {
	Apps []AppParameters `json:"apps,omitempty"`
}

type AppParameters struct {
	Name       string                 `json:"name,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
}

type NamespacedApp struct {
	App       *managementv1.App
	Namespace string
}

type NamespacedAppWithParameters struct {
	App        *managementv1.App
	Namespace  string
	Parameters string
}

func SetDeepValue(parameters interface{}, path string, value interface{}) {
	if parameters == nil {
		return
	}

	pathSegments := strings.Split(path, ".")
	switch t := parameters.(type) {
	case map[string]interface{}:
		if len(pathSegments) == 1 {
			t[pathSegments[0]] = value
			return
		}

		_, ok := t[pathSegments[0]]
		if !ok {
			t[pathSegments[0]] = map[string]interface{}{}
		}

		SetDeepValue(t[pathSegments[0]], strings.Join(pathSegments[1:], "."), value)
	}
}

func GetDeepValue(parameters interface{}, path string) interface{} {
	if parameters == nil {
		return nil
	}

	pathSegments := strings.Split(path, ".")
	switch t := parameters.(type) {
	case map[string]interface{}:
		val, ok := t[pathSegments[0]]
		if !ok {
			return nil
		} else if len(pathSegments) == 1 {
			return val
		}

		return GetDeepValue(val, strings.Join(pathSegments[1:], "."))
	case []interface{}:
		index, err := strconv.Atoi(pathSegments[0])
		if err != nil {
			return nil
		} else if index < 0 || index >= len(t) {
			return nil
		}

		val := t[index]
		if len(pathSegments) == 1 {
			return val
		}

		return GetDeepValue(val, strings.Join(pathSegments[1:], "."))
	}

	return nil
}

func ResolveTemplateParameters(set []string, parameters []storagev1.AppParameter, fileName string) (string, error) {
	var parametersFile map[string]interface{}
	if fileName != "" {
		out, err := os.ReadFile(fileName)
		if err != nil {
			return "", errors.Wrap(err, "read parameters file")
		}

		parametersFile = map[string]interface{}{}
		err = yaml.Unmarshal(out, &parametersFile)
		if err != nil {
			return "", errors.Wrap(err, "parse parameters file")
		}
	}

	return fillParameters(parameters, set, parametersFile)
}

func ResolveAppParameters(apps []NamespacedApp, appFilename string, log log.Logger) ([]NamespacedAppWithParameters, error) {
	var appFile *AppFile
	if appFilename != "" {
		out, err := os.ReadFile(appFilename)
		if err != nil {
			return nil, errors.Wrap(err, "read parameters file")
		}

		appFile = &AppFile{}
		err = yaml.Unmarshal(out, appFile)
		if err != nil {
			return nil, errors.Wrap(err, "parse parameters file")
		}
	}

	ret := []NamespacedAppWithParameters{}
	for _, app := range apps {
		if len(app.App.Spec.Parameters) == 0 {
			ret = append(ret, NamespacedAppWithParameters{
				App:       app.App,
				Namespace: app.Namespace,
			})
			continue
		}

		if appFile != nil {
			parameters, err := getParametersInAppFile(app.App, appFile)
			if err != nil {
				return nil, err
			}

			ret = append(ret, NamespacedAppWithParameters{
				App:        app.App,
				Namespace:  app.Namespace,
				Parameters: parameters,
			})
			continue
		}

		log.WriteString(logrus.InfoLevel, "\n")
		if app.Namespace != "" {
			log.Infof("Please specify parameters for app %s in namespace %s", clihelper.GetDisplayName(app.App.Name, app.App.Spec.DisplayName), app.Namespace)
		} else {
			log.Infof("Please specify parameters for app %s", clihelper.GetDisplayName(app.App.Name, app.App.Spec.DisplayName))
		}

		parameters := map[string]interface{}{}
		for _, parameter := range app.App.Spec.Parameters {
			question := parameter.Label
			if parameter.Required {
				question += " (Required)"
			}

			for {
				value, err := log.Question(&survey.QuestionOptions{
					Question:     question,
					DefaultValue: parameter.DefaultValue,
					Options:      parameter.Options,
					IsPassword:   parameter.Type == "password",
				})
				if err != nil {
					return nil, err
				}

				outVal, err := VerifyValue(value, parameter)
				if err != nil {
					log.Errorf(err.Error())
					continue
				}

				SetDeepValue(parameters, parameter.Variable, outVal)
				break
			}
		}

		out, err := yaml.Marshal(parameters)
		if err != nil {
			return nil, errors.Wrapf(err, "marshal app %s parameters", clihelper.GetDisplayName(app.App.Name, app.App.Spec.DisplayName))
		}
		ret = append(ret, NamespacedAppWithParameters{
			App:        app.App,
			Namespace:  app.Namespace,
			Parameters: string(out),
		})
	}

	return ret, nil
}

func VerifyValue(value string, parameter storagev1.AppParameter) (interface{}, error) {
	switch parameter.Type {
	case "":
		fallthrough
	case "password":
		fallthrough
	case "string":
		fallthrough
	case "multiline":
		if parameter.DefaultValue != "" && value == "" {
			value = parameter.DefaultValue
		}

		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}
		for _, option := range parameter.Options {
			if option == value {
				return value, nil
			}
		}
		if parameter.Validation != "" {
			regEx, err := regexp.Compile(parameter.Validation)
			if err != nil {
				return nil, errors.Wrap(err, "compile validation regex "+parameter.Validation)
			}

			if !regEx.MatchString(value) {
				return nil, fmt.Errorf("parameter %s (%s) needs to match regex %s", parameter.Label, parameter.Variable, parameter.Validation)
			}
		}
		if parameter.Invalidation != "" {
			regEx, err := regexp.Compile(parameter.Invalidation)
			if err != nil {
				return nil, errors.Wrap(err, "compile invalidation regex "+parameter.Invalidation)
			}

			if regEx.MatchString(value) {
				return nil, fmt.Errorf("parameter %s (%s) cannot match regex %s", parameter.Label, parameter.Variable, parameter.Invalidation)
			}
		}

		return value, nil
	case "boolean":
		if parameter.DefaultValue != "" && value == "" {
			boolValue, err := strconv.ParseBool(parameter.DefaultValue)
			if err != nil {
				return nil, errors.Wrapf(err, "parse default value for parameter %s (%s)", parameter.Label, parameter.Variable)
			}

			return boolValue, nil
		}
		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return nil, errors.Wrapf(err, "parse value for parameter %s (%s)", parameter.Label, parameter.Variable)
		}
		return boolValue, nil
	case "number":
		if parameter.DefaultValue != "" && value == "" {
			intValue, err := strconv.Atoi(parameter.DefaultValue)
			if err != nil {
				return nil, errors.Wrapf(err, "parse default value for parameter %s (%s)", parameter.Label, parameter.Variable)
			}

			return intValue, nil
		}
		if parameter.Required && value == "" {
			return nil, fmt.Errorf("parameter %s (%s) is required", parameter.Label, parameter.Variable)
		}
		num, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.Wrapf(err, "parse value for parameter %s (%s)", parameter.Label, parameter.Variable)
		}
		if parameter.Min != nil && num < *parameter.Min {
			return nil, fmt.Errorf("parameter %s (%s) cannot be smaller than %d", parameter.Label, parameter.Variable, *parameter.Min)
		}
		if parameter.Max != nil && num > *parameter.Max {
			return nil, fmt.Errorf("parameter %s (%s) cannot be greater than %d", parameter.Label, parameter.Variable, *parameter.Max)
		}

		return num, nil
	}

	return nil, fmt.Errorf("unrecognized type %s for parameter %s (%s)", parameter.Type, parameter.Label, parameter.Variable)
}

func getParametersInAppFile(appObj *managementv1.App, appFile *AppFile) (string, error) {
	if appFile == nil {
		return "", nil
	}

	for _, app := range appFile.Apps {
		if app.Name == appObj.Name {
			return fillParameters(appObj.Spec.Parameters, nil, app.Parameters)
		}
	}

	return "", fmt.Errorf("couldn't find app %s (%s) in provided parameters file", clihelper.GetDisplayName(appObj.Name, appObj.Spec.DisplayName), appObj.Name)
}

func fillParameters(parameters []storagev1.AppParameter, set []string, values map[string]interface{}) (string, error) {
	if values == nil {
		values = map[string]interface{}{}
	}

	// parse set array
	setMap, err := parseSet(parameters, set)
	if err != nil {
		return "", err
	}

	// apply parameters
	for _, parameter := range parameters {
		strVal, ok := setMap[parameter.Variable]
		if !ok {
			val := GetDeepValue(values, parameter.Variable)
			if val != nil {
				switch t := val.(type) {
				case string:
					strVal = t
				case int:
					strVal = strconv.Itoa(t)
				case bool:
					strVal = strconv.FormatBool(t)
				default:
					return "", fmt.Errorf("unrecognized type for parameter %s (%s) in file: %v", parameter.Label, parameter.Variable, t)
				}
			}
		}

		outVal, err := VerifyValue(strVal, parameter)
		if err != nil {
			return "", errors.Wrap(err, "validate parameters")
		}

		SetDeepValue(values, parameter.Variable, outVal)
	}

	out, err := yaml.Marshal(values)
	if err != nil {
		return "", errors.Wrap(err, "marshal parameters")
	}

	return string(out), nil
}

func parseSet(parameters []storagev1.AppParameter, set []string) (map[string]string, error) {
	setValues := map[string]string{}
	for _, s := range set {
		splitted := strings.Split(s, "=")
		if len(splitted) <= 1 {
			return nil, fmt.Errorf("error parsing --set %s: need parameter=value format", s)
		}

		key := splitted[0]
		value := strings.Join(splitted[1:], "=")
		found := false
		for _, parameter := range parameters {
			if parameter.Variable == key {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("parameter %s doesn't exist on template", key)
		}

		setValues[key] = value
	}

	return setValues, nil
}
