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


// Package ai provides abstractions for AI services used in Memorit.
//
// This package defines interfaces for AI operations including text embeddings
// and concept extraction. It follows the dependency inversion principle,
// allowing the core domain and business logic to depend on abstractions
// rather than concrete implementations.
//
// # Design Principles
//
// The package is designed around three key interfaces:
//
//   - Embedder: Generates vector embeddings from text
//   - ConceptExtractor: Extracts semantic concepts from text
//   - AIProvider: Aggregates AI services for convenient initialization
//
// # Implementation Packages
//
// The ai package includes two implementation sub-packages:
//
//   - ai/openai: Production implementation using OpenAI-compatible APIs
//   - ai/mock: Test doubles for unit testing without external dependencies
//
// # Constructor Return Type Pattern
//
// This package follows a mixed constructor pattern based on use case:
//
// Public constructors (openai.NewProvider, openai.NewEmbedder, etc.) return
// INTERFACE types to enforce abstraction and prevent accidental coupling to
// concrete implementations. This is essential for dependency injection and
// supporting multiple implementations.
//
//	provider, err := openai.NewProvider(config)  // returns ai.AIProvider
//
// Test utility constructors (mock.NewMockEmbedder, mock.NewMockConceptExtractor)
// return CONCRETE types to enable test assertions and behavior injection via
// the mock's public methods (CallCount, WithXFunc, Reset, etc.).
//
//	mockEmbed := mock.NewMockEmbedder()  // returns *mock.MockEmbedder
//	mockEmbed.WithEmbedTextFunc(...)     // needs concrete type
//	count := mockEmbed.CallCount()       // test assertion
//
// The mock.NewMockProvider() returns an interface since it's the primary entry
// point, but provides GetMockEmbedder()/GetMockExtractor() methods to access
// concrete types for assertions when needed.
//
// # Usage Example
//
//	// Production usage with OpenAI provider
//	config := ai.DefaultConfig()
//	provider, err := openai.NewProvider(config, 6)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	embeddings, err := provider.Embedder().EmbedText(ctx, "Hello world")
//	concepts, err := provider.ConceptExtractor().ExtractConcepts(ctx, "The Eiffel Tower is in Paris")
//
//	// Testing usage with mocks
//	mockProvider := mock.NewMockProvider()
//	embeddings, err := mockProvider.Embedder().EmbedText(ctx, "test text")
//
// # Architecture Benefits
//
//   - Testability: Business logic can be tested without external AI services
//   - Flexibility: AI providers can be swapped without changing business logic
//   - Maintainability: Clear boundaries between AI services and domain logic
//   - Extensibility: New providers can be added by implementing interfaces
package ai
