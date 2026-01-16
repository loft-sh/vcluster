package fs

import (
	"context"
	"io"
	"os"

	"github.com/vladimirvivien/gexe/vars"
)

type FileWriter struct {
	path  string
	err   error
	finfo os.FileInfo
	mode  os.FileMode
	flags int
	vars  *vars.Variables
	ctx   context.Context
}

// WriteWithVars uses the specified context and session variables to create a new FileWriter
// that can be used to write content to file at path.  The file will be created with:
//
// os.O_CREATE | os.O_TRUNC | os.O_WRONLY
func WriteWithContextVars(ctx context.Context, path string, variables *vars.Variables) *FileWriter {
	if variables == nil {
		variables = &vars.Variables{}
	}
	filePath := variables.Eval(path)

	fw := &FileWriter{
		path:  filePath,
		flags: os.O_CREATE | os.O_TRUNC | os.O_WRONLY,
		mode:  0644,
		vars:  variables,
		ctx:   ctx,
	}

	info, err := os.Stat(fw.path)
	if err == nil {
		fw.finfo = info
	}
	return fw
}

// WriteWithVars uses sesison variables to create a new FileWriter to write content to file
func WriteWithVars(path string, variables *vars.Variables) *FileWriter {
	return WriteWithContextVars(context.Background(), path, variables)
}

// Write creates a new FileWriter to write file content
func Write(path string) *FileWriter {
	return WriteWithContextVars(context.Background(), path, &vars.Variables{})
}

// AppendWithContextVars uses the specified context and session variables to create a new FileWriter
// that can be used to append content existing file at path.  The file will be open with:
//
// os.O_CREATE | os.O_APPEND | os.O_WRONLY
//
// and mode 0644
func AppendWithContextVars(ctx context.Context, path string, variables *vars.Variables) *FileWriter {
	if variables == nil {
		variables = &vars.Variables{}
	}
	filePath := variables.Eval(path)

	fw := &FileWriter{
		path:  filePath,
		flags: os.O_CREATE | os.O_APPEND | os.O_WRONLY,
		mode:  0644,
		vars:  variables,
		ctx:   ctx,
	}

	info, err := os.Stat(fw.path)
	if err != nil {
		fw.err = err
		return fw
	}
	fw.finfo = info
	return fw
}

// AppendWithVars uses the specified session variables to create a FileWriter
// to write content to file at path.
func AppendWitVars(path string, variables *vars.Variables) *FileWriter {
	return AppendWithContextVars(context.Background(), path, variables)
}

// Append creates FileWriter to write content to file at path
func Append(path string) *FileWriter {
	return AppendWithContextVars(context.Background(), path, &vars.Variables{})
}

// SetVars sets session variables for FileWriter
func (fw *FileWriter) SetVars(variables *vars.Variables) *FileWriter {
	if variables != nil {
		fw.vars = variables
	}
	return fw
}

func (fw *FileWriter) WithMode(mode os.FileMode) *FileWriter {
	fw.mode = mode
	return fw
}

// WithContext sets an execution context for the FileWriter operations
func (fw *FileWriter) WithContext(ctx context.Context) *FileWriter {
	fw.ctx = ctx
	return fw
}

// Err returns FileWriter error during execution
func (fw *FileWriter) Err() error {
	return fw.err
}

// Info returns the os.FileInfo for the associated file
func (fw *FileWriter) Info() os.FileInfo {
	return fw.finfo
}

// String writes the provided str into the file. Any
// error that occurs can be accessed with FileWriter.Err().
func (fw *FileWriter) String(str string) *FileWriter {
	if fw.err != nil {
		return fw
	}
	file, err := os.OpenFile(fw.path, fw.flags, fw.mode)
	if err != nil {
		fw.err = err
		return fw
	}
	defer file.Close()
	if fw.finfo, fw.err = file.Stat(); fw.err != nil {
		return fw
	}

	if _, err := file.WriteString(str); err != nil {
		fw.err = err
	}
	return fw
}

// Lines writes the slice of strings into the file.
// Any error will be captured and returned via FileWriter.Err().
func (fw *FileWriter) Lines(lines []string) *FileWriter {
	if fw.err != nil {
		return fw
	}

	if err := fw.ctx.Err(); err != nil {
		fw.err = err
		return fw
	}

	file, err := os.OpenFile(fw.path, fw.flags, fw.mode)
	if err != nil {
		fw.err = err
		return fw
	}
	defer file.Close()
	if fw.finfo, fw.err = file.Stat(); fw.err != nil {
		return fw
	}

	len := len(lines)
	for i, line := range lines {
		if err := fw.ctx.Err(); err != nil {
			fw.err = err
			break
		}

		if _, err := file.WriteString(line); err != nil {
			fw.err = err
			return fw
		}
		if len > (i + 1) {
			if _, err := file.Write([]byte{'\n'}); err != nil {
				fw.err = err
				return fw
			}
		}
	}
	return fw
}

// Bytes writes the []bytre provided into the file.
// Any error can be accessed using FileWriter.Err().
func (fw *FileWriter) Bytes(data []byte) *FileWriter {
	if fw.err != nil {
		return fw
	}

	if err := fw.ctx.Err(); err != nil {
		fw.err = err
		return fw
	}

	file, err := os.OpenFile(fw.path, fw.flags, fw.mode)
	if err != nil {
		fw.err = err
		return fw
	}
	defer file.Close()
	if fw.finfo, fw.err = file.Stat(); fw.err != nil {
		return fw
	}

	if _, err := file.Write(data); err != nil {
		fw.err = err
	}
	return fw
}

// From streams bytes from the provided io.Reader r and
// writes them to the file. Any error will be captured
// and returned by fw.Err().
func (fw *FileWriter) From(r io.Reader) *FileWriter {
	if fw.err != nil {
		return fw
	}

	if err := fw.ctx.Err(); err != nil {
		fw.err = err
		return fw
	}

	file, err := os.OpenFile(fw.path, fw.flags, fw.mode)
	if err != nil {
		fw.err = err
		return fw
	}
	defer file.Close()
	if fw.finfo, fw.err = file.Stat(); fw.err != nil {
		return fw
	}

	if _, err := io.Copy(file, r); err != nil {
		fw.err = err
	}
	return fw
}
