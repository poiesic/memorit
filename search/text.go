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
