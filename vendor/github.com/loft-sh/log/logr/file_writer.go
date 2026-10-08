package logr

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// One writer for the process lifetime; logging is set up once at startup.
var fileWriter *registeredFileWriter

const (
	logFileMaxSizeMB   = 100
	logFileMaxBackups  = 5
	logFileDefaultMode = os.FileMode(0644)
	// bytesPerMB matches lumberjack's megabyte conversion.
	bytesPerMB = 1024 * 1024
)

type registeredFileWriter struct {
	path        string
	writeSyncer zapcore.WriteSyncer
	// requestedMode is what the caller asked for, not necessarily what the writer
	// enforces. Kept only to reject conflicting requests.
	requestedMode os.FileMode
}

type rotatingLogger interface {
	Write([]byte) (int, error)
	Rotate() error
	Close() error
}

type rotatingFileWriter struct {
	logger   rotatingLogger
	path     string
	mode     os.FileMode
	maxBytes int64
	// size mirrors lumberjack's byte counter for the active file and is only
	// meaningful while opened. Every reopen re-reads the real file size.
	size   int64
	opened bool
	mu     sync.Mutex
}

// newLogFileOption tees the console core with a file core and applies one
// sampler around both destinations so they accept the same events.
//
// It clears the config's sampling settings before Build and reapplies them
// around the tee, so sampling is not attached to the console core alone.
func newLogFileOption(config *zap.Config, logFile string, logFileMode os.FileMode) (zap.Option, error) {
	sampling := config.Sampling
	config.Sampling = nil

	fileCore, err := newLogFileCore(*config, logFile, logFileMode)
	if err != nil {
		return nil, fmt.Errorf("build log file core: %w", err)
	}

	return zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		combinedCore := zapcore.NewTee(core, fileCore)
		if sampling == nil {
			return combinedCore
		}

		var samplerOptions []zapcore.SamplerOption
		if sampling.Hook != nil {
			samplerOptions = append(samplerOptions, zapcore.SamplerHook(sampling.Hook))
		}
		return zapcore.NewSamplerWithOptions(
			combinedCore,
			time.Second,
			sampling.Initial,
			sampling.Thereafter,
			samplerOptions...,
		)
	}), nil
}

// newLogFileCore builds a zapcore.Core that writes to the configured log file
// through the rotating file write syncer, using a plain (non-color) level
// encoder.
func newLogFileCore(config zap.Config, logFile string, logFileMode os.FileMode) (zapcore.Core, error) {
	writeSyncer, err := getFileWriteSyncer(logFile, logFileMode)
	if err != nil {
		return nil, fmt.Errorf("open log file output: %w", err)
	}

	encoderConfig := config.EncoderConfig
	var encoder zapcore.Encoder
	if config.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// Keep the console encoder's ANSI color codes out of the persisted file
		// so it stays readable by grep and log shippers.
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	return zapcore.NewCore(encoder, writeSyncer, config.Level), nil
}

func getFileWriteSyncer(path string, logFileMode os.FileMode) (zapcore.WriteSyncer, error) {
	canonicalPath, err := canonicalLogFilePath(path)
	if err != nil {
		return nil, err
	}

	logFileMode = normalizeLogFileMode(logFileMode)
	if fileWriter != nil {
		if fileWriter.path != canonicalPath {
			return nil, fmt.Errorf("log file already enabled at %q, requested %q", fileWriter.path, canonicalPath)
		}
		if fileWriter.requestedMode != logFileMode {
			return nil, fmt.Errorf("log file %q already registered with requested mode %04o, requested %04o", canonicalPath, fileWriter.requestedMode, logFileMode)
		}
		return fileWriter.writeSyncer, nil
	}

	mode, err := initializeLogFile(canonicalPath, logFileMode)
	if err != nil {
		return nil, err
	}
	writer := zapcore.AddSync(newRotatingWriter(canonicalPath, mode))
	fileWriter = &registeredFileWriter{path: canonicalPath, writeSyncer: writer, requestedMode: logFileMode}
	return writer, nil
}

func normalizeLogFileMode(mode os.FileMode) os.FileMode {
	if mode.Perm() == 0 {
		return logFileDefaultMode
	}
	return mode.Perm()
}

func canonicalLogFilePath(path string) (string, error) {
	canonicalPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve log file path: %w", err)
	}
	return canonicalPath, nil
}

func newRotatingWriter(path string, mode os.FileMode) *rotatingFileWriter {
	return &rotatingFileWriter{
		// Lumberjack owns every reopen from here on, using plain os.Stat and
		// os.OpenFile, so it will follow a symlink planted at the path mid-rotation.
		// predictReplacement's Lstat catches a swap made while the writer is idle,
		// but not one raced against a rotation.
		logger: &lumberjack.Logger{
			Filename:   path,
			MaxSize:    logFileMaxSizeMB,
			MaxBackups: logFileMaxBackups,
		},
		path:     path,
		mode:     mode.Perm(),
		maxBytes: int64(logFileMaxSizeMB) * bytesPerMB,
	}
}

func (w *rotatingFileWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	replaced, err := w.predictReplacement(len(data))
	if err != nil {
		return 0, err
	}

	n, writeErr := w.logger.Write(data)
	w.opened = true
	if replaced {
		w.size = int64(n)
	} else {
		w.size += int64(n)
	}

	var modeErr error
	if replaced {
		modeErr = chmodRegularFile(w.path, w.mode)
	}
	if writeErr != nil {
		return n, w.resynchronize(writeErr, modeErr)
	}
	if modeErr != nil {
		// The log bytes landed; this error only reports that the mode could not
		// be restored after rotation.
		return n, fmt.Errorf("set rotated log file %q permissions: %w", w.path, modeErr)
	}
	return n, nil
}

// predictReplacement mirrors lumberjack's own decision about whether this write
// makes it put a fresh file at the active path, because a file it creates never
// carries the configured mode: openNew hardcodes 0600 for a path that does not
// exist, and copies the previous file's mode through the process umask otherwise.
func (w *rotatingFileWriter) predictReplacement(writeLen int) (bool, error) {
	if w.opened {
		// Holding the active file, lumberjack rotates strictly above the limit.
		return w.size+int64(writeLen) > w.maxBytes, nil
	}

	// Not holding a file, lumberjack re-stats the path before writing. Stat it
	// here too rather than carrying a mirror across the gap, which an external
	// delete or truncate between writes would leave overstating the file.
	info, err := statRegularFile(w.path)
	if errors.Is(err, fs.ErrNotExist) {
		// The active file was removed underneath us. Lumberjack recreates it at
		// a hardcoded 0600, so this write always needs the mode reapplied.
		w.size = 0
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat log file %q: %w", w.path, err)
	}
	w.size = info.Size()
	// Opening an existing file, lumberjack rotates at equality.
	return w.size+int64(writeLen) >= w.maxBytes, nil
}

// resynchronize drops the descriptor after a failed write so the next write
// re-reads the real file, and repairs the mode if the failure left it wrong.
//
// The close is unconditional because lumberjack.Write can fail inside its own
// open, and its API does not say whether it still holds the file. Closing makes
// opened knowable instead of guessed, and guessing wrong flips the rotation
// boundary predictReplacement uses -- equality when reopening, strictly greater
// when already open -- costing a missed chmod, the failure this package exists
// to prevent. The extra syscalls during an outage are the deliberate trade.
func (w *rotatingFileWriter) resynchronize(writeErr, modeErr error) error {
	closeErr := w.logger.Close()
	w.opened = false

	info, statErr := statRegularFile(w.path)
	if statErr == nil && info.Mode().Perm() != w.mode {
		modeErr = errors.Join(modeErr, chmodRegularFile(w.path, w.mode))
	}
	return errors.Join(writeErr, closeErr, statErr, modeErr)
}

func (w *rotatingFileWriter) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.logger.Rotate(); err != nil {
		return err
	}
	w.size = 0
	w.opened = true
	if err := chmodRegularFile(w.path, w.mode); err != nil {
		return fmt.Errorf("set rotated log file %q permissions: %w", w.path, err)
	}
	return nil
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.logger.Close()
	w.opened = false
	return err
}

// initializeLogFile fails fast at startup if the directory is missing or
// unwritable, and returns the mode the writer has to enforce from then on.
func initializeLogFile(path string, configuredMode os.FileMode) (os.FileMode, error) {
	configuredMode = normalizeLogFileMode(configuredMode)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_APPEND|os.O_WRONLY, configuredMode)
	if err == nil {
		// Apply the configured mode exactly to a newly-created file, so the
		// process umask cannot narrow it away from the sidecar that reads it.
		if chmodErr := chmodOpenFile(file, configuredMode); chmodErr != nil {
			_ = file.Close()
			return 0, fmt.Errorf("set log file %q permissions: %w", path, chmodErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return 0, fmt.Errorf("close initialized log file %q: %w", path, closeErr)
		}
		return configuredMode, nil
	}
	if !os.IsExist(err) {
		return 0, fmt.Errorf("open log file %q: %w", path, err)
	}

	file, info, err := openVerifiedRegularFile(path)
	if err != nil {
		return 0, fmt.Errorf("open log file %q: %w", path, err)
	}
	// Keep a pre-existing file's mode, but only where it is narrower than
	// requested. Adopting wider bits would let whoever can pre-place a file at
	// the configured path override WithLogFileMode on every rotation.
	resolvedMode := info.Mode().Perm() & configuredMode
	if resolvedMode != info.Mode().Perm() {
		if chmodErr := chmodOpenFile(file, resolvedMode); chmodErr != nil {
			_ = file.Close()
			return 0, fmt.Errorf("set log file %q permissions: %w", path, chmodErr)
		}
	}
	if closeErr := file.Close(); closeErr != nil {
		return 0, fmt.Errorf("close initialized log file %q: %w", path, closeErr)
	}
	return resolvedMode, nil
}
