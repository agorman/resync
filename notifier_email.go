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
		return fmt.Errorf("Mailer: failed to attach stderr log: %w", err)
	}))

	message.Attach("stderr.log", gomail.SetCopyFunc(func(w io.Writer) error {
		stderr, err := m.logger.Stderr(stat.Name)
		if err != nil {
			return fmt.Errorf("Mailer: failed to read stderr log: %w", err)
		}

		_, err = io.Copy(w, stderr)
		return fmt.Errorf("Mailer: failed to attach stderr log: %w", err)
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

	formatted := make(map[string][]Stat)

	for _, stat := range stats {
		formatted[stat.Name] = append(formatted[stat.Name], stat)
	}

	var emailTmpl *template.Template
	if m.config.Email.HistoryTemplate != nil {
		fmt.Println(StringValue((m.config.Email.HistoryTemplate)))
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
	if err := emailTmpl.Execute(&tpl, formatted); err != nil {
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

var emailTemplate = `<style type="text/css">
.tg  {border-collapse:collapse;border-color:#9ABAD9;border-spacing:0;}
.tg td{background-color:#EBF5FF;border-color:#9ABAD9;border-style:solid;border-width:1px;color:#444;
    font-family:Arial, sans-serif;font-size:14px;overflow:hidden;padding:10px 5px;word-break:normal;}
.tg th{background-color:#409cff;border-color:#9ABAD9;border-style:solid;border-width:1px;color:#fff;
    font-family:Arial, sans-serif;font-size:14px;font-weight:normal;overflow:hidden;padding:10px 5px;word-break:normal;}
.tg .tg-baqh{text-align:center;vertical-align:top}
.tg .tg-lqy6{text-align:right;vertical-align:top}
.tg .tg-0lax{text-align:left;vertical-align:top}
</style>
{{ range $name, $stats := . }}
<table class="tg">
	<thead>
		<tr>
			<th class="tg-baqh" colspan="6">{{$name}}</th>
		</tr>
	</thead>
	<tbody>
		<tr>
			<td class="tg-0pky">Success</td>
			<td class="tg-0pky">Start</td>
			<td class="tg-0pky">End</td>
			<td class="tg-0pky">Duration</td>
		</tr>
		{{ range $stats}}
		<tr>
			<td class="tg-0pky">{{.Success}}</td>
			<td class="tg-0pky">{{.Start}}</td>
			<td class="tg-0pky">{{.End}}</td>
			<td class="tg-0pky">{{.Duration}}</td>
		</tr>
		{{ end}}
	</tbody>
</table>
<br>
{{ end }}`