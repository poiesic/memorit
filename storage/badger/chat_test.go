package badger

import (
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
)

func TestChatRecordBasics(t *testing.T) {
	// Create in-memory repositories
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()

	// Test adding a chat record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Hello, world!",
		Timestamp: time.Now().UTC(),
	}

	added, err := chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add chat record: %v", err)
	}

	if len(added) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(added))
	}

	if added[0].Id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	// Test retrieving the record
	retrieved, err := chatRepo.GetChatRecord(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to get chat record: %v", err)
	}

	if retrieved.Contents != "Hello, world!" {
		t.Fatalf("Expected 'Hello, world!', got '%s'", retrieved.Contents)
	}
}

func TestChatRecordDateRange(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add records with different timestamps
	now := time.Now().UTC()
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: now.Add(-2 * time.Hour)},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2", Timestamp: now.Add(-1 * time.Hour)},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 3", Timestamp: now},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add chat records: %v", err)
	}

	// Query for records in the last 90 minutes
	start := now.Add(-90 * time.Minute)
	end := now.Add(1 * time.Minute)

	results, err := chatRepo.GetChatRecordsByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("Failed to get records by date range: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(results))
	}
}

func TestConceptRefIndex(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add a concept
	concept := &core.Concept{
		Name: "golang",
		Type: "technology",
	}
	addedConcepts, err := conceptRepo.AddConcepts(ctx, concept)
	if err != nil {
		t.Fatalf("Failed to add concept: %v", err)
	}
	conceptID := addedConcepts[0].Id

	// Add a chat record referencing the concept
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "I love golang",
		Timestamp: time.Now().UTC(),
		Concepts: []core.ConceptRef{
			{ConceptId: conceptID, Importance: 8},
		},
	}
	_, err = chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add chat record: %v", err)
	}

	// Query for records by concept
	recordIDs, err := chatRepo.GetChatRecordsByConcept(ctx, conceptID)
	if err != nil {
		t.Fatalf("Failed to get records by concept: %v", err)
	}

	if len(recordIDs) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(recordIDs))
	}
}
