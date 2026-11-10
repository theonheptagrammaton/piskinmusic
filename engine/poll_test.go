package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Synthetic cancellation checks: no browser, credentials or subscription.
func TestPollStopsWhenContextCanceled(t *testing.T) {
	for _, immediate := range []bool{true, false} {
		name := "during retry"
		if immediate {
			name = "before poll"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			want := context.DeadlineExceeded
			if immediate {
				cancel()
				want = context.Canceled
			}
			start := time.Now()
			err := poll(ctx, "false", time.Second, "login")
			if !errors.Is(err, want) {
				t.Fatalf("poll error = %v, want %v", err, want)
			}
			if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
				t.Fatalf("cancelled poll took %v", elapsed)
			}
		})
	}
}
