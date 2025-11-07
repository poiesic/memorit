// Package mock provides test double implementations of AI service interfaces.
//
// This package contains mock implementations of ai.Embedder, ai.ConceptExtractor,
// and ai.AIProvider for use in unit tests. The mocks allow tests to run without
// external AI service dependencies and enable controlled, deterministic behavior.
//
// # Usage in Tests
//
//	// Basic usage with default behavior
//	mockProvider := mock.NewMockProvider()
//	embeddings, err := mockProvider.Embedder().EmbedText(ctx, "test")
//
//	// Custom behavior injection
//	mockEmbedder := mock.NewMockEmbedder().
//	    WithEmbedTextFunc(func(ctx context.Context, text string) ([]float32, error) {
//	        return []float32{0.1, 0.2, 0.3}, nil
//	    })
//
//	// Check call counts
//	count := mockEmbedder.CallCount()
//
// # Default Behavior
//
// The mock implementations provide sensible defaults:
//
//   - MockEmbedder: Returns deterministic vectors based on text hash
//   - MockConceptExtractor: Extracts simple concepts from words in text
//   - MockProvider: Aggregates mock embedder and extractor
package mock
