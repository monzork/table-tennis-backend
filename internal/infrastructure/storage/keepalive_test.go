package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestKeepAlive(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	calls := 0
	failing := false
	k := NewKeepAlive(func(context.Context) error {
		calls++
		if failing {
			return errors.New("down")
		}
		return nil
	})
	k.now = func() time.Time { return now }
	k.rnd = func(max time.Duration) time.Duration { return max - 1 } // latest possible draw

	if pinged, err := k.Run(context.Background()); !pinged || err != nil {
		t.Fatalf("first call must ping: %v %v", pinged, err)
	}
	now = now.Add(47 * time.Hour)
	if pinged, _ := k.Run(context.Background()); pinged || calls != 1 {
		t.Fatal("must not ping again before the minimum interval")
	}
	now = now.Add(keepAliveSpread + time.Hour) // past min+spread
	failing = true
	if pinged, err := k.Run(context.Background()); pinged || err == nil {
		t.Fatal("a failed ping must surface the error")
	}
	failing = false
	if pinged, _ := k.Run(context.Background()); !pinged {
		t.Fatal("must retry right after a failure")
	}
}
