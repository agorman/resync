package resync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	config, err := OpenConfig("./testdata/resync.yaml")
	assert.Nil(t, err)

	assert.Equal(t, StringValue(config.RsyncPath), "my_rsync")
	assert.Equal(t, StringValue(config.LogPath), "/var/log/changed")
	assert.Equal(t, StringValue(config.LogLevel), "info")
	assert.Equal(t, StringValue(config.LibPath), "/var/lib/changed")
	assert.Equal(t, IntValue(config.Retention), 10)
	assert.Equal(t, BoolValue(config.SecondsField), true)
	assert.Equal(t, StringValue(config.TimeLimit), "5h")
	d, err := time.ParseDuration("5h")
	assert.Nil(t, err)
	assert.Equal(t, config.timeLimit, d)

	assert.NotNil(t, config.HTTP)
	assert.Equal(t, StringValue(config.HTTP.Addr), "0.0.0.0")
	assert.Equal(t, IntValue(config.HTTP.Port), 4070)

	assert.NotNil(t, config.Email)
	assert.Equal(t, StringValue(config.Email.Host), "1.1.1.1.1")
	assert.Equal(t, IntValue(config.Email.Port), 25)
	assert.Equal(t, StringValue(config.Email.User), "me")
	assert.Equal(t, StringValue(config.Email.Pass), "pass")
	assert.Equal(t, BoolValue(config.Email.StartTLS), true)
	assert.Equal(t, BoolValue(config.Email.InsecureSkipVerify), false)
	assert.Equal(t, BoolValue(config.Email.SSL), false)
	assert.Equal(t, StringValue(config.Email.From), "me@me.com")
	assert.Contains(t, config.Email.To, "they@me.com")
	assert.Contains(t, config.Email.To, "them@me.com")
	assert.Equal(t, StringValue(config.Email.HistorySubject), "Resync History")

	assert.Len(t, config.Syncs, 3)

	media, err := config.GetSync("media")
	assert.Nil(t, err)

	assert.Equal(t, StringValue(media.Schedule), "* * * * * *")
	assert.ElementsMatch(t, media.Args(), []string{"-a", "/files/MEDIA/", "/mnt/MEDIA_BK/"})

	video, err := config.GetSync("video")
	assert.Nil(t, err)

	assert.Equal(t, StringValue(video.Schedule), "0 0 * * * *")
	assert.ElementsMatch(t, video.Args(), []string{"-a", "--stats", "/files/VIDEO/", "/mnt/VIDEO_BK/"})

	audio, err := config.GetSync("audio")
	assert.Nil(t, err)

	assert.Equal(t, StringValue(audio.Schedule), "0 0 * * * *")
	assert.ElementsMatch(t, audio.Args(), []string{"-a", "--stats", "/files/AUDIO1/", "/files/AUDIO2/", "/mnt/AUDIO_BK/"})

	_, err = config.GetSync("foo")
	assert.Error(t, err)

	dur, err := config.GetTimeLimit("media")
	assert.Nil(t, err)
	assert.Equal(t, dur, d)

	d, err = time.ParseDuration("1h")
	assert.Nil(t, err)
	dur, err = config.GetTimeLimit("video")
	assert.Nil(t, err)
	assert.Equal(t, dur, d)
}

func TestDefaults(t *testing.T) {
	config, err := OpenConfig("./testdata/defaults.yaml")
	assert.Nil(t, err)

	assert.Equal(t, StringValue(config.LogPath), "/var/log/resync")
	assert.Equal(t, IntValue(config.Retention), 7)
	assert.Equal(t, StringValue(config.LibPath), "/var/lib/resync")
	assert.Equal(t, IntValue(config.Retention), 7)
	assert.Len(t, config.Syncs, 2)

	_, err = config.GetTimeLimit("foo")
	assert.Error(t, err)
}

func TestMissingCommand(t *testing.T) {
	_, err := OpenConfig("./testdata/sync_missing_rsync_fields.yaml")
	assert.Error(t, err)
}

func TestMissingSchedule(t *testing.T) {
	_, err := OpenConfig("./testdata/sync_missing_schedule.yaml")
	assert.Error(t, err)
}

func TestEmpty(t *testing.T) {
	_, err := OpenConfig("./testdata/empty.yaml")
	assert.Error(t, err)
}

func TestEmptySyncs(t *testing.T) {
	config := &Config{}
	err := config.validate()
	assert.Error(t, err)
}

func TestSyncInvalidTimeLimit(t *testing.T) {
	config := &Config{
		Syncs: map[string]*Sync{
			"invalid time": {
				RsyncArgs:        String("-a"),
				RsyncSource:      []string{"/a/b/c"},
				RsyncDestination: String("/d/e/f"),
				Schedule:         String("* * * * * *"),
				TimeLimit:        String("invalid"),
			},
		},
	}
	err := config.validate()
	assert.Error(t, err)
}

func TestInvalid(t *testing.T) {
	_, err := OpenConfig("./testdata/invalid.yaml")
	assert.Error(t, err)
}

func TestLogLevel(t *testing.T) {
	config := &Config{
		Syncs: map[string]*Sync{
			"test": {
				RsyncArgs:        String("-a"),
				RsyncSource:      []string{"/a/b/c"},
				RsyncDestination: String("/d/e/f"),
				Schedule:         String("* * * * * *"),
			},
		},
	}

	config.LogLevel = String("panic")
	err := config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("fatal")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("trace")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("debug")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("warn")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("info")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("error")
	err = config.validate()
	assert.Nil(t, err)

	config.LogLevel = String("bad")
	err = config.validate()
	assert.Error(t, err)
}

func TestEmail(t *testing.T) {
	config := &Config{
		Email: &Email{},
		HTTP:  &HTTP{},
		Syncs: map[string]*Sync{
			"test": {
				RsyncArgs:        String("-a"),
				RsyncSource:      []string{"/a/b/c"},
				RsyncDestination: String("/d/e/f"),
				Schedule:         String("* * * * * *"),
			},
		},
	}

	err := config.validate()
	assert.Error(t, err)

	config.Email.Host = String("smtp.me.com")
	err = config.validate()
	assert.Error(t, err)

	config.Email.InsecureSkipVerify = Bool(true)
	err = config.validate()
	assert.Error(t, err)

	config.Email.From = String("me@me.com")
	err = config.validate()
	assert.Error(t, err)

	config.Email.To = []string{"you@me.com"}
	err = config.validate()
	assert.Nil(t, err)

	assert.Equal(t, StringValue(config.Email.Host), "smtp.me.com")
	assert.Equal(t, IntValue(config.Email.Port), 25)
	assert.Nil(t, config.Email.User)
	assert.Nil(t, config.Email.Pass)
	assert.Equal(t, BoolValue(config.Email.StartTLS), false)
	assert.Equal(t, BoolValue(config.Email.InsecureSkipVerify), true)
	assert.Equal(t, BoolValue(config.Email.SSL), false)
	assert.Equal(t, StringValue(config.Email.From), "me@me.com")
	assert.Contains(t, config.Email.To, "you@me.com")
	assert.Equal(t, StringValue(config.Email.HistorySubject), "Resync History")
	assert.Len(t, config.Email.To, 1)
	assert.Nil(t, config.Email.HistorySchedule)
	assert.Equal(t, BoolValue(config.Email.OnFailure), false)
}

func TestMissingConfig(t *testing.T) {
	_, err := OpenConfig("./testdata/missing.yaml")
	assert.Error(t, err)
}
