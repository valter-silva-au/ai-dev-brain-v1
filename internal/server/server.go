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

	// TODO: Phase 2 — orchestrator processes the message and responds
	// For now, echo back an acknowledgement
	ack := ChatEntry{
		Time:    time.Now().UTC().Format("15:04"),
		From:    "ADB",
		To:      "Human",
		Message: fmt.Sprintf("Received: %q — orchestrator not yet active.", message),
	}
	html, err = render(s.templates.chat, ack)
	if err == nil {
		s.hub.Broadcast(html)
	}

	w.WriteHeader(204)
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
