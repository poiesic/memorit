package badger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

const (
	defaultSequenceBandwidth = 100
)

// Backend wraps a BadgerDB instance and provides low-level operations.
type Backend struct {
	db     *badger.DB
	logger *slog.Logger
}

// badgerLoggerAdapter adapts slog.Logger to badger.Logger interface.
type badgerLoggerAdapter struct {
	logger *slog.Logger
}

var _ badger.Logger = (*badgerLoggerAdapter)(nil)

func (bl *badgerLoggerAdapter) Errorf(msg string, items ...any) {
	bl.logger.Error(fmt.Sprintf(msg, items...))
}

func (bl *badgerLoggerAdapter) Warningf(msg string, items ...any) {
	bl.logger.Warn(fmt.Sprintf(msg, items...))
}

func (bl *badgerLoggerAdapter) Infof(msg string, items ...any) {
	bl.logger.Info(fmt.Sprintf(msg, items...))
}

func (bl *badgerLoggerAdapter) Debugf(msg string, items ...any) {
	bl.logger.Debug(fmt.Sprintf(msg, items...))
}

// openBackend opens a BadgerDB database at the specified path.
// Creates the directory if it doesn't exist.
func OpenBackend(filePath string, inMemory bool) (*Backend, error) {
	var opts badger.Options

	if inMemory {
		opts = badger.DefaultOptions("").WithInMemory(true)
	} else {
		// Ensure directory exists
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filePath, 0755); err != nil {
					return nil, err
				}
				info, err = os.Stat(filePath)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", filePath)
		}
		opts = badger.DefaultOptions(filePath)
	}

	opts.Logger = &badgerLoggerAdapter{logger: slog.Default()}
	opts.Compression = options.None

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Backend{
		db:     db,
		logger: slog.Default(),
	}, nil
}

// Close closes the BadgerDB database.
func (b *Backend) Close() error {
	return b.db.Close()
}

// IsClosed returns true if the database is closed.
func (b *Backend) IsClosed() bool {
	return b.db.IsClosed()
}

// WithTx executes a function within a BadgerDB transaction.
// If isWrite is true, creates a read-write transaction.
// The transaction is automatically discarded if fn returns an error.
func (b *Backend) WithTx(fn func(tx *badger.Txn) error, isWrite bool) error {
	tx := b.db.NewTransaction(isWrite)
	defer tx.Discard()
	return fn(tx)
}

// GetSequence returns a BadgerDB sequence for generating sequential IDs.
func (b *Backend) GetSequence(name string) (*badger.Sequence, error) {
	return b.db.GetSequence([]byte(name), defaultSequenceBandwidth)
}

// WithTransaction executes a function within a transaction.
// Implements storage.TransactionManager interface.
func (b *Backend) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return b.WithTx(func(tx *badger.Txn) error {
		// Execute the callback function
		if err := fn(ctx); err != nil {
			return err
		}
		// Commit the transaction
		return tx.Commit()
	}, true)
}

// FindSimilar finds chat records similar to the given vector.
// Implements storage.VectorSearcher interface.
func (b *Backend) FindSimilar(ctx context.Context, vector []float32, minSimilarity float32, limit int) ([]*core.SearchResult, error) {
	var results []*core.SearchResult

	err := b.WithTx(func(tx *badger.Txn) error {
		// Iterate through all chat records
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(chatRecordPrefix)
		iter := tx.NewIterator(opts)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()

			// Skip index keys (date index, concept index, and sequence key)
			if bytes.Equal(key, []byte(chatRecordIDSeq)) ||
				bytes.HasPrefix(key, []byte(chatRecordDatePrefix)) ||
				bytes.HasPrefix(key, []byte(chatRecordConceptPrefix)) {
				continue
			}

			// Read the record
			var record *core.ChatRecord
			err := item.Value(func(val []byte) error {
				var err error
				record, err = storage.UnmarshalChatRecord(val)
				return err
			})
			if err != nil {
				return err
			}
			if record == nil {
				continue
			}

			// Skip records without embeddings
			if len(record.Vector) == 0 {
				continue
			}

			// Calculate cosine similarity (dot product for normalized vectors)
			similarity := dotProduct(vector, record.Vector)

			// Filter by threshold
			if similarity >= minSimilarity {
				results = append(results, &core.SearchResult{
					Record: record,
					Score:  similarity,
				})
			}
		}

		return nil
	}, false)

	if err != nil {
		return nil, err
	}

	// Sort by similarity descending
	slices.SortFunc(results, func(a, b *core.SearchResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		return 0
	})

	// Limit to maxHits
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// dotProduct calculates the dot product of two vectors.
func dotProduct(a, b []float32) float32 {
	var sum float32
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		sum += a[i] * b[i]
	}
	return sum
}
