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


package openai

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"

	"github.com/poiesic/memorit/ai"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// ConceptExtractor implements ai.ConceptExtractor using OpenAI-compatible chat APIs.
type ConceptExtractor struct {
	client        llms.Model
	minImportance int
	logger        *slog.Logger
}

// concept is an internal type used for JSON unmarshaling.
// It matches the structure expected by the LLM.
type concept struct {
	Concept    string `json:"concept"`
	Type       string `json:"type"`
	Importance int    `json:"importance"`
}

// analysis is the wrapper structure for the LLM's JSON response.
type analysis struct {
	CoreConcepts []concept `json:"core_concepts"`
}

// newConceptExtractor is an internal constructor that returns the concrete type.
// Used by Provider to manage the instance.
func newConceptExtractor(config *ai.Config) (*ConceptExtractor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create OpenAI client configured for chat/classification
	// Use "none" as token for local OpenAI-compatible services that don't require authentication
	client, err := openai.New(
		openai.WithBaseURL(config.ClassifierHost),
		openai.WithToken("none"),
		openai.WithModel(config.ClassifierModel),
	)
	if err != nil {
		return nil, err
	}

	return &ConceptExtractor{
		client:        client,
		minImportance: config.MinImportance,
		logger:        slog.Default().With("component", "openai-extractor"),
	}, nil
}

// NewConceptExtractor creates a new concept extractor using the provided configuration.
//
// Returns ai.ConceptExtractor interface to enforce abstraction.
func NewConceptExtractor(config *ai.Config) (ai.ConceptExtractor, error) {
	return newConceptExtractor(config)
}

// ExtractConcepts extracts semantic concepts from text using an LLM.
// It applies importance filtering and returns only concepts above the minimum threshold.
func (e *ConceptExtractor) ExtractConcepts(ctx context.Context, text string) ([]ai.ExtractedConcept, error) {
	// Scrub input text
	text = scrubString(text)

	// Build the system and user prompts
	systemPrompt := buildSystemPrompt()
	content := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(systemPrompt),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(text),
			},
		},
	}

	// Try up to 3 times in case of malformed JSON
	var result analysis
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		response, err := e.client.GenerateContent(ctx, content, llms.WithTemperature(0.0), llms.WithJSONMode())
		if err != nil {
			e.logger.Error("failed to generate content", "attempt", attempt+1, "err", err)
			return nil, err
		}

		if len(response.Choices) < 1 {
			e.logger.Debug("no choices returned from model")
			return []ai.ExtractedConcept{}, nil
		}

		choice := response.Choices[0]

		// Strip markdown code fences if present
		responseText := strings.TrimSpace(choice.Content)
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
		responseText = strings.TrimSpace(responseText)

		// Try to repair common JSON issues
		responseText = repairJSON(responseText)

		if err := json.Unmarshal([]byte(responseText), &result); err != nil {
			lastErr = err
			e.logger.Warn("error parsing classifier response",
				"attempt", attempt+1,
				"response", responseText,
				"err", err)
			continue
		}

		// Success
		lastErr = nil
		break
	}

	if lastErr != nil {
		e.logger.Error("failed to parse classifier response after retries", "err", lastErr)
		return nil, lastErr
	}

	// Filter by importance and convert to ai.ExtractedConcept
	extracted := make([]ai.ExtractedConcept, 0, len(result.CoreConcepts))
	for _, c := range result.CoreConcepts {
		if c.Importance >= e.minImportance {
			extracted = append(extracted, ai.ExtractedConcept{
				Name:       c.Concept,
				Type:       c.Type,
				Importance: c.Importance,
			})
		}
	}

	// Sort by importance (descending)
	slices.SortFunc(extracted, func(a, b ai.ExtractedConcept) int {
		if a.Importance == b.Importance {
			return 0
		}
		if a.Importance < b.Importance {
			return 1
		}
		return -1
	})

	e.logger.Debug("extracted concepts",
		"total", len(result.CoreConcepts),
		"filtered", len(extracted))

	for i, c := range extracted {
		extracted[i].Type = strings.ReplaceAll(c.Type, " ", "_")
	}
	return extracted, nil
}
