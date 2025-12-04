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


package reembed

import (
	"context"
	"fmt"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// BatchProcessor handles embedding generation for batches of chat records.
type BatchProcessor struct {
	repo           storage.ChatRepository
	embedder       ai.Embedder
	maxRetries     int
	retryBaseDelay time.Duration
}

// NewBatchProcessor creates a new batch processor.
// maxRetries: maximum number of retry attempts for embedding API calls
// retryBaseDelay: base delay for exponential backoff
func NewBatchProcessor(repo storage.ChatRepository, embedder ai.Embedder, maxRetries int, retryBaseDelay time.Duration) *BatchProcessor {
	return &BatchProcessor{
		repo:           repo,
		embedder:       embedder,
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
	}
}

// Process generates embeddings for a batch of records and updates them in the database.
// Vectors are normalized after embedding to ensure compatibility with cosine similarity.
func (bp *BatchProcessor) Process(ctx context.Context, records []*core.ChatRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Extract text content
	texts := make([]string, len(records))
	for i, record := range records {
		texts[i] = record.Contents
	}

	// Generate embeddings with retry
	var embeddings [][]float32
	err := RetryWithBackoff(ctx, func() error {
		var err error
		embeddings, err = bp.embedder.EmbedTexts(ctx, texts)
		return err
	}, bp.maxRetries, bp.retryBaseDelay)

	if err != nil {
		return fmt.Errorf("failed to generate embeddings after %d attempts: %w", bp.maxRetries, err)
	}

	if len(embeddings) != len(records) {
		return fmt.Errorf("embedding count mismatch: expected %d, got %d", len(records), len(embeddings))
	}

	// Normalize vectors and assign to records
	for i := range records {
		records[i].Vector = NormalizeVector(embeddings[i])
	}

	// Update records in database
	_, err = bp.repo.UpdateChatRecords(ctx, records...)
	if err != nil {
		return fmt.Errorf("failed to update records: %w", err)
	}

	return nil
}
