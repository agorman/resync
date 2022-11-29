package resync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailer(t *testing.T) {
	config := &Config{
		Retention: Int(0),
	}

	db, err := NewBoltDB(config)
	assert.Nil(t, err)

	logger := NewFSLogger(config)

	mailer := NewMailer(config, db, logger)
	assert.NotNil(t, mailer)

	// Testing that mailing with retention = 0 is a noop
	err = mailer.Mail(NewStat("TEST", "Mon Jan 02 03:04:05 PM MST"))
	assert.Nil(t, err)

	err = mailer.MailStats()
	assert.Nil(t, err)
}
