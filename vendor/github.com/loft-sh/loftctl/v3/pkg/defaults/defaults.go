package defaults

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/pkg/errors"
)

const (
	KeyProject = "project"
)

var (
	ConfigFile   = "defaults.json"
	ConfigFolder = client.CacheFolder

	DefaultKeys = []string{KeyProject}
)

// Defaults holds the default values
type Defaults struct {
	folderPath string
	fileName   string
	fullPath   string

	values map[string]string
}

// NewFromPath creates a new defaults instance from the given path
func NewFromPath(folderPath string, fileName string) (*Defaults, error) {
	fullPath := filepath.Join(folderPath, fileName)
	defaults := &Defaults{folderPath, fileName, fullPath, make(map[string]string)}

	if err := defaults.ensureConfigFile(); err != nil {
		return defaults, errors.Wrap(err, "no config file")
	}

	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return defaults, errors.Wrap(err, "read config file")
	}
	if len(contents) == 0 {
		return defaults, nil
	}
	if err = json.Unmarshal(contents, &defaults.values); err != nil {
		return defaults, errors.Wrap(err, "invalid json")
	}

	return defaults, nil
}

// Set sets the given key to the given value and persists the defaults on disk
func (d *Defaults) Set(key string, value string) error {
	if !IsSupportedKey(key) {
		return errors.Errorf("key %s is not supported", key)
	}

	d.values[key] = value
	json, err := json.Marshal(d.values)
	if err != nil {
		return errors.Wrap(err, "invalid json")
	}
	if err = os.WriteFile(d.fullPath, json, os.ModePerm); err != nil {
		return errors.Wrap(err, "write config file")
	}

	return nil
}

// Get returns the value for the given key
func (d *Defaults) Get(key string, fallback string) (string, error) {
	if !IsSupportedKey(key) {
		return fallback, errors.Errorf("key %s is not supported", key)
	}

	return d.values[key], nil
}

// IsSupportedKey returns true if the given key is supported
func IsSupportedKey(key string) bool {
	for _, k := range DefaultKeys {
		if k == key {
			return true
		}
	}

	return false
}

func (d *Defaults) ensureConfigFile() error {
	_, err := os.Stat(d.fullPath)
	// file exists
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) {
		if err := os.MkdirAll(d.folderPath, os.ModePerm); err != nil {
			return errors.Wrap(err, "create cache folder")
		}
		if _, err := os.Create(d.fullPath); err != nil {
			return errors.Wrap(err, "create defaults file")
		}

		return nil
	} else {
		return err
	}
}
