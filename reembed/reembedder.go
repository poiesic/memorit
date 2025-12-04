// Copyright 2025 Poiesic Systems
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


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

// Config holds configuration for the reembedding operation.
type Config struct {
	// BatchSize is the number of records to process in each batch
	BatchSize int

	// ReportInterval is how often to report progress (number of records)
	ReportInterval int

	// MaxRetries is the maximum number of retry attempts for failed operations
	MaxRetries int

	// RetryDelay is the base delay for exponential backoff
	RetryDelay time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BatchSize:      100,
		ReportInterval: 100,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
	}
}

// Reembedder orchestrates the reembedding of all chat records in a database.
type Reembedder struct {
	repo      storage.ChatRepository
	embedder  ai.Embedder
	config    *Config
	progress  io.Writer
	processor *BatchProcessor
	iterator  *RecordIterator
}

// NewReembedder creates a new reembedder.
// progress: where to write progress output (typically os.Stderr)
func NewReembedder(repo storage.ChatRepository, embedder ai.Embedder, config *Config, progress io.Writer) *Reembedder {
	if config == nil {
		config = DefaultConfig()
	}

	processor := NewBatchProcessor(repo, embedder, config.MaxRetries, config.RetryDelay)
	iterator := NewRecordIterator(repo, config.BatchSize)

	return &Reembedder{
		repo:      repo,
		embedder:  embedder,
		config:    config,
		progress:  progress,
		processor: processor,
		iterator:  iterator,
	}
}

// Run executes the reembedding operation.
// All chat records in the database will be reembedded with the configured embedder.
// Progress is reported to the configured writer.
func (r *Reembedder) Run(ctx context.Context) error {
	// First, count total records
	startTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC)

	allRecords, err := r.repo.GetChatRecordsByDateRange(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query records: %w", err)
	}

	totalRecords := len(allRecords)
	if totalRecords == 0 {
		fmt.Fprintf(r.progress, "No records found in database (0 records)\n")
		return nil
	}

	fmt.Fprintf(r.progress, "Starting reembedding of %d records (batch size: %d)\n",
		totalRecords, r.config.BatchSize)

	// Initialize progress tracker
	tracker := NewProgressTracker(r.progress, totalRecords, r.config.ReportInterval)
	tracker.Start()

	processed := 0

	// Process all records in batches
	err = r.iterator.ForEach(ctx, func(records []*core.ChatRecord) error {
		// Process this batch
		if err := r.processor.Process(ctx, records); err != nil {
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
	fmt.Fprintf(r.progress, "Reembedding complete. Processed %d records in %v (%.1f records/sec)\n",
		totalRecords, elapsed.Round(time.Second), float64(totalRecords)/elapsed.Seconds())

	return nil
}
