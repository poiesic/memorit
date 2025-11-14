package storage

import (
	"context"
	"time"

	"github.com/poiesic/memorit/core"
)

// Repository provides common storage operations shared across all repositories.
// Implementations must be thread-safe and support concurrent access.
type Repository interface {
	// FindSimilar finds chat records similar to the given vector.
	// Returns records with similarity >= minSimilarity, up to limit results.
	// Results are ordered by similarity score (highest first).
	FindSimilar(ctx context.Context, vector []float32, minSimilarity float32, limit int) ([]*core.SearchResult, error)

	// WithTransaction executes a function within a transaction.
	// If fn returns an error, the transaction is rolled back.
	// If fn returns nil, the transaction is committed.
	// The context passed to fn may contain transaction state.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// Close closes the storage backend and releases resources.
	Close() error
}

// ChatRepository provides operations for managing chat records.
type ChatRepository interface {
	Repository
	// AddChatRecords adds one or more chat records to storage.
	// For records with ID=0, generates new IDs from sequence.
	// Sets InsertedAt timestamp if not already set.
	// Returns the records with generated IDs and timestamps populated.
	AddChatRecords(ctx context.Context, records ...*core.ChatRecord) ([]*core.ChatRecord, error)

	// UpdateChatRecords updates existing chat records.
	// Updates the UpdatedAt timestamp automatically.
	// Returns ErrNotFound if any record doesn't exist.
	UpdateChatRecords(ctx context.Context, records ...*core.ChatRecord) ([]*core.ChatRecord, error)

	// DeleteChatRecords removes chat records by their IDs.
	// Also removes associated indices.
	// Returns ErrNotFound if any record doesn't exist.
	DeleteChatRecords(ctx context.Context, ids ...core.ID) error

	// GetChatRecord retrieves a single chat record by ID.
	// Returns ErrNotFound if the record doesn't exist.
	GetChatRecord(ctx context.Context, id core.ID) (*core.ChatRecord, error)

	// GetChatRecords retrieves multiple chat records by their IDs.
	// Returns only the records that exist (no error for missing records).
	GetChatRecords(ctx context.Context, ids ...core.ID) ([]*core.ChatRecord, error)

	// GetChatRecordsByDateRange retrieves chat records within a time range.
	// Returns records where start <= Timestamp < end, ordered by timestamp.
	GetChatRecordsByDateRange(ctx context.Context, start, end time.Time) ([]*core.ChatRecord, error)

	// GetRecentChatRecords retrieves the N most recent chat records, ordered by timestamp descending.
	// Returns up to limit records, with the most recent first.
	GetRecentChatRecords(ctx context.Context, limit int) ([]*core.ChatRecord, error)

	// GetChatRecordsByConcept retrieves IDs of chat records associated with a concept.
	// Returns only record IDs, not full records.
	GetChatRecordsByConcept(ctx context.Context, conceptID core.ID) ([]core.ID, error)
}

// ConceptRepository provides operations for managing concepts.
type ConceptRepository interface {
	Repository
	// AddConcepts adds one or more concepts to storage.
	// Uses content-based IDs (IDFromContent of concept tuple).
	// Sets InsertedAt timestamp if not already set.
	// Returns the concepts with timestamps populated.
	AddConcepts(ctx context.Context, concepts ...*core.Concept) ([]*core.Concept, error)

	// UpdateConcepts updates existing concepts.
	// Updates the UpdatedAt timestamp automatically.
	// Returns ErrNotFound if any concept doesn't exist.
	UpdateConcepts(ctx context.Context, concepts ...*core.Concept) ([]*core.Concept, error)

	// DeleteConcepts removes concepts by their IDs.
	// Returns ErrNotFound if any concept doesn't exist.
	DeleteConcepts(ctx context.Context, ids ...core.ID) error

	// GetConcept retrieves a single concept by ID.
	// Returns ErrNotFound if the concept doesn't exist.
	GetConcept(ctx context.Context, id core.ID) (*core.Concept, error)

	// GetConcepts retrieves multiple concepts by their IDs.
	// Returns only the concepts that exist (no error for missing concepts).
	GetConcepts(ctx context.Context, ids ...core.ID) ([]*core.Concept, error)

	// FindConceptByNameAndType finds a concept by its name and type tuple.
	// Returns ErrNotFound if no matching concept exists.
	FindConceptByNameAndType(ctx context.Context, name, conceptType string) (*core.Concept, error)

	// GetOrCreateConcept finds or creates a concept by name and type.
	// If the concept exists, returns it.
	// If not, creates it with the provided vector.
	// Thread-safe: handles concurrent creation attempts.
	GetOrCreateConcept(ctx context.Context, name, conceptType string, vector []float32) (*core.Concept, error)
}
