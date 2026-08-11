package retry

import (
	"errors"
	"testing"
	"time"
)

func swapIntervals(t *testing.T) {
	t.Helper()

	original := intervals
	intervals = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { intervals = original })
}

func TestIntervalsMatchIncrementPolicy(t *testing.T) {
	want := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	if len(intervals) != len(want) {
		t.Fatalf("intervals count = %d, want %d", len(intervals), len(want))
	}
	for i := range want {
		if intervals[i] != want[i] {
			t.Fatalf("intervals[%d] = %v, want %v", i, intervals[i], want[i])
		}
	}
}

func TestDoReturnsAfterFirstSuccess(t *testing.T) {
	swapIntervals(t)

	calls := 0
	err := Do(func() error {
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
	swapIntervals(t)

	calls := 0
	err := Do(func() error {
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
	swapIntervals(t)

	calls := 0
	wantErr := errors.New("temporary")
	err := Do(func() error {
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
	swapIntervals(t)

	calls := 0
	err := Do(func() error {
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
