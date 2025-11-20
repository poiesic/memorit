package reembed

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ProgressTracker tracks and reports progress of reembedding operations.
type ProgressTracker struct {
	writer         io.Writer
	total          int
	current        int
	reportInterval int
	lastReported   int
	startTime      time.Time
	started        bool
	mu             sync.Mutex
}

// NewProgressTracker creates a new progress tracker.
// writer: where to write progress output (typically os.Stderr)
// total: total number of items to process
// reportInterval: report progress every N items
func NewProgressTracker(writer io.Writer, total, reportInterval int) *ProgressTracker {
	return &ProgressTracker{
		writer:         writer,
		total:          total,
		reportInterval: reportInterval,
	}
}

// Start begins tracking progress.
func (p *ProgressTracker) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.startTime = time.Now()
	p.started = true
	p.current = 0
	p.lastReported = 0
}

// Update sets the current progress to the specified value.
func (p *ProgressTracker) Update(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// Cap at total
	if current > p.total {
		current = p.total
	}

	p.current = current

	// Report if we've crossed a report interval
	if p.current-p.lastReported >= p.reportInterval {
		p.report()
		p.lastReported = p.current
	}
}

// Increment increases the current progress by the specified amount.
func (p *ProgressTracker) Increment(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	p.current += delta
	if p.current > p.total {
		p.current = p.total
	}

	// Report if we've crossed a report interval
	if p.current-p.lastReported >= p.reportInterval {
		p.report()
		p.lastReported = p.current
	}
}

// Finish marks the operation as complete and prints final progress.
func (p *ProgressTracker) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	p.current = p.total
	p.report()
	fmt.Fprintln(p.writer) // Print newline after final progress
}

// Elapsed returns the time elapsed since Start was called.
func (p *ProgressTracker) Elapsed() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return 0
	}

	return time.Since(p.startTime)
}

// report prints the current progress. Must be called with lock held.
func (p *ProgressTracker) report() {
	elapsed := time.Since(p.startTime)
	rate := float64(p.current) / elapsed.Seconds()

	percentage := 0.0
	if p.total > 0 {
		percentage = float64(p.current) / float64(p.total) * 100.0
	}

	fmt.Fprintf(p.writer, "\rProgress: %d/%d (%.1f%%) - %.1f records/s",
		p.current, p.total, percentage, rate)
}
