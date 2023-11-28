package telemetry

import (
	"encoding/json"
	"errors"
	"strconv"
)

var (
	// ErrUnsupportedType is returned if the type is not implemented
	ErrUnsupportedType = errors.New("unsupported type")
)

type ChartInfo struct {
	Name    string
	Version string

	Values map[string]interface{}
}

type Config struct {
	Disabled           StrBool `json:"disabled,omitempty"`
	InstanceCreator    string  `json:"instanceCreator,omitempty"`
	PlatformUserID     string  `json:"platformUserID,omitempty"`
	PlatformInstanceID string  `json:"platformInstanceID,omitempty"`
	MachineID          string  `json:"machineID,omitempty"`
}

type KubernetesVersion struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
}

type StrBool string

// UnmarshalJSON parses fields that may be numbers or booleans.
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
