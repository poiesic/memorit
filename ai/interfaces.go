package ai

import "context"

// Embedder generates vector embeddings from text for semantic similarity search.
// Implementations must be thread-safe for concurrent use.
type Embedder interface {
	// EmbedText generates a vector embedding for a single text string.
	// The returned vector represents the semantic meaning of the text.
	// Returns an error if the embedding generation fails.
	EmbedText(ctx context.Context, text string) ([]float32, error)

	// EmbedTexts generates vector embeddings for multiple text strings in a batch.
	// Batch processing is more efficient than calling EmbedText multiple times.
	// The returned slice contains embeddings in the same order as the input texts.
	// Returns an error if any embedding generation fails.
	EmbedTexts(ctx context.Context, texts []string) ([][]float32, error)
}

// ConceptExtractor extracts semantic concepts from text.
// Implementations must be thread-safe for concurrent use.
type ConceptExtractor interface {
	// ExtractConcepts analyzes text and extracts key concepts with their types
	// and importance scores. Concepts represent the main semantic entities
	// mentioned or implied in the text.
	// Returns an empty slice if no concepts are found.
	// Returns an error if concept extraction fails.
	ExtractConcepts(ctx context.Context, text string) ([]ExtractedConcept, error)
}

// ExtractedConcept represents a semantic concept identified in text.
// Each concept has a name (the concept itself), a type (category),
// and an importance score indicating its relevance to the text.
type ExtractedConcept struct {
	// Name is the concept identifier in lowercase, 1-3 words, singular form.
	// Example: "eiffel tower", "paris", "dog"
	Name string

	// Type categorizes the concept (e.g., "building", "place", "animal").
	// Must match one of the predefined concept types.
	Type string

	// Importance is a score from 1-10 indicating how central this concept
	// is to understanding the text. Higher scores = more important.
	Importance int
}

// AIProvider aggregates AI services for convenient initialization and lifecycle management.
// A provider creates and manages Embedder and ConceptExtractor instances,
// ensuring they share configuration and resources appropriately.
type AIProvider interface {
	// Embedder returns the text embedding service.
	// The returned Embedder is safe for concurrent use.
	Embedder() Embedder

	// ConceptExtractor returns the concept extraction service.
	// The returned ConceptExtractor is safe for concurrent use.
	ConceptExtractor() ConceptExtractor

	// Close releases resources held by the provider and its services.
	// After Close is called, the provider and its services should not be used.
	Close() error
}
