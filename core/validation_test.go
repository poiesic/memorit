package core

import (
	"errors"
	"testing"
	"time"
)

func TestValidateChatRecord(t *testing.T) {
	validTime := time.Now().Add(-1 * time.Hour)
	futureTime := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name    string
		record  *ChatRecord
		wantErr error
	}{
		{
			name: "valid record",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerTypeHuman,
				Contents:  "Hello world",
				Timestamp: validTime,
			},
			wantErr: nil,
		},
		{
			name: "valid record with empty vector",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerTypeAI,
				Contents:  "Response",
				Timestamp: validTime,
				Vector:    nil,
			},
			wantErr: nil,
		},
		{
			name: "valid record with empty concepts",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerTypeHuman,
				Contents:  "Message",
				Timestamp: validTime,
				Concepts:  nil,
			},
			wantErr: nil,
		},
		{
			name: "valid record with ID 0",
			record: &ChatRecord{
				Id:        0,
				Speaker:   SpeakerTypeHuman,
				Contents:  "Message",
				Timestamp: validTime,
			},
			wantErr: nil,
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: ErrInvalidChatRecord,
		},
		{
			name: "empty contents",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerTypeHuman,
				Contents:  "",
				Timestamp: validTime,
			},
			wantErr: ErrEmptyContent,
		},
		{
			name: "invalid speaker type",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerType(999),
				Contents:  "Hello",
				Timestamp: validTime,
			},
			wantErr: ErrInvalidSpeakerType,
		},
		{
			name: "future timestamp",
			record: &ChatRecord{
				Id:        1,
				Speaker:   SpeakerTypeHuman,
				Contents:  "Hello",
				Timestamp: futureTime,
			},
			wantErr: ErrInvalidTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChatRecord(tt.record)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateChatRecord() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Errorf("ValidateChatRecord() error = nil, want %v", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateChatRecord() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConcept(t *testing.T) {
	tests := []struct {
		name    string
		concept *Concept
		wantErr error
	}{
		{
			name: "valid concept",
			concept: &Concept{
				Id:   1,
				Name: "example",
				Type: "thing",
			},
			wantErr: nil,
		},
		{
			name: "valid concept with empty vector",
			concept: &Concept{
				Id:     1,
				Name:   "example",
				Type:   "thing",
				Vector: nil,
			},
			wantErr: nil,
		},
		{
			name: "valid concept with ID 0",
			concept: &Concept{
				Id:   0,
				Name: "example",
				Type: "thing",
			},
			wantErr: nil,
		},
		{
			name:    "nil concept",
			concept: nil,
			wantErr: ErrInvalidConcept,
		},
		{
			name: "empty name",
			concept: &Concept{
				Id:   1,
				Name: "",
				Type: "thing",
			},
			wantErr: ErrEmptyConceptName,
		},
		{
			name: "empty type",
			concept: &Concept{
				Id:   1,
				Name: "example",
				Type: "",
			},
			wantErr: ErrEmptyConceptType,
		},
		{
			name: "empty name and type",
			concept: &Concept{
				Id:   1,
				Name: "",
				Type: "",
			},
			wantErr: ErrEmptyConceptName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConcept(tt.concept)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateConcept() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Errorf("ValidateConcept() error = nil, want %v", tt.wantErr)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateConcept() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSpeakerType(t *testing.T) {
	tests := []struct {
		name    string
		speaker SpeakerType
		wantErr bool
	}{
		{
			name:    "human speaker",
			speaker: SpeakerTypeHuman,
			wantErr: false,
		},
		{
			name:    "AI speaker",
			speaker: SpeakerTypeAI,
			wantErr: false,
		},
		{
			name:    "invalid speaker (0)",
			speaker: SpeakerType(0),
			wantErr: true,
		},
		{
			name:    "invalid speaker (999)",
			speaker: SpeakerType(999),
			wantErr: true,
		},
		{
			name:    "invalid speaker (-1)",
			speaker: SpeakerType(-1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSpeakerType(tt.speaker)

			if tt.wantErr && err == nil {
				t.Error("ValidateSpeakerType() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateSpeakerType() error = %v, want nil", err)
			}

			if err != nil && !errors.Is(err, ErrInvalidSpeakerType) {
				t.Errorf("ValidateSpeakerType() error = %v, want %v", err, ErrInvalidSpeakerType)
			}
		})
	}
}

func TestIsValidTimestamp(t *testing.T) {
	tests := []struct {
		name string
		ts   time.Time
		want bool
	}{
		{
			name: "past timestamp",
			ts:   time.Now().Add(-1 * time.Hour),
			want: true,
		},
		{
			name: "current time (approximately)",
			ts:   time.Now(),
			want: true,
		},
		{
			name: "future timestamp",
			ts:   time.Now().Add(1 * time.Hour),
			want: false,
		},
		{
			name: "zero time",
			ts:   time.Time{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTimestamp(tt.ts)
			if got != tt.want {
				t.Errorf("IsValidTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}
