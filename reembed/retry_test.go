package reembed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return nil
	}

	err := RetryWithBackoff(context.Background(), operation, 3, 10*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 1, attempts, "should succeed on first try")
}

func TestRetryWithBackoff_EventualSuccess(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := RetryWithBackoff(context.Background(), operation, 5, 10*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 3, attempts, "should succeed on third attempt")
}

func TestRetryWithBackoff_AllAttemptsFail(t *testing.T) {
	attempts := 0
	expectedErr := errors.New("persistent error")
	operation := func() error {
		attempts++
		return expectedErr
	}

	err := RetryWithBackoff(context.Background(), operation, 3, 10*time.Millisecond)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err, "should return the original error")
	assert.Equal(t, 3, attempts, "should attempt exactly maxAttempts times")
}

func TestRetryWithBackoff_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	operation := func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("error")
	}

	err := RetryWithBackoff(ctx, operation, 10, 10*time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "should return context.Canceled")
	assert.LessOrEqual(t, attempts, 2, "should stop when context is canceled")
}

func TestRetryWithBackoff_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	operation := func() error {
		attempts++
		time.Sleep(30 * time.Millisecond) // Slow operation
		return errors.New("error")
	}

	err := RetryWithBackoff(ctx, operation, 10, 10*time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "should return context.DeadlineExceeded")
	assert.LessOrEqual(t, attempts, 3, "should stop when context times out")
}

func TestRetryWithBackoff_ExponentialBackoff(t *testing.T) {
	attempts := 0
	var delays []time.Duration
	lastTime := time.Now()

	operation := func() error {
		attempts++
		if attempts > 1 {
			delays = append(delays, time.Since(lastTime))
		}
		lastTime = time.Now()
		if attempts < 4 {
			return errors.New("error")
		}
		return nil
	}

	err := RetryWithBackoff(context.Background(), operation, 5, 10*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, 4, attempts)

	// Verify exponential backoff (each delay should be roughly 2x the previous)
	require.Len(t, delays, 3, "should have 3 delays")

	// Allow some tolerance for timing variance
	assert.Greater(t, delays[1], delays[0], "second delay should be greater than first")
	assert.Greater(t, delays[2], delays[1], "third delay should be greater than second")
}

func TestRetryWithBackoff_ZeroMaxAttempts(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("error")
	}

	err := RetryWithBackoff(context.Background(), operation, 0, 10*time.Millisecond)
	require.Error(t, err)
	assert.Equal(t, 0, attempts, "should not attempt with maxAttempts=0")
}

func TestRetryWithBackoff_NegativeMaxAttempts(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("error")
	}

	err := RetryWithBackoff(context.Background(), operation, -1, 10*time.Millisecond)
	require.Error(t, err)
	assert.Equal(t, 0, attempts, "should not attempt with negative maxAttempts")
}
