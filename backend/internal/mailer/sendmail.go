package mailer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dauxuanhoanghung/url-shortener/internal/config"
)

type sendmailMailer struct {
	cfg config.MailerConfig
}

func NewSendmailMailer(cfg config.MailerConfig) Mailer {
	return &sendmailMailer{cfg: cfg}
}

// Probe checks that the sendmail binary exists and is executable.
func (m *sendmailMailer) Probe() error {
	if !filepath.IsAbs(m.cfg.SendmailPath) {
		return fmt.Errorf("sendmail: path must be absolute, got %q", m.cfg.SendmailPath)
	}
	_, err := exec.LookPath(m.cfg.SendmailPath)
	return err
}

func (m *sendmailMailer) Send(ctx context.Context, msg Message) error {
	if !filepath.IsAbs(m.cfg.SendmailPath) {
		return fmt.Errorf("sendmail: path must be absolute, got %q", m.cfg.SendmailPath)
	}

	raw := m.buildRaw(msg)

	cmd := exec.CommandContext(ctx, m.cfg.SendmailPath, "-t", "-oi")
	cmd.Stdin = bytes.NewBufferString(raw)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sendmail: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *sendmailMailer) buildRaw(msg Message) string {
	var b strings.Builder
	from := m.cfg.From
	if m.cfg.FromName != "" {
		from = fmt.Sprintf("%q <%s>", m.cfg.FromName, m.cfg.From)
	}
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	if msg.ReplyTo != "" {
		b.WriteString("Reply-To: " + msg.ReplyTo + "\r\n")
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	return b.String()
}
