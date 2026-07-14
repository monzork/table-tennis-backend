package tournaments

import (
	"context"
	"sync"
)

type Tournament interface {
	EventName() string
}

type EventHandler func(ctx context.Context, tournament Tournament) error

type Dispatcher interface {
	Subscribe(eventName string, handler EventHandler)
	Dispatch(ctx context.Context, tournament Tournament) error
	DispatchAsync(ctx context.Context, tournament Tournament)
}

type InMemoryDispatcher struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

func NewInMemoryDispatcher() *InMemoryDispatcher {
	return &InMemoryDispatcher{
		handlers: make(map[string][]EventHandler),
	}
}

func (d *InMemoryDispatcher) Subscribe(eventName string, handler EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventName] = append(d.handlers[eventName], handler)
}

func (d *InMemoryDispatcher) Dispatch(ctx context.Context, tournament Tournament) error {
	d.mu.RLock()
	handlers := d.handlers[tournament.EventName()]
	d.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, tournament); err != nil {
			return err
		}
	}
	return nil
}

func (d *InMemoryDispatcher) DispatchAsync(ctx context.Context, tournament Tournament) {
	// Use context.Background() for async to detach from the incoming HTTP request context
	// so the background task doesn't get cancelled if the client disconnects.
	bgCtx := context.Background()
	go func() {
		_ = d.Dispatch(bgCtx, tournament)
	}()
}
