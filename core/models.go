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


package core

//go:generate go run ../cmd/musgen

import (
	"encoding/binary"
	"time"

	"github.com/go-crypt/x/blake2b"
)

// ID is a unique identifier for domain entities.
// It is generated using content-based hashing or database sequences.
type ID uint64

// IDFromContent generates a deterministic ID from text content using BLAKE2b hashing.
// This ensures that identical content produces identical IDs.
func IDFromContent(text string) ID {
	h, _ := blake2b.New(8, nil) // 8 bytes = 64 bits
	h.Write([]byte(text))
	sum := h.Sum(nil)
	return ID(binary.LittleEndian.Uint64(sum))
}

// SpeakerType identifies the source of a chat message.
type SpeakerType int

const (
	// SpeakerTypeHuman represents a human user.
	SpeakerTypeHuman SpeakerType = iota + 1
	// SpeakerTypeAI represents an AI assistant.
	SpeakerTypeAI
)

// ChatRecord represents a single message in a conversation.
// It may be enriched with embeddings and concepts during processing.
type ChatRecord struct {
	Id         ID
	Speaker    SpeakerType
	Contents   string
	Timestamp  time.Time      // When the message was originally sent
	InsertedAt time.Time      // When the record was inserted into the database
	UpdatedAt  time.Time      // When the record was last updated
	Concepts   []ConceptRef   // Concepts extracted from the message (populated by processors)
	Vector     []float32      // Embedding vector for semantic search (populated by processors)
	Metadata   map[string]string // Optional metadata (e.g., "role", "provider", "model")
}

// Concept represents a domain concept extracted from chat messages.
type Concept struct {
	Id         ID
	Name       string
	Type       string
	Vector     []float32 // Embedding vector for the concept (populated by processors)
	InsertedAt time.Time
	UpdatedAt  time.Time
}

// Tuple returns a string representation of the concept as "(Type,Name)".
// This is used for generating deterministic IDs.
func (c *Concept) Tuple() string {
	return "(" + c.Type + "," + c.Name + ")"
}

// ConceptRef represents a reference to a concept with an importance score.
type ConceptRef struct {
	ConceptId  ID
	Importance int // Importance score from 1-10
}

// SimilarityMatch represents a chat record match from vector similarity search.
type SimilarityMatch struct {
	RecordId ID
	Score    float32
}

// SearchResult represents a search result with the full record and relevance score.
type SearchResult struct {
	Record *ChatRecord
	Score  float32
}

// Checkpoint represents the processing state for a processor type.
// Used to track progress and enable recovery after restarts.
type Checkpoint struct {
	ProcessorType string    // Processor identifier (e.g., "embedding", "concept")
	LastID        ID        // Highest successfully processed record ID
	UpdatedAt     time.Time // When the checkpoint was last saved
}
