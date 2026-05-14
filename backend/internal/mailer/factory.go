package mailer

import (
	"github.com/dauxuanhoanghung/url-shortener/internal/config"
	"go.uber.org/zap"
)

// prober is implemented by transports that can check their own connectivity
// before being put into service.
type prober interface {
	Probe() error
}

// New selects a transport based on cfg.Transport, probes it, and falls back
// to ConsoleMailer if the probe fails. This guarantees the application always
// starts even when SMTP is misconfigured.
//
// Selection priority:
//  1. cfg.Transport value ("smtp", "sendmail", "console")
//  2. console — automatic fallback if anything above fails its probe
func New(cfg config.MailerConfig, log *zap.Logger) Mailer {
	m, label := build(cfg, log)

	if p, ok := m.(prober); ok {
		if err := p.Probe(); err != nil {
			log.Warn("mailer: transport probe failed, falling back to console",
				zap.String("transport", label),
				zap.Error(err),
			)
			return NewConsoleMailer(log)
		}
	}

	log.Info("mailer: transport ready", zap.String("transport", label))
	return m
}

func build(cfg config.MailerConfig, log *zap.Logger) (Mailer, string) {
	switch cfg.Transport {
	case "smtp":
		return NewSMTPMailer(cfg), "smtp"
	case "sendmail":
		return NewSendmailMailer(cfg), "sendmail"
	case "console", "":
		return NewConsoleMailer(log), "console"
	default:
		log.Warn("mailer: unknown MAIL_TRANSPORT, falling back to console",
			zap.String("transport", cfg.Transport),
		)
		return NewConsoleMailer(log), "console"
	}
}
