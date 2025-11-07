# Memorit

A semantic memory system for storing and retrieving conversational data using embeddings and concept extraction.

## Architecture

Memorit follows Clean Architecture principles with clear separation of concerns:

### Core Domain (`core/`)

The heart of the application containing pure domain models and business rules with zero external dependencies (except standard library).

- **Models**: `ChatRecord`, `Concept`, `ConceptRef`, `ID`, `SpeakerType`
- **Validation**: Domain-specific validation rules
- **Errors**: Domain error types

The core domain is dependency-free and can be imported by all other layers.

### Types (`types/`)

Temporary serialization layer that re-exports core types for backward compatibility. Contains marshaling/unmarshaling code for database persistence using mus-go.

**Note**: This package will be moved to the storage layer in Phase 2 of the refactoring.

### Application Layer (root package)

Current monolithic implementation containing:

- Storage layer (BadgerDB)
- AI integrations (OpenAI, embeddings)
- Processing pipeline (embedding and concept processors)
- Search functionality
- Ingestion system

**Note**: This layer will be refactored into separate packages (`storage/`, `ai/`, `pipeline/`, `search/`, `app/`) in future phases.

## Building

This project uses [Task](https://taskfile.dev/) for build automation:

```bash
# Run tests
task test

# Build all binaries
task build

# Run static analysis
task vet

# Clean build artifacts
task clean

# Tidy dependencies
task tidy
```

## Commands

### Seeder

Seeds the database with conversation data.

```bash
./bin/seeder
```

### Searcher

Performs semantic search over stored conversations.

```bash
./bin/searcher
```

### Musgen

Generates serialization code using mus-go.

```bash
./bin/musgen
```

## Development

### Project Structure

```
memorit/
├── core/               # Pure domain models (Phase 1 ✓)
│   ├── models.go      # Domain entities
│   ├── errors.go      # Domain errors
│   ├── validation.go  # Business rules
│   └── doc.go         # Package documentation
├── types/             # Serialization layer (temporary)
│   ├── types.go       # Re-exports core types
│   ├── marshal.go     # Marshaling functions
│   └── *.gen.go       # Generated code
├── cmd/               # Command-line applications
│   ├── seeder/
│   ├── searcher/
│   └── musgen/
├── *.go               # Current monolithic implementation
└── *_test.go          # Tests
```

### Refactoring Status

- **Phase 1: Core Domain Extraction** ✓ Complete
- **Phase 2: Storage Abstraction** - Planned
- **Phase 3: AI Services Abstraction** - Planned
- **Phase 4: Pipeline Refactoring** - Planned
- **Phase 5: Search Service** - Planned
- **Phase 6: Application Layer** - Planned
- **Phase 7: Dependency Injection** - Planned
- **Phase 8: Testing & Documentation** - Planned

See `refactor.md` for the complete refactoring plan.

## Testing

```bash
# Run all tests
task test

# Run tests for specific package
go test -v ./core/

# Run with coverage
go test -cover ./...
```

## Domain Validation

The core domain implements validation rules for business invariants:

### ChatRecord Validation

- Contents must not be empty
- SpeakerType must be valid (Human or AI)
- Timestamp must not be in the future

**Not validated** (populated by processors):
- Vector embeddings
- Concept references
- ID values (0 is valid)

### Concept Validation

- Name must not be empty
- Type must not be empty

**Not validated**:
- Vector embeddings (populated by embedding processor)
- ID values (0 is valid)

## Dependencies

- Go 1.25.3+
- BadgerDB v4 - Embedded key-value database
- langchaingo - LLM and embedding integrations
- mus-go - Fast binary serialization
- Task - Build automation

## License

[License information]
