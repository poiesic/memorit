package mock

import (
	"context"
	"strings"

	"github.com/poiesic/memorit/ai"
)

// MockConceptExtractor is a test double for ai.ConceptExtractor.
// It allows custom behavior injection via function fields.
type MockConceptExtractor struct {
	// ExtractConceptsFunc is called by ExtractConcepts if set.
	// If nil, uses default simple word extraction.
	ExtractConceptsFunc func(ctx context.Context, text string) ([]ai.ExtractedConcept, error)

	callCount int
}

// NewMockConceptExtractor creates a mock concept extractor with default behavior.
// Note: Returns concrete type to allow test assertions via GetMockExtractor().
func NewMockConceptExtractor() *MockConceptExtractor {
	return &MockConceptExtractor{}
}

// ExtractConcepts extracts simple mock concepts from text.
// Default behavior: splits text by spaces and creates concepts from words.
func (m *MockConceptExtractor) ExtractConcepts(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
	m.callCount++

	if m.ExtractConceptsFunc != nil {
		return m.ExtractConceptsFunc(ctx, text)
	}

	// Default: extract simple concepts from words
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return []ai.ExtractedConcept{}, nil
	}

	// Create mock concepts from first few words
	concepts := make([]ai.ExtractedConcept, 0, len(words))
	importance := 10
	for i, word := range words {
		if i >= 5 { // Limit to 5 concepts
			break
		}

		// Clean the word
		word = strings.Trim(word, ".,!?;:\"'()[]{}—–-")
		if word == "" {
			continue
		}

		// Assign a simple type
		conceptType := "abstract_concept"
		if len(word) > 5 {
			conceptType = "thing"
		}

		concepts = append(concepts, ai.ExtractedConcept{
			Name:       word,
			Type:       conceptType,
			Importance: importance,
		})

		// Decrease importance for each subsequent concept
		if importance > 1 {
			importance--
		}
	}

	return concepts, nil
}

// CallCount returns the number of times ExtractConcepts was called.
func (m *MockConceptExtractor) CallCount() int {
	return m.callCount
}

// Reset clears the call count and custom functions.
func (m *MockConceptExtractor) Reset() {
	m.callCount = 0
	m.ExtractConceptsFunc = nil
}
