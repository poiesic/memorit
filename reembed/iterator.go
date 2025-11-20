package reembed

import (
	"context"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

const (
	// DefaultBatchSize is the default number of records to fetch in each batch
	DefaultBatchSize = 100
)

// RecordIterator iterates over all chat records in batches.
type RecordIterator struct {
	repo      storage.ChatRepository
	batchSize int
}

// NewRecordIterator creates a new record iterator.
// batchSize: number of records to fetch in each batch (must be > 0)
func NewRecordIterator(repo storage.ChatRepository, batchSize int) *RecordIterator {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	return &RecordIterator{
		repo:      repo,
		batchSize: batchSize,
	}
}

// ForEach iterates over all chat records, calling fn for each batch.
// Iteration stops on first error from fn or when all records are processed.
// Context cancellation is checked between batches.
func (it *RecordIterator) ForEach(ctx context.Context, fn func([]*core.ChatRecord) error) error {
	// Use a very wide date range to get all records
	startTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC)

	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Fetch all records using date range query
	records, err := it.repo.GetChatRecordsByDateRange(ctx, startTime, endTime)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		// No records to process
		return nil
	}

	// Process records in batches of batchSize
	for i := 0; i < len(records); i += it.batchSize {
		end := i + it.batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]

		// Call user function with batch
		if err := fn(batch); err != nil {
			return err
		}

		// Check context after each batch
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
