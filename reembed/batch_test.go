package reembed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbedder for testing
type mockEmbedder struct {
	embedTextFunc  func(ctx context.Context, text string) ([]float32, error)
	embedTextsFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	if m.embedTextFunc != nil {
		return m.embedTextFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedTextsFunc != nil {
		return m.embedTextsFunc(ctx, texts)
	}
	// Default: return unnormalized vectors for each text
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1.0, 2.0, 2.0} // magnitude = 3.0
	}
	return result, nil
}

func TestBatchProcessor_Process(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test 1", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "test 2", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)

	// Verify records were updated with normalized vectors
	updated, err := repo.GetChatRecords(ctx, added[0].Id, added[1].Id)
	require.NoError(t, err)
	require.Len(t, updated, 2)

	for _, record := range updated {
		require.NotEmpty(t, record.Vector, "should have embedding")
		// Verify normalization: magnitude should be ~1.0
		var magnitude float32
		for _, v := range record.Vector {
			magnitude += v * v
		}
		assert.InDelta(t, 1.0, magnitude, 0.01, "vector should be normalized")
	}
}

func TestBatchProcessor_EmptyBatch(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	embedder := &mockEmbedder{}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err := processor.Process(ctx, []*core.ChatRecord{})
	require.NoError(t, err, "empty batch should not error")
}

func TestBatchProcessor_EmbeddingError(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	expectedErr := errors.New("embedding error")
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return nil, expectedErr
		},
	}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.Error(t, err)
	// With retry, should eventually return the error
	assert.Contains(t, err.Error(), "embedding error")
}

func TestBatchProcessor_Retry(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	attempts := 0
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			attempts++
			if attempts < 2 {
				return nil, errors.New("temporary error")
			}
			// Success on second attempt
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{1.0, 0.0, 0.0}
			}
			return result, nil
		},
	}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts, "should retry on failure")

	// Verify record was updated
	updated, err := repo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, updated, 1)
	require.NotEmpty(t, updated[0].Vector)
}

func TestBatchProcessor_ContextCancellation(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(context.Background(), records...)
	require.NoError(t, err)

	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			cancel() // Cancel during embedding
			return nil, errors.New("error")
		},
	}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestBatchProcessor_VectorNormalization(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test record
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	// Return a known unnormalized vector
	embedder := &mockEmbedder{
		embedTextsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			// Vector (3, 4) has magnitude 5
			return [][]float32{{3.0, 4.0}}, nil
		},
	}
	processor := NewBatchProcessor(repo, embedder, 3, 10*time.Millisecond)

	err = processor.Process(ctx, added)
	require.NoError(t, err)

	// Verify normalization
	updated, err := repo.GetChatRecords(ctx, added[0].Id)
	require.NoError(t, err)
	require.Len(t, updated, 1)

	vec := updated[0].Vector
	require.Len(t, vec, 2)

	// Should be normalized to (0.6, 0.8)
	assert.InDelta(t, 0.6, vec[0], 0.001)
	assert.InDelta(t, 0.8, vec[1], 0.001)

	// Verify magnitude is 1.0
	magnitude := vec[0]*vec[0] + vec[1]*vec[1]
	assert.InDelta(t, 1.0, magnitude, 0.001)
}
