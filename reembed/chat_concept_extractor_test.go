package reembed

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatConceptExtractor_Run(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test message about testing",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 10)

	// Run concept extraction
	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			return []ai.ExtractedConcept{
				{Name: "testing", Type: "topic", Importance: 8},
			}, nil
		},
	}

	config := &Config{
		BatchSize:      3,
		ReportInterval: 3,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err = extractor1.Run(ctx)
	require.NoError(t, err)

	// Verify all records have concepts
	updated, err := chatRepo.GetChatRecordsByDateRange(ctx,
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, updated, 10)

	for _, record := range updated {
		require.NotEmpty(t, record.Concepts, "record %d should have concepts", record.Id)
		assert.Len(t, record.Concepts, 1)
		assert.Equal(t, 8, record.Concepts[0].Importance)
	}

	// Verify concepts were created with embeddings
	allConcepts, err := conceptRepo.GetAllConcepts(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, allConcepts, "should have created concepts")

	for _, concept := range allConcepts {
		assert.NotEmpty(t, concept.Vector, "concept should have embedding")
		// Verify normalization
		var magnitude float32
		for _, v := range concept.Vector {
			magnitude += v * v
		}
		assert.InDelta(t, 1.0, magnitude, 0.01, "vector should be normalized")
	}

	// Check progress output
	output := buf.String()
	assert.Contains(t, output, "10/10", "should show completion")
	assert.Contains(t, output, "Concept extraction complete", "should show completion message")
}

func TestChatConceptExtractor_EmptyDatabase(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{}
	config := DefaultConfig()

	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err := extractor1.Run(ctx)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "0 records", "should report zero records")
}

func TestChatConceptExtractor_ContextCancellation(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Add records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test content",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
	}
	_, err := chatRepo.AddChatRecords(context.Background(), records...)
	require.NoError(t, err)

	// Cancel after processing a few
	callCount := 0
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			callCount++
			if callCount == 2 {
				cancel()
			}
			return []ai.ExtractedConcept{
				{Name: "test", Type: "topic", Importance: 7},
			}, nil
		},
	}

	var buf bytes.Buffer
	config := &Config{
		BatchSize:      3,
		ReportInterval: 3,
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err = extractor1.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestChatConceptExtractor_ExtractionError(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	_, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	// Extractor that always fails
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
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

	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err = extractor1.Run(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persistent error")
}

func TestChatConceptExtractor_ProgressTracking(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add enough records to trigger progress updates
	records := make([]*core.ChatRecord, 25)
	for i := 0; i < 25; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test message",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
	}
	_, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			return []ai.ExtractedConcept{
				{Name: "test", Type: "topic", Importance: 7},
			}, nil
		},
	}

	config := &Config{
		BatchSize:      5,
		ReportInterval: 10, // Report every 10 records
		MaxRetries:     3,
		RetryDelay:     10 * time.Millisecond,
	}

	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err = extractor1.Run(ctx)
	require.NoError(t, err)

	output := buf.String()
	// Should have progress output
	assert.Contains(t, output, "Progress:", "should show progress")
	assert.Contains(t, output, "25/25", "should show final count")
}

func TestChatConceptExtractor_MultipleConcepts(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "discussing paris and the eiffel tower", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "talking about france", Timestamp: time.Now().Add(time.Second)},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	var buf bytes.Buffer
	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			if len(text) > 30 {
				return []ai.ExtractedConcept{
					{Name: "paris", Type: "place", Importance: 9},
					{Name: "eiffel tower", Type: "landmark", Importance: 8},
				}, nil
			}
			return []ai.ExtractedConcept{
				{Name: "france", Type: "place", Importance: 7},
			}, nil
		},
	}

	config := DefaultConfig()
	extractor1 := NewChatConceptExtractor(chatRepo, conceptRepo, embedder, extractor, config, &buf)
	err = extractor1.Run(ctx)
	require.NoError(t, err)

	// Verify records have correct number of concepts
	updated, err := chatRepo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	require.Len(t, updated, 2)

	assert.Len(t, updated[0].Concepts, 2, "first record should have 2 concepts")
	assert.Len(t, updated[1].Concepts, 1, "second record should have 1 concept")

	// Verify all concepts exist in repository
	allConcepts, err := conceptRepo.GetAllConcepts(ctx)
	require.NoError(t, err)
	assert.Len(t, allConcepts, 3, "should have created 3 unique concepts")
}
