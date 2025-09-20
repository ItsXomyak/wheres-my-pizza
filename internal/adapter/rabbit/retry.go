package rabbit

import (
	"context"
	"time"
)

// retry funciton to attempt function multiple times
func retry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var err error

	for i := range attempts {
		if err := fn(); err == nil {
			return nil
		}

		if i < attempts-1 {
			select {
			case <-time.After(delay):
				// Wait for the delay period
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return err
}
