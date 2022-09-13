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

package storage // import "helm.sh/helm/v3/pkg/storage"

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	rspb "helm.sh/helm/v3/pkg/release"
	relutil "helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/storage/driver"
)

// The type field of the Kubernetes storage object which stores the Helm release
// version. It is modified slightly replacing the '/': sh.helm/release.v1
// Note: The version 'v1' is incremented if the release object metadata is
// modified between major releases.
// This constant is used as a prefix for the Kubernetes storage object name.
const HelmStorageType = "sh.helm.release.v1"

// Storage represents a storage engine for a Release.
type Storage struct {
	driver.Driver

	// MaxHistory specifies the maximum number of historical releases that will
	// be retained, including the most recent release. Values of 0 or less are
	// ignored (meaning no limits are imposed).
	MaxHistory int

	Log func(string, ...interface{})
}

// Get retrieves the release from storage. An error is returned
// if the storage driver failed to fetch the release, or the
// release identified by the key, version pair does not exist.
func (s *Storage) Get(name string, version int) (*rspb.Release, error) {
	s.Log("getting release %q", makeKey(name, version))
	return s.Driver.Get(makeKey(name, version))
}

// Create creates a new storage entry holding the release. An
// error is returned if the storage driver failed to store the
// release, or a release with identical an key already exists.
func (s *Storage) Create(rls *rspb.Release) error {
	s.Log("creating release %q", makeKey(rls.Name, rls.Version))
	if s.MaxHistory > 0 {
		// Want to make space for one more release.
		s.removeLeastRecent(rls.Name, s.MaxHistory-1)
	}
	return s.Driver.Create(makeKey(rls.Name, rls.Version), rls)
}

// Update update the release in storage. An error is returned if the
// storage backend fails to update the release or if the release
// does not exist.
func (s *Storage) Update(rls *rspb.Release) error {
	s.Log("updating release %q", makeKey(rls.Name, rls.Version))
	return s.Driver.Update(makeKey(rls.Name, rls.Version), rls)
}

// Delete deletes the release from storage. An error is returned if
// the storage backend fails to delete the release or if the release
// does not exist.
func (s *Storage) Delete(name string, version int) (*rspb.Release, error) {
	s.Log("deleting release %q", makeKey(name, version))
	return s.Driver.Delete(makeKey(name, version))
}

// ListReleases returns all releases from storage. An error is returned if the
// storage backend fails to retrieve the releases.
func (s *Storage) ListReleases() ([]*rspb.Release, error) {
	s.Log("listing all releases in storage")
	return s.Driver.List(func(_ *rspb.Release) bool { return true })
}

// ListUninstalled returns all releases with Status == UNINSTALLED. An error is returned
// if the storage backend fails to retrieve the releases.
func (s *Storage) ListUninstalled() ([]*rspb.Release, error) {
	s.Log("listing uninstalled releases in storage")
	return s.Driver.List(func(rls *rspb.Release) bool {
		return relutil.StatusFilter(rspb.StatusUninstalled).Check(rls)
	})
}

// ListDeployed returns all releases with Status == DEPLOYED. An error is returned
// if the storage backend fails to retrieve the releases.
func (s *Storage) ListDeployed() ([]*rspb.Release, error) {
	s.Log("listing all deployed releases in storage")
	return s.Driver.List(func(rls *rspb.Release) bool {
		return relutil.StatusFilter(rspb.StatusDeployed).Check(rls)
	})
}

// Deployed returns the last deployed release with the provided release name, or
// returns ErrReleaseNotFound if not found.
func (s *Storage) Deployed(name string) (*rspb.Release, error) {
	ls, err := s.DeployedAll(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, errors.Errorf("%q has no deployed releases", name)
		}
		return nil, err
	}

	if len(ls) == 0 {
		return nil, errors.Errorf("%q has no deployed releases", name)
	}

	return ls[0], err
}

// DeployedAll returns all deployed releases with the provided name, or
// returns ErrReleaseNotFound if not found.
func (s *Storage) DeployedAll(name string) ([]*rspb.Release, error) {
	s.Log("getting deployed releases from %q history", name)

	ls, err := s.Driver.Query(map[string]string{
		"name":   name,
		"owner":  "helm",
		"status": "deployed",
	})
	if err == nil {
		return ls, nil
	}
	if strings.Contains(err.Error(), "not found") {
		return nil, errors.Errorf("%q has no deployed releases", name)
	}
	return nil, err
}

// History returns the revision history for the release with the provided name, or
// returns ErrReleaseNotFound if no such release name exists.
func (s *Storage) History(name string) ([]*rspb.Release, error) {
	s.Log("getting release history for %q", name)

	return s.Driver.Query(map[string]string{"name": name, "owner": "helm"})
}

// removeLeastRecent removes items from history until the lengh number of releases
// does not exceed max.
//
// We allow max to be set explicitly so that calling functions can "make space"
// for the new records they are going to write.
func (s *Storage) removeLeastRecent(name string, max int) error {
	if max < 0 {
		return nil
	}
	h, err := s.History(name)
	if err != nil {
		return err
	}
	if len(h) <= max {
		return nil
	}
	overage := len(h) - max

	// We want oldest to newest
	relutil.SortByRevision(h)

	// Delete as many as possible. In the case of API throughput limitations,
	// multiple invocations of this function will eventually delete them all.
	toDelete := h[0:overage]
	errs := []error{}
	for _, rel := range toDelete {
		key := makeKey(name, rel.Version)
		_, innerErr := s.Delete(name, rel.Version)
		if innerErr != nil {
			s.Log("error pruning %s from release history: %s", key, innerErr)
			errs = append(errs, innerErr)
		}
	}

	s.Log("Pruned %d record(s) from %s with %d error(s)", len(toDelete), name, len(errs))
	switch c := len(errs); c {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Errorf("encountered %d deletion errors. First is: %s", c, errs[0])
	}
}

// Last fetches the last revision of the named release.
func (s *Storage) Last(name string) (*rspb.Release, error) {
	s.Log("getting last revision of %q", name)
	h, err := s.History(name)
	if err != nil {
		return nil, err
	}
	if len(h) == 0 {
		return nil, errors.Errorf("no revision for release %q", name)
	}

	relutil.Reverse(h, relutil.SortByRevision)
	return h[0], nil
}

// makeKey concatenates the Kubernetes storage object type, a release name and version
// into a string with format:```<helm_storage_type>.<release_name>.v<release_version>```.
// The storage type is prepended to keep name uniqueness between different
// release storage types. An example of clash when not using the type:
// https://github.com/helm/helm/issues/6435.
// This key is used to uniquely identify storage objects.
func makeKey(rlsname string, version int) string {
	return fmt.Sprintf("%s.%s.v%d", HelmStorageType, rlsname, version)
}

// Init initializes a new storage backend with the driver d.
// If d is nil, the default in-memory driver is used.
func Init(d driver.Driver) *Storage {
	// default driver is in memory
	if d == nil {
		d = driver.NewMemory()
	}
	return &Storage{
		Driver: d,
		Log:    func(_ string, _ ...interface{}) {},
	}
}
