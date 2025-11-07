// Package ingestion provides pipeline orchestration for processing chat records.
//
// The Pipeline type manages the ingestion workflow for chat records, including:
//   - Adding records to storage
//   - Generating embeddings asynchronously
//   - Extracting and assigning concepts asynchronously
//
// Processing is performed concurrently using worker pools to maximize throughput.
// Errors during async processing are logged but do not fail the ingestion operation.
package ingestion
