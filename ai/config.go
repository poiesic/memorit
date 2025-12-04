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


package ai

import (
	"errors"
	"strings"
)

// Config holds configuration for AI service providers.
type Config struct {
	// EmbeddingHost is the base URL for the embedding service API.
	// Example: "http://localhost:11434/v1" for local OpenAI-compatible server
	EmbeddingHost string

	// ClassifierHost is the base URL for the classification/extraction service API.
	// Example: "http://localhost:11434/v1" for local OpenAI-compatible server
	ClassifierHost string

	// EmbeddingModel is the model identifier to use for text embeddings.
	// Example: "embeddinggemma", "text-embedding-3-small"
	EmbeddingModel string

	// ClassifierModel is the model identifier to use for concept extraction.
	// Example: "qwen2.5:3b", "gpt-4o-mini"
	ClassifierModel string

	// MinImportance is the minimum importance score (1-10) for extracted concepts.
	// Concepts with importance below this threshold are filtered out.
	// Default: 6
	MinImportance int
}

// ConfigOption is a functional option for configuring a Config.
type ConfigOption func(*Config)

// WithEmbeddingHost sets the embedding service host URL.
func WithEmbeddingHost(host string) ConfigOption {
	return func(c *Config) {
		c.EmbeddingHost = host
	}
}

// WithClassifierHost sets the classifier service host URL.
func WithClassifierHost(host string) ConfigOption {
	return func(c *Config) {
		c.ClassifierHost = host
	}
}

// WithHost sets both embedding and classifier hosts to the same URL.
func WithHost(host string) ConfigOption {
	return func(c *Config) {
		c.EmbeddingHost = host
		c.ClassifierHost = host
	}
}

// WithEmbeddingModel sets the embedding model identifier.
func WithEmbeddingModel(model string) ConfigOption {
	return func(c *Config) {
		c.EmbeddingModel = model
	}
}

// WithClassifierModel sets the classifier model identifier.
func WithClassifierModel(model string) ConfigOption {
	return func(c *Config) {
		c.ClassifierModel = model
	}
}

// WithMinImportance sets the minimum importance threshold for concept extraction.
func WithMinImportance(min int) ConfigOption {
	return func(c *Config) {
		c.MinImportance = min
	}
}

// DefaultConfig returns a Config with sensible defaults for local OpenAI-compatible services.
// By default, both embedding and classifier use the same host.
func DefaultConfig() *Config {
	defaultHost := "http://localhost:11434/v1"
	return &Config{
		EmbeddingHost:   defaultHost,
		ClassifierHost:  defaultHost,
		EmbeddingModel:  "embeddinggemma",
		ClassifierModel: "qwen2.5:3b",
		MinImportance:   6,
	}
}

// NewConfig creates a Config with the default values and applies the provided options.
// This is the recommended way to create a Config with custom settings.
//
// Example:
//   cfg := NewConfig(
//       WithHost("http://localhost:11434/v1"),
//       WithEmbeddingModel("text-embedding-3-small"),
//   )
//
// Example with different hosts:
//   cfg := NewConfig(
//       WithEmbeddingHost("http://localhost:11434/v1"),
//       WithClassifierHost("http://localhost:9100/v1"),
//   )
func NewConfig(opts ...ConfigOption) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Normalize ensures the configuration is in a canonical form.
// It automatically adds the /v1 suffix to hosts if missing, which is required
// by most OpenAI-compatible APIs (Ollama, LocalAI, vLLM, etc).
func (c *Config) Normalize() {
	// Ensure EmbeddingHost ends with /v1 for OpenAI-compatible APIs
	if c.EmbeddingHost != "" && !strings.HasSuffix(c.EmbeddingHost, "/v1") {
		// Remove trailing slash if present before adding /v1
		c.EmbeddingHost = strings.TrimSuffix(c.EmbeddingHost, "/")
		c.EmbeddingHost = c.EmbeddingHost + "/v1"
	}
	// Ensure ClassifierHost ends with /v1 for OpenAI-compatible APIs
	if c.ClassifierHost != "" && !strings.HasSuffix(c.ClassifierHost, "/v1") {
		// Remove trailing slash if present before adding /v1
		c.ClassifierHost = strings.TrimSuffix(c.ClassifierHost, "/")
		c.ClassifierHost = c.ClassifierHost + "/v1"
	}
}

// Validate checks that the configuration is valid and complete.
// It automatically normalizes the configuration before validation.
func (c *Config) Validate() error {
	// Normalize first to ensure hosts are in correct format
	c.Normalize()

	if c.EmbeddingHost == "" {
		return errors.New("ai config: EmbeddingHost is required")
	}
	if c.ClassifierHost == "" {
		return errors.New("ai config: ClassifierHost is required")
	}
	if c.EmbeddingModel == "" {
		return errors.New("ai config: EmbeddingModel is required")
	}
	if c.ClassifierModel == "" {
		return errors.New("ai config: ClassifierModel is required")
	}
	if c.MinImportance < 1 || c.MinImportance > 10 {
		return errors.New("ai config: MinImportance must be between 1 and 10")
	}
	return nil
}
