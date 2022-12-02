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

	re, config := build(*conf, *debug)

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
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if config.HTTP != nil {
		http.Handle("/healthcheck", healthcheck.Handler(
			healthcheck.WithTimeout(5*time.Second),
			healthcheck.WithChecker(
				"ready", healthcheck.CheckerFunc(
					func(ctx context.Context) error {
						return nil
					},
				),
			),
		))

		go func() {
			errc <- http.ListenAndServe(fmt.Sprintf("%s:%d", resync.StringValue(config.HTTP.Addr), resync.IntValue(config.HTTP.Port)), nil)
		}()
	}

	for {
		select {
		case sig := <-sigc:
			switch sig {
			case syscall.SIGHUP:
				// restart rather then exit on SIGHUP
				re.Stop()
				re, _ = build(*conf, *debug)
				if err := re.Start(); err != nil {
					log.Fatal(err)
				}
			default:
				fmt.Printf("Got signal %s: stopping\n", sig)
				re.Stop()
				return
			}
		case err := <-errc:
			log.Errorf("Run error: %s", err)
			re.Stop()
			return
		}
	}
}

func build(path string, debug bool) (*resync.Resync, *resync.Config) {
	config, err := resync.OpenConfig(path)
	if err != nil {
		log.Fatal(err)
	}

	if !debug {
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

	logger := resync.NewFSLogger(config)

	notifier := resync.NewEmailNotifier(config, db, logger)

	return resync.New(config, db, logger, notifier), config
}
