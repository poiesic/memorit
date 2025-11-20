// Package reembed provides functionality for reembedding existing chat records
// with new or updated embedding models.
//
// This package supports batch processing of chat records, progress tracking,
// retry logic with exponential backoff, and vector normalization to ensure
// compatibility with cosine similarity search.
package reembed
