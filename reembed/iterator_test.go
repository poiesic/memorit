package reembed

import (
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (storage.ChatRepository, func()) {
	backend, err := badger.OpenBackend("", true) // in-memory
	require.NoError(t, err)

	repo, err := badger.NewChatRepository(backend)
	require.NoError(t, err)

	cleanup := func() {
		repo.Close()
		backend.Close()
	}

	return repo, cleanup
}

func TestRecordIterator_Basic(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add test records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test 1", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "test 2", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeHuman, Contents: "test 3", Timestamp: time.Now()},
	}
	added, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)
	require.Len(t, added, 3)

	// Iterate all records
	iter := NewRecordIterator(repo, 2) // Batch size of 2
	count := 0
	var ids []core.ID

	err = iter.ForEach(ctx, func(records []*core.ChatRecord) error {
		count += len(records)
		for _, r := range records {
			ids = append(ids, r.Id)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, count, "should iterate all 3 records")
	assert.Len(t, ids, 3, "should have 3 IDs")
}

func TestRecordIterator_BatchSizes(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add 10 records
	records := make([]*core.ChatRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test",
			Timestamp: time.Now(),
		}
	}
	_, err := repo.AddChatRecords(ctx, records...)
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
			iter := NewRecordIterator(repo, tt.batchSize)
			batchCount := 0
			totalRecords := 0

			err := iter.ForEach(ctx, func(records []*core.ChatRecord) error {
				batchCount++
				totalRecords += len(records)
				assert.LessOrEqual(t, len(records), tt.batchSize, "batch should not exceed batchSize")
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expectedBatch, batchCount, "batch count")
			assert.Equal(t, 10, totalRecords, "total records")
		})
	}
}

func TestRecordIterator_EmptyDatabase(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	iter := NewRecordIterator(repo, 10)
	called := false

	err := iter.ForEach(ctx, func(records []*core.ChatRecord) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.False(t, called, "callback should not be called for empty database")
}

func TestRecordIterator_ErrorHandling(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Add records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "test 1", Timestamp: time.Now()},
		{Speaker: core.SpeakerTypeAI, Contents: "test 2", Timestamp: time.Now()},
	}
	_, err := repo.AddChatRecords(ctx, records...)
	require.NoError(t, err)

	iter := NewRecordIterator(repo, 1)
	called := 0

	expectedErr := assert.AnError
	err = iter.ForEach(ctx, func(records []*core.ChatRecord) error {
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

func TestRecordIterator_ContextCancellation(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Add records
	records := make([]*core.ChatRecord, 5)
	for i := 0; i < 5; i++ {
		records[i] = &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "test",
			Timestamp: time.Now(),
		}
	}
	_, err := repo.AddChatRecords(context.Background(), records...)
	require.NoError(t, err)

	iter := NewRecordIterator(repo, 1)
	called := 0

	err = iter.ForEach(ctx, func(records []*core.ChatRecord) error {
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

func TestRecordIterator_InvalidBatchSize(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	// Zero batch size should be handled gracefully
	iter := NewRecordIterator(repo, 0)
	assert.Greater(t, iter.batchSize, 0, "should use default batch size for invalid input")

	// Negative batch size
	iter = NewRecordIterator(repo, -10)
	assert.Greater(t, iter.batchSize, 0, "should use default batch size for negative input")
}
