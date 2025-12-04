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

import "errors"

// Domain validation errors
var (
	// ErrInvalidChatRecord indicates a ChatRecord failed validation.
	ErrInvalidChatRecord = errors.New("invalid chat record")

	// ErrInvalidConcept indicates a Concept failed validation.
	ErrInvalidConcept = errors.New("invalid concept")

	// ErrInvalidTimestamp indicates a timestamp is in the future.
	ErrInvalidTimestamp = errors.New("timestamp cannot be in the future")

	// ErrEmptyContent indicates the Contents field is empty.
	ErrEmptyContent = errors.New("content cannot be empty")

	// ErrInvalidSpeakerType indicates an invalid SpeakerType value.
	ErrInvalidSpeakerType = errors.New("invalid speaker type")

	// ErrEmptyConceptName indicates the concept Name field is empty.
	ErrEmptyConceptName = errors.New("concept name cannot be empty")

	// ErrEmptyConceptType indicates the concept Type field is empty.
	ErrEmptyConceptType = errors.New("concept type cannot be empty")
)
