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


// Package search provides hybrid semantic and conceptual search capabilities.
//
// The Searcher type implements a multi-stage search algorithm that combines:
//   - Semantic search using vector embeddings
//   - Conceptual search using extracted concepts
//   - Verbatim keyword matching with stop-word filtering
//
// Search results are scored and ranked based on multiple signals to provide
// the most relevant results for a given query.
package search
