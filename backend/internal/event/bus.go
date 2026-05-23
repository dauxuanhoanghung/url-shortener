package event

import (
	"context"
	"reflect"
	"sync"

	"go.uber.org/zap"
)

// HandlerFunc is the signature every subscriber must implement.
type HandlerFunc func(ctx context.Context, event any) error

// DispatchMode controls whether a handler blocks or runs in a new goroutine.
type DispatchMode int

const (
	Sync  DispatchMode = iota // blocks; errors surface to Publish caller
	Async                     // goroutine; errors are logged, not returned
)

// TypeOf is a convenience helper for Subscribe calls:
//
//	bus.Subscribe(event.TypeOf[event.UserRegistered](), handler, event.Async)
func TypeOf[E any]() reflect.Type {
	return reflect.TypeOf((*E)(nil)).Elem()
}

// EventBus is the interface services depend on.
type EventBus interface {
	Publish(ctx context.Context, event any) error
	Subscribe(eventType reflect.Type, handler HandlerFunc, mode DispatchMode)
}

type subscription struct {
	handler HandlerFunc
	mode    DispatchMode
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
				// Use a fresh context so the goroutine is not cancelled when
				// the HTTP request that triggered Publish returns.
				if err := sub.handler(context.Background(), ev); err != nil {
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
