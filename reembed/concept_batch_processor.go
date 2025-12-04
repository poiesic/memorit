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

// ConceptBatchProcessor handles embedding generation for batches of concepts.
type ConceptBatchProcessor struct {
	repo           storage.ConceptRepository
	embedder       ai.Embedder
	maxRetries     int
	retryBaseDelay time.Duration
}

// NewConceptBatchProcessor creates a new concept batch processor.
// maxRetries: maximum number of retry attempts for embedding API calls
// retryBaseDelay: base delay for exponential backoff
func NewConceptBatchProcessor(repo storage.ConceptRepository, embedder ai.Embedder, maxRetries int, retryBaseDelay time.Duration) *ConceptBatchProcessor {
	return &ConceptBatchProcessor{
		repo:           repo,
		embedder:       embedder,
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
	}
}

// Process generates embeddings for a batch of concepts and updates them in the database.
// Vectors are normalized after embedding to ensure compatibility with cosine similarity.
// Concepts are embedded using their Tuple() representation: "(Type,Name)"
func (bp *ConceptBatchProcessor) Process(ctx context.Context, concepts []*core.Concept) error {
	if len(concepts) == 0 {
		return nil
	}

	// Extract tuple representations (Type,Name)
	tuples := make([]string, len(concepts))
	for i, concept := range concepts {
		tuples[i] = concept.Tuple()
	}

	// Generate embeddings with retry
	var embeddings [][]float32
	err := RetryWithBackoff(ctx, func() error {
		var err error
		embeddings, err = bp.embedder.EmbedTexts(ctx, tuples)
		return err
	}, bp.maxRetries, bp.retryBaseDelay)

	if err != nil {
		return fmt.Errorf("failed to generate embeddings after %d attempts: %w", bp.maxRetries, err)
	}

	if len(embeddings) != len(concepts) {
		return fmt.Errorf("embedding count mismatch: expected %d, got %d", len(concepts), len(embeddings))
	}

	// Normalize vectors and assign to concepts
	for i := range concepts {
		concepts[i].Vector = NormalizeVector(embeddings[i])
	}

	// Update concepts in database
	_, err = bp.repo.UpdateConcepts(ctx, concepts...)
	if err != nil {
		return fmt.Errorf("failed to update concepts: %w", err)
	}

	return nil
}
