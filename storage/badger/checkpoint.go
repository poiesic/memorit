package badger

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// CheckpointRepository implements storage.CheckpointRepository for BadgerDB.
type CheckpointRepository struct {
	backend *Backend
}

var _ storage.CheckpointRepository = (*CheckpointRepository)(nil)

// NewCheckpointRepository creates a new CheckpointRepository.
func NewCheckpointRepository(backend *Backend) *CheckpointRepository {
	return &CheckpointRepository{
		backend: backend,
	}
}

// SaveCheckpoint persists a checkpoint for a processor type.
func (r *CheckpointRepository) SaveCheckpoint(ctx context.Context, checkpoint *core.Checkpoint) error {
	return r.backend.WithTx(func(tx *badger.Txn) error {
		checkpoint.UpdatedAt = time.Now().UTC()
		key := makeCheckpointKey(checkpoint.ProcessorType)
		value := storage.MarshalCheckpoint(checkpoint)
		if err := tx.Set(key, value); err != nil {
			return err
		}
		return tx.Commit()
	}, true)
}

// LoadCheckpoint retrieves the checkpoint for a processor type.
// Returns nil, nil if no checkpoint exists.
func (r *CheckpointRepository) LoadCheckpoint(ctx context.Context, processorType string) (*core.Checkpoint, error) {
	var checkpoint *core.Checkpoint
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		key := makeCheckpointKey(processorType)
		item, err := tx.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			var unmarshalErr error
			checkpoint, unmarshalErr = storage.UnmarshalCheckpoint(val)
			return unmarshalErr
		})
	}, false)

	return checkpoint, err
}
