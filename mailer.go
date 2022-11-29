package resync

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"sync"
	"text/template"

	gomail "gopkg.in/gomail.v2"
)

// Mailer sends emails based on stats saved in the DB.
type Mailer struct {
	config *Config
	db     DB
	logger Logger
	mu     sync.Mutex
}

// NewMailer creates a Mailer using config, db, and logger.
func NewMailer(config *Config, db DB, logger Logger) *Mailer {
	return &Mailer{
		config: config,
		db:     db,
		logger: logger,
		mu:     sync.Mutex{},
	}
}

// Mail sends a single email for the stat based on the configured email settings.
func (m *Mailer) Mail(stat Stat) error {
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
			return err
		}

		_, err = io.Copy(w, stdout)
		return err
	}))

	message.Attach("stderr.log", gomail.SetCopyFunc(func(w io.Writer) error {
		stderr, err := m.logger.Stderr(stat.Name)
		if err != nil {
			return err
		}

		_, err = io.Copy(w, stderr)
		return err
	}))

	return m.send(message)
}

// Mail sends status email for the stats in the db based on the configured email settings.
func (m *Mailer) MailStats() error {
	if IntValue(m.config.Retention) < 1 || m.config.Email == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	message := gomail.NewMessage()
	message.SetHeader("From", StringValue(m.config.Email.From))
	message.SetHeader("To", m.config.Email.To...)
	message.SetHeader("Subject", "Resync: Statistics")

	zipFile, err := m.logger.Zip()
	if err != nil {
		return err
	}
	defer os.Remove(zipFile.Name())
	message.Attach(zipFile.Name())

	stats, err := m.db.List()
	if err != nil {
		return err
	}

	formatted := make(map[string][]Stat)

	for _, stat := range stats {
		formatted[stat.Name] = append(formatted[stat.Name], stat)
	}

	tmpl := template.New("history")
	emailTmpl, err := tmpl.Parse(emailTemplate)
	if err != nil {
		return err
	}

	var tpl bytes.Buffer
	if err := emailTmpl.Execute(&tpl, formatted); err != nil {
		return err
	}

	message.SetBody("text/html", tpl.String())

	return m.send(message)
}

func (m *Mailer) send(message *gomail.Message) error {
	dialer := gomail.NewDialer(
		StringValue(m.config.Email.Host),
		IntValue(m.config.Email.Port),
		StringValue(m.config.Email.User),
		StringValue(m.config.Email.Pass),
	)

	if BoolValue(m.config.Email.StartTLS) {
		dialer.TLSConfig = &tls.Config{
			ServerName: StringValue(m.config.Email.Host),
		}
	}
	dialer.SSL = BoolValue(m.config.Email.SSL)

	return dialer.DialAndSend(message)
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
	</thead>
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
{{ end }}`
