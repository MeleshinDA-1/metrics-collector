package retry

import "time"

var intervals = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

func Do(operation func() error, retriable func(error) bool) error {
	for attempt := 0; ; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if attempt == len(intervals) || !retriable(err) {
			return err
		}

		time.Sleep(intervals[attempt])
	}
}
