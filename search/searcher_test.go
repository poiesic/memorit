package search

import (
	"context"
	"iter"
	"log/slog"
	"testing"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/ai/mock"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearcher(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	provider := mock.NewMockProvider()

	t.Run("valid configuration", func(t *testing.T) {
		searcher, err := NewSearcher(chatRepo, conceptRepo, provider)
		require.NoError(t, err)
		assert.NotNil(t, searcher)
	})

	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.Default()
		searcher, err := NewSearcher(chatRepo, conceptRepo, provider, WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, searcher)
	})

	t.Run("with nil logger falls back to default", func(t *testing.T) {
		searcher, err := NewSearcher(chatRepo, conceptRepo, provider, WithLogger(nil))
		require.NoError(t, err)
		assert.NotNil(t, searcher)
	})

	t.Run("nil chat repository", func(t *testing.T) {
		_, err := NewSearcher(nil, conceptRepo, provider)
		assert.Equal(t, ErrChatRepositoryRequired, err)
	})

	t.Run("nil concept repository", func(t *testing.T) {
		_, err := NewSearcher(chatRepo, nil, provider)
		assert.Equal(t, ErrConceptRepositoryRequired, err)
	})

	t.Run("nil provider", func(t *testing.T) {
		_, err := NewSearcher(chatRepo, conceptRepo, nil)
		assert.Equal(t, ErrAIProviderRequired, err)
	})
}

func TestFindSimilar_EmptyDatabase(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	provider := mock.NewMockProvider()
	searcher, err := NewSearcher(chatRepo, conceptRepo, provider)
	require.NoError(t, err)

	ctx := context.Background()
	results, err := searcher.FindSimilar(ctx, "test query", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFindSimilar_SemanticSearchOnly(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add chat records with vectors
	records := []*core.ChatRecord{
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "This is about artificial intelligence",
			Timestamp:  now,
			Vector:     []float32{0.9, 0.1, 0.0},
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "This is about machine learning",
			Timestamp:  now,
			Vector:     []float32{0.85, 0.15, 0.0},
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "This is about cooking recipes",
			Timestamp:  now,
			Vector:     []float32{0.1, 0.1, 0.8},
		},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Create mock provider with custom embedder
	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		// Return vector similar to first two records
		return []float32{0.88, 0.12, 0.0}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mock.NewMockConceptExtractor())

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	results, err := searcher.FindSimilar(ctx, "artificial intelligence query", 10)
	require.NoError(t, err)

	// Should find records above similarity threshold (0.60)
	assert.NotEmpty(t, results)

	// Results should be sorted by score
	for i := 0; i < len(results)-1; i++ {
		assert.GreaterOrEqual(t, results[i].Score, results[i+1].Score)
	}
}

func TestFindSimilar_ConceptualSearchOnly(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create concepts
	concepts := []*core.Concept{
		{
			Name:       "python",
			Type:       "programming_language",
			InsertedAt: now,
			UpdatedAt:  now,
		},
	}

	// Compute IDs from tuples
	for _, c := range concepts {
		c.Id = core.IDFromContent(c.Tuple())
	}

	addedConcepts, err := conceptRepo.AddConcepts(ctx, concepts...)
	require.NoError(t, err)
	require.Len(t, addedConcepts, 1)

	// Add chat records with low similarity vectors but matching concepts
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "I love programming in Python",
			Timestamp: now,
			Vector:    []float32{0.1, 0.1, 0.1}, // Low similarity
			Concepts: []core.ConceptRef{
				{ConceptId: addedConcepts[0].Id, Importance: 8},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "JavaScript is also great",
			Timestamp: now,
			Vector:    []float32{0.1, 0.1, 0.1}, // Low similarity, no matching concepts
		},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 2)

	// Create mock provider that extracts "python" concept
	mockExtractor := mock.NewMockConceptExtractor()
	mockExtractor.ExtractConceptsFunc = func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
		return []ai.ExtractedConcept{
			{Name: "python", Type: "programming_language", Importance: 9},
		}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mock.NewMockEmbedder(), mockExtractor)

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	results, err := searcher.FindSimilar(ctx, "tell me about python", 10)
	require.NoError(t, err)

	// Should find the record with matching concept
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Record.Contents, "Python")
	assert.Equal(t, float32(1.2), results[0].Score) // Conceptual-only score
}

func TestFindSimilar_HybridSearch(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create concepts
	concepts := []*core.Concept{
		{
			Name:       "machine",
			Type:       "thing",
			InsertedAt: now,
			UpdatedAt:  now,
		},
	}

	for _, c := range concepts {
		c.Id = core.IDFromContent(c.Tuple())
	}

	addedConcepts, err := conceptRepo.AddConcepts(ctx, concepts...)
	require.NoError(t, err)

	// Add records with both semantic similarity AND concept match
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Machine learning is fascinating",
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0}, // High similarity
			Concepts: []core.ConceptRef{
				{ConceptId: addedConcepts[0].Id, Importance: 8},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "AI and machine intelligence",
			Timestamp: now,
			Vector:    []float32{0.85, 0.15, 0.0}, // Medium-high similarity, no concept
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "The machine in the factory",
			Timestamp: now,
			Vector:    []float32{0.2, 0.1, 0.7}, // Low similarity, has concept
			Concepts: []core.ConceptRef{
				{ConceptId: addedConcepts[0].Id, Importance: 7},
			},
		},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Create mock provider
	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.9, 0.1, 0.0}, nil // High similarity
	}
	mockExtractor := mock.NewMockConceptExtractor()
	mockExtractor.ExtractConceptsFunc = func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
		return []ai.ExtractedConcept{
			{Name: "machine", Type: "thing", Importance: 8},
		}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mockExtractor)

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	results, err := searcher.FindSimilar(ctx, "machine learning", 10)
	require.NoError(t, err)

	// Should find all three records
	require.Len(t, results, 3)

	// First result should be hybrid (semantic + conceptual) with highest score
	assert.Contains(t, results[0].Record.Contents, "Machine learning is fascinating")
	assert.Greater(t, results[0].Score, float32(1.2)) // Should have 1.5x boost
}

func TestFindSimilar_VerbatimBoost(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "machine learning is fascinating", // Contains both query words
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "AI is the future",
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0}, // Same vector, different content
		},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 2)

	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.9, 0.1, 0.0}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mock.NewMockConceptExtractor())

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	results, err := searcher.FindSimilar(ctx, "machine learning", 10)
	require.NoError(t, err)

	require.Len(t, results, 2)

	// First result should have verbatim boost
	assert.Contains(t, results[0].Record.Contents, "machine learning")
	// Score should include 0.3 boost
	assert.Greater(t, results[0].Score, results[1].Score)
}

func TestFindSimilar_WithMaxHits(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add 10 records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Test message",
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0},
		}
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.9, 0.1, 0.0}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mock.NewMockConceptExtractor())

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	results, err := searcher.FindSimilar(ctx, "query", 5)
	require.NoError(t, err)

	// Should limit to 5 results
	assert.Len(t, results, 5)
}

func TestFindSimilarWithMonitor(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add a simple record
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Test message",
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0},
		},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.9, 0.1, 0.0}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mock.NewMockConceptExtractor())

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	// Create a test monitor
	monitor := &testMonitor{}

	results, err := searcher.FindSimilarWithMonitor(ctx, "test query", 10, monitor)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Verify monitor was called
	assert.True(t, monitor.startCalled)
	assert.True(t, monitor.finishCalled)
}

// testMonitor is a simple test implementation of SearchMonitor
type testMonitor struct {
	startCalled  bool
	finishCalled bool
}

func (m *testMonitor) Start(query string) {
	m.startCalled = true
}

func (m *testMonitor) AfterSemanticSearch(ids []uint64) {}

func (m *testMonitor) AfterQueryConceptExtraction(concepts []*core.Concept) {}

func (m *testMonitor) FoundRelatedConcepts(tuple string, conceptIds []uint64) {}

func (m *testMonitor) AfterConceptuallyRelatedSearch(seq iter.Seq[uint64]) {}

func (m *testMonitor) AfterRecordRetrieval(records []*core.ChatRecord) {}

func (m *testMonitor) SemanticAndConceptualHit(record *core.ChatRecord) {}

func (m *testMonitor) SemanticHit(record *core.ChatRecord) {}

func (m *testMonitor) ConceptualHit(record *core.ChatRecord) {}

func (m *testMonitor) Finish(results []*core.SearchResult) {
	m.finishCalled = true
}

func TestFindSimilar_ConceptNotInDatabase(t *testing.T) {
	chatRepo, conceptRepo, backend, err := badger.NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add records but no concepts
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Test message",
			Timestamp: now,
			Vector:    []float32{0.9, 0.1, 0.0},
		},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	// Mock extractor returns concepts that don't exist in DB
	mockEmbedder := mock.NewMockEmbedder()
	mockEmbedder.EmbedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
		// Return similar vector to find the record via semantic search
		return []float32{0.9, 0.1, 0.0}, nil
	}
	mockExtractor := mock.NewMockConceptExtractor()
	mockExtractor.ExtractConceptsFunc = func(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
		return []ai.ExtractedConcept{
			{Name: "nonexistent", Type: "thing", Importance: 8},
		}, nil
	}
	mockProvider := mock.NewMockProviderWithServices(mockEmbedder, mockExtractor)

	searcher, err := NewSearcher(chatRepo, conceptRepo, mockProvider)
	require.NoError(t, err)

	// Should not error, just fall back to semantic search only
	results, err := searcher.FindSimilar(ctx, "query", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}
