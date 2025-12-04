package badger

import (
	"context"
	"slices"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// ChatRepository implements storage.ChatRepository for BadgerDB.
type ChatRepository struct {
	backend *Backend
	idSeq   *badger.Sequence
}

var _ storage.ChatRepository = (*ChatRepository)(nil)

// NewChatRepository creates a new ChatRepository.
func NewChatRepository(backend *Backend) (*ChatRepository, error) {
	idSeq, err := backend.GetSequence(chatRecordIDSeq)
	if err != nil {
		return nil, err
	}

	return &ChatRepository{
		backend: backend,
		idSeq:   idSeq,
	}, nil
}

// Close releases the ID sequence.
func (r *ChatRepository) Close() error {
	return r.idSeq.Release()
}

// FindSimilar delegates to the backend.
func (r *ChatRepository) FindSimilar(ctx context.Context, vector []float32, minSimilarity float32, limit int) ([]*core.SearchResult, error) {
	return r.backend.FindSimilar(ctx, vector, minSimilarity, limit)
}

// WithTransaction delegates to the backend.
func (r *ChatRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.backend.WithTransaction(ctx, fn)
}

// AddChatRecords adds one or more chat records to storage.
func (r *ChatRepository) AddChatRecords(ctx context.Context, records ...*core.ChatRecord) ([]*core.ChatRecord, error) {
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		// Generate IDs and set timestamps
		for _, record := range records {
			// Always generate new ID from sequence
			nextID, err := r.idSeq.Next()
			if err != nil {
				return err
			}
			// BadgerDB sequences can return 0 on first call, so we skip it
			if nextID == 0 {
				nextID, err = r.idSeq.Next()
				if err != nil {
					return err
				}
			}
			record.Id = core.ID(nextID)

			record.InsertedAt = time.Now().UTC()
			record.UpdatedAt = record.InsertedAt

			// Store primary record
			key := makeChatRecordKey(record.Id)
			value := storage.MarshalChatRecord(record)
			if err := tx.Set(key, value); err != nil {
				return err
			}

			// Update date index
			dateKey := makeChatDateKey(record.Timestamp, record.Id)
			if err := tx.Set(dateKey, storage.MarshalID(record.Id)); err != nil {
				return err
			}

			// Update concept index
			if err := r.updateConceptIndex(tx, record); err != nil {
				return err
			}
		}
		return tx.Commit()
	}, true)

	return records, err
}

// UpdateChatRecords updates existing chat records.
func (r *ChatRepository) UpdateChatRecords(ctx context.Context, records ...*core.ChatRecord) ([]*core.ChatRecord, error) {
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		for _, record := range records {
			key := makeChatRecordKey(record.Id)

			// Read old record to detect changes
			old, err := r.readChatRecord(tx, key)
			if err != nil {
				return err
			}
			if old == nil {
				return storage.ErrNotFound
			}

			// Update timestamp
			record.UpdatedAt = time.Now().UTC()

			// Store updated record
			value := storage.MarshalChatRecord(record)
			if err := tx.Set(key, value); err != nil {
				return err
			}

			// Update date index if timestamp changed
			if !old.Timestamp.Equal(record.Timestamp) {
				oldDateKey := makeChatDateKey(old.Timestamp, old.Id)
				if err := tx.Delete(oldDateKey); err != nil {
					return err
				}
				newDateKey := makeChatDateKey(record.Timestamp, record.Id)
				if err := tx.Set(newDateKey, storage.MarshalID(record.Id)); err != nil {
					return err
				}
			}

			// Update concept index if concepts changed
			if !conceptsEqual(old.Concepts, record.Concepts) {
				if err := r.deleteConceptIndex(tx, old); err != nil {
					return err
				}
				if err := r.updateConceptIndex(tx, record); err != nil {
					return err
				}
			}
		}
		return tx.Commit()
	}, true)

	return records, err
}

// DeleteChatRecords removes chat records by their IDs.
func (r *ChatRepository) DeleteChatRecords(ctx context.Context, ids ...core.ID) error {
	return r.backend.WithTx(func(tx *badger.Txn) error {
		for _, id := range ids {
			key := makeChatRecordKey(id)

			// Read record to get metadata for index cleanup
			record, err := r.readChatRecord(tx, key)
			if err != nil {
				return err
			}
			if record == nil {
				return storage.ErrNotFound
			}

			// Delete from date index
			dateKey := makeChatDateKey(record.Timestamp, record.Id)
			if err := tx.Delete(dateKey); err != nil {
				return err
			}

			// Delete from concept index
			if err := r.deleteConceptIndex(tx, record); err != nil {
				return err
			}

			// Delete primary record
			if err := tx.Delete(key); err != nil {
				return err
			}
		}
		return tx.Commit()
	}, true)
}

// GetChatRecord retrieves a single chat record by ID.
func (r *ChatRepository) GetChatRecord(ctx context.Context, id core.ID) (*core.ChatRecord, error) {
	var result *core.ChatRecord
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		key := makeChatRecordKey(id)
		var err error
		result, err = r.readChatRecord(tx, key)
		if err != nil {
			return err
		}
		if result == nil {
			return storage.ErrNotFound
		}
		return nil
	}, false)
	return result, err
}

// GetChatRecords retrieves multiple chat records by their IDs.
func (r *ChatRepository) GetChatRecords(ctx context.Context, ids ...core.ID) ([]*core.ChatRecord, error) {
	var result []*core.ChatRecord
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		for _, id := range ids {
			key := makeChatRecordKey(id)
			record, err := r.readChatRecord(tx, key)
			if err != nil {
				return err
			}
			if record != nil {
				result = append(result, record)
			}
		}
		return nil
	}, false)
	return result, err
}

// GetChatRecordsByDateRange retrieves chat records within a time range.
func (r *ChatRepository) GetChatRecordsByDateRange(ctx context.Context, start, end time.Time) ([]*core.ChatRecord, error) {
	if start.Equal(end) {
		end = start.Add(1 * time.Microsecond)
	}

	var results []*core.ChatRecord
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		startKey := makePartialChatDateKey(start)
		endKey := makePartialChatDateKey(end)
		iter := tx.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(startKey); iter.Valid(); iter.Next() {
			key := iter.Item().Key()
			if slices.Compare(key, endKey) > 0 {
				break
			}

			// Read the ID from the index
			var recordID core.ID
			if err := iter.Item().Value(func(val []byte) error {
				var err error
				recordID, err = storage.UnmarshalID(val)
				return err
			}); err != nil {
				return err
			}

			// Look up the full record
			recordKey := makeChatRecordKey(recordID)
			record, err := r.readChatRecord(tx, recordKey)
			if err != nil {
				return err
			}
			if record != nil {
				results = append(results, record)
			}
		}
		return nil
	}, false)

	return results, err
}

// GetRecentChatRecords retrieves the N most recent chat records, ordered by timestamp descending.
func (r *ChatRepository) GetRecentChatRecords(ctx context.Context, limit int) ([]*core.ChatRecord, error) {
	var results []*core.ChatRecord
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		// Use reverse iterator to get most recent records first
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true

		iter := tx.NewIterator(opts)
		defer iter.Close()

		// Start from the end of the chat date prefix (to get all date-based records)
		// We seek to the last possible key with this prefix
		startKey := makePartialChatDateKey(time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC))

		// Prefix for chat date index keys
		prefix := []byte(chatRecordDatePrefix + ":")

		count := 0
		for iter.Seek(startKey); iter.Valid() && count < limit; iter.Next() {
			key := iter.Item().Key()

			// Check if we're still in the chat date index
			if len(key) < len(prefix) || slices.Compare(key[:len(prefix)], prefix) != 0 {
				break
			}

			// Read the ID from the index
			var recordID core.ID
			if err := iter.Item().Value(func(val []byte) error {
				var err error
				recordID, err = storage.UnmarshalID(val)
				return err
			}); err != nil {
				return err
			}

			// Look up the full record
			recordKey := makeChatRecordKey(recordID)
			record, err := r.readChatRecord(tx, recordKey)
			if err != nil {
				return err
			}
			if record != nil {
				results = append(results, record)
				count++
			}
		}
		return nil
	}, false)

	return results, err
}

// GetChatRecordsBeforeID retrieves chat records that occurred before the specified record ID,
// ordered by timestamp descending (newest first). This is used for lazy loading older messages.
func (r *ChatRepository) GetChatRecordsBeforeID(ctx context.Context, beforeID core.ID, limit int) ([]*core.ChatRecord, error) {
	var results []*core.ChatRecord

	err := r.backend.WithTx(func(tx *badger.Txn) error {
		// First, get the reference record to find its timestamp
		refKey := makeChatRecordKey(beforeID)
		refRecord, err := r.readChatRecord(tx, refKey)
		if err != nil {
			return err
		}
		if refRecord == nil {
			return storage.ErrNotFound
		}

		// Use reverse iterator to go backwards in time from this record
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true

		iter := tx.NewIterator(opts)
		defer iter.Close()

		// Start seeking from the reference record's date key
		// This will position us at or just before this record
		startKey := makeChatDateKey(refRecord.Timestamp, beforeID)

		// Prefix for chat date index keys
		prefix := []byte(chatRecordDatePrefix + ":")

		count := 0
		foundRef := false

		for iter.Seek(startKey); iter.Valid() && count < limit; iter.Next() {
			key := iter.Item().Key()

			// Check if we're still in the chat date index
			if len(key) < len(prefix) || slices.Compare(key[:len(prefix)], prefix) != 0 {
				break
			}

			// Read the ID from the index
			var recordID core.ID
			if err := iter.Item().Value(func(val []byte) error {
				var err error
				recordID, err = storage.UnmarshalID(val)
				return err
			}); err != nil {
				return err
			}

			// Skip the reference record itself
			if recordID == beforeID {
				foundRef = true
				continue
			}

			// Only include records after we've passed the reference
			if !foundRef {
				continue
			}

			// Look up the full record
			recordKey := makeChatRecordKey(recordID)
			record, err := r.readChatRecord(tx, recordKey)
			if err != nil {
				return err
			}
			if record != nil {
				results = append(results, record)
				count++
			}
		}
		return nil
	}, false)

	return results, err
}

// GetChatRecordsByConcept retrieves IDs of chat records associated with a concept.
func (r *ChatRepository) GetChatRecordsByConcept(ctx context.Context, conceptID core.ID) ([]core.ID, error) {
	var recordIDs []core.ID
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		startKey := makePartialChatConceptKey(conceptID)
		iter := tx.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(startKey); iter.Valid(); iter.Next() {
			key := iter.Item().Key()
			// Check if key still has our conceptID prefix
			if len(key) < len(startKey) {
				break
			}
			if slices.Compare(key[:len(startKey)], startKey) != 0 {
				break
			}

			// Read the recordID from the value
			var recordID core.ID
			err := iter.Item().Value(func(val []byte) error {
				var err error
				recordID, err = storage.UnmarshalID(val)
				return err
			})
			if err != nil {
				return err
			}
			recordIDs = append(recordIDs, recordID)
		}
		return nil
	}, false)

	return recordIDs, err
}

// GetConceptsByDateRange returns concepts referenced in messages falling within a date range
func (r *ChatRepository) GetConceptsByDateRange(ctx context.Context, start, end time.Time) ([]*core.Concept, error) {
	records, err := r.GetChatRecordsByDateRange(ctx, start, end)
	if err != nil {
		return nil, err
	}
	ids := make(map[core.ID]bool)
	for _, r := range records {
		for _, c := range r.Concepts {
			_, exists := ids[c.ConceptId]
			if !exists {
				ids[c.ConceptId] = true
			}
		}
	}
	var result []*core.Concept
	err = r.backend.WithTx(func(tx *badger.Txn) error {
		for id := range ids {
			key := makeConceptKey(id)
			concept, readErr := readConcept(tx, key)
			if readErr != nil {
				return readErr
			}
			if concept != nil {
				result = append(result, concept)
			}
		}
		return nil
	}, false)
	return result, err
}

// Helper methods

// readChatRecord reads a chat record from the transaction.
func (r *ChatRepository) readChatRecord(tx *badger.Txn, key []byte) (*core.ChatRecord, error) {
	item, err := tx.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var record *core.ChatRecord
	err = item.Value(func(val []byte) error {
		var unmarshalErr error
		record, unmarshalErr = storage.UnmarshalChatRecord(val)
		return unmarshalErr
	})
	return record, err
}

// updateConceptIndex adds concept index entries for a record.
func (r *ChatRepository) updateConceptIndex(tx *badger.Txn, record *core.ChatRecord) error {
	if len(record.Concepts) == 0 {
		return nil
	}
	for _, conceptRef := range record.Concepts {
		key := makeChatConceptKey(conceptRef.ConceptId, record.Id)
		value := storage.MarshalID(record.Id)
		if err := tx.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

// deleteConceptIndex removes concept index entries for a record.
func (r *ChatRepository) deleteConceptIndex(tx *badger.Txn, record *core.ChatRecord) error {
	if len(record.Concepts) == 0 {
		return nil
	}
	for _, conceptRef := range record.Concepts {
		key := makeChatConceptKey(conceptRef.ConceptId, record.Id)
		if err := tx.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

// conceptsEqual compares two concept slices for equality.
func conceptsEqual(a, b []core.ConceptRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ConceptId != b[i].ConceptId || a[i].Importance != b[i].Importance {
			return false
		}
	}
	return true
}
