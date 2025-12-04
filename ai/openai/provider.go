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


package openai

import (
	"log/slog"

	"github.com/poiesic/memorit/ai"
)

// Provider implements ai.AIProvider using OpenAI-compatible services.
// It manages embedder and concept extractor instances.
type Provider struct {
	config    *ai.Config
	embedder  *Embedder
	extractor *ConceptExtractor
	logger    *slog.Logger
}

// NewProvider creates a new AI provider with OpenAI-compatible services.
// The config is validated and normalized before use.
//
// Returns ai.AIProvider interface (not *Provider) to enforce abstraction
// and prevent coupling to OpenAI-specific implementation details.
func NewProvider(config *ai.Config) (ai.AIProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create embedder (using internal constructor for concrete type)
	embedder, err := newEmbedder(config)
	if err != nil {
		return nil, err
	}

	// Create concept extractor (using internal constructor for concrete type)
	extractor, err := newConceptExtractor(config)
	if err != nil {
		return nil, err
	}

	return &Provider{
		config:    config,
		embedder:  embedder,
		extractor: extractor,
		logger:    slog.Default().With("component", "openai-provider"),
	}, nil
}

// Embedder returns the text embedding service.
func (p *Provider) Embedder() ai.Embedder {
	return p.embedder
}

// ConceptExtractor returns the concept extraction service.
func (p *Provider) ConceptExtractor() ai.ConceptExtractor {
	return p.extractor
}

// Close releases resources held by the provider.
// Currently a no-op as the underlying clients don't require explicit cleanup.
func (p *Provider) Close() error {
	p.logger.Debug("closing OpenAI provider")
	return nil
}
