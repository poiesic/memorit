package reembed

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// ChatConceptExtractor orchestrates the extraction of concepts from all chat records in a database.
type ChatConceptExtractor struct {
	chatRepo  storage.ChatRepository
	embedder  ai.Embedder
	extractor ai.ConceptExtractor
	config    *Config
	progress  io.Writer
	processor *ChatConceptExtractProcessor
	iterator  *RecordIterator
}

// NewChatConceptExtractor creates a new chat concept extractor.
// progress: where to write progress output (typically os.Stderr)
func NewChatConceptExtractor(
	chatRepo storage.ChatRepository,
	conceptRepo storage.ConceptRepository,
	embedder ai.Embedder,
	extractor ai.ConceptExtractor,
	config *Config,
	progress io.Writer,
) *ChatConceptExtractor {
	if config == nil {
		config = DefaultConfig()
	}

	processor := NewChatConceptExtractProcessor(
		chatRepo,
		conceptRepo,
		embedder,
		extractor,
		config.MaxRetries,
		config.RetryDelay,
	)
	iterator := NewRecordIterator(chatRepo, config.BatchSize)

	return &ChatConceptExtractor{
		chatRepo:  chatRepo,
		embedder:  embedder,
		extractor: extractor,
		config:    config,
		progress:  progress,
		processor: processor,
		iterator:  iterator,
	}
}

// Run executes the concept extraction operation.
// All chat records in the database will have concepts re-extracted and assigned.
// Progress is reported to the configured writer.
func (e *ChatConceptExtractor) Run(ctx context.Context) error {
	// First, count total records
	startTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC)

	allRecords, err := e.chatRepo.GetChatRecordsByDateRange(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query records: %w", err)
	}

	totalRecords := len(allRecords)
	if totalRecords == 0 {
		fmt.Fprintf(e.progress, "No records found in database (0 records)\n")
		return nil
	}

	fmt.Fprintf(e.progress, "Starting concept extraction for %d records (batch size: %d)\n",
		totalRecords, e.config.BatchSize)

	// Initialize progress tracker
	tracker := NewProgressTracker(e.progress, totalRecords, e.config.ReportInterval)
	tracker.Start()

	processed := 0

	// Process all records in batches
	err = e.iterator.ForEach(ctx, func(records []*core.ChatRecord) error {
		// Process this batch
		if err := e.processor.Process(ctx, records); err != nil {
			return fmt.Errorf("failed to process batch: %w", err)
		}

		// Update progress
		processed += len(records)
		tracker.Update(processed)

		return nil
	})

	if err != nil {
		return err
	}

	// Finish progress tracking
	tracker.Finish()

	elapsed := tracker.Elapsed()
	fmt.Fprintf(e.progress, "Concept extraction complete. Processed %d records in %v (%.1f records/sec)\n",
		totalRecords, elapsed.Round(time.Second), float64(totalRecords)/elapsed.Seconds())

	return nil
}
