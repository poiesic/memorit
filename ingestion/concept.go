package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// concept is an internal type used for processing extracted concepts.
// It wraps ai.ExtractedConcept with additional helper methods.
type concept struct {
	Concept    string
	Type       string
	Importance int
}

// Tuple returns a string representation of the concept as "(Type,Name)".
// This is used for generating deterministic IDs.
func (c concept) Tuple() string {
	return "(" + c.Type + "," + c.Concept + ")"
}

// fromExtractedConcept converts ai.ExtractedConcept to internal concept type.
func fromExtractedConcept(ec ai.ExtractedConcept) concept {
	return concept{
		Concept:    ec.Name,
		Type:       ec.Type,
		Importance: ec.Importance,
	}
}

// conceptProcessor extracts concepts from chat records and assigns them.
type conceptProcessor struct {
	chatRepository    storage.ChatRepository
	conceptRepository storage.ConceptRepository
	embedder          ai.Embedder
	extractor         ai.ConceptExtractor
	contextTurns      int // Number of previous turns to include for context (0 = current message only)
	lastID            core.ID
	logger            *slog.Logger
}

var _ processor = (*conceptProcessor)(nil)

// recordConceptPos tracks where a concept should be assigned in the records
type recordConceptPos struct {
	recordIdx  int
	conceptIdx int
	importance int
}

// newConceptProcessor creates a new concept processor.
func newConceptProcessor(
	chatRepository storage.ChatRepository,
	conceptRepository storage.ConceptRepository,
	embedder ai.Embedder,
	extractor ai.ConceptExtractor,
	contextTurns int,
	logger *slog.Logger,
) (processor, error) {
	if chatRepository == nil {
		return nil, fmt.Errorf("chat repository required")
	}
	if conceptRepository == nil {
		return nil, fmt.Errorf("concept repository required")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder required")
	}
	if extractor == nil {
		return nil, fmt.Errorf("concept extractor required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &conceptProcessor{
		chatRepository:    chatRepository,
		conceptRepository: conceptRepository,
		embedder:          embedder,
		extractor:         extractor,
		contextTurns:      contextTurns,
		logger:            logger.With("processor", "concepts"),
	}, nil
}

// buildContextWindow builds the text context for concept extraction.
// If contextTurns > 0, fetches the most recent messages before the current record
// and concatenates them with the current record.
// A "turn" is a single message exchange, so contextTurns=2 means include up to 4 previous messages
// (2 turns = 2 user + 2 assistant messages, though the actual count may vary).
// If contextTurns == 0, returns only the current record's contents.
func (cp *conceptProcessor) buildContextWindow(ctx context.Context, record *core.ChatRecord) (string, error) {
	if cp.contextTurns == 0 {
		return record.Contents, nil
	}

	// Fetch recent messages to build context
	// We fetch more than we need to ensure we can find messages before the current one
	// contextTurns * 2 gives us enough headroom (2 messages per turn)
	limit := cp.contextTurns * 2
	recentRecords, err := cp.chatRepository.GetRecentChatRecords(ctx, limit+10) // +10 buffer for safety
	if err != nil {
		return "", fmt.Errorf("failed to fetch recent records: %w", err)
	}

	// Find messages that come before the current record and take the last N
	var contextRecords []*core.ChatRecord
	for _, msg := range recentRecords {
		// Only include messages that are strictly before the current record (by timestamp)
		if msg.Timestamp.Before(record.Timestamp) {
			contextRecords = append(contextRecords, msg)
		}
	}

	// Reverse the context records so they're in chronological order (oldest first)
	// GetRecentChatRecords returns newest first, we want oldest first for context
	for i, j := 0, len(contextRecords)-1; i < j; i, j = i+1, j-1 {
		contextRecords[i], contextRecords[j] = contextRecords[j], contextRecords[i]
	}

	// Take only the last N messages (most recent before current)
	maxMessages := cp.contextTurns * 2
	if len(contextRecords) > maxMessages {
		contextRecords = contextRecords[len(contextRecords)-maxMessages:]
	}

	// Concatenate context messages + current message
	var parts []string
	for _, msg := range contextRecords {
		parts = append(parts, msg.Contents)
	}
	parts = append(parts, record.Contents)

	return strings.Join(parts, "\n\n"), nil
}

// process extracts concepts from the specified chat records and assigns them.
func (cp *conceptProcessor) process(ctx context.Context, ids ...core.ID) error {
	cp.logger.Info("processing records for concepts", "records", len(ids))

	// Sort to ensure checkpointing works correctly
	slices.Sort(ids)

	records, err := cp.chatRepository.GetChatRecords(ctx, ids...)
	if err != nil {
		return err
	}

	// Step 1: Classify all records (sequential - classifier doesn't support batching)
	// Build mapping of conceptID -> positions where it should be assigned
	conceptMapping := make(map[core.ID][]recordConceptPos)
	allConcepts := make([]concept, 0)
	conceptIDToIdx := make(map[core.ID]int) // track position in allConcepts slice
	var classificationErrors []error

	for recordIdx, record := range records {
		// Build context window for this record
		contextText, err := cp.buildContextWindow(ctx, record)
		if err != nil {
			classificationErrors = append(classificationErrors, fmt.Errorf("record %d context window failed: %w", recordIdx, err))
			continue
		}

		// Extract concepts from the windowed context
		extracted, err := cp.extractor.ExtractConcepts(ctx, contextText)
		if err != nil {
			classificationErrors = append(classificationErrors, fmt.Errorf("record %d classification failed: %w", recordIdx, err))
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

	// Step 2: GetOrCreate all concepts
	var getOrCreateErr error
	var resolvedConcepts []*core.Concept
	if len(allConcepts) > 0 {
		resolvedConcepts, getOrCreateErr = cp.getOrCreateConcepts(ctx, allConcepts)
		if getOrCreateErr != nil {
			classificationErrors = append(classificationErrors, fmt.Errorf("GetOrCreate failed: %w", getOrCreateErr))
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

	// Update records
	_, updateErr := cp.chatRepository.UpdateChatRecords(ctx, records...)
	if updateErr != nil {
		classificationErrors = append(classificationErrors, fmt.Errorf("update records failed: %w", updateErr))
	} else if len(records) > 0 {
		cp.lastID = records[len(records)-1].Id
	}

	// Return combined errors if any occurred
	if len(classificationErrors) > 0 {
		return errors.Join(classificationErrors...)
	}

	return nil
}

// getOrCreateConcepts gets or creates concepts with embeddings
func (cp *conceptProcessor) getOrCreateConcepts(ctx context.Context, rawConcepts []concept) ([]*core.Concept, error) {
	// Generate embeddings for all concepts
	tuples := make([]string, len(rawConcepts))
	for i := range rawConcepts {
		tuples[i] = rawConcepts[i].Tuple()
	}

	embeddings, err := cp.embedder.EmbedTexts(ctx, tuples)
	if err != nil {
		return nil, err
	}

	// Try to get or create each concept
	result := make([]*core.Concept, 0, len(rawConcepts))
	for i, rawConcept := range rawConcepts {
		// Use the repository's GetOrCreateConcept
		concept, err := cp.conceptRepository.GetOrCreateConcept(ctx, rawConcept.Concept, rawConcept.Type, embeddings[i])
		if err != nil {
			return nil, err
		}
		result = append(result, concept)
	}

	return result, nil
}

// checkpoint saves the processor's current state.
// Currently unimplemented - reserved for future checkpointing support.
func (cp *conceptProcessor) checkpoint() error {
	// TODO: Implement checkpoint storage via repository
	return nil
}
