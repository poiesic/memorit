package reembed

import (
	"context"
	"log/slog"
	"time"
)

// RetryWithBackoff retries an operation with exponential backoff.
// maxAttempts: maximum number of attempts (must be > 0)
// baseDelay: base delay between retries (doubles on each retry)
// Returns the error from the last attempt if all attempts fail.
func RetryWithBackoff(ctx context.Context, operation func() error, maxAttempts int, baseDelay time.Duration) error {
	if maxAttempts <= 0 {
		return ErrInvalidMaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 1 {
				slog.Debug("operation succeeded after retry", "attempt", attempt)
			}
			return nil // Success
		}

		slog.Debug("operation failed, will retry", "attempt", attempt, "maxAttempts", maxAttempts, "error", lastErr)

		// Don't sleep after the last attempt
		if attempt == maxAttempts {
			break
		}

		// Calculate exponential backoff: baseDelay * 2^(attempt-1)
		delay := baseDelay
		for i := 1; i < attempt; i++ {
			delay *= 2
		}

		// Sleep with context awareness
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next attempt
		}
	}

	return lastErr
}
