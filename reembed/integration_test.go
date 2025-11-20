package reembed

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/ai/openai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullReembeddingWorkflow tests the complete reembedding workflow
// from database setup through completion using a mock embedder.
func TestIntegration_FullReembeddingWorkflow(t *testing.T) {
	// Skip if short tests
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create in-memory database
	backend, err := badger.OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	repo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)
	defer repo.Close()

	// Seed database with records WITHOUT embeddings
	records := make([]*core.ChatRecord, 50)
	for i := 0; i < 50; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test message",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Vector:    nil, // No embedding initially
		}
	}

	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 50)

	// Verify records don't have embeddings
	for _, record := range added {
		assert.Empty(t, record.Vector, "initial records should not have embeddings")
	}

	// Create embedder
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			// Return unique vectors for each text based on length
			result := make([][]float32, len(texts))
			for i := range texts {
				// Create a vector based on text index
				result[i] = []float32{
					float32(i+1) * 0.1,
					float32(i+1) * 0.2,
					float32(i+1) * 0.3,
				}
			}
			return result, nil
		},
	}

	// Configure reembedding
	config := &Config{
		BatchSize:      10,
		ReportInterval: 10,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	var buf bytes.Buffer
	reembedder := NewReembedder(repo, embedder, config, &buf)

	// Run reembedding
	err = reembedder.Run(ctx)
	require.NoError(t, err)

	// Verify all records now have normalized embeddings
	allRecords, err := repo.GetChatRecordsByDateRange(ctx,
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, allRecords, 50, "should have all 50 records")

	for i, record := range allRecords {
		require.NotEmpty(t, record.Vector, "record %d should have embedding", i)

		// Verify normalization
		var magnitude float32
		for _, v := range record.Vector {
			magnitude += v * v
		}
		assert.InDelta(t, 1.0, magnitude, 0.01, "record %d vector should be normalized", i)
	}

	// Verify progress output
	output := buf.String()
	assert.Contains(t, output, "Starting reembedding of 50 records")
	assert.Contains(t, output, "50/50")
	assert.Contains(t, output, "100.0%")
	assert.Contains(t, output, "Reembedding complete")
}

// TestIntegration_WithRealEmbedder tests with a real OpenAI-compatible embedder
// This test requires a running embedding service and is skipped by default.
func TestIntegration_WithRealEmbedder(t *testing.T) {
	t.Skip("Requires running embedding service - enable manually for testing")

	ctx := context.Background()

	// Create in-memory database
	backend, err := badger.OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	repo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)
	defer repo.Close()

	// Add test records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Hello, how are you?", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "I'm doing well, thanks!", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeHuman, Contents: "What's the weather like?", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	// Create AI config
	aiConfig := ai.NewConfig(
		ai.WithHost("http://localhost:11434/v1"),
		ai.WithEmbeddingModel("embeddinggemma"),
		ai.WithClassifierModel("qwen2.5:3b"),
	)

	// Create real embedder
	embedder, err := openai.NewEmbedder(aiConfig)
	require.NoError(t, err)

	// Run reembedding
	config := DefaultConfig()
	var buf bytes.Buffer
	reembedder := NewReembedder(repo, embedder, config, &buf)

	err = reembedder.Run(ctx)
	require.NoError(t, err)

	// Verify embeddings
	updated, err := repo.GetChatRecords(ctx, added[0].Id, added[1].Id, added[2].Id)
	require.NoError(t, err)
	require.Len(t, updated, 3)

	for _, record := range updated {
		require.NotEmpty(t, record.Vector)
		// Real embeddings should have a consistent dimension
		assert.Greater(t, len(record.Vector), 0)
	}
}

// TestIntegration_IdempotentReembedding tests that reembedding can be run multiple times
func TestIntegration_IdempotentReembedding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create in-memory database
	backend, err := badger.OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	repo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)
	defer repo.Close()

	// Add records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test message",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	config := &Config{
		BatchSize:      5,
		ReportInterval: 5,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	// First run
	var buf1 bytes.Buffer
	reembedder1 := NewReembedder(repo, embedder, config, &buf1)
	err = reembedder1.Run(ctx)
	require.NoError(t, err)

	// Get embeddings after first run
	records1, err := repo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	vec1 := records1[0].Vector

	// Second run (should overwrite with same vectors)
	var buf2 bytes.Buffer
	reembedder2 := NewReembedder(repo, embedder, config, &buf2)
	err = reembedder2.Run(ctx)
	require.NoError(t, err)

	// Get embeddings after second run
	records2, err := repo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	vec2 := records2[0].Vector

	// Verify vectors are the same (idempotent)
	require.Equal(t, len(vec1), len(vec2))
	for i := range vec1 {
		assert.InDelta(t, vec1[i], vec2[i], 0.001, "vectors should be identical after re-embedding")
	}
}
