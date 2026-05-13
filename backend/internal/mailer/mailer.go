package mailer

import "context"

type Message struct {
	To      string
	Subject string
	Body    string // plain text for now; HTML added when a real provider lands
}

// Mailer is a transport-agnostic interface. Services depend on this,
// never on a concrete provider. See docs/24-user-account-lifecycle.md §5.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
