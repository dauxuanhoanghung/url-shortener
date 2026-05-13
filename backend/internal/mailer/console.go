package mailer

import (
	"context"

	"go.uber.org/zap"
)

// ConsoleMailer "sends" by logging the rendered message. Dev-only — lets
// you copy the verification link from stdout without configuring SMTP.
type ConsoleMailer struct {
	log *zap.Logger
}

func NewConsoleMailer(log *zap.Logger) *ConsoleMailer {
	return &ConsoleMailer{log: log}
}

func (m *ConsoleMailer) Send(_ context.Context, msg Message) error {
	m.log.Info("mailer: console send",
		zap.String("to", msg.To),
		zap.String("subject", msg.Subject),
		zap.String("body", msg.Body),
	)
	return nil
}
