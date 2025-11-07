package search

import (
	"context"
	"log/slog"
	"maps"
	"sort"

	"github.com/poiesic/memorit/ai"
	"github.com/poiesic/memorit/core"
	"github.com/poiesic/memorit/storage"
)

// Searcher provides hybrid semantic and conceptual search over chat records.
type Searcher struct {
	chatRepository    storage.ChatRepository
	conceptRepository storage.ConceptRepository
	embedder          ai.Embedder
	extractor         ai.ConceptExtractor
	logger            *slog.Logger
}

// Option configures a Searcher.
type Option func(*Searcher) error

// WithLogger sets a custom logger.
// Default is slog.Default().
func WithLogger(logger *slog.Logger) Option {
	return func(s *Searcher) error {
		if logger == nil {
			logger = slog.Default()
		}
		s.logger = logger
		return nil
	}
}

// NewSearcher creates a new searcher.
func NewSearcher(
	chatRepository storage.ChatRepository,
	conceptRepository storage.ConceptRepository,
	provider ai.AIProvider,
	opts ...Option,
) (*Searcher, error) {
	if chatRepository == nil {
		return nil, ErrChatRepositoryRequired
	}
	if conceptRepository == nil {
		return nil, ErrConceptRepositoryRequired
	}
	if provider == nil {
		return nil, ErrAIProviderRequired
	}

	s := &Searcher{
		chatRepository:    chatRepository,
		conceptRepository: conceptRepository,
		embedder:          provider.Embedder(),
		extractor:         provider.ConceptExtractor(),
		logger:            slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// FindSimilar searches for chat records similar to the query.
// Returns up to maxHits results, ranked by relevance score.
func (s *Searcher) FindSimilar(ctx context.Context, query string, maxHits int) ([]*core.SearchResult, error) {
	return s.FindSimilarWithMonitor(ctx, query, maxHits, nil)
}

// FindSimilarWithMonitor searches for chat records similar to the query with monitoring.
// The monitor receives callbacks at each stage of the search process.
// Returns up to maxHits results, ranked by relevance score.
func (s *Searcher) FindSimilarWithMonitor(ctx context.Context, query string, maxHits int, monitor SearchMonitor) ([]*core.SearchResult, error) {
	// Use noop monitor if none provided
	if monitor == nil {
		monitor = &noopMonitor{}
	}

	monitor.Start(query)

	// 1. Perform semantic search
	embedding, err := s.embedder.EmbedText(ctx, query)
	if err != nil {
		s.logger.Error("error generating embedding for query", "query", query, "err", err)
		return nil, err
	}

	// Find similar embeddings - use 0.60 threshold as in original
	matches, err := s.chatRepository.FindSimilar(ctx, embedding, 0.60, maxHits)
	if err != nil {
		s.logger.Error("error querying for similar records", "err", err)
		return nil, err
	}

	// Track semantic results
	semanticSet := make(map[uint64]bool)
	semanticScores := make(map[uint64]float32)
	semanticIds := make([]uint64, 0, len(matches))
	for _, match := range matches {
		semanticSet[uint64(match.Record.Id)] = true
		semanticScores[uint64(match.Record.Id)] = match.Score
		semanticIds = append(semanticIds, uint64(match.Record.Id))
	}
	monitor.AfterSemanticSearch(semanticIds)

	// 2. Extract concepts from query
	extracted, err := s.extractor.ExtractConcepts(ctx, query)
	if err != nil {
		s.logger.Error("error extracting concepts from query", "err", err)
		return nil, err
	}

	// Convert to full concepts by computing IDs and looking them up
	concepts := make([]*core.Concept, 0, len(extracted))
	for _, ec := range extracted {
		tuple := "(" + ec.Type + "," + ec.Name + ")"
		conceptID := core.IDFromContent(tuple)
		concept, err := s.conceptRepository.GetConcept(ctx, conceptID)
		if err != nil {
			s.logger.Warn("error looking up concept", "tuple", tuple, "err", err)
			continue
		}
		if concept == nil {
			// Concept doesn't exist in database, skip
			s.logger.Debug("concept not found in database", "tuple", tuple)
			continue
		}
		concepts = append(concepts, concept)
	}
	monitor.AfterQueryConceptExtraction(concepts)

	// 3. Find messages via exact concept matching
	conceptualSet := make(map[uint64]bool)
	for _, concept := range concepts {
		tuple := concept.Tuple()
		monitor.FoundRelatedConcepts(tuple, []uint64{uint64(concept.Id)})

		// Get messages for this concept
		recordIds, err := s.chatRepository.GetChatRecordsByConcept(ctx, concept.Id)
		if err != nil {
			s.logger.Warn("failed to get records for concept", "conceptID", concept.Id, "err", err)
			continue
		}
		for _, recordId := range recordIds {
			conceptualSet[uint64(recordId)] = true
		}
	}
	monitor.AfterConceptuallyRelatedSearch(maps.Keys(conceptualSet))

	// 4. Combine and score results
	allIds := make(map[uint64]bool)
	for id := range semanticSet {
		allIds[id] = true
	}
	for id := range conceptualSet {
		allIds[id] = true
	}

	if len(allIds) == 0 {
		return []*core.SearchResult{}, nil
	}

	// Retrieve all records
	uniqueIds := make([]core.ID, 0, len(allIds))
	for id := range allIds {
		uniqueIds = append(uniqueIds, core.ID(id))
	}

	records, err := s.chatRepository.GetChatRecords(ctx, uniqueIds...)
	if err != nil {
		s.logger.Error("error retrieving chat records", "recordCount", len(uniqueIds), "err", err)
		return nil, err
	}
	monitor.AfterRecordRetrieval(records)

	// Score and build results
	results := make([]*core.SearchResult, 0, len(records))

	for _, record := range records {
		if record == nil {
			continue
		}

		inSemantic := semanticSet[uint64(record.Id)]
		inConceptual := conceptualSet[uint64(record.Id)]

		var score float32
		if inSemantic && inConceptual {
			// In both: boost by 1.5x, weighted by similarity score
			similarityScore := semanticScores[uint64(record.Id)]
			score = 1.5 * similarityScore
			monitor.SemanticAndConceptualHit(record)
		} else if inConceptual {
			// Conceptual only: 1.2
			score = 1.2
			monitor.ConceptualHit(record)
		} else {
			// Semantic only: 1.0, weighted by similarity score
			similarityScore := semanticScores[uint64(record.Id)]
			score = 1.0 * similarityScore
			monitor.SemanticHit(record)
		}

		// Apply verbatim match boost
		if containsAllQueryWords(record.Contents, query) {
			score += 0.3
		}

		results = append(results, &core.SearchResult{
			Record: record,
			Score:  score,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > maxHits {
		results = results[:maxHits]
	}
	monitor.Finish(results)

	return results, nil
}
