// Package search provides hybrid semantic and conceptual search capabilities.
//
// The Searcher type implements a multi-stage search algorithm that combines:
//   - Semantic search using vector embeddings
//   - Conceptual search using extracted concepts
//   - Verbatim keyword matching with stop-word filtering
//
// Search results are scored and ranked based on multiple signals to provide
// the most relevant results for a given query.
package search
