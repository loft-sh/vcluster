package container

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type Options struct {
	Path string `json:"path,omitempty"`
}

func NewStore(options *Options) *Store {
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
