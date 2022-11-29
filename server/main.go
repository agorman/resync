package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/agorman/resync"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	conf := flag.String("conf", "/etc/resync/resync.yaml", "Path to the resync configuration file")
	stats := flag.Bool("stats", false, "Print sync stats and exit")
	debug := flag.Bool("debug", false, "Log to STDOUT")
	flag.Parse()

	re := build(*conf, *debug)

	if *stats {
		if err := re.Dump(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := re.Start(); err != nil {
		log.Fatal(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Block until a signal is received.
	for sig := range sigc {
		switch sig {
		case syscall.SIGHUP:
			// restart rather then exit on SIGHUP
			re.Stop()
			re = build(*conf, *debug)
			if err := re.Start(); err != nil {
				log.Fatal(err)
			}
		default:
			fmt.Printf("Got signal %s: stopping\n", sig)
			re.Stop()
			return
		}
	}
}

func build(path string, debug bool) *resync.Resync {
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

	mailer := resync.NewMailer(config, db, logger)

	return resync.New(config, db, logger, mailer)
}
