package mock

import (
	"context"
	"hash/fnv"
)

// MockEmbedder is a test double for ai.Embedder.
// It allows custom behavior injection via function fields.
type MockEmbedder struct {
	// EmbedTextFunc is called by EmbedText if set.
	// If nil, uses default deterministic behavior.
	EmbedTextFunc func(ctx context.Context, text string) ([]float32, error)

	// EmbedTextsFunc is called by EmbedTexts if set.
	// If nil, uses default deterministic behavior.
	EmbedTextsFunc func(ctx context.Context, texts []string) ([][]float32, error)

	callCount int
}

// NewMockEmbedder creates a mock embedder with default deterministic behavior.
// Note: Returns concrete type to allow test assertions via GetMockEmbedder().
func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{}
}

// EmbedText generates a deterministic embedding based on text hash.
func (m *MockEmbedder) EmbedText(ctx context.Context, text string) ([]float32, error) {
	m.callCount++

	if m.EmbedTextFunc != nil {
		return m.EmbedTextFunc(ctx, text)
	}

	// Default: generate deterministic vector from text hash
	return generateDeterministicVector(text, 384), nil
}

// EmbedTexts generates deterministic embeddings for multiple texts.
func (m *MockEmbedder) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	m.callCount++

	if m.EmbedTextsFunc != nil {
		return m.EmbedTextsFunc(ctx, texts)
	}

	// Default: generate deterministic vectors for each text
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = generateDeterministicVector(text, 384)
	}
	return embeddings, nil
}

// CallCount returns the number of times any method was called.
func (m *MockEmbedder) CallCount() int {
	return m.callCount
}

// Reset clears the call count.
func (m *MockEmbedder) Reset() {
	m.callCount = 0
	m.EmbedTextFunc = nil
	m.EmbedTextsFunc = nil
}

// generateDeterministicVector creates a deterministic embedding vector from text.
// It uses FNV hash to ensure the same text always produces the same vector.
func generateDeterministicVector(text string, dim int) []float32 {
	h := fnv.New32a()
	h.Write([]byte(text))
	seed := h.Sum32()

	vector := make([]float32, dim)
	for i := 0; i < dim; i++ {
		// Simple pseudo-random generation based on seed and index
		seed = seed*1664525 + 1013904223 // LCG constants
		vector[i] = float32(seed%1000) / 1000.0
	}

	// Normalize to unit vector
	var sumSquares float32
	for _, v := range vector {
		sumSquares += v * v
	}
	norm := float32(1.0)
	if sumSquares > 0 {
		norm = float32(1.0) / float32(sumSquares)
		for i := range vector {
			vector[i] *= norm
		}
	}

	return vector
}
