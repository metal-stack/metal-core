package db

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func flag(desc string, v *atomic.Bool) Condition {
	return NewCondition(desc, func(context.Context) (bool, string, error) {
		return v.Load(), "", nil
	})
}

func TestPending(t *testing.T) {
	var met, notMet atomic.Bool
	met.Store(true)

	withDetail := NewCondition("router interface absent", func(context.Context) (bool, string, error) {
		return false, "found oid:0x6000000000daa", nil
	})

	pending, err := Pending(t.Context(), flag("met", &met), flag("not met", &notMet), withDetail)
	require.NoError(t, err)
	require.Equal(t, []string{"not met", "router interface absent (found oid:0x6000000000daa)"}, pending)
}

func TestWaitFor(t *testing.T) {
	var met, notMet, later atomic.Bool
	met.Store(true)

	err := WaitFor(t.Context(), slog.Default(), 600*time.Millisecond, flag("met", &met), flag("not met", &notMet))
	require.EqualError(t, err, "timed out after 600ms waiting for: not met")

	go func() {
		time.Sleep(300 * time.Millisecond)
		later.Store(true)
	}()
	err = WaitFor(t.Context(), slog.Default(), 5*time.Second, flag("met later", &later))
	require.NoError(t, err)
}

func TestWaitFor_CheckError(t *testing.T) {
	failing := NewCondition("failing", func(context.Context) (bool, string, error) {
		return false, "", errors.New("connection refused")
	})

	start := time.Now()
	err := WaitFor(t.Context(), slog.Default(), 5*time.Second, failing)
	require.EqualError(t, err, "could not check failing: connection refused")
	require.Less(t, time.Since(start), time.Second, "errors must not be retried until the timeout")
}

func TestWaitFor_TimeoutDuringCheck(t *testing.T) {
	var checked atomic.Bool
	slow := NewCondition("slow", func(ctx context.Context) (bool, string, error) {
		if !checked.Swap(true) {
			return false, "", nil
		}
		<-ctx.Done()
		return false, "", ctx.Err()
	})

	err := WaitFor(t.Context(), slog.Default(), 600*time.Millisecond, slow)
	require.EqualError(t, err, "timed out after 600ms waiting for: slow")
}
