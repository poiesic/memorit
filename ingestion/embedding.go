package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// ProcessorTypeEmbedding is the checkpoint key for the embedding processor.
const ProcessorTypeEmbedding = "embedding"

// embeddingProcessor generates embeddings for chat records.
type embeddingProcessor struct {
	chatRepository       storage.ChatRepository
	checkpointRepository storage.CheckpointRepository
	embedder             ai.Embedder
	lastID               core.ID
	logger               *slog.Logger
}

var _ processor = (*embeddingProcessor)(nil)

// newEmbeddingProcessor creates a new embedding processor.
func newEmbeddingProcessor(
	chatRepository storage.ChatRepository,
	checkpointRepository storage.CheckpointRepository,
	embedder ai.Embedder,
	logger *slog.Logger,
) (processor, error) {
	if chatRepository == nil {
		return nil, fmt.Errorf("chat repository required")
	}
	if checkpointRepository == nil {
		return nil, fmt.Errorf("checkpoint repository required")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &embeddingProcessor{
		chatRepository:       chatRepository,
		checkpointRepository: checkpointRepository,
		embedder:             embedder,
		logger:               logger.With("processor", "embeddings"),
	}, nil
}

// process generates embeddings for the specified chat records.
func (ep *embeddingProcessor) process(ctx context.Context, ids ...core.ID) error {
	ep.logger.Info("processing records for embeddings", "records", len(ids))

	// Sort first so checkpointing works correctly
	slices.Sort(ids)

	records, err := ep.chatRepository.GetChatRecords(ctx, ids...)
	if err != nil {
		ep.logger.Error("error retrieving chat records", "err", err)
		return err
	}

	texts := make([]string, len(records))
	for i, record := range records {
		texts[i] = record.Contents
	}

	ep.logger.Debug("generating embeddings for chat records", "records", len(texts))
	embeddings, err := ep.embedder.EmbedTexts(ctx, texts)
	if err != nil {
		ep.logger.Error("error generating embeddings", "err", err)
		return err
	}

	if len(embeddings) != len(records) {
		return fmt.Errorf("embedding result mismatch. expected %d, received %d", len(records), len(embeddings))
	}

	for i := range embeddings {
		records[i].Vector = embeddings[i]
	}

	updated, err := ep.chatRepository.UpdateChatRecords(ctx, records...)
	if err != nil {
		return err
	}

	highestID := updated[len(updated)-1].Id
	if highestID > ep.lastID {
		ep.lastID = highestID
	}

	return nil
}

// checkpoint saves the processor's current state.
func (ep *embeddingProcessor) checkpoint() error {
	if ep.lastID == 0 {
		return nil
	}
	checkpoint := &core.Checkpoint{
		ProcessorType: ProcessorTypeEmbedding,
		LastID:        ep.lastID,
		UpdatedAt:     time.Now().UTC(),
	}
	return ep.checkpointRepository.SaveCheckpoint(context.Background(), checkpoint)
}
