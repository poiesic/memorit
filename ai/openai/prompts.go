package openai

import (
	"fmt"
	"strings"

	"github.com/poiesic/memorit/ai"
)

const classificationResponseSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "core_concepts": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "concept": {
            "type": "string",
            "pattern": "^[a-z]+( [a-z]+)*$"
          },
          "type": {
            "type": "string"
          },
          "importance": {
            "type": "integer",
            "minimum": 1,
            "maximum": 10
          }
        },
        "required": ["concept", "type", "importance"],
        "additionalProperties": false
      }
    }
  },
  "required": ["core_concepts"],
  "additionalProperties": false
}`

const classificationPromptTemplate = `Extract the most important concepts from the given text and return them as JSON.

Output ONLY valid JSON which complies with the schema given below. Do not include any preamble, explanation,
greeting, or acknowledgment. Start your response directly with the opening brace { and end with the closing
brace }. Your output must exactly follow this schema:

%s

Rules:
- Concept names must be lowercase, 1-3 words, singular form only.
- Type field must match exactly one of the listed values: %s.
- Importance is an integer from 1 (least relevant) to 10 (most central). Rate based on how essential the concept is for understanding the text.
- Include only concepts that are explicitly mentioned or clearly implied by the text. Do not hallucinate.
- Weight the subject of a sentence higher.
- If no concepts can be identified, return "core_concepts": [].
- The JSON must parse without errors; no trailing commas, no extra keys, and no extraneous text outside the object.



Example (formal):
Input: "The Eiffel Tower is a famous landmark in Paris."
Output:
{
  "core_concepts": [
    {"concept":"eiffel tower","type":"building","importance":9},
    {"concept":"paris","type":"place","importance":8}
  ]
}

---  // informal / chat-style examples

Example (missing capitalization, no punctuation):
Input: "the eiffel tower is in paris"
Output:
{
  "core_concepts": [
    {"concept":"eiffel tower","type":"building","importance":9},
    {"concept":"paris","type":"place","importance":8}
  ]
}

Example (shortened pronouns, no punctuation):
Input: "hey can u tell me about big cats"
Output:
{
  "core_concepts": [
    {"concept":"big cat","type":"animal","importance":9}
  ]
}

Example (multiple animals, informal):
Input: "i love my dog and my cat"
Output:
{
  "core_concepts": [
    {"concept":"dog","type":"animal","importance":8},
    {"concept":"cat","type":"animal","importance":7}
  ]
}

Example (informal weather mention - treat as abstract event):
Input: "weather is nice today"
Output:
{
  "core_concepts": [
    {"concept":"weather","type":"abstract concept","importance":6}
  ]
}`

// buildSystemPrompt creates the system prompt with concept types embedded.
func buildSystemPrompt() string {
	return fmt.Sprintf(classificationPromptTemplate,
		classificationResponseSchema,
		strings.Join(ai.ConceptTypes, ", "))
}
