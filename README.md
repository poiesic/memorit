# Memorit

A semantic memory system for storing and retrieving conversational data using embeddings and concept extraction.

## Features

- **Hybrid Search**: Combines vector similarity, conceptual matching, and keyword search
- **Concept Extraction**: Automatically extracts semantic concepts from conversations
- **Clean Architecture**: Well-defined separation between domain, storage, and AI layers
- **Concurrent Processing**: Async embedding and concept extraction with worker pools
- **Pluggable Backends**: Abstract interfaces for storage and AI services
- **Fast Serialization**: Uses mus-go for efficient binary serialization

## Dependencies

- [BadgerDB v4](https://github.com/dgraph-io/badger) - Embedded key-value store
- [langchaingo](https://github.com/tmc/langchaingo) - LLM and embedding integrations
- [mus-go](https://github.com/mus-format/mus-go) - Fast binary serialization
- [ants](https://github.com/panjf2000/ants) - Worker pool implementation

### Core Domain (`core/`)

Pure domain models with zero external dependencies:
- Domain entities: `ChatRecord`, `Concept`, `ConceptRef`, `ID`, `SpeakerType`
- Business validation rules
- Domain-specific errors

### Storage Layer (`storage/`)

Repository pattern with BadgerDB implementation:
- `ChatRepository`: Chat record operations
- `ConceptRepository`: Concept operations
- `VectorSearcher`: Vector similarity search
- Thread-safe with context support

### AI Layer (`ai/`)

Provider abstraction for AI services:
- `Embedder`: Generate vector embeddings from text
- `ConceptExtractor`: Extract semantic concepts
- `AIProvider`: Unified interface for AI operations
- Includes OpenAI and mock implementations

### Ingestion Pipeline (`ingestion/`)

Concurrent processing pipeline:
- Stores chat records
- Generates embeddings asynchronously
- Extracts and assigns concepts in parallel
- Worker pools for maximum throughput

### Search Engine (`search/`)

Multi-stage hybrid search:
- Semantic search via vector embeddings
- Conceptual search using extracted concepts
- Keyword matching with stop-word filtering
- Ranked results with relevance scoring

## Quick Start

### Prerequisites

- Go 1.25.3+
- [Task](https://taskfile.dev/) for build automation

### Building

```bash
# Build all binaries
task build

# Run tests
task test

# Generate serialization code
task generate

# Run static analysis
task vet
```

### Usage

**Seed the database:**
```bash
# Use built-in test data
./bin/seeder

# Load from file
./bin/seeder -src data.txt
```

**Search conversations:**
```bash
./bin/searcher
```

## Development

### Running Tests

```bash
# All tests
task test

# Specific package with coverage
go test -v -cover ./core/
go test -v -cover ./storage/badger/
```

### Code Generation

Memorit uses mus-go for binary serialization. Delete `core/records_musg.gen.go` then use the `generate` task to regenerate serializable models. 


## License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE) for the full license text and [NOTICE](NOTICE) for third-party attributions.
