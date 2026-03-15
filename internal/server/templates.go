package server

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// DashboardData holds all data needed to render the dashboard
type DashboardData struct {
	Agents     []AgentView
	Tasks      []TaskView
	Metrics    MetricsView
	Alerts     []AlertView
	ChatLog    []ChatEntry
	Images     []ImageEntry
	UpdatedAt  string
	ClientCount int
}

// AgentView is a frontend-ready agent representation
type AgentView struct {
	Name         string
	DisplayName  string
	Status       string // working, idle, blocked, offline
	StatusColor  string // green, gray, red, dark
	StatusEmoji  string
	CurrentTask  string
	LastActivity string
	Capabilities []string
	Type         string // openclaw, claude-code
}

// TaskView is a frontend-ready task representation
type TaskView struct {
	ID       string
	Title    string
	Status   string
	Priority string
	Owner    string
	Repo     string
	Type     string
	StatusColor string
}

// MetricsView holds computed metrics for display
type MetricsView struct {
	TasksCreated   int
	TasksCompleted int
	TasksActive    int
	TasksBlocked   int
	AgentSessions  int
	KnowledgeItems int
}

// AlertView is a frontend-ready alert
type AlertView struct {
	Severity string
	Message  string
	TaskID   string
}

// ChatEntry represents a chat.md entry
type ChatEntry struct {
	Time    string
	From    string
	To      string
	Message string
}

// ImageEntry represents a generated image
type ImageEntry struct {
	URL     string
	Caption string
	Time    string
}

// statusColor returns the CSS color class for a task status
func statusColor(status models.TaskStatus) string {
	switch status {
	case models.TaskStatusInProgress:
		return "blue"
	case models.TaskStatusBlocked:
		return "red"
	case models.TaskStatusReview:
		return "purple"
	case models.TaskStatusDone:
		return "green"
	case models.TaskStatusArchived:
		return "gray"
	default:
		return "slate"
	}
}

// agentStatusEmoji returns an emoji for agent status
func agentStatusEmoji(status string) string {
	switch status {
	case "working":
		return "🟢"
	case "idle":
		return "⚪"
	case "blocked":
		return "🔴"
	default:
		return "⚫"
	}
}

// Templates holds all parsed HTML templates
type Templates struct {
	shell    *template.Template
	agents   *template.Template
	kanban   *template.Template
	metrics  *template.Template
	alerts   *template.Template
	chat     *template.Template
	gallery  *template.Template
}

// NewTemplates parses all dashboard templates
func NewTemplates() *Templates {
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"timeAgo": func(t string) string {
			return t // simplified for now
		},
	}

	return &Templates{
		shell:   template.Must(template.New("shell").Funcs(funcMap).Parse(shellHTML)),
		agents:  template.Must(template.New("agents").Funcs(funcMap).Parse(agentsHTML)),
		kanban:  template.Must(template.New("kanban").Funcs(funcMap).Parse(kanbanHTML)),
		metrics: template.Must(template.New("metrics").Funcs(funcMap).Parse(metricsHTML)),
		alerts:  template.Must(template.New("alerts").Funcs(funcMap).Parse(alertsHTML)),
		chat:    template.Must(template.New("chat").Funcs(funcMap).Parse(chatHTML)),
		gallery: template.Must(template.New("gallery").Funcs(funcMap).Parse(galleryHTML)),
	}
}

// RenderShell renders the base HTML shell (served once on page load)
func (t *Templates) RenderShell(data DashboardData) (string, error) {
	return render(t.shell, data)
}

// RenderAgents renders the agents section as an OOB swap fragment
func (t *Templates) RenderAgents(agents []AgentView) (string, error) {
	return render(t.agents, agents)
}

// RenderKanban renders the kanban board as an OOB swap fragment
func (t *Templates) RenderKanban(tasks []TaskView) (string, error) {
	return render(t.kanban, tasks)
}

// RenderMetrics renders the metrics panel as an OOB swap fragment
func (t *Templates) RenderMetrics(m MetricsView) (string, error) {
	return render(t.metrics, m)
}

// RenderAlerts renders alerts as an OOB swap fragment
func (t *Templates) RenderAlerts(alerts []AlertView) (string, error) {
	return render(t.alerts, alerts)
}

func render(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template render: %w", err)
	}
	return buf.String(), nil
}

// HTML Templates — these are the "disposable HTML" fragments
// The AI Dev Brain can rewrite these at any time

const shellHTML = `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MyImaginationAI — Agent Command Center</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <script src="https://unpkg.com/htmx-ext-ws@2.0.2/ws.js"></script>
  <script src="https://cdn.tailwindcss.com"></script>
  <script>
    tailwind.config = {
      darkMode: 'class',
      theme: {
        extend: {
          colors: {
            brand: { 50: '#f0f4ff', 500: '#6366f1', 600: '#4f46e5', 900: '#1e1b4b' },
          },
          animation: {
            'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
          }
        }
      }
    }
  </script>
  <style>
    body { background: #0f0f1a; color: #e2e8f0; font-family: 'Inter', system-ui, sans-serif; }
    .status-working { color: #22c55e; }
    .status-idle { color: #94a3b8; }
    .status-blocked { color: #ef4444; }
    .status-offline { color: #374151; }
    .agent-card { transition: all 0.3s ease; }
    .agent-card:hover { transform: translateY(-2px); box-shadow: 0 8px 25px rgba(99, 102, 241, 0.15); }
    @keyframes glow { 0%, 100% { box-shadow: 0 0 5px rgba(99, 102, 241, 0.3); } 50% { box-shadow: 0 0 20px rgba(99, 102, 241, 0.6); } }
    .glow { animation: glow 2s ease-in-out infinite; }
  </style>
</head>
<body hx-ext="ws" ws-connect="/ws" class="min-h-screen">

  <!-- Header -->
  <header class="border-b border-slate-800 px-6 py-3 flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div class="w-8 h-8 rounded-lg bg-brand-600 flex items-center justify-center text-white font-bold text-sm glow">AI</div>
      <h1 class="text-lg font-semibold text-white">MyImaginationAI <span class="text-slate-500 font-normal">— Agent Command Center</span></h1>
    </div>
    <div class="flex items-center gap-4 text-sm text-slate-400">
      <span id="client-count">{{.ClientCount}} viewers</span>
      <span id="updated-at">{{.UpdatedAt}}</span>
    </div>
  </header>

  <!-- Agent Grid -->
  <div id="agents" class="px-6 py-4">
    {{range .Agents}}
    <span class="inline-block px-3 py-1 rounded-full text-sm bg-slate-800 mr-2 mb-2">
      {{.StatusEmoji}} {{.DisplayName}}
    </span>
    {{end}}
  </div>

  <!-- Main Grid -->
  <div class="grid grid-cols-12 gap-4 px-6 pb-6" style="height: calc(100vh - 160px);">

    <!-- Kanban Board -->
    <div id="kanban" class="col-span-6 bg-slate-900/50 rounded-xl border border-slate-800 p-4 overflow-auto">
      <h2 class="text-sm font-semibold text-slate-300 mb-3">TASK BOARD</h2>
      <p class="text-slate-500 text-sm">Loading tasks...</p>
    </div>

    <!-- Metrics + Alerts -->
    <div class="col-span-3 flex flex-col gap-4">
      <div id="metrics" class="bg-slate-900/50 rounded-xl border border-slate-800 p-4 flex-1">
        <h2 class="text-sm font-semibold text-slate-300 mb-3">METRICS</h2>
        <p class="text-slate-500 text-sm">Loading metrics...</p>
      </div>
      <div id="alerts" class="bg-slate-900/50 rounded-xl border border-slate-800 p-4 flex-1">
        <h2 class="text-sm font-semibold text-slate-300 mb-3">ALERTS</h2>
        <p class="text-slate-500 text-sm">No alerts</p>
      </div>
    </div>

    <!-- Chat -->
    <div class="col-span-3 bg-slate-900/50 rounded-xl border border-slate-800 p-4 flex flex-col">
      <h2 class="text-sm font-semibold text-slate-300 mb-3">ORCHESTRATOR CHAT</h2>
      <div id="chat" class="flex-1 overflow-auto text-sm space-y-2">
        <p class="text-slate-500">ADB orchestrator starting...</p>
      </div>
      <form hx-post="/chat" hx-swap="none" class="mt-3 flex gap-2">
        <input name="message" type="text" placeholder="Message ADB..."
          class="flex-1 bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-white placeholder-slate-500 focus:outline-none focus:border-brand-500" />
        <button type="submit" class="bg-brand-600 hover:bg-brand-500 text-white px-4 py-2 rounded-lg text-sm font-medium">Send</button>
      </form>
    </div>

  </div>

  <!-- Gallery strip -->
  <div id="gallery" class="px-6 pb-4">
  </div>

</body>
</html>`

const agentsHTML = `<div id="agents" hx-swap-oob="innerHTML" class="px-6 py-4">
  <div class="flex flex-wrap gap-2 mb-3">
    {{range .}}{{if eq .Type "openclaw"}}
    <div class="agent-card inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-slate-800/80 border border-slate-700 hover:border-brand-500">
      <span class="text-lg">{{.StatusEmoji}}</span>
      <div>
        <span class="text-sm font-medium text-white">{{.DisplayName}}</span>
        {{if .CurrentTask}}<span class="text-xs text-slate-400 ml-2">{{.CurrentTask}}</span>{{end}}
      </div>
    </div>
    {{end}}{{end}}
  </div>
  <div class="flex flex-wrap gap-2">
    {{range .}}{{if ne .Type "openclaw"}}
    <div class="agent-card inline-flex items-center gap-2 px-4 py-2 rounded-xl {{if eq .Status "working"}}bg-green-900/30 border border-green-700 animate-pulse{{else}}bg-blue-900/30 border border-blue-700{{end}}">
      <span class="text-lg">{{.StatusEmoji}}</span>
      <div>
        <span class="text-sm font-medium text-white">{{.DisplayName}}</span>
        {{if .CurrentTask}}<span class="text-xs text-emerald-400 ml-2">{{.CurrentTask}}</span>{{end}}
      </div>
    </div>
    {{end}}{{end}}
  </div>
</div>`

const kanbanHTML = `<div id="kanban" hx-swap-oob="innerHTML" class="col-span-6 bg-slate-900/50 rounded-xl border border-slate-800 p-4 overflow-auto">
  <h2 class="text-sm font-semibold text-slate-300 mb-3">TASK BOARD</h2>
  <div class="grid grid-cols-4 gap-3">
    <div>
      <h3 class="text-xs font-semibold text-slate-500 uppercase mb-2">Backlog</h3>
      {{range .}}{{if eq .Status "backlog"}}
      <div class="bg-slate-800 rounded-lg p-2 mb-2 border-l-2 border-slate-500">
        <div class="text-xs font-mono text-slate-400">{{.ID}}</div>
        <div class="text-sm text-white truncate">{{.Title}}</div>
        <div class="text-xs text-slate-500">{{.Priority}}</div>
      </div>
      {{end}}{{end}}
    </div>
    <div>
      <h3 class="text-xs font-semibold text-blue-400 uppercase mb-2">In Progress</h3>
      {{range .}}{{if eq .Status "in_progress"}}
      <div class="bg-slate-800 rounded-lg p-2 mb-2 border-l-2 border-blue-500">
        <div class="text-xs font-mono text-slate-400">{{.ID}}</div>
        <div class="text-sm text-white truncate">{{.Title}}</div>
        <div class="text-xs text-slate-500">{{.Priority}}</div>
      </div>
      {{end}}{{end}}
    </div>
    <div>
      <h3 class="text-xs font-semibold text-red-400 uppercase mb-2">Blocked</h3>
      {{range .}}{{if eq .Status "blocked"}}
      <div class="bg-slate-800 rounded-lg p-2 mb-2 border-l-2 border-red-500">
        <div class="text-xs font-mono text-slate-400">{{.ID}}</div>
        <div class="text-sm text-white truncate">{{.Title}}</div>
        <div class="text-xs text-slate-500">{{.Priority}}</div>
      </div>
      {{end}}{{end}}
    </div>
    <div>
      <h3 class="text-xs font-semibold text-purple-400 uppercase mb-2">Review</h3>
      {{range .}}{{if eq .Status "review"}}
      <div class="bg-slate-800 rounded-lg p-2 mb-2 border-l-2 border-purple-500">
        <div class="text-xs font-mono text-slate-400">{{.ID}}</div>
        <div class="text-sm text-white truncate">{{.Title}}</div>
        <div class="text-xs text-slate-500">{{.Priority}}</div>
      </div>
      {{end}}{{end}}
    </div>
  </div>
</div>`

const metricsHTML = `<div id="metrics" hx-swap-oob="innerHTML" class="bg-slate-900/50 rounded-xl border border-slate-800 p-4 flex-1">
  <h2 class="text-sm font-semibold text-slate-300 mb-3">METRICS</h2>
  <div class="space-y-2 text-sm">
    <div class="flex justify-between"><span class="text-slate-400">Created</span><span class="text-white font-mono">{{.TasksCreated}}</span></div>
    <div class="flex justify-between"><span class="text-slate-400">Completed</span><span class="text-green-400 font-mono">{{.TasksCompleted}}</span></div>
    <div class="flex justify-between"><span class="text-slate-400">Active</span><span class="text-blue-400 font-mono">{{.TasksActive}}</span></div>
    <div class="flex justify-between"><span class="text-slate-400">Blocked</span><span class="text-red-400 font-mono">{{.TasksBlocked}}</span></div>
    <div class="flex justify-between"><span class="text-slate-400">Agent Sessions</span><span class="text-white font-mono">{{.AgentSessions}}</span></div>
    <div class="flex justify-between"><span class="text-slate-400">Knowledge Items</span><span class="text-white font-mono">{{.KnowledgeItems}}</span></div>
  </div>
</div>`

const alertsHTML = `<div id="alerts" hx-swap-oob="innerHTML" class="bg-slate-900/50 rounded-xl border border-slate-800 p-4 flex-1">
  <h2 class="text-sm font-semibold text-slate-300 mb-3">ALERTS</h2>
  {{if .}}
  <div class="space-y-2">
    {{range .}}
    <div class="text-sm px-3 py-2 rounded-lg {{if eq .Severity "high"}}bg-red-900/30 text-red-300{{else if eq .Severity "medium"}}bg-yellow-900/30 text-yellow-300{{else}}bg-slate-800 text-slate-300{{end}}">
      {{.Message}}
    </div>
    {{end}}
  </div>
  {{else}}
  <p class="text-green-400 text-sm">✓ No active alerts</p>
  {{end}}
</div>`

const chatHTML = `<div id="chat" hx-swap-oob="beforeend">
  <div class="px-2 py-1 rounded {{if eq .From "ADB"}}bg-brand-900/30{{else}}bg-slate-800{{end}}">
    <span class="text-xs text-slate-500">{{.Time}}</span>
    <span class="text-xs font-semibold {{if eq .From "ADB"}}text-brand-400{{else}}text-slate-300{{end}}">{{.From}} → {{.To}}</span>
    <p class="text-sm text-slate-200">{{.Message}}</p>
  </div>
</div>`

const galleryHTML = `<div id="gallery" hx-swap-oob="innerHTML" class="px-6 pb-4 flex gap-3 overflow-x-auto">
  {{range .}}
  <div class="flex-shrink-0">
    <img src="{{.URL}}" alt="{{.Caption}}" class="h-24 rounded-lg border border-slate-700" />
    <p class="text-xs text-slate-500 mt-1 truncate w-32">{{.Caption}}</p>
  </div>
  {{end}}
</div>`
