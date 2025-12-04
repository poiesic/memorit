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

import (
	"fmt"
	"time"
)

// ValidateChatRecord validates a ChatRecord according to domain rules.
//
// Validation rules:
//   - Contents must not be empty
//   - SpeakerType must be valid (Human or AI)
//   - Timestamp must not be in the future
//
// NOT validated (populated by processors):
//   - Vector (can be empty until embedding processor runs)
//   - Concepts (can be empty until concept processor runs)
//   - ID (0 is valid from database sequences)
func ValidateChatRecord(record *ChatRecord) error {
	if record == nil {
		return fmt.Errorf("%w: record is nil", ErrInvalidChatRecord)
	}

	if record.Contents == "" {
		return fmt.Errorf("%w: %w", ErrInvalidChatRecord, ErrEmptyContent)
	}

	if err := ValidateSpeakerType(record.Speaker); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidChatRecord, err)
	}

	if !IsValidTimestamp(record.Timestamp) {
		return fmt.Errorf("%w: %w", ErrInvalidChatRecord, ErrInvalidTimestamp)
	}

	return nil
}

// ValidateConcept validates a Concept according to domain rules.
//
// Validation rules:
//   - Name must not be empty
//   - Type must not be empty
//
// NOT validated (populated by processors):
//   - Vector (can be empty until embedded)
//   - ID (0 is valid from database sequences)
func ValidateConcept(concept *Concept) error {
	if concept == nil {
		return fmt.Errorf("%w: concept is nil", ErrInvalidConcept)
	}

	if concept.Name == "" {
		return fmt.Errorf("%w: %w", ErrInvalidConcept, ErrEmptyConceptName)
	}

	if concept.Type == "" {
		return fmt.Errorf("%w: %w", ErrInvalidConcept, ErrEmptyConceptType)
	}

	return nil
}

// ValidateSpeakerType validates that a SpeakerType has a valid value.
func ValidateSpeakerType(speaker SpeakerType) error {
	if speaker != SpeakerTypeHuman && speaker != SpeakerTypeAI {
		return fmt.Errorf("%w: value %d", ErrInvalidSpeakerType, speaker)
	}
	return nil
}

// IsValidTimestamp checks if a timestamp is valid (not in the future).
func IsValidTimestamp(ts time.Time) bool {
	return !ts.After(time.Now())
}
