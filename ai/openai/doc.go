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


// Package openai provides AI service implementations using OpenAI-compatible APIs.
//
// This package implements the ai.AIProvider interface using the langchaingo
// library to communicate with OpenAI or OpenAI-compatible services (such as
// Ollama, LocalAI, or vLLM).
//
// # Usage
//
//	config := ai.DefaultConfig()
//	// Or customize:
//	config := &ai.Config{
//	    Host:            "http://localhost:11434",  // /v1 added automatically
//	    EmbeddingModel:  "embeddinggemma",
//	    ClassifierModel: "qwen2.5:3b",
//	    MinImportance:   6,
//	}
//
//	provider, err := openai.NewProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Use the services
//	embeddings, err := provider.Embedder().EmbedText(ctx, "sample text")
//	concepts, err := provider.ConceptExtractor().ExtractConcepts(ctx, "The Eiffel Tower is in Paris")
package openai
