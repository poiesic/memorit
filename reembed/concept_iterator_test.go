package reembed

import (
	"context"
	"fmt"
	"testing"

	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConceptDB(t *testing.T) (storage.ConceptRepository, func()) {
	backend, err := badger.OpenBackend("", true) // in-memory
	require.NoError(t, err)

	repo, err := badger.NewConceptRepository(backend)
	require.NoError(t, err)

	cleanup := func() {
		repo.Close()
		backend.Close()
	}

	return repo, cleanup
}

func TestConceptIterator_Basic(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test concepts
	concepts := []*core.Concept{
		{Name: "Alice", Type: "person", Vector: []float32{0.1, 0.2}},
		{Name: "Bob", Type: "person", Vector: []float32{0.3, 0.4}},
		{Name: "Golang", Type: "technology", Vector: []float32{0.5, 0.6}},
	}
	added, err := repo.AddConcepts(ctx, concepts...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Iterate all concepts
	iter := NewConceptIterator(repo, 2) // Batch size of 2
	count := 0
	var ids []core.ID

	err = iter.ForEach(ctx, func(concepts []*core.Concept) error {
		count += len(concepts)
		for _, c := range concepts {
			ids = append(ids, c.Id)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, count, "should iterate all 3 concepts")
	assert.Len(t, ids, 3, "should have 3 IDs")
}

func TestConceptIterator_BatchSizes(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add 10 concepts (use unique names since concept IDs are content-based)
	concepts := make([]*core.Concept, 10)
	for i := 0; i < 10; i++ {
		concepts[i] = &core.Concept{
			Name:   fmt.Sprintf("concept_%d", i),
			Type:   "type",
			Vector: []float32{0.1, 0.2},
		}
	}
	_, err := repo.AddConcepts(ctx, concepts...)
	require.NoError(t, err)

	tests := []struct {
		name          string
		batchSize     int
		expectedBatch int
	}{
		{"batch size 1", 1, 10},
		{"batch size 3", 3, 4}, // 3+3+3+1
		{"batch size 5", 5, 2}, // 5+5
		{"batch size 10", 10, 1},
		{"batch size 100", 100, 1}, // All in one batch
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := NewConceptIterator(repo, tt.batchSize)
			batchCount := 0
			totalConcepts := 0

			err := iter.ForEach(ctx, func(concepts []*core.Concept) error {
				batchCount++
				totalConcepts += len(concepts)
				assert.LessOrEqual(t, len(concepts), tt.batchSize, "batch should not exceed batchSize")
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expectedBatch, batchCount, "batch count")
			assert.Equal(t, 10, totalConcepts, "total concepts")
		})
	}
}

func TestConceptIterator_EmptyDatabase(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	ctx := context.Background()

	iter := NewConceptIterator(repo, 10)
	called := false

	err := iter.ForEach(ctx, func(concepts []*core.Concept) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.False(t, called, "callback should not be called for empty database")
}

func TestConceptIterator_ErrorHandling(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add concepts
	concepts := []*core.Concept{
		{Name: "Alice", Type: "person", Vector: []float32{0.1, 0.2}},
		{Name: "Bob", Type: "person", Vector: []float32{0.3, 0.4}},
	}
	_, err := repo.AddConcepts(ctx, concepts...)
	require.NoError(t, err)

	iter := NewConceptIterator(repo, 1)
	called := 0

	expectedErr := assert.AnError
	err = iter.ForEach(ctx, func(concepts []*core.Concept) error {
		called++
		if called == 1 {
			return expectedErr
		}
		return nil
	})

	require.Error(t, err)
	assert.Equal(t, expectedErr, err, "should return callback error")
	assert.Equal(t, 1, called, "should stop on first error")
}

func TestConceptIterator_ContextCancellation(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Add concepts (use unique names since concept IDs are content-based)
	concepts := make([]*core.Concept, 5)
	for i := 0; i < 5; i++ {
		concepts[i] = &core.Concept{
			Name:   fmt.Sprintf("concept_%d", i),
			Type:   "type",
			Vector: []float32{0.1, 0.2},
		}
	}
	_, err := repo.AddConcepts(context.Background(), concepts...)
	require.NoError(t, err)

	iter := NewConceptIterator(repo, 1)
	called := 0

	err = iter.ForEach(ctx, func(concepts []*core.Concept) error {
		called++
		if called == 2 {
			cancel()
		}
		return nil
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 2, called, "should process until context canceled")
}

func TestConceptIterator_InvalidBatchSize(t *testing.T) {
	repo, cleanup := setupTestConceptDB(t)
	defer cleanup()

	// Zero batch size should be handled gracefully
	iter := NewConceptIterator(repo, 0)
	assert.Greater(t, iter.batchSize, 0, "should use default batch size for invalid input")

	// Negative batch size
	iter = NewConceptIterator(repo, -10)
	assert.Greater(t, iter.batchSize, 0, "should use default batch size for negative input")
}
