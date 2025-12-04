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


package memorit

import (
	"log/slog"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/ai/openai"
	"github.com/poiesic/memorit/ingestion"
	"github.com/poiesic/memorit/search"
	"github.com/poiesic/memorit/storage"
	"github.com/poiesic/memorit/storage/badger"
)

type Database struct {
	backend        *badger.Backend
	chatRepo       storage.ChatRepository
	conceptRepo    storage.ConceptRepository
	checkpointRepo storage.CheckpointRepository
	provider       ai.AIProvider
	logger         *slog.Logger
}

// DatabaseOption configures a Database.
type DatabaseOption func(*databaseOptions)

type databaseOptions struct {
	aiConfig *ai.Config
}

func NewDatabase(filePath string, opts ...DatabaseOption) (*Database, error) {
	// Apply options
	options := &databaseOptions{
		aiConfig: ai.DefaultConfig(), // Default if not provided
	}
	for _, opt := range opts {
		opt(options)
	}
	// Open backend
	backend, err := badger.OpenBackend(filePath, false)
	if err != nil {
		return nil, err
	}

	// Create chat repository
	chatRepo, err := badger.NewChatRepository(backend)
	if err != nil {
		backend.Close()
		return nil, err
	}

	// Create concept repository
	conceptRepo, err := badger.NewConceptRepository(backend)
	if err != nil {
		chatRepo.Close()
		backend.Close()
		return nil, err
	}

	// Create checkpoint repository
	checkpointRepo := badger.NewCheckpointRepository(backend)

	// Create AI provider with configured settings
	provider, err := openai.NewProvider(options.aiConfig)
	if err != nil {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
		return nil, err
	}

	return &Database{
		backend:        backend,
		chatRepo:       chatRepo,
		conceptRepo:    conceptRepo,
		checkpointRepo: checkpointRepo,
		provider:       provider,
		logger:         slog.Default(),
	}, nil
}

func (db *Database) Close() error {
	// Close AI provider first
	if err := db.provider.Close(); err != nil {
		db.logger.Error("error closing AI provider", "err", err)
	}

	// Close repositories
	if err := db.conceptRepo.Close(); err != nil {
		db.logger.Error("error closing concept repository", "err", err)
		return err
	}
	if err := db.chatRepo.Close(); err != nil {
		db.logger.Error("error closing chat repository", "err", err)
		return err
	}

	// Close backend
	if err := db.backend.Close(); err != nil {
		db.logger.Error("error closing backend storage", "err", err)
		return err
	}
	return nil
}

func (db *Database) ChatRepository() storage.ChatRepository {
	return db.chatRepo
}

func (db *Database) ConceptRepository() storage.ConceptRepository {
	return db.conceptRepo
}

func (db *Database) NewIngestionPipeline(opts ...ingestion.Option) (*ingestion.Pipeline, error) {
	return ingestion.NewPipeline(db.chatRepo, db.conceptRepo, db.checkpointRepo, db.provider, opts...)
}

func (db *Database) CheckpointRepository() storage.CheckpointRepository {
	return db.checkpointRepo
}

func (db *Database) NewSearcher(opts ...search.Option) (*search.Searcher, error) {
	return search.NewSearcher(db.chatRepo, db.conceptRepo, db.provider, opts...)
}
