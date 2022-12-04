package resync

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		LibPath:   String(dir),
		Retention: Int(3),
	}

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	stat1 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat2 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat3 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat4 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)

	stat5 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat6 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat7 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat8 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)

	err = db.Insert(stat1)
	assert.Nil(t, err)

	err = db.Insert(stat2)
	assert.Nil(t, err)

	err = db.Insert(stat3)
	assert.Nil(t, err)

	err = db.Insert(stat4)
	assert.Nil(t, err)

	err = db.Insert(stat5)
	assert.Nil(t, err)

	err = db.Insert(stat6)
	assert.Nil(t, err)

	err = db.Insert(stat7)
	assert.Nil(t, err)

	err = db.Insert(stat8)
	assert.Nil(t, err)

	stats, err := db.List()
	assert.Nil(t, err)
	assert.Len(t, stats, 2)
	assert.Len(t, stats["TEST"], int(3))
	assert.Len(t, stats["TEST2"], 3)

	err = db.Prune()
	assert.Nil(t, err)
	assert.Len(t, stats, 2)
	assert.Len(t, stats["TEST"], int(3))
	assert.Len(t, stats["TEST2"], 3)
}

func TestDBWithoutRetention(t *testing.T) {
	dir, err := os.MkdirTemp("", "resync_test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	config := &Config{
		LibPath:   String(dir),
		Retention: Int(0),
	}

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	stat1 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat2 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat3 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat4 := NewStat("TEST", "Mon Jan 02 03:04:05 PM MST").Finish(nil)

	stat5 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat6 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat7 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	stat8 := NewStat("TEST2", "Mon Jan 02 03:04:05 PM MST").Finish(nil)

	err = db.Insert(stat1)
	assert.Nil(t, err)

	err = db.Insert(stat2)
	assert.Nil(t, err)

	err = db.Insert(stat3)
	assert.Nil(t, err)

	err = db.Insert(stat4)
	assert.Nil(t, err)

	err = db.Insert(stat5)
	assert.Nil(t, err)

	err = db.Insert(stat6)
	assert.Nil(t, err)

	err = db.Insert(stat7)
	assert.Nil(t, err)

	err = db.Insert(stat8)
	assert.Nil(t, err)

	stats, err := db.List()
	assert.Nil(t, err)
	assert.Len(t, stats, 0)

	err = db.Prune()
	assert.Nil(t, err)
	assert.Len(t, stats, 0)
}
