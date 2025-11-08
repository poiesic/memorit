package core

import (
	"testing"
)

func TestIDFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantSame bool
	}{
		{
			name:     "same content produces same ID",
			content:  "test content",
			wantSame: true,
		},
		{
			name:     "empty string",
			content:  "",
			wantSame: true,
		},
		{
			name:     "long content",
			content:  "This is a much longer piece of content that should still hash consistently",
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1 := IDFromContent(tt.content)
			id2 := IDFromContent(tt.content)

			if tt.wantSame && id1 != id2 {
				t.Errorf("IDFromContent() produced different IDs for same content: %d vs %d", id1, id2)
			}
		})
	}
}

func TestIDFromContent_Different(t *testing.T) {
	id1 := IDFromContent("content1")
	id2 := IDFromContent("content2")

	if id1 == id2 {
		t.Errorf("IDFromContent() produced same ID for different content")
	}
}

func TestConcept_Tuple(t *testing.T) {
	tests := []struct {
		name    string
		concept Concept
		want    string
	}{
		{
			name: "basic concept",
			concept: Concept{
				Name: "example",
				Type: "thing",
			},
			want: "(thing,example)",
		},
		{
			name: "concept with spaces",
			concept: Concept{
				Name: "example name",
				Type: "thing type",
			},
			want: "(thing type,example name)",
		},
		{
			name: "empty concept",
			concept: Concept{
				Name: "",
				Type: "",
			},
			want: "(,)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.concept.Tuple()
			if got != tt.want {
				t.Errorf("Concept.Tuple() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test MUS serialization for ID
func TestIDMUS_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		id   ID
	}{
		{"zero ID", ID(0)},
		{"small ID", ID(42)},
		{"large ID", ID(18446744073709551615)},
		{"content-based ID", IDFromContent("test")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, IDMUS.Size(tt.id))
			n := IDMUS.Marshal(tt.id, buf)
			if n != len(buf) {
				t.Errorf("Marshal wrote %d bytes, expected %d", n, len(buf))
			}

			decoded, m, err := IDMUS.Unmarshal(buf)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if m != len(buf) {
				t.Errorf("Unmarshal read %d bytes, expected %d", m, len(buf))
			}
			if decoded != tt.id {
				t.Errorf("Unmarshal returned %v, expected %v", decoded, tt.id)
			}
		})
	}
}

func TestIDMUS_Skip(t *testing.T) {
	id := ID(12345)
	buf := make([]byte, IDMUS.Size(id))
	IDMUS.Marshal(id, buf)

	n, err := IDMUS.Skip(buf)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if n != len(buf) {
		t.Errorf("Skip read %d bytes, expected %d", n, len(buf))
	}
}

func TestSpeakerTypeMUS_Skip(t *testing.T) {
	speaker := SpeakerTypeHuman
	buf := make([]byte, SpeakerTypeMUS.Size(speaker))
	SpeakerTypeMUS.Marshal(speaker, buf)

	n, err := SpeakerTypeMUS.Skip(buf)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if n != len(buf) {
		t.Errorf("Skip read %d bytes, expected %d", n, len(buf))
	}
}

func TestConceptRefMUS_Skip(t *testing.T) {
	ref := ConceptRef{ConceptId: ID(100), Importance: 8}
	buf := make([]byte, ConceptRefMUS.Size(ref))
	ConceptRefMUS.Marshal(ref, buf)

	n, err := ConceptRefMUS.Skip(buf)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if n != len(buf) {
		t.Errorf("Skip read %d bytes, expected %d", n, len(buf))
	}
}

func TestChatRecordMUS_Skip(t *testing.T) {
	record := ChatRecord{
		Id:       ID(1),
		Speaker:  SpeakerTypeHuman,
		Contents: "Test",
		Concepts: []ConceptRef{{ConceptId: ID(10), Importance: 5}},
		Vector:   []float32{0.1, 0.2},
	}
	buf := make([]byte, ChatRecordMUS.Size(record))
	ChatRecordMUS.Marshal(record, buf)

	n, err := ChatRecordMUS.Skip(buf)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if n != len(buf) {
		t.Errorf("Skip read %d bytes, expected %d", n, len(buf))
	}
}

func TestConceptMUS_Skip(t *testing.T) {
	concept := Concept{
		Id:     ID(1),
		Name:   "test",
		Type:   "entity",
		Vector: []float32{0.1, 0.2, 0.3},
	}
	buf := make([]byte, ConceptMUS.Size(concept))
	ConceptMUS.Marshal(concept, buf)

	n, err := ConceptMUS.Skip(buf)
	if err != nil {
		t.Fatalf("Skip failed: %v", err)
	}
	if n != len(buf) {
		t.Errorf("Skip read %d bytes, expected %d", n, len(buf))
	}
}

// Test MUS serialization for SpeakerType
func TestSpeakerTypeMUS_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		speakerType SpeakerType
	}{
		{"human", SpeakerTypeHuman},
		{"AI", SpeakerTypeAI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, SpeakerTypeMUS.Size(tt.speakerType))
			n := SpeakerTypeMUS.Marshal(tt.speakerType, buf)
			if n != len(buf) {
				t.Errorf("Marshal wrote %d bytes, expected %d", n, len(buf))
			}

			decoded, m, err := SpeakerTypeMUS.Unmarshal(buf)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if m != len(buf) {
				t.Errorf("Unmarshal read %d bytes, expected %d", m, len(buf))
			}
			if decoded != tt.speakerType {
				t.Errorf("Unmarshal returned %v, expected %v", decoded, tt.speakerType)
			}
		})
	}
}

// Test MUS serialization for ConceptRef
func TestConceptRefMUS_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		ref  ConceptRef
	}{
		{
			name: "basic ref",
			ref:  ConceptRef{ConceptId: ID(100), Importance: 8},
		},
		{
			name: "zero ID",
			ref:  ConceptRef{ConceptId: ID(0), Importance: 1},
		},
		{
			name: "max importance",
			ref:  ConceptRef{ConceptId: ID(999), Importance: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, ConceptRefMUS.Size(tt.ref))
			n := ConceptRefMUS.Marshal(tt.ref, buf)
			if n != len(buf) {
				t.Errorf("Marshal wrote %d bytes, expected %d", n, len(buf))
			}

			decoded, m, err := ConceptRefMUS.Unmarshal(buf)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if m != len(buf) {
				t.Errorf("Unmarshal read %d bytes, expected %d", m, len(buf))
			}
			if decoded != tt.ref {
				t.Errorf("Unmarshal returned %v, expected %v", decoded, tt.ref)
			}
		})
	}
}

// Test MUS serialization for ChatRecord
func TestChatRecordMUS_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name   string
		record ChatRecord
	}{
		{
			name: "minimal record",
			record: ChatRecord{
				Id:       ID(1),
				Speaker:  SpeakerTypeHuman,
				Contents: "Hello",
			},
		},
		{
			name: "with concepts",
			record: ChatRecord{
				Id:       ID(2),
				Speaker:  SpeakerTypeAI,
				Contents: "Response",
				Concepts: []ConceptRef{
					{ConceptId: ID(10), Importance: 8},
					{ConceptId: ID(20), Importance: 6},
				},
			},
		},
		{
			name: "with vector",
			record: ChatRecord{
				Id:       ID(3),
				Speaker:  SpeakerTypeHuman,
				Contents: "Query",
				Vector:   []float32{0.1, 0.2, 0.3},
			},
		},
		{
			name: "complete record",
			record: ChatRecord{
				Id:       ID(4),
				Speaker:  SpeakerTypeAI,
				Contents: "Complete response",
				Concepts: []ConceptRef{
					{ConceptId: ID(100), Importance: 9},
				},
				Vector: []float32{0.5, 0.6},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, ChatRecordMUS.Size(tt.record))
			n := ChatRecordMUS.Marshal(tt.record, buf)
			if n != len(buf) {
				t.Errorf("Marshal wrote %d bytes, expected %d", n, len(buf))
			}

			decoded, m, err := ChatRecordMUS.Unmarshal(buf)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if m != len(buf) {
				t.Errorf("Unmarshal read %d bytes, expected %d", m, len(buf))
			}

			// Compare fields
			if decoded.Id != tt.record.Id {
				t.Errorf("Id: got %v, want %v", decoded.Id, tt.record.Id)
			}
			if decoded.Speaker != tt.record.Speaker {
				t.Errorf("Speaker: got %v, want %v", decoded.Speaker, tt.record.Speaker)
			}
			if decoded.Contents != tt.record.Contents {
				t.Errorf("Contents: got %v, want %v", decoded.Contents, tt.record.Contents)
			}
			if len(decoded.Concepts) != len(tt.record.Concepts) {
				t.Errorf("Concepts length: got %v, want %v", len(decoded.Concepts), len(tt.record.Concepts))
			}
			if len(decoded.Vector) != len(tt.record.Vector) {
				t.Errorf("Vector length: got %v, want %v", len(decoded.Vector), len(tt.record.Vector))
			}
		})
	}
}

// Test MUS serialization for Concept
func TestConceptMUS_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		concept Concept
	}{
		{
			name: "minimal concept",
			concept: Concept{
				Id:   ID(1),
				Name: "test",
				Type: "entity",
			},
		},
		{
			name: "with vector",
			concept: Concept{
				Id:     ID(2),
				Name:   "vectorized",
				Type:   "topic",
				Vector: []float32{0.1, 0.2, 0.3, 0.4},
			},
		},
		{
			name: "unicode name",
			concept: Concept{
				Id:   ID(3),
				Name: "世界",
				Type: "location",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, ConceptMUS.Size(tt.concept))
			n := ConceptMUS.Marshal(tt.concept, buf)
			if n != len(buf) {
				t.Errorf("Marshal wrote %d bytes, expected %d", n, len(buf))
			}

			decoded, m, err := ConceptMUS.Unmarshal(buf)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if m != len(buf) {
				t.Errorf("Unmarshal read %d bytes, expected %d", m, len(buf))
			}

			if decoded.Id != tt.concept.Id {
				t.Errorf("Id: got %v, want %v", decoded.Id, tt.concept.Id)
			}
			if decoded.Name != tt.concept.Name {
				t.Errorf("Name: got %v, want %v", decoded.Name, tt.concept.Name)
			}
			if decoded.Type != tt.concept.Type {
				t.Errorf("Type: got %v, want %v", decoded.Type, tt.concept.Type)
			}
			if len(decoded.Vector) != len(tt.concept.Vector) {
				t.Errorf("Vector length: got %v, want %v", len(decoded.Vector), len(tt.concept.Vector))
			}
		})
	}
}
