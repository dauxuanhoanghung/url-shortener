# 29 — Mailer Transports

## 1. Problem

The original mailer had a single `ConsoleMailer` that logged to stdout with no way
to configure a real transport without editing Go source. Production requires SMTP;
other environments may use a local MTA (sendmail).

---

## 2. Transport Catalogue

| Transport  | `MAIL_TRANSPORT` value                | Use case                                       |
| ---------- | ------------------------------------- | ---------------------------------------------- |
| `console`  | `console` (default when var is unset) | Local dev — prints to stdout via zap           |
| `smtp`     | `smtp`                                | Production / staging with a real SMTP relay    |
| `sendmail` | `sendmail`                            | Server with a local MTA (`/usr/sbin/sendmail`) |

### Fallback guarantee

Every non-console transport implements a `Probe()` method that is called at startup.
If the probe fails (connection refused, binary not found, bad path), the factory logs
a warning and falls back to `ConsoleMailer`. The application **always starts**.

---

## 3. Env Variables

```env
# Which transport to use. Omit or set to "console" for local dev.
MAIL_TRANSPORT=smtp

# Shared — used by all transports
MAIL_FROM=noreply@example.com
MAIL_FROM_NAME=URL Shortener

# SMTP only
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=user@example.com
SMTP_PASSWORD=secret
SMTP_TLS=true          # true = STARTTLS (port 587); false = plain (port 25)

# Sendmail only
SENDMAIL_PATH=/usr/sbin/sendmail   # must be absolute; defaults to /usr/sbin/sendmail
```

---

## 4. Directory Layout

```
internal/mailer/
  mailer.go      ← Message struct + Mailer interface
  console.go     ← ConsoleMailer (logs to stdout; dev / fallback)
  smtp.go        ← SMTPMailer   (gomail + net/smtp STARTTLS)
  sendmail.go    ← SendmailMailer (exec.CommandContext pipes to sendmail -t -oi)
  factory.go     ← New(cfg, log) — probes transport, falls back to console
```

---

## 5. Library Choice

`gopkg.in/mail.v2` (gomail) is used for SMTP instead of raw `net/smtp` because:

- Handles `multipart/alternative` (text + HTML) automatically
- Correct RFC-2047 encoding of non-ASCII display names in `From` header
- Clean STARTTLS dial in one call via `gomail.Dialer`
- Single small dependency, zero transitive deps beyond `quotedprintable`

---

## 6. Message Struct

```go
type Message struct {
    To      string
    Subject string
    Body    string  // plain text (always required)
    HTML    string  // optional — transports send multipart/alternative when set
    ReplyTo string  // optional
}
```

Existing callers set only `To/Subject/Body`; zero values for new fields are safe.

**TODO**: add HTML template rendering. Event handlers (`SendVerificationEmail`,
`SendPasswordResetEmail`) currently send plain text. Wire up Go `html/template`
or a template-per-event approach and populate `msg.HTML` before publishing.

---

## 7. Config Struct

`MailerConfig` in `internal/config/config.go`:

```go
type MailerConfig struct {
    Transport    string // "console" | "smtp" | "sendmail"
    From         string
    FromName     string
    SMTPHost     string
    SMTPPort     int
    SMTPUsername string
    SMTPPassword string
    SMTPTLS      bool   // true = STARTTLS
    SendmailPath string // must be absolute path
}
```

Loaded via env in `config.Load()` with `getEnvInt` / `getEnvBool` helpers.

---

## 8. Factory and Probe Pattern

```go
// factory.go
func New(cfg config.MailerConfig, log *zap.Logger) Mailer {
    m, label := build(cfg, log)        // select transport
    if p, ok := m.(prober); ok {
        if err := p.Probe(); err != nil {
            log.Warn("mailer: transport probe failed, falling back to console", ...)
            return NewConsoleMailer(log)
        }
    }
    log.Info("mailer: transport ready", zap.String("transport", label))
    return m
}
```

Each non-console transport implements:

```go
type prober interface {
    Probe() error
}
```

- **SMTPMailer.Probe** — dials `SMTP_HOST:SMTP_PORT`, attempts STARTTLS if configured
- **SendmailMailer.Probe** — verifies the path is absolute and the binary exists via `exec.LookPath`

---

## 9. Security Notes

- SMTP password comes from env only — never logged, never in error strings
- `gomail.Dialer` validates the server TLS certificate (no `InsecureSkipVerify`)
- Sendmail path must be **absolute**; relative paths are rejected at probe time and at `Send` time
- `exec.CommandContext` is used so a server shutdown cancels in-flight sendmail processes

---

## 10. Adding a New Transport (e.g. SendGrid API)

1. Create `internal/mailer/sendgrid.go` implementing `Mailer` (and optionally `prober`)
2. Add a `case "sendgrid":` branch in `factory.go`
3. Add config fields to `MailerConfig` and the corresponding env vars
4. Update this doc and the env-var table in §3

No other files need to change.
