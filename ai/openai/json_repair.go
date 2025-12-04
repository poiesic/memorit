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

// repairJSON attempts to fix common JSON formatting issues from LLM responses.
// It specifically handles missing opening quotes before keys in JSON objects.
func repairJSON(s string) string {
	// Fix missing opening quote before keys
	// Pattern: after { or , followed by optional whitespace, then a word followed by ":
	// Example: `, type":` -> `, "type":`
	result := []rune(s)
	fixed := make([]rune, 0, len(result)+100)

	i := 0
	for i < len(result) {
		ch := result[i]

		// After { or , look for unquoted keys
		if ch == '{' || ch == ',' {
			fixed = append(fixed, ch)
			i++

			// Skip whitespace
			for i < len(result) && (result[i] == ' ' || result[i] == '\n' || result[i] == '\t') {
				fixed = append(fixed, result[i])
				i++
			}

			// Check if we have an unquoted key (starts with letter, not with quote)
			if i < len(result) && result[i] != '"' && isLetter(result[i]) {
				keyStart := i
				// Find the end of the key name
				for i < len(result) && (isLetter(result[i]) || result[i] == '_' || result[i] == ' ') {
					i++
				}
				keyEnd := i

				// Check if this is followed by ": which indicates a missing opening quote
				if i+1 < len(result) && result[i] == '"' && result[i+1] == ':' {
					// Add opening quote, key, keep closing quote
					fixed = append(fixed, '"')
					for j := keyStart; j < keyEnd; j++ {
						if result[j] != ' ' || (j > keyStart && j < keyEnd-1) {
							fixed = append(fixed, result[j])
						}
					}
					// Don't add closing quote - it's already there at result[i]
					continue
				} else {
					// Not an unquoted key, just copy what we skipped
					for j := keyStart; j < i; j++ {
						fixed = append(fixed, result[j])
					}
				}
			}
		} else {
			fixed = append(fixed, ch)
			i++
		}
	}

	return string(fixed)
}
