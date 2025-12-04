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


package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/ai/openai"
	"github.com/poiesic/memorit/reembed"
	"github.com/poiesic/memorit/storage/badger"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "memorit",
		Usage: "Semantic memory system for conversational data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Usage:   "Set logging level (debug, info, warn, error)",
				Value:   "info",
			},
		},
		Before:   setupLogger,
		Commands: []*cli.Command{
			{
				Name:   "reembed",
				Usage:  "Reembed all chat records with new embeddings",
				Action: reembedCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Usage:    "Path to BadgerDB database directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "embedding-host",
						Usage: "Embedding service host URL",
						Value: "http://localhost:11434/v1",
					},
					&cli.StringFlag{
						Name:     "embedding-model",
						Usage:    "Embedding model name",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Number of records to process in each batch",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "report-interval",
						Usage: "Report progress every N records",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "max-retries",
						Usage: "Maximum retry attempts for failed operations",
						Value: 3,
					},
					&cli.DurationFlag{
						Name:  "retry-delay",
						Usage: "Base delay for exponential backoff",
						Value: 1 * time.Second,
					},
				},
			},
			{
				Name:   "reembed-concepts",
				Usage:  "Reembed all concepts with new embeddings",
				Action: reembedConceptsCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Usage:    "Path to BadgerDB database directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "embedding-host",
						Usage: "Embedding service host URL",
						Value: "http://localhost:11434/v1",
					},
					&cli.StringFlag{
						Name:     "embedding-model",
						Usage:    "Embedding model name",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Number of concepts to process in each batch",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "report-interval",
						Usage: "Report progress every N concepts",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "max-retries",
						Usage: "Maximum retry attempts for failed operations",
						Value: 3,
					},
					&cli.DurationFlag{
						Name:  "retry-delay",
						Usage: "Base delay for exponential backoff",
						Value: 1 * time.Second,
					},
				},
			},
			{
				Name:   "extract-concepts",
				Usage:  "Re-extract concepts from all chat records",
				Action: extractConceptsCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "db",
						Aliases:  []string{"d"},
						Usage:    "Path to BadgerDB database directory",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "classifier-host",
						Usage:    "Classifier service host URL for concept extraction",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "classifier-model",
						Usage:    "Classifier model name for concept extraction",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "embedding-host",
						Usage: "Embedding service host URL (defaults to classifier-host if not specified)",
					},
					&cli.StringFlag{
						Name:     "embedding-model",
						Usage:    "Embedding model name for concept embeddings",
						Required: true,
					},
					&cli.IntFlag{
						Name:  "batch-size",
						Usage: "Number of records to process in each batch",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "report-interval",
						Usage: "Report progress every N records",
						Value: 100,
					},
					&cli.IntFlag{
						Name:  "max-retries",
						Usage: "Maximum retry attempts for failed operations",
						Value: 3,
					},
					&cli.DurationFlag{
						Name:  "retry-delay",
						Usage: "Base delay for exponential backoff",
						Value: 1 * time.Second,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func reembedCommand(c *cli.Context) error {
	ctx := context.Background()

	// Validate flags
	dbPath := c.String("db")
	if dbPath == "" {
		return fmt.Errorf("database path is required")
	}

	// Open database
	backend, err := badger.OpenBackend(dbPath, false)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer backend.Close()

	repo, err := badger.NewChatRepository(backend)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Create AI config
	aiConfig := ai.NewConfig(
		ai.WithEmbeddingHost(c.String("embedding-host")),
		ai.WithEmbeddingModel(c.String("embedding-model")),
		// Use dummy classifier values (not needed for reembedding)
		ai.WithClassifierHost(c.String("embedding-host")),
		ai.WithClassifierModel("dummy"),
	)

	if err := aiConfig.Validate(); err != nil {
		return fmt.Errorf("invalid AI configuration: %w", err)
	}

	// Create embedder
	embedder, err := openai.NewEmbedder(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create reembedding config
	reembedConfig := &reembed.Config{
		BatchSize:      c.Int("batch-size"),
		ReportInterval: c.Int("report-interval"),
		MaxRetries:     c.Int("max-retries"),
		RetryDelay:     c.Duration("retry-delay"),
	}

	// Validate config
	if reembedConfig.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}
	if reembedConfig.ReportInterval <= 0 {
		return fmt.Errorf("report-interval must be greater than 0")
	}
	if reembedConfig.MaxRetries <= 0 {
		return fmt.Errorf("max-retries must be greater than 0")
	}

	// Create reembedder
	reembedder := reembed.NewReembedder(repo, embedder, reembedConfig, os.Stderr)

	// Run reembedding
	fmt.Fprintf(os.Stderr, "Database: %s\n", dbPath)
	fmt.Fprintf(os.Stderr, "Embedding host: %s\n", c.String("embedding-host"))
	fmt.Fprintf(os.Stderr, "Embedding model: %s\n", c.String("embedding-model"))
	fmt.Fprintln(os.Stderr)

	if err := reembedder.Run(ctx); err != nil {
		return fmt.Errorf("reembedding failed: %w", err)
	}

	return nil
}

func reembedConceptsCommand(c *cli.Context) error {
	ctx := context.Background()

	// Validate flags
	dbPath := c.String("db")
	if dbPath == "" {
		return fmt.Errorf("database path is required")
	}

	// Open database
	backend, err := badger.OpenBackend(dbPath, false)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer backend.Close()

	repo, err := badger.NewConceptRepository(backend)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Create AI config
	aiConfig := ai.NewConfig(
		ai.WithEmbeddingHost(c.String("embedding-host")),
		ai.WithEmbeddingModel(c.String("embedding-model")),
		// Use dummy classifier values (not needed for reembedding)
		ai.WithClassifierHost(c.String("embedding-host")),
		ai.WithClassifierModel("dummy"),
	)

	if err := aiConfig.Validate(); err != nil {
		return fmt.Errorf("invalid AI configuration: %w", err)
	}

	// Create embedder
	embedder, err := openai.NewEmbedder(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create reembedding config
	reembedConfig := &reembed.Config{
		BatchSize:      c.Int("batch-size"),
		ReportInterval: c.Int("report-interval"),
		MaxRetries:     c.Int("max-retries"),
		RetryDelay:     c.Duration("retry-delay"),
	}

	// Validate config
	if reembedConfig.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}
	if reembedConfig.ReportInterval <= 0 {
		return fmt.Errorf("report-interval must be greater than 0")
	}
	if reembedConfig.MaxRetries <= 0 {
		return fmt.Errorf("max-retries must be greater than 0")
	}

	// Create concept reembedder
	reembedder := reembed.NewConceptReembedder(repo, embedder, reembedConfig, os.Stderr)

	// Run reembedding
	fmt.Fprintf(os.Stderr, "Database: %s\n", dbPath)
	fmt.Fprintf(os.Stderr, "Embedding host: %s\n", c.String("embedding-host"))
	fmt.Fprintf(os.Stderr, "Embedding model: %s\n", c.String("embedding-model"))
	fmt.Fprintln(os.Stderr)

	if err := reembedder.Run(ctx); err != nil {
		return fmt.Errorf("concept reembedding failed: %w", err)
	}

	return nil
}

func extractConceptsCommand(c *cli.Context) error {
	ctx := context.Background()

	// Validate flags
	dbPath := c.String("db")
	if dbPath == "" {
		return fmt.Errorf("database path is required")
	}

	// Get classifier host (required)
	classifierHost := c.String("classifier-host")
	if classifierHost == "" {
		return fmt.Errorf("classifier-host is required")
	}

	// Get embedding host (defaults to classifier host if not specified)
	embeddingHost := c.String("embedding-host")
	if embeddingHost == "" {
		embeddingHost = classifierHost
	}

	// Open database
	backend, err := badger.OpenBackend(dbPath, false)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer backend.Close()

	chatRepo, err := badger.NewChatRepository(backend)
	if err != nil {
		return fmt.Errorf("failed to create chat repository: %w", err)
	}
	defer chatRepo.Close()

	conceptRepo, err := badger.NewConceptRepository(backend)
	if err != nil {
		return fmt.Errorf("failed to create concept repository: %w", err)
	}
	defer conceptRepo.Close()

	// Create AI config
	aiConfig := ai.NewConfig(
		ai.WithEmbeddingHost(embeddingHost),
		ai.WithEmbeddingModel(c.String("embedding-model")),
		ai.WithClassifierHost(classifierHost),
		ai.WithClassifierModel(c.String("classifier-model")),
	)

	if err := aiConfig.Validate(); err != nil {
		return fmt.Errorf("invalid AI configuration: %w", err)
	}

	// Create embedder and extractor
	embedder, err := openai.NewEmbedder(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	extractor, err := openai.NewConceptExtractor(aiConfig)
	if err != nil {
		return fmt.Errorf("failed to create concept extractor: %w", err)
	}

	// Create extraction config
	extractConfig := &reembed.Config{
		BatchSize:      c.Int("batch-size"),
		ReportInterval: c.Int("report-interval"),
		MaxRetries:     c.Int("max-retries"),
		RetryDelay:     c.Duration("retry-delay"),
	}

	// Validate config
	if extractConfig.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}
	if extractConfig.ReportInterval <= 0 {
		return fmt.Errorf("report-interval must be greater than 0")
	}
	if extractConfig.MaxRetries <= 0 {
		return fmt.Errorf("max-retries must be greater than 0")
	}

	// Create extractor
	conceptExtractor := reembed.NewChatConceptExtractor(
		chatRepo,
		conceptRepo,
		embedder,
		extractor,
		extractConfig,
		os.Stderr,
	)

	// Run extraction
	fmt.Fprintf(os.Stderr, "Database: %s\n", dbPath)
	fmt.Fprintf(os.Stderr, "Classifier host: %s\n", classifierHost)
	fmt.Fprintf(os.Stderr, "Classifier model: %s\n", c.String("classifier-model"))
	fmt.Fprintf(os.Stderr, "Embedding host: %s\n", embeddingHost)
	fmt.Fprintf(os.Stderr, "Embedding model: %s\n", c.String("embedding-model"))
	fmt.Fprintln(os.Stderr)

	if err := conceptExtractor.Run(ctx); err != nil {
		return fmt.Errorf("concept extraction failed: %w", err)
	}

	return nil
}

func setupLogger(c *cli.Context) error {
	// Get log level from flag and normalize to lowercase
	levelStr := strings.ToLower(c.String("log-level"))

	// Map string to slog.Level
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", levelStr)
	}

	// Configure slog with the specified level
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	return nil
}
