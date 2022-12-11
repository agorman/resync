package resync

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// FSLoggger persists log files to a file system based on the LogPath config
// options. Beware that errors can occur when using the returned io.ReadClosers from
// Stderr and Stdout and calling Rotate.
type FSLogger struct {
	config *Config
	mu     sync.Mutex
}

// Creates an FSLogger with the given config.
func NewFSLogger(config *Config) *FSLogger {
	return &FSLogger{
		config: config,
		mu:     sync.Mutex{},
	}
}

// Rotates the log files and keeps old logs based on the configured retention value.
func (l *FSLogger) Rotate(name string) (io.WriteCloser, io.WriteCloser, error) {
	if IntValue(l.config.Retention) < 1 {
		return nil, nil, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// create a directory under the log path with the same name as the sync
	if err := os.MkdirAll(fmt.Sprintf("%s/%s", StringValue(l.config.LogPath), name), 0666); err != nil {
		return nil, nil, fmt.Errorf("FS Logger: failed to create log path: %w", err)
	}

	stdout := fmt.Sprintf("%s/%s/stdout.log", StringValue(l.config.LogPath), name)
	logger := &lumberjack.Logger{
		Filename:   stdout,
		MaxSize:    0,
		MaxBackups: IntValue(l.config.Retention),
	}
	logger.Rotate()

	stdoutLog, err := os.Create(stdout)
	if err != nil {
		return nil, nil, fmt.Errorf("FS Logger: failed to create stdout log: %w", err)
	}

	stderr := fmt.Sprintf("%s/%s/stderr.log", StringValue(l.config.LogPath), name)
	logger = &lumberjack.Logger{
		Filename:   stderr,
		MaxSize:    0,
		MaxBackups: IntValue(l.config.Retention),
	}
	logger.Rotate()

	stderrLog, err := os.Create(stderr)
	if err != nil {
		return nil, nil, fmt.Errorf("FS Logger: failed to create stderr log: %w", err)
	}

	return stdoutLog, stderrLog, nil
}

// Stdout returns the last STDOUT from the rsync command. Expect errors Rotate is called
// while stdout is still open.
func (l *FSLogger) Stdout(name string) (io.ReadCloser, error) {
	if IntValue(l.config.Retention) < 1 {
		return nil, errors.New("FS Logger: stdout log unavailable when retention is less than 1")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	logDir := filepath.Join(StringValue(l.config.LogPath), name)
	if err := os.MkdirAll(logDir, 0644); err != nil {
		return nil, fmt.Errorf("FS Logger: failed to create stdout log path for sync %s: %w", name, err)
	}

	stdout := filepath.Join(logDir, "stdout.log")

	f, err := os.Open(stdout)
	if err != nil {
		return nil, fmt.Errorf("FS Logger: failed to open stdout log for sync %s: %w", name, err)
	}
	return f, nil
}

// Stderr returns the last STDERR from the rsync command. Expect errors Rotate is called
// while stderr is still open.
func (l *FSLogger) Stderr(name string) (io.ReadCloser, error) {
	if IntValue(l.config.Retention) < 1 {
		return nil, errors.New("FS Logger: stderr log unavailable when retention is less than 1")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	logDir := filepath.Join(StringValue(l.config.LogPath), name)
	if err := os.MkdirAll(logDir, 0644); err != nil {
		return nil, fmt.Errorf("FS Logger: failed to create stderr log path for sync %s: %w", name, err)
	}

	stderr := filepath.Join(logDir, "stderr.log")

	f, err := os.Open(stderr)
	if err != nil {
		return nil, fmt.Errorf("FS Logger: failed to open stderr log for sync %s: %w", name, err)
	}
	return f, nil
}

// Zip returns file that contains all of the stored logs for all syncs. Expect errors Rotate is called
// while the zip is being created.
func (l *FSLogger) Zip() (*os.File, error) {
	if IntValue(l.config.Retention) < 1 {
		return nil, errors.New("FS Logger: zipping logs unavailable when retention is less than 1")
	}

	zipFile, err := os.CreateTemp("", "logs.*.zip")
	if err != nil {
		return nil, fmt.Errorf("FS Logger: failed to create temp file for zip: %w", err)
	}

	w := zip.NewWriter(zipFile)
	defer w.Close()

	err = filepath.Walk(StringValue(l.config.LogPath), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(StringValue(l.config.LogPath), path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("FS Logger: failed create zip file: %w", err)
	}
	return zipFile, nil
}
