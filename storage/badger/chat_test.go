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

func TestGetRecentChatRecords(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add records with incrementing timestamps
	now := time.Now().UTC().Truncate(time.Microsecond)
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: now.Add(-4 * time.Hour)},
		{Speaker: core.SpeakerTypeAI, Contents: "Response 1", Timestamp: now.Add(-3 * time.Hour)},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2", Timestamp: now.Add(-2 * time.Hour)},
		{Speaker: core.SpeakerTypeAI, Contents: "Response 2", Timestamp: now.Add(-1 * time.Hour)},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 3", Timestamp: now},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add chat records: %v", err)
	}

	// Test: Get last 3 records
	results, err := chatRepo.GetRecentChatRecords(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to get recent records: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(results))
	}

	// Verify order: most recent first
	if results[0].Contents != "Message 3" {
		t.Errorf("Expected 'Message 3' first, got '%s'", results[0].Contents)
	}
	if results[1].Contents != "Response 2" {
		t.Errorf("Expected 'Response 2' second, got '%s'", results[1].Contents)
	}
	if results[2].Contents != "Message 2" {
		t.Errorf("Expected 'Message 2' third, got '%s'", results[2].Contents)
	}

	// Test: Get all records
	allResults, err := chatRepo.GetRecentChatRecords(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get all records: %v", err)
	}

	if len(allResults) != 5 {
		t.Fatalf("Expected 5 records, got %d", len(allResults))
	}

	// Test: Get zero records
	zeroResults, err := chatRepo.GetRecentChatRecords(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to get zero records: %v", err)
	}

	if len(zeroResults) != 0 {
		t.Fatalf("Expected 0 records, got %d", len(zeroResults))
	}

	// Test: Empty database
	chatRepo2, conceptRepo2, backend2, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create second repository: %v", err)
	}
	defer func() { conceptRepo2.Close(); chatRepo2.Close(); backend2.Close() }()

	emptyResults, err := chatRepo2.GetRecentChatRecords(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to query empty database: %v", err)
	}

	if len(emptyResults) != 0 {
		t.Fatalf("Expected 0 records from empty database, got %d", len(emptyResults))
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

func TestUpdateChatRecords(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add a record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Original content",
		Timestamp: now,
	}
	added, err := chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update the record
	added[0].Contents = "Updated content"
	updated, err := chatRepo.UpdateChatRecords(ctx, added[0])
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	if updated[0].Contents != "Updated content" {
		t.Fatalf("Expected updated content, got %s", updated[0].Contents)
	}

	// Verify the update persisted
	retrieved, err := chatRepo.GetChatRecord(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if retrieved.Contents != "Updated content" {
		t.Fatalf("Expected updated content to persist, got %s", retrieved.Contents)
	}
}

func TestUpdateChatRecords_WithConceptChanges(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add a concept
	concept := &core.Concept{
		Name: "test",
		Type: "entity",
	}
	addedConcepts, err := conceptRepo.AddConcepts(ctx, concept)
	if err != nil {
		t.Fatalf("Failed to add concept: %v", err)
	}

	// Add a record with concept
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Test content",
		Timestamp: now,
		Concepts: []core.ConceptRef{
			{ConceptId: addedConcepts[0].Id, Importance: 8},
		},
	}
	added, err := chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update record to remove concepts
	added[0].Concepts = []core.ConceptRef{}
	_, err = chatRepo.UpdateChatRecords(ctx, added[0])
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Verify concept index was updated
	recordIDs, err := chatRepo.GetChatRecordsByConcept(ctx, addedConcepts[0].Id)
	if err != nil {
		t.Fatalf("Failed to get records by concept: %v", err)
	}

	if len(recordIDs) != 0 {
		t.Fatalf("Expected 0 records after concept removal, got %d", len(recordIDs))
	}
}

func TestDeleteChatRecords(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: now},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2", Timestamp: now},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add records: %v", err)
	}

	// Delete first record
	err = chatRepo.DeleteChatRecords(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify it's deleted
	_, err = chatRepo.GetChatRecord(ctx, added[0].Id)
	if err == nil {
		t.Fatal("Expected error when getting deleted record")
	}

	// Verify second record still exists
	retrieved, err := chatRepo.GetChatRecord(ctx, added[1].Id)
	if err != nil {
		t.Fatalf("Failed to get remaining record: %v", err)
	}
	if retrieved.Contents != "Message 2" {
		t.Fatalf("Expected 'Message 2', got %s", retrieved.Contents)
	}
}

func TestGetChatRecords_Multiple(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()
	now := time.Now().UTC()

	// Add records
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: now},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2", Timestamp: now},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 3", Timestamp: now},
	}
	added, err := chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add records: %v", err)
	}

	// Get multiple records
	retrieved, err := chatRepo.GetChatRecords(ctx, added[0].Id, added[2].Id)
	if err != nil {
		t.Fatalf("Failed to get records: %v", err)
	}

	if len(retrieved) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(retrieved))
	}
}
