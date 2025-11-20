package reembed

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressTracker_Basic(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 100, 10)

	tracker.Start()
	assert.True(t, tracker.started, "should be started")

	tracker.Increment(25)
	tracker.Increment(25)
	tracker.Increment(50)

	elapsed := tracker.Elapsed()
	assert.Greater(t, elapsed, time.Duration(0), "elapsed time should be positive")

	output := buf.String()
	assert.Contains(t, output, "100/100", "should show completion")
	assert.Contains(t, output, "100.0%", "should show 100%")
}

func TestProgressTracker_Update(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 1000, 100)

	tracker.Start()
	tracker.Update(250)

	time.Sleep(10 * time.Millisecond) // Allow time for potential output

	// Update again to trigger progress
	tracker.Update(500)

	output := buf.String()
	// Should have some progress output
	assert.True(t, len(output) > 0, "should have progress output")
}

func TestProgressTracker_Finish(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 100, 10)

	tracker.Start()
	tracker.Update(75)
	tracker.Finish()

	output := buf.String()
	assert.Contains(t, output, "100/100", "finish should set to total")
	assert.Contains(t, output, "100.0%", "finish should show 100%")
	assert.Contains(t, output, "\n", "finish should print newline")
}

func TestProgressTracker_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 0, 10)

	tracker.Start()
	tracker.Finish()

	output := buf.String()
	assert.Contains(t, output, "0/0", "should handle zero total")
}

func TestProgressTracker_IncrementBeyondTotal(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 100, 10)

	tracker.Start()
	tracker.Increment(150) // More than total

	output := buf.String()
	// Should cap at total
	assert.Contains(t, output, "100/100", "should not exceed total")
}

func TestProgressTracker_Rate(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 1000, 100)

	tracker.Start()
	time.Sleep(50 * time.Millisecond)
	tracker.Update(100)
	time.Sleep(50 * time.Millisecond)

	tracker.Finish()

	output := buf.String()
	assert.Contains(t, output, "records/s", "should show rate")
}

func TestProgressTracker_NotStarted(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 100, 10)

	// Should not panic when not started
	tracker.Increment(10)
	tracker.Finish()

	// No output expected since not started
	output := buf.String()
	assert.Equal(t, "", output, "should have no output when not started")
}

func TestProgressTracker_ReportInterval(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 1000, 100) // Report every 100 records

	tracker.Start()

	// First update under interval - should not print
	buf.Reset()
	tracker.Update(50)
	assert.Equal(t, "", buf.String(), "should not print under interval")

	// Update to exactly interval - should print
	buf.Reset()
	tracker.Update(100)
	output := buf.String()
	assert.True(t, len(output) > 0, "should print at interval")

	// Update beyond interval - should print
	buf.Reset()
	tracker.Update(250)
	output = buf.String()
	assert.True(t, len(output) > 0, "should print beyond interval")
}

func TestProgressTracker_FormattedOutput(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewProgressTracker(&buf, 5000, 1000)

	tracker.Start()
	tracker.Update(2500)
	time.Sleep(10 * time.Millisecond)
	tracker.Update(5000)

	output := buf.String()

	// Check format contains expected elements
	lines := strings.Split(strings.TrimSpace(output), "\r")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		assert.Contains(t, lastLine, "/", "should have progress fraction")
		assert.Contains(t, lastLine, "%", "should have percentage")
	}
}
