package fs

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/vladimirvivien/gexe/vars"
)

type FileReader struct {
	err     error
	path    string
	info    os.FileInfo
	mode    os.FileMode
	vars    *vars.Variables
	content *bytes.Buffer
	ctx     context.Context
}

// ReadWithContextVars uses specified context and session variables to read the file at path
// and returns a *FileReader to access its content
func ReadWithContextVars(ctx context.Context, path string, variables *vars.Variables) *FileReader {
	if variables == nil {
		variables = &vars.Variables{}
	}
	filePath := variables.Eval(path)

	if err := ctx.Err(); err != nil {
		return &FileReader{err: err, path: filePath}
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return &FileReader{err: err, path: filePath}
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return &FileReader{err: err, path: filePath}
	}

	return &FileReader{
		path:    filePath,
		info:    info,
		mode:    info.Mode(),
		content: bytes.NewBuffer(fileData),
		vars:    variables,
		ctx:     ctx,
	}
}

// ReadWithVars uses session variables to create  a new FileReader
func ReadWithVars(path string, variables *vars.Variables) *FileReader {
	return ReadWithContextVars(context.Background(), path, variables)
}

// Read reads the file at path and returns FileReader to access its content
func Read(path string) *FileReader {
	return ReadWithContextVars(context.Background(), path, &vars.Variables{})
}

// SetVars sets the FileReader's session variables
func (fr *FileReader) SetVars(variables *vars.Variables) *FileReader {
	fr.vars = variables
	return fr
}

// SetContext sets the context for the FileReader operations
func (fr *FileReader) SetContext(ctx context.Context) *FileReader {
	fr.ctx = ctx
	return fr
}

// Err returns an operation error during file read.
func (fr *FileReader) Err() error {
	return fr.err
}

// Info surfaces the os.FileInfo for the associated file
func (fr *FileReader) Info() os.FileInfo {
	return fr.info
}

// String returns the content of the file as a string value
func (fr *FileReader) String() string {
	return fr.content.String()
}

// Lines returns the content of the file as slice of string
func (fr *FileReader) Lines() []string {
	if fr.err != nil {
		return []string{}
	}

	if err := fr.ctx.Err(); err != nil {
		fr.err = err
		return []string{}
	}

	var lines []string
	scnr := bufio.NewScanner(fr.content)

	for scnr.Scan() {
		if err := fr.ctx.Err(); err != nil {
			fr.err = err
			break
		}
		lines = append(lines, scnr.Text())
	}

	// err should never happen, but capture it anyway
	if scnr.Err() != nil {
		fr.err = scnr.Err()
		return []string{}
	}

	return lines
}

// Bytes returns the content of the file as []byte
func (fr *FileReader) Bytes() []byte {
	if fr.err != nil {
		return []byte{}
	}

	if err := fr.ctx.Err(); err != nil {
		fr.err = err
		return []byte{}
	}

	return fr.content.Bytes()
}

// Into reads the content of the file and writes
// it into the specified Writer
func (fr *FileReader) Into(w io.Writer) *FileReader {
	if fr.err != nil {
		return fr
	}

	if err := fr.ctx.Err(); err != nil {
		fr.err = err
		return fr
	}

	if _, err := io.Copy(w, fr.content); err != nil {
		fr.err = err
	}
	return fr
}
