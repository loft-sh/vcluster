package template

import (
	"os"
	gotemplate "text/template"
)

func MustRender(template string, data interface{}) (string, func() error) {
	tempDir, cleanUp, err := Render(template, data)
	if err != nil {
		panic(err)
	}
	return tempDir, cleanUp
}

func Render(template string, data interface{}) (string, func() error, error) {
	tmpFile, err := os.CreateTemp("", "vcluster-*.yaml")
	if err != nil {
		return "", nil, err
	}
	defer func(tmpFile *os.File) {
		_ = tmpFile.Close()
	}(tmpFile)

	parsedTemplate, err := gotemplate.New("template").Parse(template)
	if err != nil {
		return "", nil, err
	}

	if err := parsedTemplate.Execute(tmpFile, data); err != nil {
		return "", nil, err
	}

	return tmpFile.Name(), func() error {
		return os.Remove(tmpFile.Name())
	}, nil
}

// RenderToFile re-renders a template with new data to an existing file path.
// This is useful for updating previously rendered templates after flag parsing.
func RenderToFile(path string, tmpl string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	parsed, err := gotemplate.New("template").Parse(tmpl)
	if err != nil {
		return err
	}

	return parsed.Execute(f, data)
}
