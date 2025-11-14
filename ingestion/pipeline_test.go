package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConceptExtractor implements ai.ConceptExtractor for testing
type testConceptExtractor struct {
	responses   map[string][]ai.ExtractedConcept // map from text to concepts
	shouldError bool
	errorOnText string
}

func (m *testConceptExtractor) ExtractConcepts(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
	if m.shouldError || text == m.errorOnText {
		return nil, errors.New("extraction error")
	}
	if concepts, ok := m.responses[text]; ok {
		return concepts, nil
	}
	return []ai.ExtractedConcept{}, nil
}

// testEmbedder implements ai.Embedder for testing
type testEmbedder struct {
	embeddings  [][]float32
	shouldError bool
}

func (m *testEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.shouldError {
		return nil, errors.New("embedder error")
	}
	if len(m.embeddings) > 0 {
		return m.embeddings[0], nil
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *testEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if m.shouldError {
		return nil, errors.New("embedder error")
	}
	if len(m.embeddings) > 0 {
		return m.embeddings, nil
	}
	// Generate dynamic embeddings
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{float32(i) * 0.1, float32(i) * 0.2, float32(i) * 0.3}
	}
	return result, nil
}

// testAIProvider implements ai.AIProvider for testing
type testAIProvider struct {
	embedder  ai.Embedder
	extractor ai.ConceptExtractor
}

func (p *testAIProvider) Embedder() ai.Embedder {
	return p.embedder
}

func (p *testAIProvider) ConceptExtractor() ai.ConceptExtractor {
	return p.extractor
}

func (p *testAIProvider) Close() error {
	return nil
}

func setupTestRepositories(t *testing.T) (storage.ChatRepository, storage.ConceptRepository, func()) {
	backend, err := badger.OpenBackend(t.TempDir(), false)
	require.NoError(t, err)

	chatRepo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)

	conceptRepo, err := badger.NewConceptRepository(backend)
	require.NoError(t, err)

	cleanup := func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}

	return chatRepo, conceptRepo, cleanup
}

func setupTestConceptProcessor(t *testing.T) (*conceptProcessor, storage.ChatRepository) {
	chatRepo, conceptRepo, cleanup := setupTestRepositories(t)
	t.Cleanup(cleanup)

	embedder := &testEmbedder{}

	extractor := &testConceptExtractor{
		responses: make(map[string][]ai.ExtractedConcept),
	}

	cp, err := newConceptProcessor(chatRepo, conceptRepo, embedder, extractor, nil)
	require.NoError(t, err)
	require.NotNil(t, cp)

	return cp.(*conceptProcessor), chatRepo
}

func TestConceptProcessor_Process_SingleRecord(t *testing.T) {
	cp, chatRepo := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Add a chat record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Alice went to Paris",
		Timestamp: time.Now().UTC(),
	}
	added, err := chatRepo.AddChatRecords(ctx, record)
	require.NoError(t, err)
	require.Len(t, added, 1)

	// Setup classifier response
	cp.extractor.(*testConceptExtractor).responses["Alice went to Paris"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 8},
		{Name: "Paris", Type: "place", Importance: 7},
	}

	// Process the record
	err = cp.process(ctx, added[0].Id)
	require.NoError(t, err)

	// Verify concepts were assigned
	processed, err := chatRepo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, processed, 1)
	require.Len(t, processed[0].Concepts, 2)

	// Verify concept details
	assert.Equal(t, 8, processed[0].Concepts[0].Importance)
	assert.Equal(t, 7, processed[0].Concepts[1].Importance)

	// Verify concepts exist in concept collection
	concept1, err := cp.conceptRepository.GetConcept(ctx, processed[0].Concepts[0].ConceptId)
	require.NoError(t, err)
	require.NotNil(t, concept1)

	concept2, err := cp.conceptRepository.GetConcept(ctx, processed[0].Concepts[1].ConceptId)
	require.NoError(t, err)
	require.NotNil(t, concept2)
}

func TestConceptProcessor_Process_MultipleBatchSizes(t *testing.T) {
	testCases := []struct {
		name       string
		numRecords int
	}{
		{"1 record", 1},
		{"10 records", 10},
		{"30 records", 30},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cp, chatRepo := setupTestConceptProcessor(t)
			ctx := context.Background()

			// Create records with unique concepts
			records := make([]*core.ChatRecord, tc.numRecords)
			ids := make([]core.ID, tc.numRecords)
			for i := 0; i < tc.numRecords; i++ {
				text := fmt.Sprintf("Message %d", i)
				records[i] = &core.ChatRecord{
					Speaker:   core.SpeakerTypeHuman,
					Contents:  text,
					Timestamp: time.Now().UTC(),
				}

				// Setup classifier response for each message
				cp.extractor.(*testConceptExtractor).responses[text] = []ai.ExtractedConcept{
					{Name: fmt.Sprintf("Person%d", i), Type: "person", Importance: 5 + i},
				}
			}

			// Add all records
			added, err := chatRepo.AddChatRecords(ctx, records...)
			require.NoError(t, err)
			require.Len(t, added, tc.numRecords)

			for i, rec := range added {
				ids[i] = rec.Id
			}

			// Process all records in batch
			err = cp.process(ctx, ids...)
			require.NoError(t, err)

			// Verify all records have concepts assigned
			processed, err := chatRepo.GetChatRecords(ctx, ids...)
			require.NoError(t, err)
			require.Len(t, processed, tc.numRecords)

			for i, rec := range processed {
				assert.Len(t, rec.Concepts, 1, "record %d should have 1 concept", i)
				assert.Equal(t, 5+i, rec.Concepts[0].Importance, "record %d importance mismatch", i)
			}
		})
	}
}

func TestConceptProcessor_Process_DuplicateConceptsAcrossRecords(t *testing.T) {
	cp, chatRepo := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Create 3 records that all mention "Alice"
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Alice is here", Timestamp: time.Now().UTC()},
		{Speaker: core.SpeakerTypeHuman, Contents: "Alice went there", Timestamp: time.Now().UTC()},
		{Speaker: core.SpeakerTypeHuman, Contents: "Alice came back", Timestamp: time.Now().UTC()},
	}

	// Setup classifier responses - all mention Alice but with different importance
	cp.extractor.(*testConceptExtractor).responses["Alice is here"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 8},
	}
	cp.extractor.(*testConceptExtractor).responses["Alice went there"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 7},
	}
	cp.extractor.(*testConceptExtractor).responses["Alice came back"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 9},
	}

	// Add all records
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Process all records
	ids := []core.ID{added[0].Id, added[1].Id, added[2].Id}
	err = cp.process(ctx, ids...)
	require.NoError(t, err)

	// Verify all records have Alice assigned
	processed, err := chatRepo.GetChatRecords(ctx, ids...)
	require.NoError(t, err)
	require.Len(t, processed, 3)

	// All should reference the same concept ID (Alice is deduplicated)
	aliceID := processed[0].Concepts[0].ConceptId
	assert.Equal(t, aliceID, processed[1].Concepts[0].ConceptId)
	assert.Equal(t, aliceID, processed[2].Concepts[0].ConceptId)

	// But importance should be preserved per record
	assert.Equal(t, 8, processed[0].Concepts[0].Importance)
	assert.Equal(t, 7, processed[1].Concepts[0].Importance)
	assert.Equal(t, 9, processed[2].Concepts[0].Importance)

	// Verify Alice exists in concept collection
	aliceConcept, err := cp.conceptRepository.GetConcept(ctx, aliceID)
	require.NoError(t, err)
	require.NotNil(t, aliceConcept)
	assert.Equal(t, "Alice", aliceConcept.Name)
	assert.Equal(t, "person", aliceConcept.Type)
}

func TestConceptProcessor_Process_PartialFailureWithErrorsJoin(t *testing.T) {
	cp, chatRepo := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Create 3 records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: time.Now().UTC()},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2 FAIL", Timestamp: time.Now().UTC()},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 3", Timestamp: time.Now().UTC()},
	}

	// Setup classifier - one will fail
	cp.extractor.(*testConceptExtractor).responses["Message 1"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 8},
	}
	cp.extractor.(*testConceptExtractor).errorOnText = "Message 2 FAIL"
	cp.extractor.(*testConceptExtractor).responses["Message 3"] = []ai.ExtractedConcept{
		{Name: "Bob", Type: "person", Importance: 7},
	}

	// Add all records
	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Process all records - should return error but continue processing
	ids := []core.ID{added[0].Id, added[1].Id, added[2].Id}
	err = cp.process(ctx, ids...)

	// Should have an error
	require.Error(t, err)
	// Should contain the classification error
	assert.Contains(t, err.Error(), "classification failed")
	assert.Contains(t, err.Error(), "record 1")

	// Verify records 1 and 3 were processed successfully
	processed, err := chatRepo.GetChatRecords(ctx, ids...)
	require.NoError(t, err)
	require.Len(t, processed, 3)

	// Record 0 should have Alice
	assert.Len(t, processed[0].Concepts, 1)
	assert.Equal(t, 8, processed[0].Concepts[0].Importance)

	// Record 1 should have empty concepts (failed)
	assert.Len(t, processed[1].Concepts, 0)

	// Record 2 should have Bob
	assert.Len(t, processed[2].Concepts, 1)
	assert.Equal(t, 7, processed[2].Concepts[0].Importance)
}

func TestConceptProcessor_Process_MultipleConceptsPerRecord(t *testing.T) {
	cp, chatRepo := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Create record with multiple concepts
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Alice and Bob went to Paris and London",
		Timestamp: time.Now().UTC(),
	}

	// Setup classifier response with 4 concepts
	cp.extractor.(*testConceptExtractor).responses["Alice and Bob went to Paris and London"] = []ai.ExtractedConcept{
		{Name: "Alice", Type: "person", Importance: 8},
		{Name: "Bob", Type: "person", Importance: 7},
		{Name: "Paris", Type: "place", Importance: 9},
		{Name: "London", Type: "place", Importance: 6},
	}

	// Add record
	added, err := chatRepo.AddChatRecords(ctx, record)
	require.NoError(t, err)
	require.Len(t, added, 1)

	// Process
	err = cp.process(ctx, added[0].Id)
	require.NoError(t, err)

	// Verify all concepts assigned correctly
	processed, err := chatRepo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, processed, 1)
	require.Len(t, processed[0].Concepts, 4)

	// Verify importance values preserved in correct order
	assert.Equal(t, 8, processed[0].Concepts[0].Importance)
	assert.Equal(t, 7, processed[0].Concepts[1].Importance)
	assert.Equal(t, 9, processed[0].Concepts[2].Importance)
	assert.Equal(t, 6, processed[0].Concepts[3].Importance)

	// Verify all concepts exist
	for _, conceptRef := range processed[0].Concepts {
		concept, err := cp.conceptRepository.GetConcept(ctx, conceptRef.ConceptId)
		require.NoError(t, err)
		require.NotNil(t, concept)
	}
}

func TestConceptProcessor_Process_EmptyRecords(t *testing.T) {
	cp, _ := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Test with empty ID list
	err := cp.process(ctx)
	require.NoError(t, err)
}

func TestConceptProcessor_Process_NoConceptsClassified(t *testing.T) {
	cp, chatRepo := setupTestConceptProcessor(t)
	ctx := context.Background()

	// Create record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Simple text",
		Timestamp: time.Now().UTC(),
	}

	// Setup extractor to return empty concepts
	cp.extractor.(*testConceptExtractor).responses["Simple text"] = []ai.ExtractedConcept{}

	// Add record
	added, err := chatRepo.AddChatRecords(ctx, record)
	require.NoError(t, err)
	require.Len(t, added, 1)

	// Process
	err = cp.process(ctx, added[0].Id)
	require.NoError(t, err)

	// Verify no concepts assigned
	processed, err := chatRepo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, processed, 1)
	assert.Len(t, processed[0].Concepts, 0)
}

func TestEmbeddingProcessor_Process(t *testing.T) {
	chatRepo, _, cleanup := setupTestRepositories(t)
	defer cleanup()
	ctx := context.Background()

	embedder := &testEmbedder{
		embeddings: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
	}

	ep, err := newEmbeddingProcessor(chatRepo, embedder, nil)
	require.NoError(t, err)

	// Add records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "First message", Timestamp: time.Now().UTC()},
		{Speaker: core.SpeakerTypeHuman, Contents: "Second message", Timestamp: time.Now().UTC()},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 2)

	// Process
	ids := []core.ID{added[0].Id, added[1].Id}
	err = ep.process(ctx, ids...)
	require.NoError(t, err)

	// Verify embeddings assigned
	processed, err := chatRepo.GetChatRecords(ctx, ids...)
	require.NoError(t, err)
	require.Len(t, processed, 2)

	assert.Equal(t, []float32{0.1, 0.2, 0.3}, processed[0].Vector)
	assert.Equal(t, []float32{0.4, 0.5, 0.6}, processed[1].Vector)
}

func TestEmbeddingProcessor_Process_EmbedderError(t *testing.T) {
	chatRepo, _, cleanup := setupTestRepositories(t)
	defer cleanup()
	ctx := context.Background()

	embedder := &testEmbedder{
		shouldError: true,
	}

	ep, err := newEmbeddingProcessor(chatRepo, embedder, nil)
	require.NoError(t, err)

	// Add record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Test message",
		Timestamp: time.Now().UTC(),
	}

	added, err := chatRepo.AddChatRecords(ctx, record)
	require.NoError(t, err)
	require.Len(t, added, 1)

	// Process should fail
	err = ep.process(ctx, added[0].Id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedder error")
}

func TestNewPipeline(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepositories(t)
	defer cleanup()

	embedder := &testEmbedder{}
	extractor := &testConceptExtractor{responses: make(map[string][]ai.ExtractedConcept)}
	provider := &testAIProvider{embedder: embedder, extractor: extractor}

	t.Run("valid pipeline", func(t *testing.T) {
		pipeline, err := NewPipeline(chatRepo, conceptRepo, provider)
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()

		assert.NotNil(t, pipeline.chatRepository)
		assert.NotNil(t, pipeline.conceptRepository)
		assert.NotNil(t, pipeline.embeddingPool)
		assert.NotNil(t, pipeline.conceptPool)
	})

	t.Run("nil chat repository", func(t *testing.T) {
		_, err := NewPipeline(nil, conceptRepo, provider)
		assert.Equal(t, ErrChatRepositoryRequired, err)
	})

	t.Run("nil concept repository", func(t *testing.T) {
		_, err := NewPipeline(chatRepo, nil, provider)
		assert.Equal(t, ErrConceptRepositoryRequired, err)
	})

	t.Run("nil provider", func(t *testing.T) {
		_, err := NewPipeline(chatRepo, conceptRepo, nil)
		assert.Equal(t, ErrAIProviderRequired, err)
	})
}

func TestPipeline_WithOptions(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepositories(t)
	defer cleanup()

	embedder := &testEmbedder{}
	extractor := &testConceptExtractor{responses: make(map[string][]ai.ExtractedConcept)}
	provider := &testAIProvider{embedder: embedder, extractor: extractor}

	t.Run("with pool size", func(t *testing.T) {
		pipeline, err := NewPipeline(chatRepo, conceptRepo, provider, WithPoolSize(4))
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()

		// Pool exists and can accept work
		assert.NotNil(t, pipeline.embeddingPool)
		assert.NotNil(t, pipeline.conceptPool)
	})

	t.Run("with pool size zero defaults to 1", func(t *testing.T) {
		pipeline, err := NewPipeline(chatRepo, conceptRepo, provider, WithPoolSize(0))
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()
	})

	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.Default()
		pipeline, err := NewPipeline(chatRepo, conceptRepo, provider, WithLogger(logger))
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()

		assert.Equal(t, logger, pipeline.logger)
	})

	t.Run("with nil logger falls back to default", func(t *testing.T) {
		pipeline, err := NewPipeline(chatRepo, conceptRepo, provider, WithLogger(nil))
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()

		assert.NotNil(t, pipeline.logger)
	})

	t.Run("with multiple options", func(t *testing.T) {
		logger := slog.Default()
		pipeline, err := NewPipeline(
			chatRepo,
			conceptRepo,
			provider,
			WithPoolSize(2),
			WithLogger(logger),
		)
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		defer pipeline.Release()

		assert.Equal(t, logger, pipeline.logger)
	})
}

func TestPipeline_Ingest(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepositories(t)
	defer cleanup()

	embedder := &testEmbedder{
		embeddings: [][]float32{{0.1, 0.2, 0.3}},
	}
	extractor := &testConceptExtractor{
		responses: map[string][]ai.ExtractedConcept{
			"Hello world": {
				{Name: "greeting", Type: "action", Importance: 8},
			},
		},
	}
	provider := &testAIProvider{embedder: embedder, extractor: extractor}

	pipeline, err := NewPipeline(chatRepo, conceptRepo, provider, WithPoolSize(1))
	require.NoError(t, err)
	defer pipeline.Release()

	ctx := context.Background()

	t.Run("ingest single message", func(t *testing.T) {
		err := pipeline.Ingest(ctx, core.SpeakerTypeHuman, []string{"Hello world"}, nil)
		require.NoError(t, err)

		// Give async processors time to complete
		time.Sleep(100 * time.Millisecond)

		// Verify record was added
		records, err := chatRepo.GetChatRecordsByDateRange(ctx, time.Now().Add(-1*time.Minute), time.Now().Add(1*time.Minute))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 1)
	})

	t.Run("ingest multiple messages", func(t *testing.T) {
		err := pipeline.Ingest(ctx, core.SpeakerTypeAI, []string{"Message 1", "Message 2", "Message 3"}, nil)
		require.NoError(t, err)

		// Give async processors time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("ingest with no messages", func(t *testing.T) {
		err := pipeline.Ingest(ctx, core.SpeakerTypeHuman, []string{}, nil)
		require.NoError(t, err)
	})

	t.Run("ingest with metadata", func(t *testing.T) {
		metadata := map[string]string{
			"role":     "engineer",
			"provider": "anthropic",
		}
		err := pipeline.Ingest(ctx, core.SpeakerTypeHuman, []string{"Test with metadata"}, &IngestOptions{
			Metadata: metadata,
		})
		require.NoError(t, err)

		// Give async processors time to complete
		time.Sleep(100 * time.Millisecond)

		// Verify metadata was stored
		records, err := chatRepo.GetChatRecordsByDateRange(ctx, time.Now().Add(-1*time.Minute), time.Now().Add(1*time.Minute))
		require.NoError(t, err)
		var found *core.ChatRecord
		for _, r := range records {
			if r.Contents == "Test with metadata" {
				found = r
				break
			}
		}
		require.NotNil(t, found, "should find record with metadata")
		assert.Equal(t, "engineer", found.Metadata["role"])
		assert.Equal(t, "anthropic", found.Metadata["provider"])
	})
}

func TestPipeline_Release(t *testing.T) {
	chatRepo, conceptRepo, cleanup := setupTestRepositories(t)
	defer cleanup()

	embedder := &testEmbedder{}
	extractor := &testConceptExtractor{responses: make(map[string][]ai.ExtractedConcept)}
	provider := &testAIProvider{embedder: embedder, extractor: extractor}

	pipeline, err := NewPipeline(chatRepo, conceptRepo, provider)
	require.NoError(t, err)

	// Release should not panic
	pipeline.Release()

	// Multiple releases should not panic
	pipeline.Release()
}

func TestConceptProcessor_Checkpoint(t *testing.T) {
	cp, _ := setupTestConceptProcessor(t)

	// Checkpoint should not error (currently a no-op)
	err := cp.checkpoint()
	require.NoError(t, err)
}

func TestEmbeddingProcessor_Checkpoint(t *testing.T) {
	chatRepo, _, cleanup := setupTestRepositories(t)
	defer cleanup()

	embedder := &testEmbedder{}
	ep, err := newEmbeddingProcessor(chatRepo, embedder, nil)
	require.NoError(t, err)

	// Checkpoint should not error (currently a no-op)
	err = ep.checkpoint()
	require.NoError(t, err)
}
