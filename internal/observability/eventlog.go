package observability

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventTaskCreated         EventType = "task.created"
	EventTaskCompleted       EventType = "task.completed"
	EventTaskStatusChanged   EventType = "task.status_changed"
	EventAgentSessionStarted EventType = "agent.session_started"
	EventKnowledgeExtracted  EventType = "knowledge.extracted"
	EventWorktreeCreated     EventType = "worktree.created"
	EventWorktreeRemoved     EventType = "worktree.removed"
)

// Event represents a single event in the system
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      EventType              `json:"type"`
	Data      map[string]interface{} `json:"data"`
}

// EventLog manages append-only JSONL event logging
type EventLog struct {
	filePath string
	mu       sync.Mutex
	enabled  bool // false if log file can't be created
}

// NewEventLog creates a new event log
func NewEventLog(filePath string) *EventLog {
	el := &EventLog{
		filePath: filePath,
		enabled:  true,
	}

	// Test if we can write to the file (non-fatal if we can't)
	if err := el.ensureFileExists(); err != nil {
		// Log to stderr but don't crash
		fmt.Fprintf(os.Stderr, "Warning: event log disabled, could not create log file: %v\n", err)
		el.enabled = false
	}

	return el
}

// ensureFileExists ensures the log file exists and is writable
func (el *EventLog) ensureFileExists() error {
	// Try to open/create the file
	f, err := os.OpenFile(el.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// Log writes an event to the log file (thread-safe, non-fatal on error)
func (el *EventLog) Log(eventType EventType, data map[string]interface{}) {
	if !el.enabled {
		return // silently skip if disabled
	}

	el.mu.Lock()
	defer el.mu.Unlock()

	event := Event{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Data:      data,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		// Non-fatal: log to stderr but don't crash
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal event: %v\n", err)
		return
	}

	// Append to file
	f, err := os.OpenFile(el.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open event log: %v\n", err)
		return
	}
	defer f.Close()

	// Write JSONL (JSON + newline)
	if _, err := f.Write(append(jsonData, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write event: %v\n", err)
		return
	}
}

// ReadAll reads all events from the log, gracefully skipping malformed lines
func (el *EventLog) ReadAll() ([]Event, error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(el.filePath); os.IsNotExist(err) {
		return []Event{}, nil
	}

	f, err := os.Open(el.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open event log: %w", err)
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Gracefully skip malformed lines (log warning but continue)
			fmt.Fprintf(os.Stderr, "Warning: skipping malformed event at line %d: %v\n", lineNum, err)
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read event log: %w", err)
	}

	return events, nil
}

// ReadByType reads all events of a specific type
func (el *EventLog) ReadByType(eventType EventType) ([]Event, error) {
	allEvents, err := el.ReadAll()
	if err != nil {
		return nil, err
	}

	var filtered []Event
	for _, event := range allEvents {
		if event.Type == eventType {
			filtered = append(filtered, event)
		}
	}

	return filtered, nil
}

// Clear clears the event log (for testing purposes)
func (el *EventLog) Clear() error {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Remove the file if it exists
	if err := os.Remove(el.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear event log: %w", err)
	}

	return nil
}
