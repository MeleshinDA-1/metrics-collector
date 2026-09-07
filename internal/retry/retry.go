package retry

import (
	"context"
	"errors"
	"time"
)

type Policy struct {
	intervals []time.Duration
}

func NewPolicy(intervals ...time.Duration) Policy {
	return Policy{intervals: intervals}
}

func DefaultPolicy() Policy {
	return NewPolicy(1*time.Second, 3*time.Second, 5*time.Second)
}

func (policy Policy) Do(
	ctx context.Context,
	operation func(context.Context) error,
	retriable func(error) bool,
) error {
	for attempt := 0; ; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}

		if attempt == len(policy.intervals) || !retriable(err) {
			return err
		}

		timer := time.NewTimer(policy.intervals[attempt])
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(err, ctx.Err())
		}
	}
}
