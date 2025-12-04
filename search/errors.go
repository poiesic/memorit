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
