package resync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResync(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		Retention:    Int(0),
		SecondsField: Bool(true),
		TimeLimit:    String("5s"),
		Syncs: map[string]*Sync{
			"test": {
				RsyncArgs:        String("-a"),
				RsyncSource:      []string{"./testdata/a/"},
				RsyncDestination: String(dir),
				Schedule:         String("* * * * * *"),
			},
		},
	}
	err = config.validate()
	assert.Nil(t, err)

	db, err := NewBoltDB(config)
	assert.Nil(t, err)
	defer db.Close()

	logger := NewFSLogger(config)

	notifer := NewEmailNotifier(config, db, logger)
	assert.NotNil(t, notifer)

	re := New(config, db, logger, notifer)
	assert.NotNil(t, re)

	// very basic tests
	err = re.Start()
	assert.Nil(t, err)

	time.Sleep(time.Second)
	re.Stop()

	b, err := os.ReadFile(filepath.Join(dir, "test"))
	assert.Nil(t, err)
	assert.Equal(t, b, []byte("Hello World"))

	err = re.Dump()
	assert.Error(t, err)
}

func TestStart(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		LogPath: String(dir),
		LibPath: String(dir),
		Syncs: map[string]*Sync{
			"test": {
				RsyncArgs:        String("-a"),
				RsyncSource:      []string{"./testdata/a/"},
				RsyncDestination: String(dir),
				Schedule:         String("* * * * *"),
			},
		},
	}
	err = config.validate()
	assert.Nil(t, err)

	db, err := NewBoltDB(config)
	assert.Nil(t, err)
	defer db.Close()

	logger := NewFSLogger(config)

	notifier := NewEmailNotifier(config, db, logger)
	assert.NotNil(t, notifier)

	re := New(config, db, logger, notifier)
	assert.NotNil(t, re)

	err = re.Start()
	assert.Nil(t, err)
	err = re.Start()
	assert.Nil(t, err)

	re.Stop()
	re.Stop()

	err = re.Dump()
	assert.Nil(t, err)
}
