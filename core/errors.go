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
