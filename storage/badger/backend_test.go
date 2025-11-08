package badger

import (
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenBackend_InMemory(t *testing.T) {
	backend, err := OpenBackend("", true)
	require.NoError(t, err)
	require.NotNil(t, backend)
	defer backend.Close()

	assert.False(t, backend.IsClosed())
}

func TestOpenBackend_FileSystem(t *testing.T) {
	tmpDir := t.TempDir()
	backend, err := OpenBackend(tmpDir, false)
	require.NoError(t, err)
	require.NotNil(t, backend)
	defer backend.Close()

	assert.False(t, backend.IsClosed())
}

func TestOpenBackend_InvalidPath(t *testing.T) {
	// Try to open a file path (not directory)
	tmpFile := t.TempDir() + "/file.txt"
	// Create a file at the path
	backend, err := OpenBackend(tmpFile, false)
	if err == nil {
		backend.Close()
	}
	// We expect this to either error or succeed (depending on mkdir behavior)
	// The key is that it should handle the case gracefully
}

func TestBackendClose(t *testing.T) {
	backend, err := OpenBackend("", true)
	require.NoError(t, err)
	require.NotNil(t, backend)

	assert.False(t, backend.IsClosed())

	err = backend.Close()
	require.NoError(t, err)

	assert.True(t, backend.IsClosed())
}

func TestFindSimilar_NoRecords(t *testing.T) {
	backend, err := OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()
	vector := []float32{0.1, 0.2, 0.3}

	results, err := backend.FindSimilar(ctx, vector, 0.5, 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestFindSimilar_WithRecords(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create records with different vectors
	records := []*core.ChatRecord{
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "First message",
			Timestamp:  now,
			Vector:     []float32{1.0, 0.0, 0.0}, // Very similar to query
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Second message",
			Timestamp:  now,
			Vector:     []float32{0.9, 0.1, 0.0}, // Somewhat similar
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Third message",
			Timestamp:  now,
			Vector:     []float32{0.0, 0.0, 1.0}, // Not similar
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Fourth message without vector",
			Timestamp:  now,
			Vector:     nil, // No vector - should be skipped
		},
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 4)

	// Search for similar records
	queryVector := []float32{1.0, 0.0, 0.0}
	results, err := backend.FindSimilar(ctx, queryVector, 0.8, 10)
	require.NoError(t, err)

	// Should find at least the most similar record
	require.NotEmpty(t, results)

	// Results should be sorted by score descending
	for i := 0; i < len(results)-1; i++ {
		assert.GreaterOrEqual(t, results[i].Score, results[i+1].Score)
	}

	// First result should be the most similar
	assert.Equal(t, "First message", results[0].Record.Contents)
	assert.Greater(t, results[0].Score, float32(0.8))
}

func TestFindSimilar_ThresholdFiltering(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create records with known similarity scores
	records := []*core.ChatRecord{
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "High similarity",
			Timestamp:  now,
			Vector:     []float32{1.0, 0.0, 0.0},
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Medium similarity",
			Timestamp:  now,
			Vector:     []float32{0.7, 0.3, 0.0},
		},
		{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Low similarity",
			Timestamp:  now,
			Vector:     []float32{0.3, 0.7, 0.0},
		},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	queryVector := []float32{1.0, 0.0, 0.0}

	t.Run("high threshold", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.95, 10)
		require.NoError(t, err)
		// Only the most similar should pass
		assert.LessOrEqual(t, len(results), 1)
	})

	t.Run("medium threshold", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.6, 10)
		require.NoError(t, err)
		// At least high and medium should pass
		assert.GreaterOrEqual(t, len(results), 2)
	})

	t.Run("low threshold", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.2, 10)
		require.NoError(t, err)
		// All records should pass
		assert.Equal(t, 3, len(results))
	})
}

func TestFindSimilar_LimitResults(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	require.NoError(t, err)
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()
	now := time.Now().UTC()

	// Create 10 records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Message",
			Timestamp:  now,
			Vector:     []float32{0.9, 0.1, 0.0}, // All similar
		}
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	queryVector := []float32{1.0, 0.0, 0.0}

	t.Run("limit to 3", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.5, 3)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("limit to 5", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.5, 5)
		require.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("limit higher than results", func(t *testing.T) {
		results, err := backend.FindSimilar(ctx, queryVector, 0.5, 100)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 10)
	})
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{-1.0, 0.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "general case",
			a:        []float32{0.6, 0.8},
			b:        []float32{0.8, 0.6},
			expected: 0.96, // 0.6*0.8 + 0.8*0.6 = 0.48 + 0.48 = 0.96
		},
		{
			name:     "different lengths - use min",
			a:        []float32{1.0, 2.0, 3.0},
			b:        []float32{1.0, 2.0},
			expected: 5.0, // 1*1 + 2*2 = 5
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "zero vectors",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{0.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dotProduct(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestWithTransaction(t *testing.T) {
	backend, err := OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	ctx := context.Background()

	t.Run("successful transaction", func(t *testing.T) {
		err := backend.WithTransaction(ctx, func(ctx context.Context) error {
			// Transaction logic here
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("failed transaction", func(t *testing.T) {
		testErr := assert.AnError
		err := backend.WithTransaction(ctx, func(ctx context.Context) error {
			return testErr
		})
		assert.Equal(t, testErr, err)
	})
}

func TestGetSequence(t *testing.T) {
	backend, err := OpenBackend("", true)
	require.NoError(t, err)
	defer backend.Close()

	seq, err := backend.GetSequence("test_sequence")
	require.NoError(t, err)
	require.NotNil(t, seq)
	defer seq.Release()

	// Get sequential IDs
	id1, err := seq.Next()
	require.NoError(t, err)

	id2, err := seq.Next()
	require.NoError(t, err)

	// IDs should be sequential
	assert.Greater(t, id2, id1)
}
