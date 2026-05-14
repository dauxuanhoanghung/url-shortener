package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/dauxuanhoanghung/url-shortener/internal/config"
	gomail "gopkg.in/mail.v2"
)

type smtpMailer struct {
	cfg config.MailerConfig
}

func NewSMTPMailer(cfg config.MailerConfig) Mailer {
	return &smtpMailer{cfg: cfg}
}

// Probe dials the configured SMTP server and returns any connection error.
// Used by the factory to decide whether to fall back to console.
func (m *smtpMailer) Probe() error {
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	if m.cfg.SMTPTLS {
		c, err := smtp.Dial(addr)
		if err != nil {
			return err
		}
		defer c.Close()
		tlsCfg := &tls.Config{ServerName: m.cfg.SMTPHost} //nolint:gosec
		return c.StartTLS(tlsCfg)
	}
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	c.Close()
	return nil
}

func (m *smtpMailer) Send(_ context.Context, msg Message) error {
	gm := gomail.NewMessage()

	from := m.cfg.From
	if m.cfg.FromName != "" {
		from = gm.FormatAddress(m.cfg.From, m.cfg.FromName)
	}
	gm.SetHeader("From", from)
	gm.SetHeader("To", msg.To)
	gm.SetHeader("Subject", msg.Subject)
	if msg.ReplyTo != "" {
		gm.SetHeader("Reply-To", msg.ReplyTo)
	}

	if msg.HTML != "" {
		gm.SetBody("text/plain", msg.Body)
		gm.AddAlternative("text/html", msg.HTML)
	} else {
		gm.SetBody("text/plain", msg.Body)
	}

	var dialer *gomail.Dialer
	if m.cfg.SMTPTLS {
		dialer = gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUsername, m.cfg.SMTPPassword)
		dialer.TLSConfig = &tls.Config{ServerName: m.cfg.SMTPHost} //nolint:gosec
	} else {
		dialer = gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUsername, m.cfg.SMTPPassword)
		dialer.SSL = false
	}

	return dialer.DialAndSend(gm)
}
