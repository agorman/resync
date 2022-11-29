package resync

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

// Config is an object representation of the YAML configuration file. Config is read from in multiple goroutines and
// should not be written after the NewConfig is called or data races will happen. Config is not thread safe and should
// be considered read only after being passed to any other object. Writing any of it's fields will cause a data race.
type Config struct {
	// RsyncPath is the path to the rsync binary. Defaults to rsync or rsync.exe on Windows.
	RsyncPath *string `yaml:"rsync_path"`

	// LogPath is the directory on disk where resync logs will be stored. Defaults to /var/log/resync.
	LogPath *string `yaml:"log_path"`

	// LogLevel sets the level of logging. Valid levels are: panic, fatal, trace, debug, warn, info, and error. Defaults to error
	LogLevel *string `yaml:"log_level"`

	// LibPath is the directory on disk where resync lib files are stored. Defaults to /var/lib/resync.
	LibPath *string `yaml:"lib_path"`

	// The time format used when displaying sync stats. See formatting options in the go time.Time package.
	// Defaults to Mon Jan 02 03:04:05 PM MST
	TimeFormat *string `yaml:"time_format"`

	// Retention is the number of logs and stats that are stored for each sync. If set to less than 1 no
	// logs or are stats are saved. Defaults to 7.
	Retention *int `yaml:"retention"`

	// Enable the cron seconds field. This makes the first field in the cron expression handle seconds
	// changes the expression to 6 fields. Defaults to false.
	SecondsField *bool `yaml:"seconds_field"`

	// TimeLimit is the maximum amount of time that a sync job will run before being killed. TimeLimit
	// must be a string that can be passed to the time.Duration.ParseDuration() function. Default is no
	// time limit.
	TimeLimit *string `yaml:"time_limit"`

	Email     *Email           `yaml:"email"`
	Syncs     map[string]*Sync `yaml:"syncs"`
	timeLimit time.Duration
}

// GetSync returns the Sync object by name. Otherwise it returns an error.
func (c *Config) GetSync(name string) (*Sync, error) {
	sync, ok := c.Syncs[name]
	if !ok {
		return nil, fmt.Errorf("Sync doesn't exist with name: %s", name)
	}

	return sync, nil
}

// GetTimeLimit returns the time limit for name if it exists. If it doesn't exist it then tries to return the
// global time limit. If that also doesn't exist then this method returns an error.
func (c *Config) GetTimeLimit(name string) (time.Duration, error) {
	if sync, err := c.GetSync(name); err == nil {
		if sync.TimeLimit != nil {
			return sync.timeLimit, nil
		}
	}

	if c.TimeLimit != nil {
		return c.timeLimit, nil
	}

	return c.timeLimit, fmt.Errorf("time_limit undefined for %s and no global time_limit is set", name)
}

// validate both validates the configuration and sets the default options.
func (c *Config) validate() error {
	if c.RsyncPath == nil {
		os := runtime.GOOS
		switch os {
		case "windows":
			c.RsyncPath = String("rsync.exe")
		default:
			c.RsyncPath = String("rsync")
		}
	}

	if c.LogPath == nil {
		c.LogPath = String("/var/log/resync")
	}

	if c.LogLevel == nil {
		log.SetLevel(log.ErrorLevel)
	} else {
		switch StringValue(c.LogLevel) {
		case "panic":
			log.SetLevel(log.PanicLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "trace":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			return fmt.Errorf("Invalid log_level: %s", StringValue(c.LogLevel))
		}
	}

	if c.LibPath == nil {
		c.LibPath = String("/var/lib/resync")
	}

	if c.TimeFormat == nil {
		c.TimeFormat = String("Mon Jan 02 03:04:05 PM MST")
	}

	if c.Retention == nil {
		c.Retention = Int(7)
	}

	if c.SecondsField == nil {
		c.SecondsField = Bool(false)
	}

	if c.TimeLimit != nil {
		var err error
		c.timeLimit, err = time.ParseDuration(StringValue(c.TimeLimit))
		if err != nil {
			return err
		}
	}

	if c.Email != nil {
		if c.Email.Host == nil {
			return errors.New("Missing host entry for smtp")
		}

		if c.Email.Port == nil {
			c.Email.Port = Int(25)
		}

		if c.Email.StartTLS == nil {
			c.Email.StartTLS = Bool(false)
		}

		if c.Email.SSL == nil {
			c.Email.SSL = Bool(false)
		}

		// StartTLS takes presidence over SSL
		if BoolValue(c.Email.StartTLS) {
			c.Email.SSL = Bool(false)
		}

		if c.Email.From == nil {
			return errors.New("Missing from entry for smtp")
		}

		if len(c.Email.To) == 0 {
			return errors.New("Missing to entry for smtp")
		}

		if c.Email.OnFailure == nil {
			c.Email.OnFailure = Bool(false)
		}
	}

	if len(c.Syncs) == 0 {
		return errors.New("Resync configuration file doesn't contain any sync entries")
	}

	for name, sync := range c.Syncs {
		if sync.Command == nil {
			return fmt.Errorf("Missing command entry for sync: %s", name)
		}

		if sync.Schedule == nil {
			return fmt.Errorf("Missing schedule entry for sync: %s", name)
		}

		if sync.TimeLimit != nil {
			var err error
			sync.timeLimit, err = time.ParseDuration(StringValue(sync.TimeLimit))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Sync defines a single rsync command, cron expression, and other related options.
type Sync struct {
	// Command is the rsync command that will be run. It should be idential to an rsync command on the command
	// line with just the rsync command itself omitted.
	Command *string `yaml:"command"`

	// Schedule is the cron expresion for this sync.
	Schedule *string `yaml:"schedule"`

	// TimeLimit is the maximum amount of time that a sync job will run before being killed. TimeLimit
	// must be a string that can be passed to the time.Duration.ParseDuration() function.
	TimeLimit *string `yaml:"time_limit"`
	timeLimit time.Duration
}

// Email defines the SMTP configuration options needed when sending email notifications.
type Email struct {
	// Host is the hostname or IP of the SMTP server.
	Host *string `yaml:"host"`

	// Port is the port of the SMTP server.
	Port *int `yaml:"port"`

	// User is the username used to authenticate.
	User *string `yaml:"user"`

	// Pass is the password used to authenticate.
	Pass *string `yaml:"pass"`

	// StartTLS enables TLS security. If both StartTLS and SSL are true then StartTLS will be used.
	StartTLS *bool `yaml:"starttls"`

	// SSL enables SSL security. If both StartTLS and SSL are true then StartTLS will be used.
	SSL *bool `yaml:"ssl"`

	// From is the email address the email will be sent from.
	From *string `yaml:"from"`

	// To is an array of email addresses for which emails will be sent.
	To []string `yaml:"to"`

	// Schedule is a cron expression. If set then an email with sync history will be sent based on the schedule.
	Schedule *string `yaml:"schedule"`

	// OnFailure will send an email for each sync failure if true.
	OnFailure *bool `yaml:"on_failure"`
}

// OpenConfig returns a new Config option by reading the YAML file at path. If the file
// doesn't exist, can't be read, is invalid YAML, or doesn't match the resync spec then
// an error is returned.
func OpenConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	config := new(Config)
	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	return config, config.validate()
}
