package resync

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		LogPath:   String(dir),
		Retention: Int(3),
	}

	logger := NewFSLogger(config)
	assert.NotNil(t, logger)

	stdout, stderr, err := logger.Rotate("TEST")
	assert.Nil(t, err)

	_, err = fmt.Fprint(stdout, "STDOUT")
	assert.Nil(t, err)
	assert.Nil(t, stdout.Close())

	_, err = fmt.Fprint(stderr, "STDERR")
	assert.Nil(t, err)
	assert.Nil(t, stderr.Close())

	r, err := logger.Stdout("TEST")
	assert.Nil(t, err)
	b, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte("STDOUT"))
	assert.Nil(t, r.Close())

	r, err = logger.Stderr("TEST")
	assert.Nil(t, err)
	b, err = io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte("STDERR"))
	assert.Nil(t, r.Close())

	stdout, stderr, err = logger.Rotate("TEST")
	assert.Nil(t, err)
	assert.Nil(t, stdout.Close())
	assert.Nil(t, stderr.Close())

	entries, err := os.ReadDir(filepath.Join(dir, "TEST"))
	assert.Nil(t, err)
	assert.Len(t, entries, 4)

	// test zip file
	zipFile, err := logger.Zip()
	assert.Nil(t, err)

	zipStat, err := zipFile.Stat()
	assert.Nil(t, err)

	zipReader, err := zip.NewReader(zipFile, zipStat.Size())
	assert.Nil(t, err)
	assert.Len(t, zipReader.File, 4)
}

func TestLoggerWithoutRetention(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		LogPath:   String(dir),
		Retention: Int(0),
	}

	logger := NewFSLogger(config)
	assert.NotNil(t, logger)

	stdout, stderr, err := logger.Rotate("TEST")
	assert.Nil(t, stdout)
	assert.Nil(t, stderr)
	assert.Nil(t, err)

	_, err = logger.Stdout("TEST")
	assert.NotNil(t, err)

	_, err = logger.Stderr("TEST")
	assert.NotNil(t, err)

	_, err = logger.Zip()
	assert.NotNil(t, err)
}
