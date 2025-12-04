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


package ingestion

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// progressInterval is the number of records between progress log messages during recovery.
const progressInterval = 10

// Pipeline orchestrates the ingestion and processing of chat records.
// It manages concurrent processing of embeddings and concept extraction.
type Pipeline struct {
	chatRepository       storage.ChatRepository
	conceptRepository    storage.ConceptRepository
	checkpointRepository storage.CheckpointRepository
	embeddingPool        *ants.Pool
	conceptPool          *ants.Pool
	embeddingProc        processor
	conceptProc          processor
	contextTurns         int // Number of previous turns to include for concept extraction context
	logger               *slog.Logger
}

// Option configures a Pipeline.
type Option func(*Pipeline) error

// WithPoolSize sets the worker pool size for concurrent processing.
// Default is runtime.NumCPU() / 2, with a minimum of 1.
func WithPoolSize(size int) Option {
	return func(p *Pipeline) error {
		if size < 1 {
			size = 1
		}

		// Release old pools
		if p.embeddingPool != nil {
			p.embeddingPool.Release()
		}
		if p.conceptPool != nil {
			p.conceptPool.Release()
		}

		// Create new pools
		embeddingPool, err := ants.NewPool(size)
		if err != nil {
			return err
		}

		conceptPool, err := ants.NewPool(size)
		if err != nil {
			embeddingPool.Release()
			return err
		}

		p.embeddingPool = embeddingPool
		p.conceptPool = conceptPool
		return nil
	}
}

// WithLogger sets a custom logger.
// Default is slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(p *Pipeline) error {
		if logger == nil {
			logger = slog.Default()
		}
		p.logger = logger
		return nil
	}
}

// NewPipeline creates a new ingestion pipeline.
// On startup, it loads checkpoints and synchronously processes any pending records
// before returning. This ensures the pipeline is in a consistent state.
func NewPipeline(
	chatRepository storage.ChatRepository,
	conceptRepository storage.ConceptRepository,
	checkpointRepository storage.CheckpointRepository,
	provider ai.AIProvider,
	opts ...Option,
) (*Pipeline, error) {
	if chatRepository == nil {
		return nil, ErrChatRepositoryRequired
	}
	if conceptRepository == nil {
		return nil, ErrConceptRepositoryRequired
	}
	if checkpointRepository == nil {
		return nil, ErrCheckpointRepositoryRequired
	}
	if provider == nil {
		return nil, ErrAIProviderRequired
	}

	// Default logger
	logger := slog.Default()

	// Default pool size
	poolSize := runtime.NumCPU() / 2
	if poolSize < 1 {
		poolSize = 1
	}

	embeddingPool, err := ants.NewPool(poolSize)
	if err != nil {
		return nil, err
	}

	conceptsPool, err := ants.NewPool(poolSize)
	if err != nil {
		embeddingPool.Release()
		return nil, err
	}

	// Create pipeline with defaults
	p := &Pipeline{
		chatRepository:       chatRepository,
		conceptRepository:    conceptRepository,
		checkpointRepository: checkpointRepository,
		embeddingPool:        embeddingPool,
		conceptPool:          conceptsPool,
		contextTurns:         2, // Default: 2 turns (up to 4 previous messages)
		logger:               logger,
	}

	// Apply options (may override defaults)
	for _, opt := range opts {
		if optErr := opt(p); optErr != nil {
			p.Release()
			return nil, optErr
		}
	}

	// Create processors after options are applied (so they get final config)
	embeddingProc, err := newEmbeddingProcessor(chatRepository, checkpointRepository, provider.Embedder(), p.logger)
	if err != nil {
		p.Release()
		return nil, err
	}

	conceptProc, err := newConceptProcessor(chatRepository, conceptRepository, checkpointRepository,
		provider.Embedder(), provider.ConceptExtractor(), p.contextTurns, p.logger)
	if err != nil {
		p.Release()
		return nil, err
	}

	p.embeddingProc = embeddingProc
	p.conceptProc = conceptProc

	// Recover any pending records synchronously before returning
	if err := p.recover(context.Background()); err != nil {
		p.Release()
		return nil, err
	}

	return p, nil
}

// recover processes any pending records from before the last checkpoint.
// This is called synchronously during pipeline startup.
func (p *Pipeline) recover(ctx context.Context) error {
	// Load embedding checkpoint
	embeddingCheckpoint, err := p.checkpointRepository.LoadCheckpoint(ctx, ProcessorTypeEmbedding)
	if err != nil {
		return err
	}

	// Load concept checkpoint
	conceptCheckpoint, err := p.checkpointRepository.LoadCheckpoint(ctx, ProcessorTypeConcept)
	if err != nil {
		return err
	}

	// Determine the lowest checkpoint ID (we need to process records after this)
	// If either checkpoint is nil, we need to start from 0 to ensure that processor
	// gets all records it needs.
	var lowestCheckpointID core.ID
	if embeddingCheckpoint == nil || conceptCheckpoint == nil {
		// At least one processor has never run, start from beginning
		lowestCheckpointID = 0
	} else {
		// Both have checkpoints, use the lower one
		lowestCheckpointID = embeddingCheckpoint.LastID
		if conceptCheckpoint.LastID < lowestCheckpointID {
			lowestCheckpointID = conceptCheckpoint.LastID
		}
	}

	// Get all records after the lowest checkpoint
	pendingRecords, err := p.chatRepository.GetChatRecordsAfterID(ctx, lowestCheckpointID)
	if err != nil {
		return err
	}

	if len(pendingRecords) == 0 {
		p.logger.Info("no pending records to recover")
		return nil
	}

	p.logger.Info("recovering pending records", "count", len(pendingRecords))

	// Extract IDs
	allIDs := make([]core.ID, len(pendingRecords))
	for i, record := range pendingRecords {
		allIDs[i] = record.Id
	}

	// Process embeddings for records after embedding checkpoint
	var embeddingLastID core.ID
	if embeddingCheckpoint != nil {
		embeddingLastID = embeddingCheckpoint.LastID
	}
	embeddingIDs := filterIDsAfter(allIDs, embeddingLastID)
	if len(embeddingIDs) > 0 {
		p.logger.Info("recovering embeddings", "count", len(embeddingIDs))
		if err := p.processWithProgress(ctx, p.embeddingProc, "embeddings", embeddingIDs); err != nil {
			return err
		}
	}

	// Process concepts for records after concept checkpoint
	var conceptLastID core.ID
	if conceptCheckpoint != nil {
		conceptLastID = conceptCheckpoint.LastID
	}
	conceptIDs := filterIDsAfter(allIDs, conceptLastID)
	if len(conceptIDs) > 0 {
		p.logger.Info("recovering concepts", "count", len(conceptIDs))
		if err := p.processWithProgress(ctx, p.conceptProc, "concepts", conceptIDs); err != nil {
			return err
		}
	}

	p.logger.Info("recovery complete")
	return nil
}

// processWithProgress processes records in batches with progress logging.
func (p *Pipeline) processWithProgress(ctx context.Context, proc processor, name string, ids []core.ID) error {
	total := len(ids)
	for i := 0; i < total; i += progressInterval {
		end := i + progressInterval
		if end > total {
			end = total
		}

		batch := ids[i:end]
		if err := proc.process(ctx, batch...); err != nil {
			return err
		}

		if err := proc.checkpoint(); err != nil {
			p.logger.Error("error saving checkpoint during recovery", "processor", name, "err", err)
		}

		p.logger.Info("recovery progress", "processor", name, "processed", end, "total", total)
	}
	return nil
}

// filterIDsAfter returns IDs that are greater than afterID.
func filterIDsAfter(ids []core.ID, afterID core.ID) []core.ID {
	var result []core.ID
	for _, id := range ids {
		if id > afterID {
			result = append(result, id)
		}
	}
	return result
}

// IngestOptions holds optional parameters for ingestion.
type IngestOptions struct {
	Metadata  map[string]string // Optional metadata to attach to records
	Timestamp time.Time         // Optional timestamp (uses current time if zero)
}

// Ingest adds messages as chat records and processes them asynchronously.
// The speakerType is applied to all messages in the batch.
// Processing includes generating embeddings and extracting concepts.
// Errors during async processing are logged but do not fail the ingestion.
func (p *Pipeline) Ingest(ctx context.Context, speakerType core.SpeakerType, messages []string, opts *IngestOptions) error {
	if opts == nil {
		opts = &IngestOptions{}
	}

	// Create records
	records := make([]*core.ChatRecord, len(messages))
	for i, message := range messages {
		timestamp := opts.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now().UTC()
		}

		records[i] = &core.ChatRecord{
			Speaker:   speakerType,
			Contents:  message,
			Timestamp: timestamp,
			Metadata:  opts.Metadata,
		}
	}

	// Add to storage
	added, err := p.chatRepository.AddChatRecords(ctx, records...)
	if err != nil {
		return err
	}

	if len(added) == 0 {
		return nil
	}

	// Extract IDs
	ids := make([]core.ID, len(added))
	for i, record := range added {
		ids[i] = record.Id
	}

	// Submit for async processing
	p.embeddingPool.Submit(func() {
		if err := p.embeddingProc.process(context.Background(), ids...); err != nil {
			p.logger.Error("error processing embeddings", "err", err)
			return
		}
		if err := p.embeddingProc.checkpoint(); err != nil {
			p.logger.Error("error applying embedding checkpoint", "err", err)
		}
	})

	p.conceptPool.Submit(func() {
		if err := p.conceptProc.process(context.Background(), ids...); err != nil {
			p.logger.Error("error processing concepts", "err", err)
			return
		}
		if err := p.conceptProc.checkpoint(); err != nil {
			p.logger.Error("error applying concept checkpoint", "err", err)
		}
	})

	return nil
}

// Release releases resources including worker pools.
// The pipeline should not be used after calling Release.
func (p *Pipeline) Release() {
	if p.embeddingPool != nil {
		p.embeddingPool.Release()
	}
	if p.conceptPool != nil {
		p.conceptPool.Release()
	}
}
