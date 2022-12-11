package resync

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"text/template"

	gomail "gopkg.in/gomail.v2"
)

// EmailNotifier sends emails based on stats saved in the DB.
type EmailNotifier struct {
	config *Config
	db     DB
	logger Logger
	mu     sync.Mutex
}

// NewEmailNotifier creates a Mailer using config, db, and logger.
func NewEmailNotifier(config *Config, db DB, logger Logger) *EmailNotifier {
	return &EmailNotifier{
		config: config,
		db:     db,
		logger: logger,
		mu:     sync.Mutex{},
	}
}

// Notify sends a single email for the stat based on the configured email settings.
func (m *EmailNotifier) Notify(stat Stat) error {
	if IntValue(m.config.Retention) < 1 || m.config.Email == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	message := gomail.NewMessage()
	message.SetHeader("From", StringValue(m.config.Email.From))
	message.SetHeader("To", m.config.Email.To...)

	if stat.Success {
		message.SetHeader("Subject", fmt.Sprintf("Resync: Sync %s Complete", stat.Name))
	} else {
		message.SetHeader("Subject", fmt.Sprintf("Resync: Sync %s Failed", stat.Name))
	}

	message.Attach("stdout.log", gomail.SetCopyFunc(func(w io.Writer) error {
		stdout, err := m.logger.Stdout(stat.Name)
		if err != nil {
			return fmt.Errorf("Mailer: failed to read stdout log: %w", err)
		}

		_, err = io.Copy(w, stdout)
		if err != nil {
			return fmt.Errorf("Mailer: failed to attach stdout log: %w", err)
		}
		return nil
	}))

	message.Attach("stderr.log", gomail.SetCopyFunc(func(w io.Writer) error {
		stderr, err := m.logger.Stderr(stat.Name)
		if err != nil {
			return fmt.Errorf("Mailer: failed to read stderr log: %w", err)
		}

		_, err = io.Copy(w, stderr)
		if err != nil {
			return fmt.Errorf("Mailer: failed to attach stderr log: %w", err)
		}
		return nil
	}))

	return m.send(message)
}

// NotifyHistory sends status email for the stats in the db based on the configured email settings.
func (m *EmailNotifier) NotifyHistory() error {
	if IntValue(m.config.Retention) < 1 || m.config.Email == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stats, err := m.db.List()
	if err != nil {
		return fmt.Errorf("Mailer: failed to get list from stats db: %w", err)
	}

	if len(stats) == 0 {
		return errors.New("Mailer: no stats found when trying to send stats email")
	}

	message := gomail.NewMessage()
	message.SetHeader("From", StringValue(m.config.Email.From))
	message.SetHeader("To", m.config.Email.To...)
	message.SetHeader("Subject", StringValue(m.config.Email.HistorySubject))

	zipFile, err := m.logger.Zip()
	if err != nil {
		return fmt.Errorf("Mailer: failed to get zip from logger: %w", err)
	}
	defer os.Remove(zipFile.Name())
	message.Attach(zipFile.Name())

	var emailTmpl *template.Template
	if m.config.Email.HistoryTemplate != nil {
		emailTmpl, err = template.ParseFiles(StringValue(m.config.Email.HistoryTemplate))
		if err != nil {
			return fmt.Errorf("Mailer: failed to parse custom email template %s: %w", StringValue(m.config.Email.HistoryTemplate), err)
		}
	} else {
		tmpl := template.New("history")
		emailTmpl, err = tmpl.Parse(emailTemplate)
		if err != nil {
			return fmt.Errorf("Mailer: failed to parse email template: %w", err)
		}
	}

	var tpl bytes.Buffer
	if err := emailTmpl.Execute(&tpl, stats); err != nil {
		return fmt.Errorf("Mailer: failed to execute email template: %w", err)
	}

	message.SetBody("text/html", tpl.String())

	return m.send(message)
}

func (m *EmailNotifier) send(message *gomail.Message) error {
	dialer := gomail.NewDialer(
		StringValue(m.config.Email.Host),
		IntValue(m.config.Email.Port),
		StringValue(m.config.Email.User),
		StringValue(m.config.Email.Pass),
	)

	if BoolValue(m.config.Email.StartTLS) {
		dialer.TLSConfig = &tls.Config{
			ServerName:         StringValue(m.config.Email.Host),
			InsecureSkipVerify: BoolValue(m.config.Email.InsecureSkipVerify),
		}
	}
	dialer.SSL = BoolValue(m.config.Email.SSL)

	err := dialer.DialAndSend(message)
	if err != nil {
		return fmt.Errorf("Mailer: failed to send email: %w", err)
	}
	return nil
}

var emailTemplate = `<html>
<head>

<style type="text/css">
.tg {
  border-collapse:separate;
  border-spacing:0;
  border-radius:10px;
  width: 100%;
  font-family:Roboto,"Helvetica Neue",sans-serif;
}

.tg td {
  color:#444;
  font-size:14px;
  overflow:hidden;
  padding:3px 10px 3px 0px;
  word-break:normal;
  border-bottom: 1px solid;
  border-bottom-color: #BDBDBD;
  height: 40px;
}

.tg th {
  background-color:#424242;
  color:#FFFFFF;
  font-family:Roboto,"Helvetica Neue",sans-serif;
  font-size:12px;
  font-weight:bold;
  overflow:hidden;
  padding:3px 10px 3px 0px;
  word-break:normal;
  border:0;
  height: 60px;
}

.tg .tg-title {
  text-align:center;
  font-size: 14px;
}

.tg .tg-data {
  text-align:left;
}

.tg .tg-header {
  text-align:left;
  font-weight: bold;
  color: #9E9E9E;
}

.success {
  color: #2E7D32 !important;
  font-weight: bold;
}

.failure {
  color: #D32F2F !important;
  font-weight: bold;
}
</style>

</head>
<body>

{{ range $name, $stats := . }}
<table class="tg">
        <thead>
                <tr>
                        <th class="tg-title" colspan="6">{{$name}}</th>
                </tr>
        </thead>
        <tbody>
                <tr>
                        <td class="tg-header">Status</td>
                        <td class="tg-header">Start</td>
                        <td class="tg-header">End</td>
                        <td class="tg-header">Duration</td>
                </tr>
                {{ range $stats}}
                <tr>
                        {{if .Success}}
                          <td class="success">Success</td>
                        {{else}}
                          <td class="failure">Failed</td>
                        {{end}}
                        <td class="tg-data">{{.Start}}</td>
                        <td class="tg-data">{{.End}}</td>
                        <td class="tg-data">{{.Duration}}</td>
                </tr>
                {{ end}}
        </tbody>
</table>
<br>
{{ end }}

</body>
</html>`
