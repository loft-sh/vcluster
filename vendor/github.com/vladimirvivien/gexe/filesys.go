package gexe

import (
	"context"
	"os"

	"github.com/vladimirvivien/gexe/fs"
)

// PathExists returns true if path exists.
// All errors causes to return false.
func (e *Echo) PathExists(path string) bool {
	return fs.PathWithVars(path, e.vars).Exists()
}

// MkDir creates a directory at specified path with mode value.
// FSInfo contains information about the path or error if occured
func (e *Echo) MkDir(path string, mode os.FileMode) *fs.FSInfo {
	p := fs.PathWithVars(path, e.vars)
	return p.MkDir(mode)
}

// RmPath removes specified path (dir or file).
// Error is returned FSInfo.Err()
func (e *Echo) RmPath(path string) *fs.FSInfo {
	p := fs.PathWithVars(path, e.vars)
	return p.Remove()
}

// PathInfo
func (e *Echo) PathInfo(path string) *fs.FSInfo {
	return fs.PathWithVars(path, e.vars).Info()
}

// FileReadWithContext uses specified context to provide methods to read file
// content at path.
func (e *Echo) FileReadWithContext(ctx context.Context, path string) *fs.FileReader {
	return fs.ReadWithContextVars(ctx, path, e.vars)
}

// FileRead provides methods to read file content
func (e *Echo) FileRead(path string) *fs.FileReader {
	return fs.ReadWithContextVars(context.Background(), path, e.vars)
}

// FileWriteWithContext uses context ctx to create a fs.FileWriter to write content to provided path
func (e *Echo) FileWriteWithContext(ctx context.Context, path string) *fs.FileWriter {
	return fs.WriteWithContextVars(ctx, path, e.vars)
}

// FileWrite creates a fs.FileWriter to write content to provided path
func (e *Echo) FileWrite(path string) *fs.FileWriter {
	return fs.WriteWithContextVars(context.Background(), path, e.vars)
}

// FileAppend creates a new fs.FileWriter to append content to provided path
func (e *Echo) FileAppendWithContext(ctx context.Context, path string) *fs.FileWriter {
	return fs.AppendWithContextVars(ctx, path, e.vars)
}

// FileAppend creates a new fs.FileWriter to append content to provided path
func (e *Echo) FileAppend(path string) *fs.FileWriter {
	return fs.AppendWithContextVars(context.Background(), path, e.vars)
}
