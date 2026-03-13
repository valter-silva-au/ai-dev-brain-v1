package observability

import (
	"time"
)

// Metrics represents aggregated metrics derived from the event log
type Metrics struct {
	TasksCreated        int                    `json:"tasks_created"`
	TasksCompleted      int                    `json:"tasks_completed"`
	TasksByStatus       map[string]int         `json:"tasks_by_status"`
	TasksByType         map[string]int         `json:"tasks_by_type"`
	AgentSessions       int                    `json:"agent_sessions"`
	KnowledgeExtracts   int                    `json:"knowledge_extracts"`
	WorktreesCreated    int                    `json:"worktrees_created"`
	WorktreesRemoved    int                    `json:"worktrees_removed"`
	LastEventTimestamp  time.Time              `json:"last_event_timestamp"`
	TaskStatusHistory   map[string][]StatusChange `json:"task_status_history"` // task_id -> status changes
}

// StatusChange represents a status change for a task
type StatusChange struct {
	Timestamp time.Time `json:"timestamp"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
}

// MetricsCalculator computes metrics on-demand from event log
type MetricsCalculator struct {
	eventLog *EventLog
}

// NewMetricsCalculator creates a new metrics calculator
func NewMetricsCalculator(eventLog *EventLog) *MetricsCalculator {
	return &MetricsCalculator{
		eventLog: eventLog,
	}
}

// ComputeMetrics derives all metrics from the event log
func (mc *MetricsCalculator) ComputeMetrics() (*Metrics, error) {
	events, err := mc.eventLog.ReadAll()
	if err != nil {
		return nil, err
	}

	metrics := &Metrics{
		TasksByStatus:     make(map[string]int),
		TasksByType:       make(map[string]int),
		TaskStatusHistory: make(map[string][]StatusChange),
	}

	// Process each event
	for _, event := range events {
		// Update last event timestamp
		if event.Timestamp.After(metrics.LastEventTimestamp) {
			metrics.LastEventTimestamp = event.Timestamp
		}

		switch event.Type {
		case EventTaskCreated:
			metrics.TasksCreated++

			// Extract task type if available
			if taskType, ok := event.Data["type"].(string); ok {
				metrics.TasksByType[taskType]++
			}

			// Extract initial status if available
			if status, ok := event.Data["status"].(string); ok {
				metrics.TasksByStatus[status]++
			}

		case EventTaskCompleted:
			metrics.TasksCompleted++

		case EventTaskStatusChanged:
			// Track status changes
			taskID, hasTaskID := event.Data["task_id"].(string)
			oldStatus, hasOldStatus := event.Data["old_status"].(string)
			newStatus, hasNewStatus := event.Data["new_status"].(string)

			if hasTaskID && hasOldStatus && hasNewStatus {
				// Add to history
				if metrics.TaskStatusHistory[taskID] == nil {
					metrics.TaskStatusHistory[taskID] = []StatusChange{}
				}
				metrics.TaskStatusHistory[taskID] = append(
					metrics.TaskStatusHistory[taskID],
					StatusChange{
						Timestamp: event.Timestamp,
						OldStatus: oldStatus,
						NewStatus: newStatus,
					},
				)

				// Update status counts
				metrics.TasksByStatus[oldStatus]--
				if metrics.TasksByStatus[oldStatus] <= 0 {
					delete(metrics.TasksByStatus, oldStatus)
				}
				metrics.TasksByStatus[newStatus]++
			}

		case EventAgentSessionStarted:
			metrics.AgentSessions++

		case EventKnowledgeExtracted:
			metrics.KnowledgeExtracts++

		case EventWorktreeCreated:
			metrics.WorktreesCreated++

		case EventWorktreeRemoved:
			metrics.WorktreesRemoved++
		}
	}

	return metrics, nil
}

// GetTaskDuration calculates how long a task has been in a specific status
func (mc *MetricsCalculator) GetTaskDuration(taskID, status string) (time.Duration, error) {
	events, err := mc.eventLog.ReadAll()
	if err != nil {
		return 0, err
	}

	var lastStatusChange time.Time
	currentStatus := ""

	// Find when the task entered the given status
	for _, event := range events {
		if event.Type == EventTaskCreated {
			if tid, ok := event.Data["task_id"].(string); ok && tid == taskID {
				if s, ok := event.Data["status"].(string); ok {
					currentStatus = s
					lastStatusChange = event.Timestamp
				}
			}
		} else if event.Type == EventTaskStatusChanged {
			if tid, ok := event.Data["task_id"].(string); ok && tid == taskID {
				if newStatus, ok := event.Data["new_status"].(string); ok {
					currentStatus = newStatus
					lastStatusChange = event.Timestamp
				}
			}
		}
	}

	// If the task is currently in the requested status, return duration
	if currentStatus == status {
		return time.Since(lastStatusChange), nil
	}

	return 0, nil
}

// GetTasksInStatus returns task IDs currently in a specific status
func (mc *MetricsCalculator) GetTasksInStatus(status string) ([]string, error) {
	events, err := mc.eventLog.ReadAll()
	if err != nil {
		return nil, err
	}

	// Track current status of each task
	taskStatuses := make(map[string]string)

	for _, event := range events {
		if event.Type == EventTaskCreated {
			if taskID, ok := event.Data["task_id"].(string); ok {
				if s, ok := event.Data["status"].(string); ok {
					taskStatuses[taskID] = s
				}
			}
		} else if event.Type == EventTaskStatusChanged {
			if taskID, ok := event.Data["task_id"].(string); ok {
				if newStatus, ok := event.Data["new_status"].(string); ok {
					taskStatuses[taskID] = newStatus
				}
			}
		}
	}

	// Filter by requested status
	var taskIDs []string
	for taskID, s := range taskStatuses {
		if s == status {
			taskIDs = append(taskIDs, taskID)
		}
	}

	return taskIDs, nil
}
