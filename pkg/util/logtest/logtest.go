// Package logtest provides small helpers for tests that assert on the
// JSON-encoded log file written by the file logging setup.
package logtest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	loftlogr "github.com/loft-sh/log/logr"
)

var (
	fileLoggerOnce sync.Once
	fileLogger     logr.Logger
	fileLoggerPath string
	errFileLogger  error
)

// NewFileJSONLogger returns a logger that writes JSON encoded records to a log
// file, together with the path of that file. Read the most recent record back
// with ReadJSONRecord.
//
// loft-sh/log/logr registers its file sink once per process and rejects a
// second registration at a different path, so the logger and its file are
// created on the first call and reused by every later call in this process
// (other tests in the same package, or the same test under -count=2).
func NewFileJSONLogger(t *testing.T) (logr.Logger, string) {
	t.Helper()

	fileLoggerOnce.Do(func() {
		dir, err := os.MkdirTemp("", "logtest-")
		if err != nil {
			errFileLogger = err
			return
		}
		fileLoggerPath = filepath.Join(dir, "vcluster.log")
		fileLogger, errFileLogger = loftlogr.NewLoggerWithOptions(
			loftlogr.WithLogFile(fileLoggerPath),
			loftlogr.WithLogEncoding("json"),
		)
	})
	if errFileLogger != nil {
		t.Fatal(errFileLogger)
	}

	return fileLogger, fileLoggerPath
}

// ReadJSONRecord reads the log file at path and decodes its last line as a
// single JSON log record. The file is shared across every NewFileJSONLogger
// caller in this process, so the last line is the most recently written
// record rather than necessarily the only one.
func ReadJSONRecord(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	line := lines[len(lines)-1]
	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("file record is not JSON: %v: %q", err, line)
	}

	return record
}
