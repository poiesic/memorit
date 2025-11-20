package reembed

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReembedder_Run(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test message",
			Timestamp: time.Now(),
		}
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 10)

	// Run reembedding
	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	config := &Config{
		BatchSize:      3,
		ReportInterval: 3,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	reembedder := NewReembedder(repo, embedder, config, &buf)
	err = reembedder.Run(ctx)
	require.NoError(t, err)

	// Verify all records have embeddings
	updated, err := repo.GetChatRecordsByDateRange(ctx,
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, updated, 10)

	for _, record := range updated {
		require.NotEmpty(t, record.Vector, "record %d should have embedding", record.Id)
		// Verify normalization
		var magnitude float32
		for _, v := range record.Vector {
			magnitude += v * v
		}
		assert.InDelta(t, 1.0, magnitude, 0.01, "vector should be normalized")
	}

	// Check progress output
	output := buf.String()
	assert.Contains(t, output, "10/10", "should show completion")
}

func TestReembedder_EmptyDatabase(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	config := DefaultConfig()

	reembedder := NewReembedder(repo, embedder, config, &buf)
	err := reembedder.Run(ctx)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "0 records", "should report zero records")
}

func TestReembedder_ContextCancellation(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Add records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test",
			Timestamp: time.Now(),
		}
	}
	_, err := repo.AddChatRecords(context.Background(), records...)
	require.NoError(t, err)

	// Cancel after processing a few
	callCount := 0
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			callCount++
			if callCount == 2 {
				cancel()
			}
			result := make([][]float32, len(texts))
			for i := range result {
				result[i] = []float32{1.0, 0.0, 0.0}
			}
			return result, nil
		},
	}

	var buf bytes.Buffer
	config := &Config{
		BatchSize:      3,
		ReportInterval: 3,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	reembedder := NewReembedder(repo, embedder, config, &buf)
	err = reembedder.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestReembedder_EmbeddingError(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	_, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	// Embedder that always fails
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return nil, errors.New("persistent error")
		},
	}

	var buf bytes.Buffer
	config := &Config{
		BatchSize:      1,
		ReportInterval: 1,
		MaxRetries:     2,
		RetryDelay:     10 * time.Millisecond,
	}

	reembedder := NewReembedder(repo, embedder, config, &buf)
	err = reembedder.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persistent error")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Greater(t, config.BatchSize, 0, "batch size should be positive")
	assert.Greater(t, config.ReportInterval, 0, "report interval should be positive")
	assert.Greater(t, config.MaxRetries, 0, "max retries should be positive")
	assert.Greater(t, config.RetryDelay, time.Duration(0), "retry delay should be positive")
}

func TestReembedder_ProgressTracking(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add enough records to trigger progress updates
	records := make([]*core.ChatRecord, 25)
	for i := 0; i < 25; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test",
			Timestamp: time.Now(),
		}
	}
	_, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	config := &Config{
		BatchSize:      5,
		ReportInterval: 10, // Report every 10 records
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	reembedder := NewReembedder(repo, embedder, config, &buf)
	err = reembedder.Run(ctx)
	require.NoError(t, err)

	output := buf.String()
	// Should have progress output
	assert.Contains(t, output, "Progress:", "should show progress")
	assert.Contains(t, output, "25/25", "should show final count")
}
