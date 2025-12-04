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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// concept is an internal type used for processing extracted concepts.
type concept struct {
	Name       string
	Type       string
	Importance int
}

// Tuple returns a string representation of the concept as "(Type,Name)".
func (c concept) Tuple() string {
	return "(" + c.Type + "," + c.Name + ")"
}

// fromExtractedConcept converts ai.ExtractedConcept to internal concept type.
func fromExtractedConcept(ec ai.ExtractedConcept) concept {
	correctedType := strings.ReplaceAll(ec.Type, " ", "_")
	return concept{
		Name:       ec.Name,
		Type:       correctedType,
		Importance: ec.Importance,
	}
}

// ChatConceptExtractProcessor handles concept extraction for batches of chat records.
type ChatConceptExtractProcessor struct {
	chatRepo       storage.ChatRepository
	conceptRepo    storage.ConceptRepository
	embedder       ai.Embedder
	extractor      ai.ConceptExtractor
	maxRetries     int
	retryBaseDelay time.Duration
}

// NewChatConceptExtractProcessor creates a new chat concept extract processor.
// maxRetries: maximum number of retry attempts for AI API calls
// retryBaseDelay: base delay for exponential backoff
func NewChatConceptExtractProcessor(
	chatRepo storage.ChatRepository,
	conceptRepo storage.ConceptRepository,
	embedder ai.Embedder,
	extractor ai.ConceptExtractor,
	maxRetries int,
	retryBaseDelay time.Duration,
) *ChatConceptExtractProcessor {
	return &ChatConceptExtractProcessor{
		chatRepo:       chatRepo,
		conceptRepo:    conceptRepo,
		embedder:       embedder,
		extractor:      extractor,
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
	}
}

// recordConceptPos tracks where a concept should be assigned in the records
type recordConceptPos struct {
	recordIdx  int
	conceptIdx int
	importance int
}

// Process extracts concepts from a batch of chat records and updates them.
// For each record:
//  1. Extracts concepts using the AI ConceptExtractor
//  2. Creates/updates concepts with embeddings in the ConceptRepository
//  3. Updates the chat record with new ConceptRef assignments
func (p *ChatConceptExtractProcessor) Process(ctx context.Context, records []*core.ChatRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Build mapping of conceptID -> positions where it should be assigned
	conceptMapping := make(map[core.ID][]recordConceptPos)
	allConcepts := make([]concept, 0)
	conceptIDToIdx := make(map[core.ID]int) // track position in allConcepts slice
	var extractionErrors []error

	// Step 1: Extract concepts from all records
	for recordIdx, record := range records {
		var extracted []ai.ExtractedConcept
		err := RetryWithBackoff(ctx, func() error {
			var err error
			extracted, err = p.extractor.ExtractConcepts(ctx, record.Contents)
			return err
		}, p.maxRetries, p.retryBaseDelay)

		if err != nil {
			extractionErrors = append(extractionErrors, fmt.Errorf("record %d (%v) extraction failed: %w", recordIdx, record.Id, err))
			continue
		}

		// Convert ai.ExtractedConcept to internal concept type
		concepts := make([]concept, len(extracted))
		for i, ec := range extracted {
			concepts[i] = fromExtractedConcept(ec)
		}

		// Initialize the record's concepts array
		record.Concepts = make([]core.ConceptRef, len(concepts))

		// Build mapping for this record's concepts
		for conceptIdx, c := range concepts {
			conceptID := core.IDFromContent(c.Tuple())

			// Track the position where this concept should be assigned
			conceptMapping[conceptID] = append(conceptMapping[conceptID], recordConceptPos{
				recordIdx:  recordIdx,
				conceptIdx: conceptIdx,
				importance: c.Importance,
			})

			// Add to allConcepts if we haven't seen this concept yet
			if _, exists := conceptIDToIdx[conceptID]; !exists {
				conceptIDToIdx[conceptID] = len(allConcepts)
				allConcepts = append(allConcepts, c)
			}
		}
	}

	// Step 2: Generate embeddings for all unique concepts and create/update them
	var resolvedConcepts []*core.Concept
	if len(allConcepts) > 0 {
		var err error
		resolvedConcepts, err = p.getOrCreateConcepts(ctx, allConcepts)
		if err != nil {
			extractionErrors = append(extractionErrors, fmt.Errorf("concept creation failed: %w", err))
		}
	}

	// Step 3: Distribute concepts back to records using the mapping
	for _, resolvedConcept := range resolvedConcepts {
		positions := conceptMapping[resolvedConcept.Id]
		for _, pos := range positions {
			records[pos.recordIdx].Concepts[pos.conceptIdx] = core.ConceptRef{
				ConceptId:  resolvedConcept.Id,
				Importance: pos.importance,
			}
		}
	}

	// Step 4: Update all records in the database
	_, err := p.chatRepo.UpdateChatRecords(ctx, records...)
	if err != nil {
		extractionErrors = append(extractionErrors, fmt.Errorf("update records failed: %w", err))
	}

	// Return combined errors if any occurred
	if len(extractionErrors) > 0 {
		return errors.Join(extractionErrors...)
	}

	return nil
}

// getOrCreateConcepts gets or creates concepts with embeddings
func (p *ChatConceptExtractProcessor) getOrCreateConcepts(ctx context.Context, rawConcepts []concept) ([]*core.Concept, error) {
	// Generate embeddings for all concepts
	tuples := make([]string, len(rawConcepts))
	for i := range rawConcepts {
		tuples[i] = rawConcepts[i].Tuple()
	}

	var embeddings [][]float32
	err := RetryWithBackoff(ctx, func() error {
		var err error
		embeddings, err = p.embedder.EmbedTexts(ctx, tuples)
		return err
	}, p.maxRetries, p.retryBaseDelay)

	if err != nil {
		return nil, fmt.Errorf("failed to generate concept embeddings after %d attempts: %w", p.maxRetries, err)
	}

	if len(embeddings) != len(rawConcepts) {
		return nil, fmt.Errorf("embedding count mismatch: expected %d, got %d", len(rawConcepts), len(embeddings))
	}

	// Normalize embeddings
	for i := range embeddings {
		embeddings[i] = NormalizeVector(embeddings[i])
	}

	// Try to get or create each concept
	result := make([]*core.Concept, 0, len(rawConcepts))
	for i, rawConcept := range rawConcepts {
		// Use the repository's GetOrCreateConcept
		concept, err := p.conceptRepo.GetOrCreateConcept(ctx, rawConcept.Name, rawConcept.Type, embeddings[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get/create concept %s: %w", rawConcept.Tuple(), err)
		}
		result = append(result, concept)
	}

	return result, nil
}
