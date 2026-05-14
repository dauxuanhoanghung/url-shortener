package mailer

import "context"

// Message is the transport-agnostic mail payload.
// Callers that only set To/Subject/Body are unaffected by new fields.
//
// TODO: add HTML template rendering (see docs/29-mailer-transports.md §7).
// Transports already handle HTML when the field is non-empty.
type Message struct {
	To      string
	Subject string
	Body    string // plain text
	HTML    string // optional; transports send multipart/alternative when set
	ReplyTo string // optional
}

// Mailer is the interface all transports implement.
// Services and event handlers depend on this, never on a concrete type.
// See docs/29-mailer-transports.md.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
