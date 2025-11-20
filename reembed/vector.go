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
