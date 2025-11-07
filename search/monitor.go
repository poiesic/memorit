package search

import (
	"iter"

	"github.com/poiesic/memorit/core"
)

// SearchMonitor provides hooks to observe the search process.
// Implement this interface to track intermediate steps and results during search.
type SearchMonitor interface {
	Start(query string)
	AfterSemanticSearch(ids []uint64)
	AfterQueryConceptExtraction(concepts []*core.Concept)
	FoundRelatedConcepts(tuple string, conceptIds []uint64)
	AfterConceptuallyRelatedSearch(iter.Seq[uint64])
	AfterRecordRetrieval(records []*core.ChatRecord)
	SemanticAndConceptualHit(record *core.ChatRecord)
	SemanticHit(record *core.ChatRecord)
	ConceptualHit(record *core.ChatRecord)
	Finish(results []*core.SearchResult)
}

// noopMonitor is a no-op implementation of SearchMonitor
type noopMonitor struct{}

var _ SearchMonitor = (*noopMonitor)(nil)

func (n *noopMonitor) Start(_ string)                                    {}
func (n *noopMonitor) AfterSemanticSearch(_ []uint64)                    {}
func (n *noopMonitor) AfterQueryConceptExtraction(_ []*core.Concept)    {}
func (n *noopMonitor) FoundRelatedConcepts(_ string, _ []uint64)         {}
func (n *noopMonitor) AfterConceptuallyRelatedSearch(_ iter.Seq[uint64]) {}
func (n *noopMonitor) AfterRecordRetrieval(_ []*core.ChatRecord)        {}
func (n *noopMonitor) SemanticAndConceptualHit(_ *core.ChatRecord)      {}
func (n *noopMonitor) SemanticHit(_ *core.ChatRecord)                   {}
func (n *noopMonitor) ConceptualHit(_ *core.ChatRecord)                 {}
func (n *noopMonitor) Finish(_ []*core.SearchResult)                    {}
