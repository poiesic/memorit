package reembed

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVector(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected []float32
	}{
		{
			name:     "unit vector remains unchanged",
			input:    []float32{1.0, 0.0, 0.0},
			expected: []float32{1.0, 0.0, 0.0},
		},
		{
			name:     "scale non-unit vector",
			input:    []float32{3.0, 4.0},
			expected: []float32{0.6, 0.8},
		},
		{
			name:     "negative values",
			input:    []float32{-1.0, 1.0},
			expected: []float32{-1.0 / float32(math.Sqrt(2)), 1.0 / float32(math.Sqrt(2))},
		},
		{
			name:     "small values",
			input:    []float32{0.001, 0.002, 0.003},
			expected: normalizeReference([]float32{0.001, 0.002, 0.003}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeVector(tt.input)
			require.Equal(t, len(tt.expected), len(result), "vector length mismatch")

			for i := range result {
				assert.InDelta(t, tt.expected[i], result[i], 1e-6, "element %d", i)
			}

			// Verify magnitude is 1.0
			var magnitude float32
			for _, v := range result {
				magnitude += v * v
			}
			magnitude = float32(math.Sqrt(float64(magnitude)))
			assert.InDelta(t, 1.0, magnitude, 1e-6, "magnitude should be 1.0")
		})
	}
}

func TestNormalizeVector_ZeroVector(t *testing.T) {
	input := []float32{0.0, 0.0, 0.0}
	result := NormalizeVector(input)

	// Zero vector should return zero vector (can't normalize)
	for i, v := range result {
		assert.Equal(t, float32(0.0), v, "element %d should be 0", i)
	}
}

func TestNormalizeVector_EmptyVector(t *testing.T) {
	input := []float32{}
	result := NormalizeVector(input)
	assert.Empty(t, result, "empty vector should return empty vector")
}

// normalizeReference is a reference implementation for testing
func normalizeReference(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	var magnitude float32
	for _, val := range v {
		magnitude += val * val
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))

	if magnitude == 0 {
		return v
	}

	result := make([]float32, len(v))
	for i, val := range v {
		result[i] = val / magnitude
	}
	return result
}
