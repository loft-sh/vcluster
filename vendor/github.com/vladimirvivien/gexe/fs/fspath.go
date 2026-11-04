package fs

import (
	"errors"
	"io/fs"
	"os"

	"github.com/vladimirvivien/gexe/vars"
)

type FSPath struct {
	err  error
	path string
	vars *vars.Variables
}

// Path points to a node path
func Path(path string) *FSPath {
	return &FSPath{path: path}
}

// PathWithVars points to a path and applies variables to the path value
func PathWithVars(path string, variables *vars.Variables) *FSPath {
	p := Path(variables.Eval(path))
	p.vars = variables
	return p
}

// Info returns information about the specified path
func (p *FSPath) Info() *FSInfo {
	info, err := os.Stat(p.path)
	if err != nil {
		return &FSInfo{err: err, path: p.path}
	}
	return &FSInfo{path: p.path, info: info, mode: info.Mode()}
}

// Exists returns true only if os.Stat nil error.
// Any other scenarios will return false.
func (p *FSPath) Exists() bool {
	if _, err := os.Stat(p.path); err != nil {
		return false
	}
	return true
}

// MkDir creates a directory with file mode at specified
func (p *FSPath) MkDir(mode fs.FileMode) *FSInfo {
	if err := os.MkdirAll(p.path, mode); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return &FSInfo{err: err, path: p.path}
		}
	}
	info, err := os.Stat(p.path)
	if err != nil {
		return &FSInfo{err: err, path: p.path}
	}
	return &FSInfo{path: p.path, info: info, mode: info.Mode(), vars: p.vars}
}

// Remove removes entry at path
func (p *FSPath) Remove() *FSInfo {
	info, err := os.Stat(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FSInfo{path: p.path}
		}
		return &FSInfo{err: err, path: p.path}
	}
	if err := os.RemoveAll(p.path); err != nil {
		return &FSInfo{err: err, path: p.path, info: info}
	}
	return &FSInfo{path: p.path, info: info, mode: info.Mode()}
}

// Read wraps call to create a new *FileReader instance
func (p *FSPath) Read() *FileReader {
	if p.vars != nil {
		return ReadWithVars(p.path, p.vars)
	}
	return Read(p.path)
}

// Write wraps call to create a new *FileWriter instance
func (p *FSPath) Write() *FileWriter {
	if p.vars != nil {
		return WriteWithVars(p.path, p.vars)
	}
	return Write(p.path)
}

// Append wraps call to create a new *FileWriter instance for file append operations
func (p *FSPath) Append() *FileWriter {
	if p.vars != nil {
		return AppendWitVars(p.path, p.vars)
	}
	return Append(p.path)
}

// Dirs returns info about dirs in path
func (p *FSPath) Dirs() (infos []*FSInfo) {
	entries, err := os.ReadDir(p.path)
	if err != nil {
		p.err = err
		return nil
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			infos = append(infos, &FSInfo{err: err})
		}
		infos = append(infos, &FSInfo{info: info, mode: info.Mode()})
	}
	return
}
