package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "http://localhost:11434/v1", cfg.EmbeddingHost)
	assert.Equal(t, "http://localhost:11434/v1", cfg.ClassifierHost)
	assert.Equal(t, "embeddinggemma", cfg.EmbeddingModel)
	assert.Equal(t, "qwen2.5:3b", cfg.ClassifierModel)
	assert.Equal(t, 6, cfg.MinImportance)
}

func TestNewConfig(t *testing.T) {
	t.Run("with no options", func(t *testing.T) {
		cfg := NewConfig()

		assert.NotNil(t, cfg)
		// Should have default values
		assert.Equal(t, "http://localhost:11434/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://localhost:11434/v1", cfg.ClassifierHost)
		assert.Equal(t, 6, cfg.MinImportance)
	})

	t.Run("with custom host", func(t *testing.T) {
		cfg := NewConfig(WithHost("http://custom:8080/v1"))

		assert.Equal(t, "http://custom:8080/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://custom:8080/v1", cfg.ClassifierHost)
	})

	t.Run("with separate hosts", func(t *testing.T) {
		cfg := NewConfig(
			WithEmbeddingHost("http://embed:8080/v1"),
			WithClassifierHost("http://classify:9090/v1"),
		)

		assert.Equal(t, "http://embed:8080/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://classify:9090/v1", cfg.ClassifierHost)
	})

	t.Run("with custom models", func(t *testing.T) {
		cfg := NewConfig(
			WithEmbeddingModel("text-embedding-3-small"),
			WithClassifierModel("gpt-4o-mini"),
		)

		assert.Equal(t, "text-embedding-3-small", cfg.EmbeddingModel)
		assert.Equal(t, "gpt-4o-mini", cfg.ClassifierModel)
	})

	t.Run("with custom min importance", func(t *testing.T) {
		cfg := NewConfig(WithMinImportance(8))

		assert.Equal(t, 8, cfg.MinImportance)
	})

	t.Run("with multiple options", func(t *testing.T) {
		cfg := NewConfig(
			WithHost("http://custom:8080/v1"),
			WithEmbeddingModel("custom-embed"),
			WithClassifierModel("custom-classify"),
			WithMinImportance(7),
		)

		assert.Equal(t, "http://custom:8080/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://custom:8080/v1", cfg.ClassifierHost)
		assert.Equal(t, "custom-embed", cfg.EmbeddingModel)
		assert.Equal(t, "custom-classify", cfg.ClassifierModel)
		assert.Equal(t, 7, cfg.MinImportance)
	})
}

func TestConfigNormalize(t *testing.T) {
	tests := []struct {
		name               string
		embeddingHost      string
		classifierHost     string
		expectedEmbedding  string
		expectedClassifier string
	}{
		{
			name:               "already has /v1",
			embeddingHost:      "http://localhost:11434/v1",
			classifierHost:     "http://localhost:11434/v1",
			expectedEmbedding:  "http://localhost:11434/v1",
			expectedClassifier: "http://localhost:11434/v1",
		},
		{
			name:               "missing /v1",
			embeddingHost:      "http://localhost:11434",
			classifierHost:     "http://localhost:11434",
			expectedEmbedding:  "http://localhost:11434/v1",
			expectedClassifier: "http://localhost:11434/v1",
		},
		{
			name:               "has trailing slash",
			embeddingHost:      "http://localhost:11434/",
			classifierHost:     "http://localhost:11434/",
			expectedEmbedding:  "http://localhost:11434/v1",
			expectedClassifier: "http://localhost:11434/v1",
		},
		{
			name:               "has trailing slash and v1",
			embeddingHost:      "http://localhost:11434/v1",
			classifierHost:     "http://localhost:11434/v1",
			expectedEmbedding:  "http://localhost:11434/v1",
			expectedClassifier: "http://localhost:11434/v1",
		},
		{
			name:               "empty hosts",
			embeddingHost:      "",
			classifierHost:     "",
			expectedEmbedding:  "",
			expectedClassifier: "",
		},
		{
			name:               "different formats",
			embeddingHost:      "http://embed:8080",
			classifierHost:     "http://classify:9090/v1",
			expectedEmbedding:  "http://embed:8080/v1",
			expectedClassifier: "http://classify:9090/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				EmbeddingHost:  tt.embeddingHost,
				ClassifierHost: tt.classifierHost,
			}

			cfg.Normalize()

			assert.Equal(t, tt.expectedEmbedding, cfg.EmbeddingHost)
			assert.Equal(t, tt.expectedClassifier, cfg.ClassifierHost)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434",
			ClassifierHost:  "http://localhost:11434",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   6,
		}

		err := cfg.Validate()
		assert.NoError(t, err)

		// Should also normalize
		assert.Equal(t, "http://localhost:11434/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://localhost:11434/v1", cfg.ClassifierHost)
	})

	t.Run("missing embedding host", func(t *testing.T) {
		cfg := &Config{
			ClassifierHost:  "http://localhost:11434/v1",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   6,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EmbeddingHost")
	})

	t.Run("missing classifier host", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434/v1",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   6,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ClassifierHost")
	})

	t.Run("missing embedding model", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434/v1",
			ClassifierHost:  "http://localhost:11434/v1",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   6,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EmbeddingModel")
	})

	t.Run("missing classifier model", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:  "http://localhost:11434/v1",
			ClassifierHost: "http://localhost:11434/v1",
			EmbeddingModel: "embeddinggemma",
			MinImportance:  6,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ClassifierModel")
	})

	t.Run("min importance too low", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434/v1",
			ClassifierHost:  "http://localhost:11434/v1",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   0,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MinImportance")
	})

	t.Run("min importance too high", func(t *testing.T) {
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434/v1",
			ClassifierHost:  "http://localhost:11434/v1",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   11,
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MinImportance")
	})

	t.Run("min importance at boundaries", func(t *testing.T) {
		// Test min boundary (1)
		cfg := &Config{
			EmbeddingHost:   "http://localhost:11434/v1",
			ClassifierHost:  "http://localhost:11434/v1",
			EmbeddingModel:  "embeddinggemma",
			ClassifierModel: "qwen2.5:3b",
			MinImportance:   1,
		}
		err := cfg.Validate()
		assert.NoError(t, err)

		// Test max boundary (10)
		cfg.MinImportance = 10
		err = cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestConfigOptions(t *testing.T) {
	t.Run("WithEmbeddingHost", func(t *testing.T) {
		cfg := &Config{}
		opt := WithEmbeddingHost("http://test:8080/v1")
		opt(cfg)

		assert.Equal(t, "http://test:8080/v1", cfg.EmbeddingHost)
	})

	t.Run("WithClassifierHost", func(t *testing.T) {
		cfg := &Config{}
		opt := WithClassifierHost("http://test:9090/v1")
		opt(cfg)

		assert.Equal(t, "http://test:9090/v1", cfg.ClassifierHost)
	})

	t.Run("WithHost sets both", func(t *testing.T) {
		cfg := &Config{}
		opt := WithHost("http://test:8080/v1")
		opt(cfg)

		assert.Equal(t, "http://test:8080/v1", cfg.EmbeddingHost)
		assert.Equal(t, "http://test:8080/v1", cfg.ClassifierHost)
	})

	t.Run("WithEmbeddingModel", func(t *testing.T) {
		cfg := &Config{}
		opt := WithEmbeddingModel("test-model")
		opt(cfg)

		assert.Equal(t, "test-model", cfg.EmbeddingModel)
	})

	t.Run("WithClassifierModel", func(t *testing.T) {
		cfg := &Config{}
		opt := WithClassifierModel("test-classifier")
		opt(cfg)

		assert.Equal(t, "test-classifier", cfg.ClassifierModel)
	})

	t.Run("WithMinImportance", func(t *testing.T) {
		cfg := &Config{}
		opt := WithMinImportance(7)
		opt(cfg)

		assert.Equal(t, 7, cfg.MinImportance)
	})
}

func TestConfigValidate_Integration(t *testing.T) {
	// Test that NewConfig produces a valid configuration
	cfg := NewConfig()
	err := cfg.Validate()
	require.NoError(t, err)

	// Test that DefaultConfig produces a valid configuration
	cfg = DefaultConfig()
	err = cfg.Validate()
	require.NoError(t, err)
}
