package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	internal "github.com/valter-silva-au/ai-dev-brain/internal"
	"github.com/valter-silva-au/ai-dev-brain/internal/hive"
	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// Server is the ADB web dashboard server
type Server struct {
	app        *internal.App
	hub        *WSHub
	templates  *Templates
	httpServer *http.Server

	// Hive Mind components
	agentReg    hive.AgentRegistry
	projectReg  hive.ProjectRegistry
	knowledgeAgg hive.KnowledgeAggregator
	messageBus  hive.MessageBus
}

// NewServer creates a new dashboard server
func NewServer(app *internal.App, agentReg hive.AgentRegistry, projectReg hive.ProjectRegistry, knowledgeAgg hive.KnowledgeAggregator, messageBus hive.MessageBus) *Server {
	return &Server{
		app:          app,
		hub:          NewWSHub(),
		templates:    NewTemplates(),
		agentReg:     agentReg,
		projectReg:   projectReg,
		knowledgeAgg: knowledgeAgg,
		messageBus:   messageBus,
	}
}

// Start starts the HTTP server and background state broadcaster
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Base shell (served once on page load)
	mux.HandleFunc("GET /", s.handleIndex)

	// WebSocket for live HTML fragment streaming
	mux.HandleFunc("GET /ws", s.hub.HandleWS)

	// Chat endpoint
	mux.HandleFunc("POST /chat", s.handleChat)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Create images directory
	imageDir := filepath.Join(s.app.BasePath, "images", "generated")
	os.MkdirAll(imageDir, 0o755)

	// Serve generated images
	mux.Handle("GET /images/", http.StripPrefix("/images/", http.FileServer(http.Dir(filepath.Join(s.app.BasePath, "images")))))

	// Start background loops
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.broadcastLoop(ctx)

	// Start thinking loop (text summary every 5s)
	thinking := NewThinkingLoop(s.hub, s.gatherAgents, s.gatherTasks, s.gatherMetrics)
	go thinking.Run(ctx)

	// Start image generation loop (continuous, ~70s per image)
	imageGen := NewImageGenerator(s.hub, imageDir, s.templates, func() string {
		agents := s.gatherAgents()
		tasks := s.gatherTasks()
		var parts []string
		for _, a := range agents {
			if a.Status == "working" {
				parts = append(parts, fmt.Sprintf("%s working on %s", a.DisplayName, a.CurrentTask))
			}
		}
		for _, t := range tasks {
			if t.Status == "in_progress" {
				parts = append(parts, fmt.Sprintf("task %s %s", t.ID, t.Title))
			}
		}
		return strings.Join(parts, ", ")
	})
	go imageGen.Run(ctx)

	log.Printf("ADB Dashboard serving at http://%s", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// handleIndex serves the base HTML shell with initial state
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := s.gatherDashboardData()
	html, err := s.templates.RenderShell(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("render error: %v", err), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleChat receives a chat message from the frontend
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}

	message := strings.TrimSpace(r.FormValue("message"))
	if message == "" {
		w.WriteHeader(204)
		return
	}

	// Broadcast the user's message to the dashboard
	entry := ChatEntry{
		Time:    time.Now().UTC().Format("15:04"),
		From:    "Human",
		To:      "ADB",
		Message: message,
	}
	html, err := render(s.templates.chat, entry)
	if err == nil {
		s.hub.Broadcast(html)
	}

	// ADB orchestrator responds based on the message
	response := s.processChat(message)
	ack := ChatEntry{
		Time:    time.Now().UTC().Format("15:04"),
		From:    "ADB",
		To:      "Human",
		Message: response,
	}
	html, err = render(s.templates.chat, ack)
	if err == nil {
		s.hub.Broadcast(html)
	}

	w.WriteHeader(204)
}

// processChat generates an intelligent response based on the message
func (s *Server) processChat(message string) string {
	lower := strings.ToLower(message)

	// Status report
	if strings.Contains(lower, "status") || strings.Contains(lower, "report") || strings.Contains(lower, "what") {
		return s.generateStatusReport()
	}

	// Agent-specific queries
	if strings.Contains(lower, "who") && (strings.Contains(lower, "working") || strings.Contains(lower, "active") || strings.Contains(lower, "running")) {
		return s.generateActiveAgentsReport()
	}

	// Task queries
	if strings.Contains(lower, "task") || strings.Contains(lower, "blocked") || strings.Contains(lower, "backlog") {
		return s.generateTaskReport()
	}

	// Help
	if strings.Contains(lower, "help") || lower == "?" {
		return "I can answer: <b>status report</b>, <b>who is working</b>, <b>tasks</b>, <b>blocked</b>, <b>backlog</b>, <b>metrics</b>. Just ask!"
	}

	// Metrics
	if strings.Contains(lower, "metric") {
		m := s.gatherMetrics()
		return fmt.Sprintf("📊 <b>Metrics:</b> %d tasks created, %d completed, %d active, %d blocked, %d agent sessions, %d knowledge items.",
			m.TasksCreated, m.TasksCompleted, m.TasksActive, m.TasksBlocked, m.AgentSessions, m.KnowledgeItems)
	}

	// Default — friendly response
	return fmt.Sprintf("Hey! I heard you say %q. Try asking for a <b>status report</b>, <b>who is working</b>, or about <b>tasks</b>. I'm here to help! 🧠", message)
}

// generateStatusReport creates a comprehensive status summary
func (s *Server) generateStatusReport() string {
	agents := s.gatherAgents()
	tasks := s.gatherTasks()
	metrics := s.gatherMetrics()
	procs := ScanLiveProcesses()
	summary := SummarizeProcesses(procs)

	var lines []string
	lines = append(lines, "📋 <b>Status Report</b>")
	lines = append(lines, "")

	// Live processes
	total := len(summary.ClaudeCodeSessions) + len(summary.AgentLoopsRuns)
	lines = append(lines, fmt.Sprintf("🔥 <b>%d active AI sessions</b>", total))
	for _, p := range summary.ClaudeCodeSessions {
		lines = append(lines, fmt.Sprintf("&nbsp;&nbsp;🟢 Claude Code → <b>%s</b>", p.Project))
	}
	for _, p := range summary.AgentLoopsRuns {
		lines = append(lines, fmt.Sprintf("&nbsp;&nbsp;🔵 Agent Loops → <b>%s</b>", p.Project))
	}
	if summary.OpenClawGateway != nil {
		lines = append(lines, "&nbsp;&nbsp;⚡ OpenClaw Gateway running")
	}

	// OpenClaw bots
	var idleBots []string
	for _, a := range agents {
		if a.Type == "openclaw" {
			idleBots = append(idleBots, a.DisplayName)
		}
	}
	if len(idleBots) > 0 {
		lines = append(lines, fmt.Sprintf("&nbsp;&nbsp;📡 %d OpenClaw bots: %s", len(idleBots), strings.Join(idleBots, ", ")))
	}

	// Tasks
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
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("📋 <b>Tasks:</b> %d in progress, %d blocked, %d in review, %d in backlog", inProgress, blocked, review, backlog))

	// Metrics
	lines = append(lines, fmt.Sprintf("📊 <b>Lifetime:</b> %d created, %d completed", metrics.TasksCreated, metrics.TasksCompleted))

	return strings.Join(lines, "<br/>")
}

// generateActiveAgentsReport lists who is currently working
func (s *Server) generateActiveAgentsReport() string {
	procs := ScanLiveProcesses()
	summary := SummarizeProcesses(procs)

	if summary.TotalActive == 0 {
		return "💤 Nobody is working right now. All quiet."
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("🔥 <b>%d active sessions right now:</b>", summary.TotalActive))
	for _, p := range summary.ClaudeCodeSessions {
		lines = append(lines, fmt.Sprintf("&nbsp;&nbsp;🟢 Claude Code working on <b>%s</b> (PID %d)", p.Project, p.PID))
	}
	for _, p := range summary.AgentLoopsRuns {
		lines = append(lines, fmt.Sprintf("&nbsp;&nbsp;🔵 Agent Loops building <b>%s</b> (PID %d)", p.Project, p.PID))
	}
	return strings.Join(lines, "<br/>")
}

// generateTaskReport lists tasks by status
func (s *Server) generateTaskReport() string {
	tasks := s.gatherTasks()
	if len(tasks) == 0 {
		return "No active tasks found."
	}

	var lines []string
	statuses := []struct{ name, status, emoji string }{
		{"In Progress", "in_progress", "🔵"},
		{"Blocked", "blocked", "🔴"},
		{"Review", "review", "🟣"},
		{"Backlog", "backlog", "⚪"},
	}

	for _, s := range statuses {
		var matching []string
		for _, t := range tasks {
			if t.Status == s.status {
				matching = append(matching, fmt.Sprintf("<b>%s</b> %s [%s]", t.ID, t.Title, t.Priority))
			}
		}
		if len(matching) > 0 {
			lines = append(lines, fmt.Sprintf("%s <b>%s (%d):</b>", s.emoji, s.name, len(matching)))
			for _, m := range matching {
				lines = append(lines, "&nbsp;&nbsp;• "+m)
			}
		}
	}

	return strings.Join(lines, "<br/>")
}

// broadcastLoop periodically gathers state and pushes HTML fragments to all clients
func (s *Server) broadcastLoop(ctx context.Context) {
	// Initial broadcast after short delay (let clients connect)
	time.Sleep(500 * time.Millisecond)
	s.broadcastState()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.broadcastState()
		}
	}
}

// broadcastState gathers all state and pushes updated HTML fragments
func (s *Server) broadcastState() {
	if s.hub.Count() == 0 {
		return // No clients, skip work
	}

	// Render and broadcast each section independently
	agents := s.gatherAgents()

	// Separate OpenClaw bots from live processes for display
	var openclawBots, liveProcesses []AgentView
	for _, a := range agents {
		if a.Type == "openclaw" {
			openclawBots = append(openclawBots, a)
		} else {
			liveProcesses = append(liveProcesses, a)
		}
	}

	if html, err := s.templates.RenderAgents(agents); err == nil {
		s.hub.Broadcast(html)
	}

	tasks := s.gatherTasks()
	if html, err := s.templates.RenderKanban(tasks); err == nil {
		s.hub.Broadcast(html)
	}

	metrics := s.gatherMetrics()
	if html, err := s.templates.RenderMetrics(metrics); err == nil {
		s.hub.Broadcast(html)
	}

	alerts := s.gatherAlerts()
	if html, err := s.templates.RenderAlerts(alerts); err == nil {
		s.hub.Broadcast(html)
	}
}

// gatherDashboardData collects all state for the initial page render
func (s *Server) gatherDashboardData() DashboardData {
	return DashboardData{
		Agents:    s.gatherAgents(),
		Tasks:     s.gatherTasks(),
		Metrics:   s.gatherMetrics(),
		Alerts:    s.gatherAlerts(),
		UpdatedAt: time.Now().UTC().Format("15:04:05"),
		ClientCount: s.hub.Count(),
	}
}

// gatherAgents combines registry agents with live process detection
func (s *Server) gatherAgents() []AgentView {
	// Start with OpenClaw bots from registry
	var views []AgentView

	if s.agentReg != nil {
		_ = s.agentReg.Load()
		agents, _ := s.agentReg.List(models.AgentFilter{})
		for _, a := range agents {
			status := string(a.Status)
			if status == "" {
				status = "idle"
			}
			views = append(views, AgentView{
				Name:         a.Name,
				DisplayName:  a.Name,
				Status:       status,
				StatusEmoji:  agentStatusEmoji(status),
				CurrentTask:  a.ActiveTask,
				Capabilities: a.Capabilities,
				Type:         string(a.Type),
			})
		}
	}

	if len(views) == 0 {
		names := []string{"Prime", "Nexus", "A&R-X", "Job Hunter", "Luna", "Nina", "Vanguard", "PermitAI"}
		for _, name := range names {
			views = append(views, AgentView{
				Name:        strings.ToLower(strings.ReplaceAll(name, " ", "-")),
				DisplayName: name,
				Status:      "idle",
				StatusEmoji: "⚪",
				Type:        "openclaw",
			})
		}
	}

	// Scan live processes and overlay real-time status
	procs := ScanLiveProcesses()
	summary := SummarizeProcesses(procs)

	// Add Claude Code sessions as active agents
	for _, p := range summary.ClaudeCodeSessions {
		views = append(views, AgentView{
			Name:        fmt.Sprintf("claude-%d", p.PID),
			DisplayName: fmt.Sprintf("Claude Code → %s", p.Project),
			Status:      "working",
			StatusEmoji: "🟢",
			CurrentTask: p.Project,
			Type:        "claude-code",
		})
	}

	// Add Agent Loops runs
	for _, p := range summary.AgentLoopsRuns {
		views = append(views, AgentView{
			Name:        fmt.Sprintf("agent-loops-%d", p.PID),
			DisplayName: fmt.Sprintf("Agent Loops → %s", p.Project),
			Status:      "working",
			StatusEmoji: "🔵",
			CurrentTask: p.Project,
			Type:        "agent-loops",
		})
	}

	// Mark OpenClaw gateway status
	if summary.OpenClawGateway != nil {
		for i, v := range views {
			if v.Type == "openclaw" && v.DisplayName == "Prime" {
				views[i].CurrentTask = "Gateway running"
			}
		}
	}

	// Add ADB MCP server
	for _, p := range summary.ADBServices {
		views = append(views, AgentView{
			Name:        fmt.Sprintf("adb-%d", p.PID),
			DisplayName: "ADB MCP Server",
			Status:      "working",
			StatusEmoji: "🧠",
			CurrentTask: p.Project,
			Type:        "adb-mcp",
		})
	}

	return views
}

// gatherTasks reads task state from the backlog
func (s *Server) gatherTasks() []TaskView {
	backlog, err := s.app.BacklogManager.Load()
	if err != nil {
		log.Printf("backlog load: %v", err)
		return nil
	}

	var views []TaskView
	for _, t := range backlog.Tasks {
		if t.Status == models.TaskStatusArchived {
			continue // Skip archived tasks
		}
		views = append(views, TaskView{
			ID:          t.ID,
			Title:       t.Title,
			Status:      string(t.Status),
			Priority:    string(t.Priority),
			Owner:       t.Owner,
			Repo:        t.Repo,
			Type:        string(t.Type),
			StatusColor: statusColor(t.Status),
		})
	}
	return views
}

// gatherMetrics computes metrics from the event log
func (s *Server) gatherMetrics() MetricsView {
	if s.app.MetricsCalculator == nil {
		return MetricsView{}
	}

	m, err := s.app.MetricsCalculator.ComputeMetrics()
	if err != nil {
		log.Printf("metrics calculate: %v", err)
		return MetricsView{}
	}

	return MetricsView{
		TasksCreated:   m.TasksCreated,
		TasksCompleted: m.TasksCompleted,
		TasksActive:    m.TasksByStatus["in_progress"],
		TasksBlocked:   m.TasksByStatus["blocked"],
		AgentSessions:  m.AgentSessions,
		KnowledgeItems: m.KnowledgeExtracts,
	}
}

// gatherAlerts evaluates alert conditions
func (s *Server) gatherAlerts() []AlertView {
	if s.app.AlertEvaluator == nil {
		return nil
	}

	alerts, err := s.app.AlertEvaluator.EvaluateAll()
	if err != nil {
		log.Printf("alert evaluate: %v", err)
		return nil
	}

	var views []AlertView
	for _, a := range alerts {
		views = append(views, AlertView{
			Severity: string(a.Severity),
			Message:  a.Message,
			TaskID:   a.TaskID,
		})
	}
	return views
}
