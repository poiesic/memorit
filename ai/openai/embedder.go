package openai

import (
	"context"
	"log/slog"

	"github.com/poiesic/memorit/ai"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

// Embedder implements ai.Embedder using OpenAI-compatible embedding APIs.
type Embedder struct {
	embedder embeddings.Embedder
	logger   *slog.Logger
}

// newEmbedder is an internal constructor that returns the concrete type.
// Used by Provider to manage the instance.
func newEmbedder(config *ai.Config) (*Embedder, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create OpenAI client configured for embeddings
	// Use "none" as token for local OpenAI-compatible services that don't require authentication
	client, err := openai.New(
		openai.WithBaseURL(config.EmbeddingHost),
		openai.WithToken("none"),
		openai.WithEmbeddingModel(config.EmbeddingModel),
	)
	if err != nil {
		return nil, err
	}

	// Wrap in langchaingo embedder
	embedder, err := embeddings.NewEmbedder(client, embeddings.WithStripNewLines(true))
	if err != nil {
		return nil, err
	}

	return &Embedder{
		embedder: embedder,
		logger:   slog.Default().With("component", "openai-embedder"),
	}, nil
}

// NewEmbedder creates a new embedder using the provided configuration.
//
// Returns ai.Embedder interface to enforce abstraction.
func NewEmbedder(config *ai.Config) (ai.Embedder, error) {
	return newEmbedder(config)
}

// EmbedText generates a vector embedding for a single text string.
func (e *Embedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	e.logger.Debug("generating embedding for single text", "length", len(text))

	embeddings, err := e.embedder.EmbedDocuments(ctx, []string{text})
	if err != nil {
		e.logger.Error("failed to generate embedding", "err", err)
		return nil, err
	}

	if len(embeddings) == 0 {
		e.logger.Warn("embedder returned empty result")
		return []float32{}, nil
	}

	return embeddings[0], nil
}

// EmbedTexts generates vector embeddings for multiple text strings in a batch.
func (e *Embedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	e.logger.Debug("generating embeddings for texts", "count", len(texts))

	embeddings, err := e.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		e.logger.Error("failed to generate embeddings", "count", len(texts), "err", err)
		return nil, err
	}

	return embeddings, nil
}
