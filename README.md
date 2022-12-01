[![Build Status](https://github.com/agorman/resync/workflows/resync/badge.svg)](https://github.com/agorman/resync/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/agorman/resync)](https://goreportcard.com/report/github.com/agorman/resync)
[![GoDoc](https://godoc.org/github.com/agorman/resync?status.svg)](https://godoc.org/github.com/agorman/resync)
[![codecov](https://codecov.io/gh/agorman/resync/branch/main/graph/badge.svg)](https://codecov.io/gh/agorman/resync)

# Resync

Resync is a layer on top of rsync and cron that adds some helpful functionality.

Resync aims to be a very simple replacement for people that are already familiar with Rsync and Cron. All you need to do is create the YAML file and you're ready to go.


# Why Resync?


Rsync with Cron is a popular pattern. If that's working for you then you might not have a use for Resync. However, I ran into a few pain points that I wrote resync to address.


- Skips the next cron invocation if the previous rsync command is still running
- An optional maximum time on how long each rsync command is allowed to run
- Stores and rotates logs from each rsync invocation
- Provides historical data on each rsync invocation
- Send email notifications for failures and reports


# How does it work?


Instead of using the system crontab resync runs cron jobs internally using the same syntax. A YAML file is used to configure resync and define how and when rsync is run.


1. Download the [latest release](https://github.com/agorman/resync/releases).
2. Create a YAML configuration file
3. Run it `resync -conf resync.yaml`


# Configuration file


The YAML file defines how and when each rsync command is run.


## Minimal config example

Just define the syncs you'd like to perform and use the defaults.

~~~
syncs:
  backup:
    command: -a /home/user /mnt/backup
    schedule: 0 0 * * *
~~~

## Full config example

~~~

rsync_path: rsync
log_path: /var/log/resync
log_level: error
lib_path: /var/lib/resync
time_format: Mon Jan 02 03:04:05 PM MST
retention: 7
seconds_field: false
time_limit: 5h
email:
  host: smtp.myserver.com
  port: 587
  user: myuser
  pass: mypass
  starttls: true
  insecure_skip_verify: false
  ssl: false
  from: me@myserver.com
  to:
    - user1@myserver.com
    - user2@myserver.com
  history_subject: Resync History
  history_schedule: "* * * * *"
  history_template: "/etc/resync/resync.tmpl"
  on_failure: false
syncs:
  data:
    command: -a /data/ /mnt/backup/data/
    schedule: "0 0 * * *"
  data2:
    command: -a /data2/ /mnt/backup/data2/
    schedule: "0 2 * * *"
~~~


## Global Options


### rsync_path

    Path to the rsync binary. Defaults to rsync or rsync.exe on Windows.

### log_path

    Directory on disk where resync logs will be stored. Defaults to /var/log/resync.

### log_level

    Sets the log level. Valid levels are: panic, fatal, trace, debug, warn, info, and error. Defaults to error.

### lib_path

    Directory on disk where resync lib files will be stored. Defaults to /var/lib/resync.

### time_format

    The time format used when displaying sync stats. See formatting options in the go time.Time package. Defaults to Mon Jan 02 03:04:05 PM MST

### retention

    The number of logs and stats that are stored for each sync. Defaults to 7.

### seconds_field

    Enable the cron seconds field. This makes the first field in the cron expression handle seconds changes the expression to 6 fields. Defaults to false.

### time_limit

    The maximum amount of time that a sync job will run before being killed. TimeLimit must be a string that can be passed to the time.Duration.ParseDuration() function. Default is no time limit.


## Email


### host

    The hostname or IP of the SMTP server.

### port

    The port of the SMTP server.

### user

    The username used to authenticate.

### pass

    The password used to authenticate.

### start_tls

    StartTLS enables TLS security. If both StartTLS and SSL are true then StartTLS will be used.

### insecure_skip_verify

    When using TLS skip verifying the server's certificate chain and host name.

### ssl

    SSL enables SSL security. If both StartTLS and SSL are true then StartTLS will be used.

### from

    The email address the email will be sent from.

### to

	An array of email addresses for which emails will be sent.

### history_email

    Optional subject to use when sending sync history emails.

### history_schedule

	Defines a cron expression used to send scheduled reports. If set then an email with sync history will be sent based on the schedule.

### history_template

    Optional template to use when sending history emails. See go's html/template for details. Uses the default template if blank.

### on_failure

	Send an email for each sync failure if true.

## Syncs


### command

    The rsync command that will be run. It should be idential to an rsync command on the command line with just the rsync command itself omitted.

### schedule

    The cron expression that will be used. It should be quoted as * have a special meaning in YAML files.

### time_limit

    The maximum amount of time that a sync job will run before being killed. TimeLimit must be a string that can be passed to the time.Duration.ParseDuration() function. Default is no time limit.


# Flags


## -conf

    Path to the resync configuration file

## -debug

    Log to STDOUT


## Road Map


- Docker Image
- Systemd service file
- Create rpm
- Create deb
- More persistence and logging layers
