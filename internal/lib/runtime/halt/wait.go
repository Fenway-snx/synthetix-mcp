package halt

import (
	"context"
	"time"
)

type PollCheckFunc func() (bool, error)

func WaitUntil(ctx context.Context, pollInterval time.Duration, check PollCheckFunc) error {
	if pollInterval <= 0 {
		pollInterval = 10 * time.Millisecond
	}

	done, err := check()
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, err = check()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}
