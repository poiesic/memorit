package ingestion

import (
	"context"

	"github.com/poiesic/memorit/core"
)

// processor is an internal interface for processing chat records.
// Implementations handle specific enrichment tasks like embeddings or concept extraction.
type processor interface {
	// process enriches the chat records identified by the given IDs.
	process(ctx context.Context, ids ...core.ID) error

	// checkpoint saves the processor's current state.
	// Currently unimplemented but reserved for future checkpointing support.
	checkpoint() error
}
