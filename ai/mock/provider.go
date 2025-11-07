package mock

import "github.com/poiesic/memorit/ai"

// MockProvider is a test double for ai.AIProvider.
// It aggregates mock embedder and extractor instances.
type MockProvider struct {
	embedder  *MockEmbedder
	extractor *MockConceptExtractor
}

// NewMockProvider creates a new mock provider with default mock services.
//
// Returns ai.AIProvider interface for consistency with production constructors.
// Use GetMockEmbedder()/GetMockExtractor() to access concrete types for test assertions.
func NewMockProvider() ai.AIProvider {
	return &MockProvider{
		embedder:  NewMockEmbedder(),
		extractor: NewMockConceptExtractor(),
	}
}

// NewMockProviderWithServices creates a mock provider with custom mock services.
// This allows full control over the behavior of each service.
func NewMockProviderWithServices(embedder *MockEmbedder, extractor *MockConceptExtractor) ai.AIProvider {
	return &MockProvider{
		embedder:  embedder,
		extractor: extractor,
	}
}

// Embedder returns the mock embedder.
func (p *MockProvider) Embedder() ai.Embedder {
	return p.embedder
}

// ConceptExtractor returns the mock concept extractor.
func (p *MockProvider) ConceptExtractor() ai.ConceptExtractor {
	return p.extractor
}

// Close is a no-op for mock provider.
func (p *MockProvider) Close() error {
	return nil
}

// GetMockEmbedder returns the underlying mock embedder for test assertions.
// This allows tests to check call counts and inject custom behavior.
func (p *MockProvider) GetMockEmbedder() *MockEmbedder {
	return p.embedder
}

// GetMockExtractor returns the underlying mock extractor for test assertions.
// This allows tests to check call counts and inject custom behavior.
func (p *MockProvider) GetMockExtractor() *MockConceptExtractor {
	return p.extractor
}
