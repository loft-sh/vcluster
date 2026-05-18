package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/api/v4/pkg/snapshot"
)

func NewStore(options *snapshot.ContainerOptions) *Store {
	return &Store{
		path: options.Path,
	}
}

type Store struct {
	path string
}

func (s *Store) Target() string {
	return "container://" + s.path
}

func (s *Store) GetObject(_ context.Context) (io.ReadCloser, error) {
	return os.Open(s.path)
}

func (s *Store) PutObject(_ context.Context, body io.Reader) error {
	err := os.MkdirAll(filepath.Dir(s.path), 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, body)
	return err
}

func (s *Store) List(_ context.Context) ([]snapshot.Snapshot, error) {
	path := s.path
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		path = filepath.Dir(path)
	}

	var snapshotsList []snapshot.Snapshot
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		eInfo, err := entry.Info()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		if eInfo.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), "tar.gz") {
			continue
		}

		snapshotsList = append(snapshotsList, snapshot.Snapshot{
			ID:        entry.Name(),
			URL:       "container://" + path + "/" + entry.Name(),
			Timestamp: eInfo.ModTime(),
		})
	}
	return snapshotsList, nil
}

func (s *Store) Delete(_ context.Context) error {
	fileInfo, err := os.Stat(s.path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("not a snapshot file")
	}

	if !strings.HasSuffix(s.path, "tar.gz") {
		return fmt.Errorf("not a snapshot file")
	}

	return os.Remove(s.path)
}
