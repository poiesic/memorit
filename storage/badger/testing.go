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
