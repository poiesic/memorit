package search

import "strings"

// Stop words to filter out when checking for verbatim matches
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "be": true, "is": true, "are": true,
	"was": true, "to": true, "of": true, "and": true, "in": true, "that": true,
	"have": true, "it": true, "for": true, "not": true, "on": true, "with": true,
	"as": true, "you": true, "do": true, "at": true, "this": true, "but": true,
	"by": true, "from": true,
}

// tokenizeAndFilter splits text into words, lowercases, trims punctuation, and removes stop words
func tokenizeAndFilter(text string) []string {
	words := strings.Fields(text)
	filtered := make([]string, 0, len(words))

	for _, word := range words {
		// Lowercase and trim punctuation
		cleaned := strings.ToLower(strings.Trim(word, ".,!?;:'\"-()[]{}"))

		// Skip stop words and empty strings
		if cleaned != "" && !stopWords[cleaned] {
			filtered = append(filtered, cleaned)
		}
	}

	return filtered
}

// containsAllQueryWords checks if all query words (after filtering) appear in the document
func containsAllQueryWords(document, query string) bool {
	queryWords := tokenizeAndFilter(query)
	if len(queryWords) == 0 {
		return false
	}

	docWords := tokenizeAndFilter(document)
	docWordSet := make(map[string]bool, len(docWords))
	for _, word := range docWords {
		docWordSet[word] = true
	}

	// Check if all query words exist in document
	for _, qWord := range queryWords {
		if !docWordSet[qWord] {
			return false
		}
	}

	return true
}
