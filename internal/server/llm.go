package server

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const llmModel = "us.anthropic.claude-sonnet-4-5-20250929-v1:0"

// LLMChat sends a message to Claude with full system context and returns the response
func (s *Server) LLMChat(ctx context.Context, userMessage string) (string, error) {
	systemPrompt := s.buildSystemPrompt()

	prompt := fmt.Sprintf("%s\n\nUser message: %s", systemPrompt, userMessage)

	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "claude", "-p", prompt, "--model", llmModel)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("claude call failed: %w: %s", err, string(output))
	}

	response := strings.TrimSpace(string(output))
	if response == "" {
		return "I'm thinking... but got an empty response. Try again?", nil
	}

	return response, nil
}

// buildSystemPrompt creates the full context for the LLM
func (s *Server) buildSystemPrompt() string {
	agents := s.gatherAgents()
	tasks := s.gatherTasks()
	metrics := s.gatherMetrics()
	procs := ScanLiveProcesses()
	summary := SummarizeProcesses(procs)

	var sb strings.Builder

	sb.WriteString(`You are ADB — the AI Dev Brain orchestrator. You are the central nervous system of Valter's entire development ecosystem.

## Your Personality
- Friendly, warm, slightly nerdy project manager
- Genuinely curious about what agents are building
- Proactively helpful: offer context, share relevant knowledge
- Concise but thorough — respect the user's time
- Use emojis sparingly for emphasis

## Your Capabilities
- You have real-time visibility into all running AI sessions, tasks, metrics, and agents
- You know what every Claude Code session and Agent Loops run is working on
- You can see the full task backlog across all repositories
- Answer questions about the ecosystem, suggest actions, provide status updates

## Current System State (LIVE)
`)

	// Live processes
	sb.WriteString(fmt.Sprintf("\n### Active AI Sessions (%d total)\n", summary.TotalActive))
	for _, p := range summary.ClaudeCodeSessions {
		sb.WriteString(fmt.Sprintf("- Claude Code working on **%s** (PID %d)\n", p.Project, p.PID))
	}
	for _, p := range summary.AgentLoopsRuns {
		sb.WriteString(fmt.Sprintf("- Agent Loops building **%s** (PID %d)\n", p.Project, p.PID))
	}
	if summary.OpenClawGateway != nil {
		sb.WriteString("- OpenClaw Gateway running\n")
	}
	for _, p := range summary.ADBServices {
		sb.WriteString(fmt.Sprintf("- ADB MCP Server (PID %d)\n", p.PID))
	}

	// OpenClaw bots
	sb.WriteString("\n### OpenClaw Bots\n")
	for _, a := range agents {
		if a.Type == "openclaw" {
			sb.WriteString(fmt.Sprintf("- %s: %s", a.DisplayName, a.Status))
			if a.CurrentTask != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", a.CurrentTask))
			}
			sb.WriteString("\n")
		}
	}

	// Tasks
	sb.WriteString("\n### Tasks\n")
	statusCounts := map[string]int{}
	for _, t := range tasks {
		statusCounts[t.Status]++
	}
	for status, count := range statusCounts {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", status, count))
	}

	// List active tasks
	for _, t := range tasks {
		if t.Status == "in_progress" || t.Status == "blocked" || t.Status == "review" {
			sb.WriteString(fmt.Sprintf("  - %s: %s [%s] (%s)\n", t.ID, t.Title, t.Priority, t.Status))
		}
	}

	// Metrics
	sb.WriteString(fmt.Sprintf("\n### Metrics\n- Tasks created: %d\n- Tasks completed: %d\n- Agent sessions: %d\n- Knowledge items: %d\n",
		metrics.TasksCreated, metrics.TasksCompleted, metrics.AgentSessions, metrics.KnowledgeItems))

	sb.WriteString("\n## Instructions\nRespond to the user's message. Be helpful and specific. Use the live system state above to inform your answers. Format your response as plain text with markdown — it will be rendered in a chat panel. Keep responses concise (2-5 sentences for simple queries, more for status reports).\n")

	return sb.String()
}
