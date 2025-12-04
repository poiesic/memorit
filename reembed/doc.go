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


// Package reembed provides functionality for reembedding existing chat records
// with new or updated embedding models.
//
// This package supports batch processing of chat records, progress tracking,
// retry logic with exponential backoff, and vector normalization to ensure
// compatibility with cosine similarity search.
package reembed
