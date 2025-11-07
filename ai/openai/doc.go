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
