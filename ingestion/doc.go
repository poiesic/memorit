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


// Package ingestion provides pipeline orchestration for processing chat records.
//
// The Pipeline type manages the ingestion workflow for chat records, including:
//   - Adding records to storage
//   - Generating embeddings asynchronously
//   - Extracting and assigning concepts asynchronously
//
// Processing is performed concurrently using worker pools to maximize throughput.
// Errors during async processing are logged but do not fail the ingestion operation.
package ingestion
