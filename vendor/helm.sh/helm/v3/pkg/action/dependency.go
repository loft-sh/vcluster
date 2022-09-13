/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package action

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/gosuri/uitable"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// Dependency is the action for building a given chart's dependency tree.
//
// It provides the implementation of 'helm dependency' and its respective subcommands.
type Dependency struct {
	Verify      bool
	Keyring     string
	SkipRefresh bool
}

// NewDependency creates a new Dependency object with the given configuration.
func NewDependency() *Dependency {
	return &Dependency{}
}

// List executes 'helm dependency list'.
func (d *Dependency) List(chartpath string, out io.Writer) error {
	c, err := loader.Load(chartpath)
	if err != nil {
		return err
	}

	if c.Metadata.Dependencies == nil {
		fmt.Fprintf(out, "WARNING: no dependencies at %s\n", filepath.Join(chartpath, "charts"))
		return nil
	}

	d.printDependencies(chartpath, out, c.Metadata.Dependencies)
	fmt.Fprintln(out)
	d.printMissing(chartpath, out, c.Metadata.Dependencies)
	return nil
}

func (d *Dependency) dependencyStatus(chartpath string, dep *chart.Dependency) string {
	filename := fmt.Sprintf("%s-%s.tgz", dep.Name, "*")

	switch archives, err := filepath.Glob(filepath.Join(chartpath, "charts", filename)); {
	case err != nil:
		return "bad pattern"
	case len(archives) > 1:
		return "too many matches"
	case len(archives) == 1:
		archive := archives[0]
		if _, err := os.Stat(archive); err == nil {
			c, err := loader.Load(archive)
			if err != nil {
				return "corrupt"
			}
			if c.Name() != dep.Name {
				return "misnamed"
			}

			if c.Metadata.Version != dep.Version {
				constraint, err := semver.NewConstraint(dep.Version)
				if err != nil {
					return "invalid version"
				}

				v, err := semver.NewVersion(c.Metadata.Version)
				if err != nil {
					return "invalid version"
				}

				if constraint.Check(v) {
					return "ok"
				}
				return "wrong version"
			}
			return "ok"
		}
	}

	folder := filepath.Join(chartpath, "charts", dep.Name)
	if fi, err := os.Stat(folder); err != nil {
		return "missing"
	} else if !fi.IsDir() {
		return "mispackaged"
	}

	c, err := loader.Load(folder)
	if err != nil {
		return "corrupt"
	}

	if c.Name() != dep.Name {
		return "misnamed"
	}

	if c.Metadata.Version != dep.Version {
		constraint, err := semver.NewConstraint(dep.Version)
		if err != nil {
			return "invalid version"
		}

		v, err := semver.NewVersion(c.Metadata.Version)
		if err != nil {
			return "invalid version"
		}

		if constraint.Check(v) {
			return "unpacked"
		}
		return "wrong version"
	}

	return "unpacked"
}

// printDependencies prints all of the dependencies in the yaml file.
func (d *Dependency) printDependencies(chartpath string, out io.Writer, reqs []*chart.Dependency) {
	table := uitable.New()
	table.MaxColWidth = 80
	table.AddRow("NAME", "VERSION", "REPOSITORY", "STATUS")
	for _, row := range reqs {
		table.AddRow(row.Name, row.Version, row.Repository, d.dependencyStatus(chartpath, row))
	}
	fmt.Fprintln(out, table)
}

// printMissing prints warnings about charts that are present on disk, but are
// not in Charts.yaml.
func (d *Dependency) printMissing(chartpath string, out io.Writer, reqs []*chart.Dependency) {
	folder := filepath.Join(chartpath, "charts/*")
	files, err := filepath.Glob(folder)
	if err != nil {
		fmt.Fprintln(out, err)
		return
	}

	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			fmt.Fprintf(out, "Warning: %s\n", err)
		}
		// Skip anything that is not a directory and not a tgz file.
		if !fi.IsDir() && filepath.Ext(f) != ".tgz" {
			continue
		}
		c, err := loader.Load(f)
		if err != nil {
			fmt.Fprintf(out, "WARNING: %q is not a chart.\n", f)
			continue
		}
		found := false
		for _, d := range reqs {
			if d.Name == c.Name() {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(out, "WARNING: %q is not in Chart.yaml.\n", f)
		}
	}
}
