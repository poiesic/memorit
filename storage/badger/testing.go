package badger

import "github.com/poiesic/memorit/storage"

// NewMemoryRepositories creates in-memory chat and concept repositories for testing.
// Returns chatRepo, conceptRepo, backend, and error.
// Caller must close both repos and backend when done.
func NewMemoryRepositories() (storage.ChatRepository, storage.ConceptRepository, *Backend, error) {
	backend, err := OpenBackend("", true)
	if err != nil {
		return nil, nil, nil, err
	}

	chatRepo, err := NewChatRepository(backend)
	if err != nil {
		backend.Close()
		return nil, nil, nil, err
	}

	conceptRepo, err := NewConceptRepository(backend)
	if err != nil {
		chatRepo.Close()
		backend.Close()
		return nil, nil, nil, err
	}

	return chatRepo, conceptRepo, backend, nil
}
