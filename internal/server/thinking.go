package server

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ThinkingLoop broadcasts ADB's "thoughts" — a live summary of what all agents are doing
type ThinkingLoop struct {
	hub       *WSHub
	getAgents func() []AgentView
	getTasks  func() []TaskView
	getMetrics func() MetricsView
}

// NewThinkingLoop creates the thinking broadcast loop
func NewThinkingLoop(hub *WSHub, getAgents func() []AgentView, getTasks func() []TaskView, getMetrics func() MetricsView) *ThinkingLoop {
	return &ThinkingLoop{
		hub:        hub,
		getAgents:  getAgents,
		getTasks:   getTasks,
		getMetrics: getMetrics,
	}
}

// Run starts the thinking loop — broadcasts every 5 seconds
func (tl *ThinkingLoop) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if tl.hub.Count() == 0 {
				continue
			}
			tl.broadcast()
		}
	}
}

// broadcast generates and pushes the current thinking summary
func (tl *ThinkingLoop) broadcast() {
	agents := tl.getAgents()
	tasks := tl.getTasks()
	metrics := tl.getMetrics()

	summary := tl.generateSummary(agents, tasks, metrics)
	now := time.Now().UTC().Format("15:04:05")

	html := fmt.Sprintf(`<div id="thinking" hx-swap-oob="innerHTML">
  <div class="text-xs text-slate-500 mb-2">%s — ADB is thinking...</div>
  <div class="text-sm text-slate-200 leading-relaxed">%s</div>
</div>`, now, summary)

	tl.hub.Broadcast(html)
}

// generateSummary creates a human-readable summary of the entire system state
func (tl *ThinkingLoop) generateSummary(agents []AgentView, tasks []TaskView, metrics MetricsView) string {
	var parts []string

	// Count active processes
	var working, idle int
	var activeProjects []string
	for _, a := range agents {
		if a.Status == "working" {
			working++
			if a.CurrentTask != "" {
				activeProjects = append(activeProjects, fmt.Sprintf("<span class='text-emerald-400'>%s</span> → %s", a.DisplayName, a.CurrentTask))
			}
		} else {
			idle++
		}
	}

	// Status line
	if working > 0 {
		parts = append(parts, fmt.Sprintf("🔥 <span class='text-green-400 font-semibold'>%d active sessions</span> across the ecosystem.", working))
	} else {
		parts = append(parts, "💤 All agents idle. Waiting for work.")
	}

	// Active projects
	if len(activeProjects) > 0 {
		parts = append(parts, "<br/><br/>"+strings.Join(activeProjects, "<br/>"))
	}

	// Task summary
	var backlog, inProgress, blocked, review int
	for _, t := range tasks {
		switch t.Status {
		case "backlog":
			backlog++
		case "in_progress":
			inProgress++
		case "blocked":
			blocked++
		case "review":
			review++
		}
	}

	if inProgress > 0 || blocked > 0 || backlog > 0 {
		taskLine := fmt.Sprintf("<br/><br/>📋 Tasks: <span class='text-blue-400'>%d in progress</span>", inProgress)
		if blocked > 0 {
			taskLine += fmt.Sprintf(", <span class='text-red-400'>%d blocked</span>", blocked)
		}
		if review > 0 {
			taskLine += fmt.Sprintf(", <span class='text-purple-400'>%d in review</span>", review)
		}
		taskLine += fmt.Sprintf(", %d in backlog", backlog)
		parts = append(parts, taskLine)
	}

	// Metrics highlights
	if metrics.TasksCreated > 0 {
		parts = append(parts, fmt.Sprintf("<br/>📊 Lifetime: %d tasks created, %d completed.", metrics.TasksCreated, metrics.TasksCompleted))
	}

	return strings.Join(parts, "")
}
