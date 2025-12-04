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


package reembed

import "math"

// NormalizeVector normalizes a vector to unit length.
// Returns a new vector. If the input is a zero vector, returns a zero vector.
func NormalizeVector(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	// Calculate magnitude
	var magnitude float32
	for _, val := range v {
		magnitude += val * val
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))

	// Can't normalize zero vector
	if magnitude == 0 {
		result := make([]float32, len(v))
		return result
	}

	// Normalize
	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = val / magnitude
	}
	return result
}
