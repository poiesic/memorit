package storage

import (
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalID(t *testing.T) {
	tests := []struct {
		name string
		id   core.ID
	}{
		{"zero ID", core.ID(0)},
		{"small ID", core.ID(42)},
		{"large ID", core.ID(18446744073709551615)}, // max uint64
		{"content-based ID", core.IDFromContent("test content")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data := MarshalID(tt.id)
			require.NotNil(t, data)
			require.NotEmpty(t, data)

			// Unmarshal
			decoded, err := UnmarshalID(data)
			require.NoError(t, err)
			assert.Equal(t, tt.id, decoded)
		})
	}
}

func TestUnmarshalID_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalID(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestMarshalUnmarshalChatRecord(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)

	tests := []struct {
		name   string
		record *core.ChatRecord
	}{
		{
			name: "minimal record",
			record: &core.ChatRecord{
				Id:         core.ID(1),
				Speaker:    core.SpeakerTypeHuman,
				Contents:   "Hello",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "record with concepts",
			record: &core.ChatRecord{
				Id:         core.ID(2),
				Speaker:    core.SpeakerTypeAI,
				Contents:   "I understand.",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
				Concepts: []core.ConceptRef{
					{ConceptId: core.ID(10), Importance: 8},
					{ConceptId: core.ID(20), Importance: 5},
				},
			},
		},
		{
			name: "record with vector",
			record: &core.ChatRecord{
				Id:         core.ID(3),
				Speaker:    core.SpeakerTypeHuman,
				Contents:   "Test with embedding",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
				Vector:     []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			},
		},
		{
			name: "record with everything",
			record: &core.ChatRecord{
				Id:         core.ID(4),
				Speaker:    core.SpeakerTypeAI,
				Contents:   "Complete record with all fields populated for comprehensive testing",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
				Concepts: []core.ConceptRef{
					{ConceptId: core.ID(100), Importance: 10},
					{ConceptId: core.ID(200), Importance: 7},
					{ConceptId: core.ID(300), Importance: 9},
				},
				Vector: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
			},
		},
		{
			name: "empty contents",
			record: &core.ChatRecord{
				Id:         core.ID(5),
				Speaker:    core.SpeakerTypeHuman,
				Contents:   "",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "unicode contents",
			record: &core.ChatRecord{
				Id:         core.ID(6),
				Speaker:    core.SpeakerTypeHuman,
				Contents:   "Hello 世界 🌍 émojis",
				Timestamp:  now,
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data := MarshalChatRecord(tt.record)
			require.NotNil(t, data)
			require.NotEmpty(t, data)

			// Unmarshal
			decoded, err := UnmarshalChatRecord(data)
			require.NoError(t, err)
			require.NotNil(t, decoded)

			// Verify fields
			assert.Equal(t, tt.record.Id, decoded.Id)
			assert.Equal(t, tt.record.Speaker, decoded.Speaker)
			assert.Equal(t, tt.record.Contents, decoded.Contents)
			assert.True(t, tt.record.Timestamp.Equal(decoded.Timestamp))
			assert.True(t, tt.record.InsertedAt.Equal(decoded.InsertedAt))
			assert.True(t, tt.record.UpdatedAt.Equal(decoded.UpdatedAt))
			// Use ElementsMatch to handle nil vs empty slice
			if len(tt.record.Concepts) == 0 {
				assert.Empty(t, decoded.Concepts)
			} else {
				assert.Equal(t, tt.record.Concepts, decoded.Concepts)
			}
			if len(tt.record.Vector) == 0 {
				assert.Empty(t, decoded.Vector)
			} else {
				assert.Equal(t, tt.record.Vector, decoded.Vector)
			}
		})
	}
}

func TestUnmarshalChatRecord_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"invalid data", []byte{0xFF, 0xFF, 0xFF}},
		{"partial data", []byte{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalChatRecord(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestMarshalUnmarshalConcept(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)

	tests := []struct {
		name    string
		concept *core.Concept
	}{
		{
			name: "minimal concept",
			concept: &core.Concept{
				Id:         core.ID(1),
				Name:       "TestConcept",
				Type:       "entity",
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "concept with vector",
			concept: &core.Concept{
				Id:         core.ID(2),
				Name:       "VectorConcept",
				Type:       "topic",
				Vector:     []float32{0.1, 0.2, 0.3, 0.4},
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "concept with special characters",
			concept: &core.Concept{
				Id:         core.ID(3),
				Name:       "Special-Concept_123",
				Type:       "entity",
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "concept with unicode",
			concept: &core.Concept{
				Id:         core.ID(4),
				Name:       "世界",
				Type:       "location",
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
		{
			name: "concept with long vector",
			concept: &core.Concept{
				Id:         core.IDFromContent("(type,long_vector)"),
				Name:       "LongVector",
				Type:       "type",
				Vector:     make([]float32, 1536), // typical OpenAI embedding size
				InsertedAt: now,
				UpdatedAt:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data := MarshalConcept(tt.concept)
			require.NotNil(t, data)
			require.NotEmpty(t, data)

			// Unmarshal
			decoded, err := UnmarshalConcept(data)
			require.NoError(t, err)
			require.NotNil(t, decoded)

			// Verify fields
			assert.Equal(t, tt.concept.Id, decoded.Id)
			assert.Equal(t, tt.concept.Name, decoded.Name)
			assert.Equal(t, tt.concept.Type, decoded.Type)
			assert.True(t, tt.concept.InsertedAt.Equal(decoded.InsertedAt))
			assert.True(t, tt.concept.UpdatedAt.Equal(decoded.UpdatedAt))
			// Handle nil vs empty slice
			if len(tt.concept.Vector) == 0 {
				assert.Empty(t, decoded.Vector)
			} else {
				assert.Equal(t, tt.concept.Vector, decoded.Vector)
			}
		})
	}
}

func TestUnmarshalConcept_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"invalid data", []byte{0xFF, 0xFF, 0xFF}},
		{"partial data", []byte{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalConcept(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestRoundTripConsistency(t *testing.T) {
	t.Run("multiple marshal-unmarshal cycles", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Microsecond)
		original := &core.ChatRecord{
			Id:         core.ID(999),
			Speaker:    core.SpeakerTypeHuman,
			Contents:   "Testing consistency",
			Timestamp:  now,
			InsertedAt: now,
			UpdatedAt:  now,
			Concepts: []core.ConceptRef{
				{ConceptId: core.ID(1), Importance: 8},
			},
			Vector: []float32{0.1, 0.2, 0.3},
		}

		// Perform 3 marshal-unmarshal cycles
		current := original
		for i := 0; i < 3; i++ {
			data := MarshalChatRecord(current)
			decoded, err := UnmarshalChatRecord(data)
			require.NoError(t, err)
			current = decoded
		}

		// Verify final result matches original
		assert.Equal(t, original.Id, current.Id)
		assert.Equal(t, original.Speaker, current.Speaker)
		assert.Equal(t, original.Contents, current.Contents)
		assert.Equal(t, original.Concepts, current.Concepts)
		assert.Equal(t, original.Vector, current.Vector)
	})
}
