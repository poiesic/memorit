package badger

import (
	"context"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// ConceptRepository implements storage.ConceptRepository for BadgerDB.
type ConceptRepository struct {
	backend *Backend
}

var _ storage.ConceptRepository = (*ConceptRepository)(nil)

// NewConceptRepository creates a new ConceptRepository.
func NewConceptRepository(backend *Backend) (*ConceptRepository, error) {
	return &ConceptRepository{
		backend: backend,
	}, nil
}

// Close releases resources. ConceptRepository has no resources to release.
func (r *ConceptRepository) Close() error {
	return nil
}

// FindSimilar delegates to the backend.
func (r *ConceptRepository) FindSimilar(ctx context.Context, vector []float32, minSimilarity float32, limit int) ([]*core.SearchResult, error) {
	return r.backend.FindSimilar(ctx, vector, minSimilarity, limit)
}

// WithTransaction delegates to the backend.
func (r *ConceptRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.backend.WithTransaction(ctx, fn)
}

// AddConcepts adds one or more concepts to storage.
func (r *ConceptRepository) AddConcepts(ctx context.Context, concepts ...*core.Concept) ([]*core.Concept, error) {
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		for _, concept := range concepts {
			// Use content-based ID if not set
			if concept.Id == 0 {
				concept.Id = core.IDFromContent(concept.Tuple())
			}

			// Set timestamps
			concept.InsertedAt = time.Now().UTC()
			concept.UpdatedAt = concept.InsertedAt

			// Store primary record
			key := makeConceptKey(concept.Id)
			value := storage.MarshalConcept(concept)
			if err := tx.Set(key, value); err != nil {
				return err
			}

			// Store tuple index
			tupleKey := makeConceptTupleKey(concept.Name, concept.Type)
			if err := tx.Set(tupleKey, storage.MarshalID(concept.Id)); err != nil {
				return err
			}
		}
		return tx.Commit()
	}, true)

	return concepts, err
}

// UpdateConcepts updates existing concepts.
func (r *ConceptRepository) UpdateConcepts(ctx context.Context, concepts ...*core.Concept) ([]*core.Concept, error) {
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		for _, concept := range concepts {
			key := makeConceptKey(concept.Id)

			// Read old concept to detect changes
			old, err := readConcept(tx, key)
			if err != nil {
				return err
			}
			if old == nil {
				return storage.ErrNotFound
			}

			// Update timestamp
			concept.UpdatedAt = time.Now().UTC()

			// Store updated record
			value := storage.MarshalConcept(concept)
			if err := tx.Set(key, value); err != nil {
				return err
			}

			// Update tuple index if name or type changed
			if old.Name != concept.Name || old.Type != concept.Type {
				oldTupleKey := makeConceptTupleKey(old.Name, old.Type)
				if err := tx.Delete(oldTupleKey); err != nil {
					return err
				}
				newTupleKey := makeConceptTupleKey(concept.Name, concept.Type)
				if err := tx.Set(newTupleKey, storage.MarshalID(concept.Id)); err != nil {
					return err
				}
			}
		}
		return tx.Commit()
	}, true)

	return concepts, err
}

// DeleteConcepts removes concepts by their IDs.
func (r *ConceptRepository) DeleteConcepts(ctx context.Context, ids ...core.ID) error {
	return r.backend.WithTx(func(tx *badger.Txn) error {
		for _, id := range ids {
			key := makeConceptKey(id)

			// Read concept to get metadata for index cleanup
			concept, err := readConcept(tx, key)
			if err != nil {
				return err
			}
			if concept == nil {
				return storage.ErrNotFound
			}

			// Delete from tuple index
			tupleKey := makeConceptTupleKey(concept.Name, concept.Type)
			if err := tx.Delete(tupleKey); err != nil {
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

// GetConcept retrieves a single concept by ID.
func (r *ConceptRepository) GetConcept(ctx context.Context, id core.ID) (*core.Concept, error) {
	var result *core.Concept
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		key := makeConceptKey(id)
		var err error
		result, err = readConcept(tx, key)
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

// GetConcepts retrieves multiple concepts by their IDs.
func (r *ConceptRepository) GetConcepts(ctx context.Context, ids ...core.ID) ([]*core.Concept, error) {
	var result []*core.Concept
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		for _, id := range ids {
			key := makeConceptKey(id)
			concept, err := readConcept(tx, key)
			if err != nil {
				return err
			}
			if concept != nil {
				result = append(result, concept)
			}
		}
		return nil
	}, false)
	return result, err
}

// FindConceptByNameAndType finds a concept by its name and type tuple.
func (r *ConceptRepository) FindConceptByNameAndType(ctx context.Context, name, conceptType string) (*core.Concept, error) {
	var result *core.Concept
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		// Look up ID from tuple index
		tupleKey := makeConceptTupleKey(name, conceptType)
		item, err := tx.Get(tupleKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return storage.ErrNotFound
			}
			return err
		}

		var conceptID core.ID
		err = item.Value(func(val []byte) error {
			conceptID, err = storage.UnmarshalID(val)
			return err
		})
		if err != nil {
			return err
		}

		// Look up full concept
		conceptKey := makeConceptKey(conceptID)
		result, err = readConcept(tx, conceptKey)
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

// GetOrCreateConcept finds or creates a concept by name and type.
func (r *ConceptRepository) GetOrCreateConcept(ctx context.Context, name, conceptType string, vector []float32) (*core.Concept, error) {
	// Try to find existing concept
	concept, err := r.FindConceptByNameAndType(ctx, name, conceptType)
	if err == nil {
		return concept, nil
	}
	if err != storage.ErrNotFound {
		return nil, err
	}

	// Create new concept
	newConcept := &core.Concept{
		Id:     core.IDFromContent("(" + conceptType + "," + name + ")"),
		Name:   name,
		Type:   conceptType,
		Vector: vector,
	}

	// Try to add it (may fail due to race condition)
	added, err := r.AddConcepts(ctx, newConcept)
	if err != nil {
		// If add failed, try to find it again (someone else may have created it)
		concept, findErr := r.FindConceptByNameAndType(ctx, name, conceptType)
		if findErr == nil {
			return concept, nil
		}
		return nil, err
	}

	return added[0], nil
}

// GetAllConcepts retrieves all concepts from storage.
func (r *ConceptRepository) GetAllConcepts(ctx context.Context) ([]*core.Concept, error) {
	var results []*core.Concept
	err := r.backend.WithTx(func(tx *badger.Txn) error {
		// Create iterator to scan all concept keys
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iter := tx.NewIterator(opts)
		defer iter.Close()

		// Seek to first concept key
		prefix := []byte(conceptRecordPrefix + ":")
		for iter.Seek(prefix); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()

			// Stop if we've moved past concept keys
			if !hasPrefix(key, prefix) {
				break
			}

			// Read the concept
			var concept *core.Concept
			err := item.Value(func(val []byte) error {
				var err error
				concept, err = storage.UnmarshalConcept(val)
				return err
			})
			if err != nil {
				return err
			}

			if concept != nil {
				results = append(results, concept)
			}
		}
		return nil
	}, false)

	return results, err
}

// Helper methods

// hasPrefix checks if a byte slice has a given prefix
func hasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && string(s[:len(prefix)]) == string(prefix)
}

// readConcept reads a concept from the transaction.
func readConcept(tx *badger.Txn, key []byte) (*core.Concept, error) {
	item, err := tx.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	var concept *core.Concept
	err = item.Value(func(val []byte) error {
		var err error
		concept, err = storage.UnmarshalConcept(val)
		return err
	})
	return concept, err
}
