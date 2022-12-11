package resync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"text/tabwriter"

	cron "github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

// runningSync is used to pass sync information on a channel
type runningSync struct {
	name     string
	cancel   context.CancelFunc
	runningc chan bool
}

// Resync is responsible for running rsync commands based on cron schedules.
type Resync struct {
	config   *Config
	db       DB
	logger   Logger
	notifier Notifier
	crontab  *cron.Cron
	syncs    map[string]*runningSync
	running  bool
	stopping bool
	startc   chan *runningSync
	endc     chan *runningSync
	stopc    chan struct{}
	donec    chan struct{}
}

// New creates a new Resync object.
func New(config *Config, db DB, logger Logger, notifier Notifier) *Resync {
	return &Resync{
		config:   config,
		db:       db,
		logger:   logger,
		notifier: notifier,
		syncs:    make(map[string]*runningSync),
		crontab:  cron.New(),
		startc:   make(chan *runningSync),
		endc:     make(chan *runningSync),
		stopc:    make(chan struct{}),
		donec:    make(chan struct{}),
	}
}

// Start sets up and runs the configured cron jobs.
func (re *Resync) Start() error {
	if re.running {
		return nil
	}

	// setup cron
	if BoolValue(re.config.SecondsField) {
		// Optionally create crontab with optional seconds field
		re.crontab = cron.New(cron.WithSeconds())
	} else {
		re.crontab = cron.New()
	}

	// add each cron sync job
	for name, sync := range re.config.Syncs {
		schedule := StringValue(sync.Schedule)

		// make sure variables used in different goroutine (AddFunc) aren't shadowed
		err := func(name, schedule string) error {
			_, err := re.crontab.AddFunc(schedule, func() {
				// add recovery here so entire program doesn't crash on panic
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("Panic running job %s\n%s", name, debug.Stack())
					}
				}()

				if err := re.sync(name); err != nil {
					log.Errorf("Error running job %s: %v", name, err)
				}
			})
			return err
		}(name, schedule)
		if err != nil {
			return err
		}

		log.Infof("Sync Scheduled %s: %s", name, StringValue(sync.Schedule))
	}

	// setup scheduled stats email
	if re.config.Email != nil && re.config.Email.HistorySchedule != nil {
		_, err := re.crontab.AddFunc(StringValue(re.config.Email.HistorySchedule), func() {
			// add recovery here so entire program doesn't crash on panic
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Panic in scheduled stats email: %s", debug.Stack())
				}
			}()

			if err := re.notifier.NotifyHistory(); err != nil {
				log.Error(err)
			}
		})
		if err != nil {
			return err
		}

		log.Infof("History Email Scheduled: %s", StringValue(re.config.Email.HistorySchedule))
	}

	re.running = true
	re.crontab.Start()
	go re.loop()

	return nil
}

// Stop stops running cron jobs, closes the db, and kills all running sync jobs.
func (re *Resync) Stop() {
	if re.stopping || !re.running {
		return
	}

	re.stopc <- struct{}{}
	<-re.donec

	re.stopping = false
	re.running = false
}

func (re *Resync) loop() {
	for {
		select {
		case sync := <-re.startc:
			if _, ok := re.syncs[sync.name]; ok {
				sync.runningc <- true
			} else {
				re.syncs[sync.name] = sync
				sync.runningc <- false
			}
		case sync := <-re.endc:
			delete(re.syncs, sync.name)

			if re.stopping && len(re.syncs) == 0 {
				re.donec <- struct{}{}
				return
			}
		case <-re.stopc:
			re.stopping = true
			re.crontab.Stop()
			for name, sync := range re.syncs {
				log.Infof("Cancelling running rsync: %s", name)
				sync.cancel()
			}

			if len(re.syncs) == 0 {
				re.donec <- struct{}{}
				return
			}
		}
	}
}

func (re *Resync) sync(name string) error {
	sync, err := re.config.GetSync(name)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if timeLimit, err := re.config.GetTimeLimit(name); err == nil {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(context.Background(), timeLimit)
		defer timeoutCancel()
	}

	cmd := exec.CommandContext(ctx, StringValue(re.config.RsyncPath), sync.Args()...)

	// inform main loop that we're running a sync
	rc := &runningSync{
		name:     name,
		cancel:   cancel,
		runningc: make(chan bool),
	}
	re.startc <- rc

	// check runningc to see if the sync is already running
	if running := <-rc.runningc; running {
		log.Infof("Skipping rsync %s because it's already running", name)
		return nil
	}

	// rotate logs
	stdoutLog, stderrLog, err := re.logger.Rotate(name)
	if err != nil {
		return err
	}

	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog

	log.Infof("Running %s: %s %s", name, StringValue(re.config.RsyncPath), strings.Join(sync.Args(), " "))

	stat := NewStat(name, StringValue(re.config.TimeFormat))
	err = cmd.Run()
	stat = stat.Finish(err)

	if stat.Success {
		log.Infof("Finished %s after %s", name, stat.Duration)
	} else {
		log.Errorf("Error %s: after %s: %s", name, stat.Duration, err)
	}

	if err != nil && re.config.Email != nil && BoolValue(re.config.Email.OnFailure) {
		if err := re.notifier.Notify(stat); err != nil {
			log.Error(err)
		}
	}

	if IntValue(re.config.Retention) > 0 {
		if err := re.db.Insert(stat); err != nil {
			log.Errorf("Failed to write stats for %s: %v", name, err)
		}
	}

	// inform main loop that the sync is complete
	re.endc <- rc

	return err
}

// Dump prints all of the stats to STDOUT.
func (re *Resync) Dump() error {
	if IntValue(re.config.Retention) < 1 {
		return errors.New("Unable to print stats when retention is less than 1")
	}

	stats, err := re.db.List()
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

	for _, stats := range stats {
		fmt.Fprintln(writer, "NAME\tSUCCESS\tSTART\tEND\tDURATION")
		for _, stat := range stats {
			fmt.Fprintf(writer, "%s\t%t\t%s\t%s\t%s\n", stat.Name, stat.Success, stat.Start, stat.End, stat.Duration)
		}
		fmt.Fprintln(writer)
	}

	return writer.Flush()
}
