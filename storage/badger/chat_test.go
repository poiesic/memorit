package badger

import (
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
	"github.com/stretchr/testify/require"
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

	// Add 5 records with different timestamps
	now := time.Now().UTC()
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

	// Get 3 most recent records
	recent, err := chatRepo.GetRecentChatRecords(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to get recent records: %v", err)
	}

	if len(recent) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(recent))
	}

	// Verify they're in descending order (newest first)
	if recent[0].Contents != "Message 3" {
		t.Fatalf("Expected 'Message 3' first, got '%s'", recent[0].Contents)
	}

	if recent[1].Contents != "Response 2" {
		t.Fatalf("Expected 'Response 2' second, got '%s'", recent[1].Contents)
	}

	if recent[2].Contents != "Message 2" {
		t.Fatalf("Expected 'Message 2' third, got '%s'", recent[2].Contents)
	}
}

func TestConceptIndex(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add a concept first
	concepts := []*core.Concept{
		{
			Name: "golang",
			Type: "technology",
		},
	}

	addedConcepts, err := conceptRepo.AddConcepts(ctx, concepts...)
	if err != nil {
		t.Fatalf("Failed to add concept: %v", err)
	}

	conceptID := addedConcepts[0].Id

	// Add chat records with this concept
	now := time.Now().UTC()
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "I love Go",
			Timestamp: now,
			Concepts: []core.ConceptRef{
				{ConceptId: conceptID, Importance: 8},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Go is great",
			Timestamp: now.Add(1 * time.Minute),
			Concepts: []core.ConceptRef{
				{ConceptId: conceptID, Importance: 9},
			},
		},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add chat records: %v", err)
	}

	// Query by concept
	recordIDs, err := chatRepo.GetChatRecordsByConcept(ctx, conceptID)
	if err != nil {
		t.Fatalf("Failed to get records by concept: %v", err)
	}

	if len(recordIDs) != 2 {
		t.Fatalf("Expected 2 record IDs, got %d", len(recordIDs))
	}
}

func TestUpdateChatRecord(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add a record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "Original",
		Timestamp: time.Now().UTC(),
	}

	added, err := chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update it
	added[0].Contents = "Updated"
	_, err = chatRepo.UpdateChatRecords(ctx, added[0])
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Retrieve and verify
	retrieved, err := chatRepo.GetChatRecord(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if retrieved.Contents != "Updated" {
		t.Fatalf("Expected 'Updated', got '%s'", retrieved.Contents)
	}
}

func TestDeleteChatRecord(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add a record
	record := &core.ChatRecord{
		Speaker:   core.SpeakerTypeHuman,
		Contents:  "To be deleted",
		Timestamp: time.Now().UTC(),
	}

	added, err := chatRepo.AddChatRecords(ctx, record)
	if err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Delete it
	err = chatRepo.DeleteChatRecords(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify it's gone
	_, err = chatRepo.GetChatRecord(ctx, added[0].Id)
	if err == nil {
		t.Fatal("Expected error when getting deleted record")
	}
}

func TestBulkOperations(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add multiple records at once
	now := time.Now().UTC()
	records := []*core.ChatRecord{
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 1", Timestamp: now},
		{Speaker: core.SpeakerTypeHuman, Contents: "Message 2", Timestamp: now.Add(time.Minute)},
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

func TestGetChatRecordsBeforeID(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add 10 records with different timestamps
	now := time.Now().UTC()
	records := []*core.ChatRecord{}
	for i := 0; i < 10; i++ {
		records = append(records, &core.ChatRecord{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Message " + string(rune('0'+i)),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
		})
	}

	added, err := chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add chat records: %v", err)
	}

	// Get records before the 7th message (index 6, which has content "Message 6")
	referenceID := added[6].Id
	older, err := chatRepo.GetChatRecordsBeforeID(ctx, referenceID, 3)
	if err != nil {
		t.Fatalf("Failed to get records before ID: %v", err)
	}

	// Should get messages 5, 4, 3 (in that order - descending)
	if len(older) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(older))
	}

	// Verify order (newest to oldest)
	if older[0].Contents != "Message 5" {
		t.Fatalf("Expected 'Message 5' first, got '%s'", older[0].Contents)
	}
	if older[1].Contents != "Message 4" {
		t.Fatalf("Expected 'Message 4' second, got '%s'", older[1].Contents)
	}
	if older[2].Contents != "Message 3" {
		t.Fatalf("Expected 'Message 3' third, got '%s'", older[2].Contents)
	}

	// Test with limit larger than available records
	older, err = chatRepo.GetChatRecordsBeforeID(ctx, added[2].Id, 10)
	if err != nil {
		t.Fatalf("Failed to get records before ID: %v", err)
	}

	// Should only get messages 1 and 0
	if len(older) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(older))
	}

	// Test with first message (should return empty)
	older, err = chatRepo.GetChatRecordsBeforeID(ctx, added[0].Id, 5)
	if err != nil {
		t.Fatalf("Failed to get records before first ID: %v", err)
	}

	if len(older) != 0 {
		t.Fatalf("Expected 0 records before first message, got %d", len(older))
	}
}

func TestGetConceptsByDateRange(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	require.NoError(t, err, "Failed to create repositories")
	defer func() {
		conceptRepo.Close()
		chatRepo.Close()
		backend.Close()
	}()

	ctx := context.Background()

	// Create test concepts
	concepts := []*core.Concept{
		{Name: "golang", Type: "technology"},
		{Name: "database", Type: "technology"},
		{Name: "testing", Type: "practice"},
	}

	addedConcepts, err := conceptRepo.AddConcepts(ctx, concepts...)
	require.NoError(t, err, "Failed to add concepts")
	require.Len(t, addedConcepts, 3)

	golangID := addedConcepts[0].Id
	databaseID := addedConcepts[1].Id
	testingID := addedConcepts[2].Id

	// Create chat records with different timestamps and concepts
	now := time.Now().UTC()
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "I love Go programming",
			Timestamp: now.Add(-3 * time.Hour),
			Concepts: []core.ConceptRef{
				{ConceptId: golangID, Importance: 9},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Working with databases in Go",
			Timestamp: now.Add(-2 * time.Hour),
			Concepts: []core.ConceptRef{
				{ConceptId: golangID, Importance: 7},
				{ConceptId: databaseID, Importance: 8},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Writing tests is important",
			Timestamp: now.Add(-1 * time.Hour),
			Concepts: []core.ConceptRef{
				{ConceptId: testingID, Importance: 10},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Another message about Go",
			Timestamp: now,
			Concepts: []core.ConceptRef{
				{ConceptId: golangID, Importance: 8},
			},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "Message with no concepts",
			Timestamp: now.Add(-30 * time.Minute),
		},
	}

	_, err = chatRepo.AddChatRecords(ctx, records...)
	require.NoError(t, err, "Failed to add chat records")

	t.Run("returns unique concepts from messages in date range", func(t *testing.T) {
		// Query for records from last 2.5 hours (should include records at -2h, -1h, and -30min)
		start := now.Add(-150 * time.Minute)
		end := now.Add(1 * time.Minute)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")

		// Should get golang, database, and testing concepts (deduplicated)
		require.Len(t, results, 3, "Expected 3 unique concepts")

		// Verify we got the right concepts
		conceptIDs := make(map[core.ID]bool)
		for _, c := range results {
			conceptIDs[c.Id] = true
		}

		require.True(t, conceptIDs[golangID], "Expected golang concept")
		require.True(t, conceptIDs[databaseID], "Expected database concept")
		require.True(t, conceptIDs[testingID], "Expected testing concept")
	})

	t.Run("deduplicates concepts appearing in multiple messages", func(t *testing.T) {
		// Query for all records (golang appears in 3 different messages)
		start := now.Add(-4 * time.Hour)
		end := now.Add(1 * time.Hour)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")

		// Should get 3 unique concepts even though golang appears 3 times
		require.Len(t, results, 3, "Expected 3 unique concepts")

		// Count how many times we see the golang concept (should be exactly once)
		golangCount := 0
		for _, c := range results {
			if c.Id == golangID {
				golangCount++
				require.Equal(t, "golang", c.Name)
				require.Equal(t, "technology", c.Type)
			}
		}
		require.Equal(t, 1, golangCount, "golang concept should appear exactly once")
	})

	t.Run("returns empty when no records in date range", func(t *testing.T) {
		// Query for a time range with no records
		start := now.Add(-10 * time.Hour)
		end := now.Add(-9 * time.Hour)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")
		require.Empty(t, results, "Expected no concepts in empty date range")
	})

	t.Run("handles records without concepts", func(t *testing.T) {
		// Query for range that includes the message with no concepts
		start := now.Add(-45 * time.Minute)
		end := now.Add(-15 * time.Minute)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")
		require.Empty(t, results, "Expected no concepts when messages have no concepts")
	})

	t.Run("handles equal start and end times", func(t *testing.T) {
		// When start == end, implementation adds 1 microsecond to end
		exactTime := now.Add(-2 * time.Hour)

		results, err := chatRepo.GetConceptsByDateRange(ctx, exactTime, exactTime)
		require.NoError(t, err, "Failed to get concepts with equal start and end")

		// This is testing the edge case handling in the implementation
		// Since we add 1 microsecond, and our record is at exactly -2h,
		// it should include that record
		require.NotEmpty(t, results, "Expected to find concepts when start equals end")
	})

	t.Run("returns concepts with full details", func(t *testing.T) {
		start := now.Add(-3 * time.Hour)
		end := now.Add(-90 * time.Minute)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")

		// Should get golang and database concepts
		require.Len(t, results, 2, "Expected 2 concepts")

		// Verify concepts have all their details
		for _, c := range results {
			require.NotZero(t, c.Id, "Concept ID should be set")
			require.NotEmpty(t, c.Name, "Concept name should be set")
			require.NotEmpty(t, c.Type, "Concept type should be set")
			require.NotZero(t, c.InsertedAt, "InsertedAt should be set")
			require.NotZero(t, c.UpdatedAt, "UpdatedAt should be set")
		}
	})

	t.Run("respects date range boundaries", func(t *testing.T) {
		// Query for exactly the range containing only the -1h message
		start := now.Add(-90 * time.Minute)
		end := now.Add(-30 * time.Minute)

		results, err := chatRepo.GetConceptsByDateRange(ctx, start, end)
		require.NoError(t, err, "Failed to get concepts by date range")

		// Should only get the testing concept
		require.Len(t, results, 1, "Expected 1 concept")
		require.Equal(t, testingID, results[0].Id)
		require.Equal(t, "testing", results[0].Name)
		require.Equal(t, "practice", results[0].Type)
	})
}
