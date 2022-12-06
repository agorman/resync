package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/agorman/resync"
	"github.com/etherlabsio/healthcheck/v2"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	conf := flag.String("conf", "/etc/resync/resync.yaml", "Path to the resync configuration file")
	stats := flag.Bool("stats", false, "Print sync stats and exit")
	debug := flag.Bool("debug", false, "Log to STDOUT")
	flag.Parse()

	config, err := resync.OpenConfig(*conf)
	if err != nil {
		log.Fatal(err)
	}

	if !*debug {
		logfile := &lumberjack.Logger{
			Filename:   filepath.Join(resync.StringValue(config.LogPath), "resync.log"),
			MaxSize:    10,
			MaxBackups: 4,
		}
		log.SetOutput(logfile)
	}

	db, err := resync.NewBoltDB(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	logger := resync.NewFSLogger(config)

	notifier := resync.NewEmailNotifier(config, db, logger)

	re := resync.New(config, db, logger, notifier)

	if *stats {
		if err := re.Dump(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := re.Start(); err != nil {
		log.Fatal(err)
	}

	errc := make(chan error, 1)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// get most recent status for each sync and if any have failed then return an error
	if config.HTTP != nil {
		http.Handle("/live", healthcheck.Handler(
			healthcheck.WithTimeout(5*time.Second),
			healthcheck.WithChecker(
				"live", healthcheck.CheckerFunc(
					func(ctx context.Context) error {
						return nil
					},
				),
			),
		))

		http.Handle("/health", healthcheck.Handler(
			healthcheck.WithTimeout(5*time.Second),
			healthcheck.WithChecker(
				"health", healthcheck.CheckerFunc(
					func(ctx context.Context) error {
						statMap, err := db.List()
						if err != nil {
							return err
						}

						for name, stats := range statMap {
							if len(stats) > 0 && !stats[0].Success {
								return fmt.Errorf("One more more syncs failed including %s", name)
							}
						}

						return nil
					},
				),
			),
		))

		go func() {
			errc <- http.ListenAndServe(fmt.Sprintf("%s:%d", resync.StringValue(config.HTTP.Addr), resync.IntValue(config.HTTP.Port)), nil)
		}()
	}

	select {
	case s := <-sigc:
		log.Warnf("Received signal %s, exiting", s)
		return
	case e := <-errc:
		log.Errorf("Run error: %s", e)
		return
	}
}
