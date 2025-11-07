package search

import "errors"

var (
	// ErrChatRepositoryRequired is returned when a chat repository is not provided.
	ErrChatRepositoryRequired = errors.New("chat repository required")

	// ErrConceptRepositoryRequired is returned when a concept repository is not provided.
	ErrConceptRepositoryRequired = errors.New("concept repository required")

	// ErrAIProviderRequired is returned when an AI provider is not provided.
	ErrAIProviderRequired = errors.New("AI provider required")
)
