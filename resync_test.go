package resync

import (
	"fmt"
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
				Command:  String(fmt.Sprintf("-a ./testdata/a/ %s", dir)),
				Schedule: String("* * * * * *"),
			},
		},
	}
	err = config.validate()
	assert.Nil(t, err)

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	logger := NewFSLogger(config)

	mailer := NewMailer(config, db, logger)
	assert.NotNil(t, mailer)

	re := New(config, db, logger, mailer)
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
				Command:  String(fmt.Sprintf("-a ./testdata/a/ %s", dir)),
				Schedule: String("* * * * *"),
			},
		},
	}
	err = config.validate()
	assert.Nil(t, err)

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	logger := NewFSLogger(config)

	mailer := NewMailer(config, db, logger)
	assert.NotNil(t, mailer)

	re := New(config, db, logger, mailer)
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
