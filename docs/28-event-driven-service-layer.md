# 28 — Event-Driven Service Layer (Observer Pattern)

## 1. Problem

Current services are monolithic procedures. `authService.Register` alone does five distinct things in a fixed, synchronous sequence:

1. Validate input (duplicate email check)
2. Hash password + persist user
3. Create `user_plans` row
4. Send verification email
5. Map to response DTO

Adding any new side-effect (audit log, Slack notification, analytics event) requires editing the core service function. The same pattern appears in `urlService.Create` (validate → enforce plan limit → persist → dispatch metadata job → map response).

**Consequences:**

- Side-effects are coupled to core writes — impossible to reorder or disable without touching business logic
- Testing requires stubbing every dependency even when only one step is under test
- The "why is this here?" question is hard to answer — is email sending a business rule or a notification?

---

## 2. Solution: In-Process Event Bus

Introduce a lightweight **event bus** in `internal/event/`. Services publish a typed domain event after their core write succeeds. Side-effect handlers subscribe and execute independently.

No external broker (Kafka, RabbitMQ). No event sourcing. No new dependencies — pure Go stdlib.

---

## 3. Directory Layout

```
internal/
  event/
    bus.go              ← EventBus interface + default implementation
    events.go           ← typed event structs
    handler/
      send_verification_email.go
      enqueue_metadata_fetch.go
      (future: send_slack_notification.go, record_audit_log.go, ...)
```

---

## 4. Event Structs (`internal/event/events.go`)

```go
package event

import (
    "github.com/dauxuanhoanghung/url-shortener/internal/model"
    "github.com/google/uuid"
)

type UserRegistered struct {
    User     *model.User
    UserPlan *model.UserPlan
}

type UserLoggedIn struct {
    User     *model.User
    UserPlan *model.UserPlan
}

type EmailVerified struct {
    UserID uuid.UUID
}

type PasswordReset struct {
    UserID uuid.UUID
}

type URLCreated struct {
    URL        *model.ShortURL
    MetadataID uuid.UUID
    UserID     uuid.UUID
}

type URLDeleted struct {
    URLID  uuid.UUID
    UserID uuid.UUID
}
```

---

## 5. Event Bus Interface (`internal/event/bus.go`)

```go
package event

import (
    "context"
    "reflect"
    "sync"

    "go.uber.org/zap"
)

// HandlerFunc is the signature every subscriber must implement.
type HandlerFunc func(ctx context.Context, event any) error

// DispatchMode controls whether a handler runs in the same goroutine or a new one.
type DispatchMode int

const (
    Sync  DispatchMode = iota // blocks; error surfaces to Publish caller
    Async                     // goroutine; error is logged, not returned
)

type subscription struct {
    handler HandlerFunc
    mode    DispatchMode
}

// EventBus is the interface services depend on.
type EventBus interface {
    Publish(ctx context.Context, event any) error
    Subscribe(eventType reflect.Type, handler HandlerFunc, mode DispatchMode)
}

type bus struct {
    mu     sync.RWMutex
    subs   map[reflect.Type][]subscription
    logger *zap.Logger
}

func NewBus(logger *zap.Logger) EventBus {
    return &bus{
        subs:   make(map[reflect.Type][]subscription),
        logger: logger,
    }
}

// TypeOf is a convenience helper: event.TypeOf[event.UserRegistered]()
func TypeOf[E any]() reflect.Type {
    return reflect.TypeOf((*E)(nil)).Elem()
}

func (b *bus) Subscribe(eventType reflect.Type, handler HandlerFunc, mode DispatchMode) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.subs[eventType] = append(b.subs[eventType], subscription{handler, mode})
}

func (b *bus) Publish(ctx context.Context, ev any) error {
    b.mu.RLock()
    subs := b.subs[reflect.TypeOf(ev)]
    b.mu.RUnlock()

    for _, s := range subs {
        if s.mode == Async {
            go func(sub subscription) {
                if err := sub.handler(ctx, ev); err != nil {
                    b.logger.Error("async event handler failed",
                        zap.String("event", reflect.TypeOf(ev).Name()),
                        zap.Error(err),
                    )
                }
            }(s)
        } else {
            if err := s.handler(ctx, ev); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

## 6. Event Handlers (`internal/event/handler/`)

### `send_verification_email.go`

```go
package handler

import (
    "context"
    "fmt"

    "github.com/dauxuanhoanghung/url-shortener/internal/event"
    "github.com/dauxuanhoanghung/url-shortener/internal/mailer"
    "github.com/dauxuanhoanghung/url-shortener/internal/model"
    "github.com/dauxuanhoanghung/url-shortener/internal/repository"
    "github.com/google/uuid"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "time"
)

const verifyEmailTokenTTL = 24 * time.Hour

func SendVerificationEmail(
    tokenRepo repository.TokenRepository,
    m mailer.Mailer,
    frontendBaseURL string,
) event.HandlerFunc {
    return func(ctx context.Context, ev any) error {
        e := ev.(event.UserRegistered)
        return issueAndSendVerification(ctx, e.User, tokenRepo, m, frontendBaseURL)
    }
}

func issueAndSendVerification(
    ctx context.Context,
    user *model.User,
    tokenRepo repository.TokenRepository,
    m mailer.Mailer,
    frontendBaseURL string,
) error {
    if err := tokenRepo.InvalidateByPurpose(ctx, user.ID, model.TokenPurposeVerifyEmail); err != nil {
        return err
    }
    raw, err := generateOpaqueToken()
    if err != nil {
        return err
    }
    now := time.Now()
    if err := tokenRepo.Create(ctx, &model.Token{
        ID:        uuid.New(),
        UserID:    user.ID,
        Purpose:   model.TokenPurposeVerifyEmail,
        TokenHash: hashToken(raw),
        ExpiresAt: now.Add(verifyEmailTokenTTL),
        CreatedAt: now,
    }); err != nil {
        return err
    }
    link := frontendBaseURL + "/verify-email?token=" + raw
    return m.Send(ctx, mailer.Message{
        To:      user.Email,
        Subject: "Verify your email",
        Body:    fmt.Sprintf("Welcome! Please verify your email (valid 24h):\n\n%s", link),
    })
}

func generateOpaqueToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
    sum := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(sum[:])
}
```

### `enqueue_metadata_fetch.go`

```go
package handler

import (
    "context"

    "github.com/dauxuanhoanghung/url-shortener/internal/event"
    "github.com/dauxuanhoanghung/url-shortener/internal/worker"
)

func EnqueueMetadataFetch(w worker.MetadataWorker) event.HandlerFunc {
    return func(ctx context.Context, ev any) error {
        e := ev.(event.URLCreated)
        w.Submit(worker.MetadataJob{
            MetadataID: e.MetadataID,
            URLID:      e.URL.ID,
            ShortCode:  e.URL.ShortCode,
            UserID:     e.UserID,
            TargetURL:  e.URL.OriginalURL,
        })
        return nil
    }
}
```

---

## 7. Refactored Services

### `authService.Register` — before vs after

**Before:**

```go
func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
    existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
    if existing != nil {
        return nil, ErrEmailAlreadyExists
    }
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    // ...
    user, err := s.userRepo.Create(ctx, &model.User{...})
    userPlan, err := s.userPlanRepo.Create(ctx, user.ID, defaultPlanCode)
    if err := s.issueVerificationEmail(ctx, user); err != nil {
        _ = err  // side-effect swallowed silently
    }
    return s.generateAuthResponse(user, userPlan)
}
```

**After:**

```go
func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
    existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
    if existing != nil {
        return nil, ErrEmailAlreadyExists
    }
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    // ...
    user, err := s.userRepo.Create(ctx, &model.User{...})
    userPlan, err := s.userPlanRepo.Create(ctx, user.ID, defaultPlanCode)
    // core writes done — publish event, handlers run independently
    _ = s.bus.Publish(ctx, event.UserRegistered{User: user, UserPlan: userPlan})
    return s.generateAuthResponse(user, userPlan)
}
```

`userPlanRepo.Create` stays in the service body because `generateAuthResponse` depends on its result (`userPlan.PlanCode`). It is a required write, not a side-effect.

### `urlService.Create` — after

```go
func (s *urlService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateURLRequest, baseURL string) (*dto.URLResponse, error) {
    if err := validateURL(req.OriginalURL); err != nil {
        return nil, err
    }
    // plan limit check (required — must block before persist)
    userPlan, _ := s.userPlanRepo.GetByUserID(ctx, userID)
    plan, _ := s.planRepo.GetByCode(ctx, userPlan.PlanCode)
    count, _ := s.repo.CountByUser(ctx, userID)
    if count >= int(plan.MaxURLs) {
        return nil, ErrPlanLimitReached
    }
    // persist
    created := s.createWithRetry(ctx, userID, req.OriginalURL)
    meta, _ := s.metaRepo.Create(ctx, created.ID)
    // publish — metadata fetch is now a handler, not inline logic
    _ = s.bus.Publish(ctx, event.URLCreated{
        URL:        created,
        MetadataID: meta.ID,
        UserID:     userID,
    })
    return toURLResponse(created, nil, baseURL), nil
}
```

---

## 8. Dependency Injection (wiring)

In `cmd/server/main.go` (or a dedicated `wire.go`):

```go
bus := event.NewBus(logger)

// Auth side-effects
bus.Subscribe(
    event.TypeOf[event.UserRegistered](),
    handler.SendVerificationEmail(tokenRepo, mailer, cfg.FrontendBaseURL),
    event.Async,
)

// URL side-effects
bus.Subscribe(
    event.TypeOf[event.URLCreated](),
    handler.EnqueueMetadataFetch(metadataWorker),
    event.Async,
)

// Inject bus into services
authSvc := service.NewAuthService(userRepo, userPlanRepo, tokenRepo, jwtSecret, bus)
urlSvc  := service.NewURLService(urlRepo, metaRepo, userPlanRepo, planRepo, bus)
```

Services receive `event.EventBus` via constructor — never build the bus inside a service.

---

## 9. Sync/Async Decision Table

| Handler                            | Mode  | Reason                                             |
| ---------------------------------- | ----- | -------------------------------------------------- |
| `SendVerificationEmail`            | Async | Delivery failure must not block registration       |
| `EnqueueMetadataFetch`             | Async | Fire-and-forget, already goroutine in current code |
| `RecordAuditLog` (future)          | Sync  | Audit record must exist before response returns    |
| `StripeWebhook → upgrade` (future) | Sync  | Must succeed or rollback the transaction           |
| `SlackNotify` (future)             | Async | Pure notification, failure is acceptable           |

---

## 10. Testing

### Bus test double

```go
// In tests, use a synchronous no-op bus to isolate service logic:
type fakeBus struct{ published []any }

func (f *fakeBus) Publish(_ context.Context, ev any) error {
    f.published = append(f.published, ev)
    return nil
}
func (f *fakeBus) Subscribe(_ reflect.Type, _ event.HandlerFunc, _ event.DispatchMode) {}
```

### Handler tests

Each handler is a standalone function. Test it directly:

```go
func TestSendVerificationEmail(t *testing.T) {
    tokenRepo := &fakeTokenRepo{}
    m := &fakeMailer{}
    h := handler.SendVerificationEmail(tokenRepo, m, "http://localhost:5173")
    err := h(context.Background(), event.UserRegistered{User: testUser, UserPlan: testPlan})
    // assert tokenRepo received Create call, mailer received Send call
}
```

Handler tests need no HTTP, no service, no database.

---

## 11. What Does NOT Change

- Handler (HTTP layer), repository, DTO, and model packages — untouched
- Error contracts (`internal/service/` sentinel errors) — unchanged
- No new external dependencies
- No breaking API changes

---

## 12. Implementation Order

1. `internal/event/events.go` — event structs
2. `internal/event/bus.go` — bus interface + implementation
3. `internal/event/handler/send_verification_email.go`
4. `internal/event/handler/enqueue_metadata_fetch.go`
5. Refactor `authService` — add `bus` field, replace `issueVerificationEmail` call with `Publish`
6. Refactor `urlService` — add `bus` field, replace inline worker dispatch with `Publish`
7. Wire bus + subscriptions in `cmd/server/main.go`
8. Delete dead helpers from `auth_service.go` (`issueVerificationEmail`, `issuePasswordResetEmail`, `issueTokenAndSend`, token helpers — moved to event handler package)
9. Add `fakeBus` to `service/testutil_test.go`, update existing service tests
