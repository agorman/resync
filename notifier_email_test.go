package resync

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailer(t *testing.T) {
	config, err := OpenConfig("./testdata/resync.yaml")
	assert.Nil(t, err)

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	successStat := NewStat("SUCCESS", "Mon Jan 02 03:04:05 PM MST").Finish(nil)
	errorStat := NewStat("SUCCESS", "Mon Jan 02 03:04:05 PM MST").Finish(errors.New("ERROR"))
	err = db.Insert(successStat)
	assert.Nil(t, err)
	err = db.Insert(errorStat)
	assert.Nil(t, err)

	logger := NewFSLogger(config)

	emailNotifier := NewEmailNotifier(config, db, logger)
	assert.NotNil(t, emailNotifier)

	// Testing that mailing with retention = 0 is a noop
	err = emailNotifier.Notify(NewStat("TEST", "Mon Jan 02 03:04:05 PM MST"))
	assert.Error(t, err)

	err = emailNotifier.NotifyHistory()
	assert.Error(t, err)

	// safe because resync isn't running
	config.Email.HistoryTemplate = String("./testdata/test.tmpl")
	err = emailNotifier.NotifyHistory()
	assert.Error(t, err)
}

func TestSkipMailer(t *testing.T) {
	config := &Config{
		Retention: Int(0),
	}

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	logger := NewFSLogger(config)

	emailNotifier := NewEmailNotifier(config, db, logger)
	assert.NotNil(t, emailNotifier)

	// Testing that mailing with retention = 0 is a noop
	err = emailNotifier.Notify(NewStat("TEST", "Mon Jan 02 03:04:05 PM MST"))
	assert.Nil(t, err)

	err = emailNotifier.NotifyHistory()
	assert.Nil(t, err)
}
