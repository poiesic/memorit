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
