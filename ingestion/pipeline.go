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

// Pipeline orchestrates the ingestion and processing of chat records.
// It manages concurrent processing of embeddings and concept extraction.
type Pipeline struct {
	chatRepository    storage.ChatRepository
	conceptRepository storage.ConceptRepository
	embeddingPool     *ants.Pool
	conceptPool       *ants.Pool
	embeddingProc     processor
	conceptProc       processor
	contextTurns      int // Number of previous turns to include for concept extraction context
	logger            *slog.Logger
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
func NewPipeline(
	chatRepository storage.ChatRepository,
	conceptRepository storage.ConceptRepository,
	provider ai.AIProvider,
	opts ...Option,
) (*Pipeline, error) {
	if chatRepository == nil {
		return nil, ErrChatRepositoryRequired
	}
	if conceptRepository == nil {
		return nil, ErrConceptRepositoryRequired
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
		chatRepository:    chatRepository,
		conceptRepository: conceptRepository,
		embeddingPool:     embeddingPool,
		conceptPool:       conceptsPool,
		contextTurns:      2, // Default: 2 turns (up to 4 previous messages)
		logger:            logger,
	}

	// Apply options (may override defaults)
	for _, opt := range opts {
		if optErr := opt(p); optErr != nil {
			p.Release()
			return nil, optErr
		}
	}

	// Create processors after options are applied (so they get final config)
	embeddingProc, err := newEmbeddingProcessor(chatRepository, provider.Embedder(), p.logger)
	if err != nil {
		p.Release()
		return nil, err
	}

	conceptProc, err := newConceptProcessor(chatRepository, conceptRepository,
		provider.Embedder(), provider.ConceptExtractor(), p.contextTurns, p.logger)
	if err != nil {
		p.Release()
		return nil, err
	}

	p.embeddingProc = embeddingProc
	p.conceptProc = conceptProc

	return p, nil
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
