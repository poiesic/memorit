package badger

import (
	"context"
	"testing"
	"time"

	"github.com/poiesic/memorit/core"
)

func TestConceptBasics(t *testing.T) {
	// Create in-memory repository
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Test adding a concept
	concept := &core.Concept{
		Name:   "test",
		Type:   "abstract concept",
		Vector: []float32{0.1, 0.2, 0.3},
	}

	addedConcepts, err := conceptRepo.AddConcepts(ctx, concept)
	if err != nil {
		t.Fatalf("Failed to add concept: %v", err)
	}

	if len(addedConcepts) != 1 {
		t.Fatalf("Expected 1 concept, got %d", len(addedConcepts))
	}

	if addedConcepts[0].Id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	// Test retrieving the concept
	retrievedConcept, err := conceptRepo.GetConcept(ctx, addedConcepts[0].Id)
	if err != nil {
		t.Fatalf("Failed to get concept: %v", err)
	}

	if retrievedConcept.Name != "test" {
		t.Fatalf("Expected 'test', got '%s'", retrievedConcept.Name)
	}

	// Test FindConceptByNameAndType
	found, err := conceptRepo.FindConceptByNameAndType(ctx, "test", "abstract concept")
	if err != nil {
		t.Fatalf("Failed to find concept: %v", err)
	}

	if found.Name != "test" {
		t.Fatalf("Expected 'test', got '%s'", found.Name)
	}
}

func TestGetOrCreateConcept(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Create a concept
	vector := []float32{0.1, 0.2, 0.3}
	concept1, err := conceptRepo.GetOrCreateConcept(ctx, "test", "abstract concept", vector)
	if err != nil {
		t.Fatalf("Failed to create concept: %v", err)
	}

	// Try to create the same concept again
	concept2, err := conceptRepo.GetOrCreateConcept(ctx, "test", "abstract concept", vector)
	if err != nil {
		t.Fatalf("Failed to get concept: %v", err)
	}

	// Should return the same concept
	if concept1.Id != concept2.Id {
		t.Fatalf("Expected same concept ID, got %d and %d", concept1.Id, concept2.Id)
	}
}

func TestUpdateConcepts(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add a concept
	concept := &core.Concept{
		Name:   "original",
		Type:   "entity",
		Vector: []float32{0.1, 0.2, 0.3},
	}
	added, err := conceptRepo.AddConcepts(ctx, concept)
	if err != nil {
		t.Fatalf("Failed to add concept: %v", err)
	}

	// Update the concept
	added[0].Name = "updated"
	updated, err := conceptRepo.UpdateConcepts(ctx, added[0])
	if err != nil {
		t.Fatalf("Failed to update concept: %v", err)
	}

	if updated[0].Name != "updated" {
		t.Fatalf("Expected updated name, got %s", updated[0].Name)
	}

	// Verify the update persisted
	retrieved, err := conceptRepo.GetConcept(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to get concept: %v", err)
	}

	if retrieved.Name != "updated" {
		t.Fatalf("Expected updated name to persist, got %s", retrieved.Name)
	}
}

func TestDeleteConcepts(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add concepts
	concepts := []*core.Concept{
		{Name: "concept1", Type: "entity"},
		{Name: "concept2", Type: "entity"},
	}
	added, err := conceptRepo.AddConcepts(ctx, concepts...)
	if err != nil {
		t.Fatalf("Failed to add concepts: %v", err)
	}

	// Delete first concept
	err = conceptRepo.DeleteConcepts(ctx, added[0].Id)
	if err != nil {
		t.Fatalf("Failed to delete concept: %v", err)
	}

	// Verify it's deleted
	_, err = conceptRepo.GetConcept(ctx, added[0].Id)
	if err == nil {
		t.Fatal("Expected error when getting deleted concept")
	}

	// Verify second concept still exists
	retrieved, err := conceptRepo.GetConcept(ctx, added[1].Id)
	if err != nil {
		t.Fatalf("Failed to get remaining concept: %v", err)
	}
	if retrieved.Name != "concept2" {
		t.Fatalf("Expected 'concept2', got %s", retrieved.Name)
	}
}

func TestGetConcepts_Multiple(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Add concepts
	concepts := []*core.Concept{
		{Name: "concept1", Type: "entity"},
		{Name: "concept2", Type: "entity"},
		{Name: "concept3", Type: "entity"},
	}
	added, err := conceptRepo.AddConcepts(ctx, concepts...)
	if err != nil {
		t.Fatalf("Failed to add concepts: %v", err)
	}

	// Get multiple concepts
	retrieved, err := conceptRepo.GetConcepts(ctx, added[0].Id, added[2].Id)
	if err != nil {
		t.Fatalf("Failed to get concepts: %v", err)
	}

	if len(retrieved) != 2 {
		t.Fatalf("Expected 2 concepts, got %d", len(retrieved))
	}
}

func TestConceptRepository_FindSimilar(t *testing.T) {
	chatRepo, conceptRepo, backend, err := NewMemoryRepositories()
	if err != nil {
		t.Fatalf("Failed to create repositories: %v", err)
	}
	defer func() { conceptRepo.Close(); chatRepo.Close(); backend.Close() }()

	ctx := context.Background()

	// Note: ConceptRepository.FindSimilar delegates to backend which searches ChatRecords
	// So we need to add chat records with vectors, not concepts
	records := []*core.ChatRecord{
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "similar message 1",
			Timestamp: time.Now().UTC(),
			Vector:    []float32{1.0, 0.0, 0.0},
		},
		{
			Speaker:   core.SpeakerTypeHuman,
			Contents:  "similar message 2",
			Timestamp: time.Now().UTC(),
			Vector:    []float32{0.9, 0.1, 0.0},
		},
	}
	_, err = chatRepo.AddChatRecords(ctx, records...)
	if err != nil {
		t.Fatalf("Failed to add chat records: %v", err)
	}

	// Search for similar records via concept repository
	queryVector := []float32{1.0, 0.0, 0.0}
	results, err := conceptRepo.FindSimilar(ctx, queryVector, 0.8, 10)
	if err != nil {
		t.Fatalf("Failed to find similar: %v", err)
	}

	// Should find at least the most similar record
	if len(results) == 0 {
		t.Fatal("Expected to find similar records")
	}

	// Results should be sorted by score
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Fatal("Results should be sorted by score descending")
		}
	}
}
