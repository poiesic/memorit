# Reembed Package

The `reembed` package provides functionality for regenerating embeddings for all chat records in a memorit database.

## Purpose

This package is useful when you need to:

- Switch to a different embedding model
- Upgrade an embedding model version
- Fix corrupted or missing embeddings
- Change embedding dimensions

## Features

- **Batch processing**: Configurable batch sizes for efficient API usage
- **Progress tracking**: Real-time progress reporting with rate calculation
- **Retry logic**: Automatic retry with exponential backoff for transient failures
- **Vector normalization**: Ensures all vectors are normalized to unit length for cosine similarity
- **Context-aware**: Supports cancellation and timeouts via context
- **Idempotent**: Can be run multiple times safely - always overwrites existing embeddings

## Usage

### As a CLI Command

```bash
# Reembed all records (embedding-model is required)
memorit reembed -d /path/to/database --embedding-model embeddinggemma

# Use a different model
memorit reembed -d /path/to/database --embedding-model text-embedding-3-small

# Custom batch size and progress reporting
memorit reembed -d /path/to/database --embedding-model embeddinggemma --batch-size 50 --report-interval 50

# With remote embedding server
memorit reembed -d /path/to/database --embedding-host http://remote-server:11434/v1 --embedding-model custom-model
```

### As a Library

```go
import (
    "context"
    "os"

    "github.com/poiesic/memorit/ai"
    "github.com/poiesic/memorit/ai/openai"
    "github.com/poiesic/memorit/reembed"
    "github.com/poiesic/memorit/storage/badger"
)

func reembedDatabase(dbPath string) error {
    ctx := context.Background()

    // Open database
    backend, err := badger.OpenBackend(dbPath, false)
    if err != nil {
        return err
    }
    defer backend.Close()

    repo, err := badger.NewChatRepository(backend)
    if err != nil {
        return err
    }
    defer repo.Close()

    // Create embedder
    aiConfig := ai.NewConfig(
        ai.WithEmbeddingHost("http://localhost:11434/v1"),
        ai.WithEmbeddingModel("embeddinggemma"),
    )
    embedder, err := openai.NewEmbedder(aiConfig)
    if err != nil {
        return err
    }

    // Configure and run reembedding
    config := &reembed.Config{
        BatchSize:      100,
        ReportInterval: 100,
        MaxRetries:     3,
        RetryDelay:     1 * time.Second,
    }

    reembedder := reembed.NewReembedder(repo, embedder, config, os.Stderr)
    return reembedder.Run(ctx)
}
```

## Architecture

### Components

1. **RecordIterator**: Iterates over all chat records in batches
2. **BatchProcessor**: Processes batches of records through the embedding API
3. **ProgressTracker**: Reports progress at configurable intervals
4. **Reembedder**: Orchestrates the full reembedding workflow
5. **Vector normalization**: Ensures all vectors have unit magnitude

### Workflow

1. Count total records in database
2. Initialize progress tracking
3. Iterate through records in batches
4. For each batch:
   - Extract text content
   - Call embedding API with retry logic
   - Normalize resulting vectors
   - Update records in database
   - Report progress
5. Print final statistics

## Configuration

### Config Options

- `BatchSize`: Number of records per API call (default: 100)
- `ReportInterval`: Report progress every N records (default: 100)
- `MaxRetries`: Maximum retry attempts for failures (default: 3)
- `RetryDelay`: Base delay for exponential backoff (default: 1s)

### Retry Behavior

The package uses exponential backoff for retries:
- Attempt 1: immediate
- Attempt 2: after `RetryDelay`
- Attempt 3: after `RetryDelay * 2`
- Attempt N: after `RetryDelay * 2^(N-2)`

## Vector Normalization

All embedding vectors are normalized to unit length (magnitude = 1.0) after retrieval from the embedding API. This ensures compatibility with cosine similarity search, which uses dot product on normalized vectors.

The normalization formula: `v_norm = v / ||v||` where `||v||` is the Euclidean magnitude.

## Error Handling

The package handles several types of errors:

- **Network errors**: Retried with exponential backoff
- **API rate limits**: Retried with exponential backoff
- **Context cancellation**: Immediately stops processing
- **Invalid configurations**: Returns error before processing
- **Database errors**: Propagated to caller

## Performance

Processing speed depends on:
- Embedding API latency (typically the bottleneck)
- Batch size (larger = fewer API calls)
- Network latency
- Database I/O speed

Typical performance with local embedding service: 50-200 records/second

## Testing

The package includes comprehensive tests:

- Unit tests for each component
- Integration tests for full workflows
- Mock embedders for testing without API dependency

Run tests:
```bash
go test -v ./reembed
```

## Safety Considerations

- **Database access**: Assumes exclusive access during operation
- **No rollback**: Overwrites embeddings immediately (operation is idempotent)
- **No dimension validation**: Always replaces existing vectors regardless of dimensions
- **Context cancellation**: Stops at batch boundaries, may leave some records updated
