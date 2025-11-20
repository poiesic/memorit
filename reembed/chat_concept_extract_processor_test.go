package reembed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConceptExtractor for testing
type mockConceptExtractor struct {
	extractConceptsFunc func(ctx context.Context, text string) ([]ai.ExtractedConcept, error)
}

func (m *mockConceptExtractor) ExtractConcepts(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
	if m.extractConceptsFunc != nil {
		return m.extractConceptsFunc(ctx, text)
	}
	// Default: return some concepts
	return []ai.ExtractedConcept{
		{Name: "test", Type: "topic", Importance: 8},
		{Name: "example", Type: "concept", Importance: 6},
	}, nil
}

func setupTestRepos(t *testing.T) (*badger.ChatRepository, *badger.ConceptRepository, func()) {
	backend, err := badger.OpenBackend("", true) // in-memory
	require.NoError(t, err)

	chatRepo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)

	conceptRepo, err := badger.NewConceptRepository(backend)
	require.NoError(t, err)

	cleanup := func() {
		chatRepo.Close()
		conceptRepo.Close()
		backend.Close()
	}

	return chatRepo, conceptRepo, cleanup
}

func TestChatConceptExtractProcessor_Process(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "discussing the Eiffel Tower in Paris", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "response about landmarks", Timestamp: time.Now()},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			// Return different concepts based on text content
			if len(text) > 30 {
				return []ai.ExtractedConcept{
					{Name: "eiffel tower", Type: "landmark", Importance: 9},
					{Name: "paris", Type: "place", Importance: 8},
				}, nil
			}
			return []ai.ExtractedConcept{
				{Name: "landmark", Type: "topic", Importance: 7},
			}, nil
		},
	}

	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)

	// Verify records were updated with concepts
	updated, err := chatRepo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	require.Len(t, updated, 2)

	// First record should have 2 concepts
	assert.Len(t, updated[0].Concepts, 2, "first record should have 2 concepts")

	// Second record should have 1 concept
	assert.Len(t, updated[1].Concepts, 1, "second record should have 1 concept")

	// Verify concepts were created in concept repository
	for _, record := range updated {
		for _, conceptRef := range record.Concepts {
			concept, err := conceptRepo.GetConcept(ctx, conceptRef.ConceptId)
			require.NoError(t, err)
			assert.NotEmpty(t, concept.Name)
			assert.NotEmpty(t, concept.Type)
			assert.NotEmpty(t, concept.Vector, "concept should have embedding")
		}
	}
}

func TestChatConceptExtractProcessor_EmptyBatch(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{}
	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err := processor.Process(ctx, []*core.ChatRecord{})
	require.NoError(t, err, "empty batch should not error")
}

func TestChatConceptExtractProcessor_ExtractionError(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	expectedErr := errors.New("extraction error")
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			return nil, expectedErr
		},
	}

	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extraction error")
}

func TestChatConceptExtractProcessor_Retry(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test content", Timestamp: time.Now()},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	attempts := 0
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			attempts++
			if attempts < 2 {
				return nil, errors.New("temporary error")
			}
			// Success on second attempt
			return []ai.ExtractedConcept{
				{Name: "test", Type: "topic", Importance: 7},
			}, nil
		},
	}

	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "should retry on failure")

	// Verify record was updated with concepts
	updated, err := chatRepo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Len(t, updated[0].Concepts, 1, "should have 1 concept")
}

func TestChatConceptExtractProcessor_DuplicateConcepts(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records that will generate the same concepts
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "discussing paris", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "more about paris", Timestamp: time.Now()},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			// Both records will extract the same concept
			return []ai.ExtractedConcept{
				{Name: "paris", Type: "place", Importance: 8},
			}, nil
		},
	}

	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)

	// Verify both records reference the same concept
	updated, err := chatRepo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	require.Len(t, updated, 2)

	assert.Len(t, updated[0].Concepts, 1)
	assert.Len(t, updated[1].Concepts, 1)

	// Both should have the same concept ID
	assert.Equal(t, updated[0].Concepts[0].ConceptId, updated[1].Concepts[0].ConceptId,
		"duplicate concepts should reference the same concept ID")

	// Verify only one concept was created
	allConcepts, err := conceptRepo.GetAllConcepts(ctx)
	require.NoError(t, err)
	assert.Len(t, allConcepts, 1, "should only create one concept for duplicates")
}

func TestChatConceptExtractProcessor_NoConcepts(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepos(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test content", Timestamp: time.Now()},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	extractor := &mockConceptExtractor{
		extractConceptsFunc: func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
			// Return no concepts
			return []ai.ExtractedConcept{}, nil
		},
	}

	processor := NewChatConceptExtractProcessor(chatRepo, conceptRepo, embedder, extractor, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)

	// Verify record has no concepts
	updated, err := chatRepo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Empty(t, updated[0].Concepts, "should have no concepts")
}
