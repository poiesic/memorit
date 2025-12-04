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


// Package storage provides the storage abstraction layer for memorit.
//
// This package defines repository interfaces that decouple storage implementation
// from business logic. It allows for different storage backends (BadgerDB, in-memory,
// etc.) to be used interchangeably.
//
// # Constructor Return Type Pattern
//
// This package follows a strict "return interface" pattern for all public constructors
// to enforce abstraction and enable multiple storage backend implementations:
//
//	repo, err := badger.NewRepository(path)  // returns storage.Repository interface
//
// This design decision prioritizes:
//   - Abstraction: Prevents accidental coupling to BadgerDB specifics
//   - Swappability: Easy to add alternative backends (PostgreSQL, in-memory, etc.)
//   - Testing: Consumers can use mock implementations without modification
//
// Internal package constructors (newChatRepository, newBackend, etc.) may return
// concrete types since they're only used within the implementation package.
//
// # Architecture
//
// The storage layer follows the Repository pattern:
//
//   - Repository: Main interface combining all storage operations
//   - ChatRepository: Operations for chat records
//   - ConceptRepository: Operations for concepts
//   - VectorSearcher: Vector similarity search operations
//   - TransactionManager: Transaction support
//
// # Usage
//
// Create a repository instance:
//
//	repo, err := badger.NewRepository("/path/to/db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer repo.Close()
//
// Use in tests with in-memory storage:
//
//	repo, err := badger.NewMemoryRepository()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer repo.Close()
//
// # Thread Safety
//
// All repository implementations must be thread-safe and support
// concurrent access from multiple goroutines.
//
// # Context Support
//
// All repository methods accept context.Context for cancellation
// and timeout support. Pass context.Background() for operations
// without specific timeout requirements.
package storage
