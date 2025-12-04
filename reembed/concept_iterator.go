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

	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// ConceptIterator iterates over all concepts in batches.
type ConceptIterator struct {
	repo      storage.ConceptRepository
	batchSize int
}

// NewConceptIterator creates a new concept iterator.
// batchSize: number of concepts to fetch in each batch (must be > 0)
func NewConceptIterator(repo storage.ConceptRepository, batchSize int) *ConceptIterator {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	return &ConceptIterator{
		repo:      repo,
		batchSize: batchSize,
	}
}

// ForEach iterates over all concepts, calling fn for each batch.
// Iteration stops on first error from fn or when all concepts are processed.
// Context cancellation is checked between batches.
func (it *ConceptIterator) ForEach(ctx context.Context, fn func([]*core.Concept) error) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Fetch all concepts
	concepts, err := it.repo.GetAllConcepts(ctx)
	if err != nil {
		return err
	}

	if len(concepts) == 0 {
		// No concepts to process
		return nil
	}

	// Process concepts in batches of batchSize
	for i := 0; i < len(concepts); i += it.batchSize {
		end := i + it.batchSize
		if end > len(concepts) {
			end = len(concepts)
		}

		batch := concepts[i:end]

		// Call user function with batch
		if err := fn(batch); err != nil {
			return err
		}

		// Check context after each batch
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}
