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

// ConceptReembedder orchestrates the reembedding of all concepts in a database.
type ConceptReembedder struct {
	repo      storage.ConceptRepository
	embedder  ai.Embedder
	config    *Config
	progress  io.Writer
	processor *ConceptBatchProcessor
	iterator  *ConceptIterator
}

// NewConceptReembedder creates a new concept reembedder.
// progress: where to write progress output (typically os.Stderr)
func NewConceptReembedder(repo storage.ConceptRepository, embedder ai.Embedder, config *Config, progress io.Writer) *ConceptReembedder {
	if config == nil {
		config = DefaultConfig()
	}

	processor := NewConceptBatchProcessor(repo, embedder, config.MaxRetries, config.RetryDelay)
	iterator := NewConceptIterator(repo, config.BatchSize)

	return &ConceptReembedder{
		repo:      repo,
		embedder:  embedder,
		config:    config,
		progress:  progress,
		processor: processor,
		iterator:  iterator,
	}
}

// Run executes the reembedding operation.
// All concepts in the database will be reembedded with the configured embedder.
// Progress is reported to the configured writer.
func (r *ConceptReembedder) Run(ctx context.Context) error {
	// First, count total concepts
	allConcepts, err := r.repo.GetAllConcepts(ctx)
	if err != nil {
		return fmt.Errorf("failed to query concepts: %w", err)
	}

	totalConcepts := len(allConcepts)
	if totalConcepts == 0 {
		fmt.Fprintf(r.progress, "No concepts found in database (0 concepts)\n")
		return nil
	}

	fmt.Fprintf(r.progress, "Starting reembedding of %d concepts (batch size: %d)\n",
		totalConcepts, r.config.BatchSize)

	// Initialize progress tracker
	tracker := NewProgressTracker(r.progress, totalConcepts, r.config.ReportInterval)
	tracker.Start()

	processed := 0

	// Process all concepts in batches
	err = r.iterator.ForEach(ctx, func(concepts []*core.Concept) error {
		// Process this batch
		if err := r.processor.Process(ctx, concepts); err != nil {
			return fmt.Errorf("failed to process batch: %w", err)
		}

		// Update progress
		processed += len(concepts)
		tracker.Update(processed)

		return nil
	})

	if err != nil {
		return err
	}

	// Finish progress tracking
	tracker.Finish()

	elapsed := tracker.Elapsed()
	fmt.Fprintf(r.progress, "Reembedding complete. Processed %d concepts in %v (%.1f concepts/sec)\n",
		totalConcepts, elapsed.Round(time.Second), float64(totalConcepts)/elapsed.Seconds())

	return nil
}
