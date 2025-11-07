package badger

import (
	"context"
	"testing"

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
