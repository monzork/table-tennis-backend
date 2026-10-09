package storage

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

const (
	keepAliveMin    = 48 * time.Hour // never ping more often than this
	keepAliveSpread = 48 * time.Hour // ...nor later than min+spread (stays under Supabase's 7-day pause)
)

// KeepAlive pings Supabase on a randomised schedule when Run is called
// (e.g. by an external scheduler hitting /health/storage): a call before the
// next due time is a no-op, so any caller frequency yields a ping every 2 to 4
// days. State is in memory, so a restart pings again on the next call.
type KeepAlive struct {
	ping func(context.Context) error
	now  func() time.Time
	rnd  func(time.Duration) time.Duration

	mu   sync.Mutex
	next time.Time
}

func NewKeepAlive(ping func(context.Context) error) *KeepAlive {
	return &KeepAlive{
		ping: ping,
		now:  time.Now,
		rnd:  func(max time.Duration) time.Duration { return time.Duration(rand.Int63n(int64(max))) },
	}
}

// Run pings Supabase if one is due and reports whether it did.
func (k *KeepAlive) Run(ctx context.Context) (pinged bool, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	now := k.now()
	if now.Before(k.next) {
		return false, nil
	}
	if err := k.ping(ctx); err != nil {
		return false, err // next stays due, so the next call retries
	}
	k.next = now.Add(keepAliveMin + k.rnd(keepAliveSpread))
	return true, nil
}
