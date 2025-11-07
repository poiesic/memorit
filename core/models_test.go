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
