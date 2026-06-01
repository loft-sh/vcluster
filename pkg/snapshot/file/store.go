package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	snapshotapi "github.com/loft-sh/api/v4/pkg/snapshot"
)

type Options struct {
	Path string `json:"path,omitempty"`
}

func NewStore(options *snapshotapi.FileOptions) *Store {
	return &Store{path: options.Path}
}

type Store struct {
	path string
}

func (s *Store) Target() string {
	return "file://" + s.path
}

func (s *Store) GetObject(_ context.Context) (io.ReadCloser, error) {
	return os.Open(s.path)
}

func (s *Store) PutObject(_ context.Context, body io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, body)
	return err
}

func (s *Store) List(_ context.Context) ([]snapshotapi.Snapshot, error) {
	path := s.path
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		path = filepath.Dir(path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var snapshots []snapshotapi.Snapshot
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if info.IsDir() || !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}
		snapshots = append(snapshots, snapshotapi.Snapshot{
			ID:        entry.Name(),
			URL:       "file://" + path + "/" + entry.Name(),
			Timestamp: info.ModTime(),
		})
	}
	return snapshots, nil
}

func (s *Store) Delete(_ context.Context) error {
	fi, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("not a snapshot file")
	}
	if !strings.HasSuffix(s.path, ".tar.gz") {
		return fmt.Errorf("not a snapshot file")
	}
	return os.Remove(s.path)
}
