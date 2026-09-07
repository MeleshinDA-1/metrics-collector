package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fastPolicy() Policy {
	return NewPolicy(time.Millisecond, time.Millisecond, time.Millisecond)
}

func TestDefaultPolicyMatchesIncrementPolicy(t *testing.T) {
	want := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	got := DefaultPolicy().intervals

	if len(got) != len(want) {
		t.Fatalf("intervals count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("intervals[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestDoReturnsAfterFirstSuccess(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), func(context.Context) error {
		calls++
		return nil
	}, func(error) bool { return true })

	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoRetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	}, func(error) bool { return true })

	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestDoGivesUpAfterThreeExtraAttempts(t *testing.T) {
	calls := 0
	wantErr := errors.New("temporary")
	err := fastPolicy().Do(context.Background(), func(context.Context) error {
		calls++
		return wantErr
	}, func(error) bool { return true })

	if !errors.Is(err, wantErr) {
		t.Fatalf("Do returned %v, want %v", err, wantErr)
	}
	if calls != 4 {
		t.Fatalf("calls = %d, want 4 (initial attempt + 3 retries)", calls)
	}
}

func TestDoDoesNotRetryNonRetriableError(t *testing.T) {
	calls := 0
	err := fastPolicy().Do(context.Background(), func(context.Context) error {
		calls++
		return errors.New("fatal")
	}, func(error) bool { return false })

	if err == nil {
		t.Fatal("Do returned nil error, want non-nil error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDoStopsWaitingOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	policy := NewPolicy(time.Hour, time.Hour, time.Hour)
	operationErr := errors.New("temporary")

	calls := 0
	done := make(chan error, 1)
	go func() {
		done <- policy.Do(ctx, func(context.Context) error {
			calls++
			cancel()
			return operationErr
		}, func(error) bool { return true })
	}()

	select {
	case err := <-done:
		if !errors.Is(err, operationErr) || !errors.Is(err, context.Canceled) {
			t.Fatalf("Do returned %v, want both the operation error and context.Canceled", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Do kept waiting after the context was cancelled")
	}
}
