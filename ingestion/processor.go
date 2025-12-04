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


package ingestion

import (
	"context"

	"github.com/poiesic/memorit/core"
)

// processor is an internal interface for processing chat records.
// Implementations handle specific enrichment tasks like embeddings or concept extraction.
type processor interface {
	// process enriches the chat records identified by the given IDs.
	process(ctx context.Context, ids ...core.ID) error

	// checkpoint saves the processor's current state.
	// Currently unimplemented but reserved for future checkpointing support.
	checkpoint() error
}
