package file

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

type Options struct {
	Path string `json:"path,omitempty"`
}

func AddFlags(fs *pflag.FlagSet, fileOptions *Options) {
	// file options
	fs.StringVar(&fileOptions.Path, "file-path", fileOptions.Path, "The file path to write the snapshot to")
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
	return "file://" + s.path
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
