# Hive Mind Architecture Design

**Status:** Draft
**Date:** 2026-03-14
**Source:** feat/ai-dev-brain/claude-code-2-1-50

## Executive Summary

The Hive Mind feature transforms AI Dev Brain (adb) from a single-workspace task management tool into the central nervous system connecting 8 OpenClaw AI bots and 30+ code repositories. This design enables unified knowledge access, project-to-project awareness, agent-to-agent communication, and cross-repository intelligence—all while maintaining adb's file-based, low-maintenance philosophy suitable for a solo developer.

## 1. Architecture Overview

### High-Level System Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Hive Mind Layer                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐            │
│  │   Project    │   │    Agent     │   │  Knowledge   │            │
│  │   Registry   │   │   Registry   │   │  Aggregator  │            │
│  │              │   │              │   │              │            │
│  │ (projects/   │   │ (agents/     │   │ (hive-mind/  │            │
│  │  index.yaml) │   │  index.yaml) │   │  index.yaml) │            │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘            │
│         │                  │                   │                    │
│         └──────────────────┴───────────────────┘                    │
│                            │                                        │
│  ┌─────────────────────────┴────────────────────────────┐           │
│  │          Message Bus (File-Based Pub/Sub)            │           │
│  │                                                       │           │
│  │  channels/                                            │           │
│  │    inbox/   - Incoming messages from agents/projects │           │
│  │    outbox/  - Outgoing messages to agents/projects   │           │
│  │    archive/ - Processed messages                     │           │
│  └───────────────────────────────────────────────────────┘           │
│                                                                       │
└───────────────────────────────────┬───────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────────┐
        │                           │                               │
┌───────▼────────┐     ┌────────────▼──────────┐     ┌─────────────▼──────┐
│ ADB MCP Server │     │   OpenClaw Bots       │     │   Claude Code      │
│                │     │   (8 agents)          │     │   Sessions         │
│ adb mcp serve  │     │                       │     │                    │
│                │     │ ~/.openclaw/          │     │ In worktrees       │
│ Tools:         │     │   workspace-{name}/   │     │                    │
│ - query_hive   │     │   agents/{name}/      │     │ .mcp.json ->       │
│ - get_project  │     │   memory/{name}.sqlite│     │   adb-hive         │
│ - get_agent    │     │   channels/           │     │                    │
│ - search_multi │     └───────────────────────┘     └────────────────────┘
│ - send_message │
└────────────────┘

         Data Flow:
         1. Knowledge extraction feeds → Knowledge Aggregator
         2. Project metadata feeds → Project Registry
         3. Agent status feeds → Agent Registry
         4. Queries flow through MCP server → Registries
         5. Messages flow through file-based pub/sub
```

### Key Design Principles

1. **File-based everything** - No always-running services, consistent with adb's current architecture
2. **Incremental aggregation** - Knowledge/project/agent data syncs on-demand or via hooks
3. **Interface-based boundaries** - All cross-package communication through interfaces and adapters
4. **Graceful degradation** - Hive Mind features non-fatal if unavailable
5. **Security by isolation** - Agents see metadata, not credentials or sensitive data
6. **Local-first** - All data lives in the life/ repo, the single source of truth

## 2. Component Design

### 2.1 Project Registry (`internal/hive/projectregistry.go`)

**Purpose:** Central catalog of all tracked repositories and their metadata.

**Package Structure:**
```
internal/
  hive/
    projectregistry.go       # ProjectRegistry interface and implementation
    projectregistry_test.go
    projectregistry_property_test.go
```

**Interface Definition:**
```go
package hive

import "github.com/valter-silva-au/ai-dev-brain/pkg/models"

// ProjectRegistry manages the catalog of all tracked projects.
type ProjectRegistry interface {
    // Register adds or updates a project in the registry.
    Register(project models.Project) error

    // Get retrieves a project by its repo path or name.
    Get(repoPathOrName string) (*models.Project, error)

    // List returns all registered projects, optionally filtered.
    List(filter models.ProjectFilter) ([]models.Project, error)

    // Query searches projects by tags, tech stack, or keywords.
    Query(query string) ([]models.Project, error)

    // UpdateStatus updates a project's status without full reload.
    UpdateStatus(repoPathOrName string, status models.ProjectStatus) error

    // Load reads the project index from disk.
    Load() error

    // Save persists the project index to disk.
    Save() error
}
```

**Storage Format:** `projects/index.yaml`
```yaml
version: "1.0"
projects:
  - name: "ai-dev-brain-v1"
    repo_path: "/home/valter/Code/repos/github.com/valter-silva-au/ai-dev-brain-v1"
    purpose: "AI-powered task management and knowledge accumulation CLI"
    tech_stack: ["go", "cobra", "yaml", "markdown"]
    status: "active"
    last_updated: "2026-03-14T10:30:00Z"
    tags: ["cli", "ai", "task-management"]
    default_ai: "kiro"
    branch_pattern: "feat/bug/spike/refactor"
    related_projects: []
    knowledge_entry_count: 45
    decision_count: 12
    active_task_count: 3
  - name: "openclaw"
    repo_path: "/home/valter/.openclaw"
    purpose: "Multi-agent AI bot framework with browser automation"
    tech_stack: ["python", "bedrock", "telegram", "sqlite"]
    status: "active"
    last_updated: "2026-03-14T09:15:00Z"
    tags: ["ai", "agents", "automation"]
    related_projects: ["ai-dev-brain-v1"]
    agent_count: 8
```

**Integration Points:**
- **CLI:** `adb hive project register <path>`, `adb hive project list`
- **Hook:** `adb-hook-post-tool-use.sh` auto-registers projects when first encountered
- **Adapter in app.go:**
```go
type projectRegistryAdapter struct {
    reg hive.ProjectRegistry
}

func (a *projectRegistryAdapter) GetProjectsForContext(tags []string) ([]models.Project, error) {
    // Implementation
}
```

### 2.2 Agent Registry (`internal/hive/agentregistry.go`)

**Purpose:** Central catalog of all AI agents (OpenClaw bots and Claude Code personas) and their capabilities.

**Interface Definition:**
```go
package hive

// AgentRegistry manages the catalog of all AI agents.
type AgentRegistry interface {
    // Register adds or updates an agent in the registry.
    Register(agent models.Agent) error

    // Get retrieves an agent by name.
    Get(name string) (*models.Agent, error)

    // List returns all registered agents, optionally filtered.
    List(filter models.AgentFilter) ([]models.Agent, error)

    // Query searches agents by capabilities, role, or keywords.
    Query(query string) ([]models.Agent, error)

    // UpdateStatus updates an agent's current status (idle, busy, offline).
    UpdateStatus(name string, status models.AgentStatus) error

    // Load reads the agent index from disk.
    Load() error

    // Save persists the agent index to disk.
    Save() error
}
```

**Storage Format:** `agents/index.yaml`
```yaml
version: "1.0"
agents:
  - name: "team-lead"
    type: "claude-code"
    model: "opus"
    capabilities: ["orchestration", "task-breakdown", "team-coordination"]
    role: "Multi-agent team orchestration and BMAD workflow routing"
    status: "idle"
    last_seen: "2026-03-14T10:00:00Z"
    home_project: "ai-dev-brain-v1"
    active_task: ""
    session_count: 45
  - name: "code-panda"
    type: "openclaw"
    model: "bedrock-opus-4.6"
    capabilities: ["coding", "debugging", "git", "browser"]
    role: "Software development with browser research"
    status: "busy"
    last_seen: "2026-03-14T10:25:00Z"
    home_project: ""
    active_task: "researching-react-patterns"
    memory_path: "/home/valter/.openclaw/memory/code-panda.sqlite"
    workspace_path: "/home/valter/.openclaw/workspace-code-panda"
  - name: "research-owl"
    type: "openclaw"
    model: "bedrock-opus-4.6"
    capabilities: ["research", "documentation", "summarization", "web-scraping"]
    role: "Deep research and information synthesis"
    status: "idle"
    last_seen: "2026-03-14T09:50:00Z"
    home_project: ""
```

**OpenClaw Integration:**
- **Discovery:** Parse `~/.openclaw/openclaw.json` to enumerate agents
- **SOUL.md parsing:** Extract capabilities and role from `agents/{name}/agent/SOUL.md`
- **Memory inspection:** Query SQLite memory for recent activity
- **Status tracking:** Monitor workspace activity via filesystem timestamps

**Integration Points:**
- **CLI:** `adb hive agent list`, `adb hive agent status`
- **Hook:** `adb-hook-session-end.sh` updates agent status after sessions
- **Adapter in app.go:**
```go
type agentRegistryAdapter struct {
    reg hive.AgentRegistry
}
```

### 2.3 Knowledge Aggregator (`internal/hive/knowledgeaggregator.go`)

**Purpose:** Unified query interface across all project knowledge stores (ADRs, decisions, learnings, wiki entries).

**Interface Definition:**
```go
package hive

// KnowledgeAggregator provides unified knowledge queries across all projects.
type KnowledgeAggregator interface {
    // SearchAcrossProjects performs a full-text search across all project knowledge stores.
    SearchAcrossProjects(query string, opts SearchOptions) ([]models.HiveKnowledgeResult, error)

    // GetDecisionsForTopic returns all decisions related to a topic from all projects.
    GetDecisionsForTopic(topic string) ([]models.HiveKnowledgeResult, error)

    // GetPatternsByTag returns patterns/learnings tagged with specific tags.
    GetPatternsByTag(tags []string) ([]models.HiveKnowledgeResult, error)

    // GetRelatedKnowledge finds knowledge related to a specific task across all repos.
    GetRelatedKnowledge(task models.Task) ([]models.HiveKnowledgeResult, error)

    // Index rebuilds the hive-level knowledge index from all project knowledge stores.
    Index() error

    // Load reads the hive knowledge index from disk.
    Load() error

    // Save persists the hive knowledge index to disk.
    Save() error
}

type SearchOptions struct {
    ProjectFilter []string // Limit to specific projects
    TypeFilter    []models.KnowledgeEntryType
    Since         time.Time
    Limit         int
}
```

**Storage Format:** `hive-mind/knowledge/index.yaml`
```yaml
version: "1.0"
indexed_at: "2026-03-14T10:30:00Z"
projects:
  - repo_path: "/home/valter/Code/repos/github.com/valter-silva-au/ai-dev-brain-v1"
    knowledge_path: "docs/knowledge/index.yaml"
    last_indexed: "2026-03-14T10:30:00Z"
    entry_count: 45
    decision_count: 12
    learning_count: 18
    pattern_count: 10
    gotcha_count: 5
entries:
  # Flattened index of all knowledge entries with project references
  - id: "ai-dev-brain-v1::K-00001"
    project: "ai-dev-brain-v1"
    type: "decision"
    topic: "architecture"
    summary: "Use local interfaces to avoid import cycles"
    date: "2026-02-01"
    tags: ["go", "architecture", "patterns"]
  - id: "life::K-00023"
    project: "life"
    type: "learning"
    topic: "knowledge-management"
    summary: "Knowledge extraction from completed tasks improves future task quality"
    date: "2026-03-10"
    tags: ["process", "knowledge"]
```

**Aggregation Strategy:**
1. **On-demand indexing:** `adb hive knowledge index` scans all registered projects
2. **Hook-driven incremental updates:** `adb-hook-task-completed.sh` triggers `adb hive knowledge update` for the current project
3. **Cache with staleness detection:** Index includes `last_indexed` timestamps per project
4. **Conflict-free merging:** Knowledge IDs namespaced by project (`<project>::<id>`)

**Query Execution:**
1. Check hive index first (fast path)
2. If stale (>24h), optionally refresh from source
3. Merge results from multiple projects, deduplicate by content similarity

**Integration Points:**
- **CLI:** `adb hive knowledge search <query>`, `adb hive knowledge index`
- **MCP Server:** Exposed as MCP tools `search_hive_knowledge`, `get_related_knowledge`
- **Adapter in app.go:**
```go
type knowledgeAggregatorAdapter struct {
    agg hive.KnowledgeAggregator
    projReg hive.ProjectRegistry
}
```

### 2.4 Message Bus (`internal/hive/messagebus.go`)

**Purpose:** File-based pub/sub for agent-to-agent and project-to-project communication.

**Interface Definition:**
```go
package hive

// MessageBus provides file-based pub/sub messaging between agents and projects.
type MessageBus interface {
    // Publish sends a message to the bus.
    Publish(msg models.HiveMessage) error

    // Subscribe retrieves pending messages for a recipient.
    Subscribe(recipient string) ([]models.HiveMessage, error)

    // MarkProcessed moves a message to the archive.
    MarkProcessed(messageID string) error

    // GetConversationHistory retrieves messages in a conversation thread.
    GetConversationHistory(conversationID string) ([]models.HiveMessage, error)
}
```

**Storage Format:**
- **Inbox:** `channels/inbox/{recipient}/{message-id}.yaml`
- **Outbox:** `channels/outbox/{sender}/{message-id}.yaml`
- **Archive:** `channels/archive/{year}/{month}/{message-id}.yaml`

**Message Schema:** `pkg/models/hivemessage.go`
```go
type HiveMessage struct {
    ID             string            `yaml:"id"`
    ConversationID string            `yaml:"conversation_id,omitempty"`
    From           string            `yaml:"from"` // Agent or project name
    To             string            `yaml:"to"`   // Agent or project name
    Subject        string            `yaml:"subject"`
    Content        string            `yaml:"content"`
    Type           HiveMessageType   `yaml:"type"`
    Priority       models.Priority   `yaml:"priority"`
    Date           string            `yaml:"date"`
    InReplyTo      string            `yaml:"in_reply_to,omitempty"`
    Tags           []string          `yaml:"tags,omitempty"`
    Metadata       map[string]string `yaml:"metadata,omitempty"`
    Status         HiveMessageStatus `yaml:"status"`
}

type HiveMessageType string
const (
    HiveMessageRequest  HiveMessageType = "request"  // Action request
    HiveMessageResponse HiveMessageType = "response" // Reply to a request
    HiveMessageNotify   HiveMessageType = "notify"   // One-way notification
    HiveMessageQuery    HiveMessageType = "query"    // Knowledge query
)

type HiveMessageStatus string
const (
    HiveMessagePending   HiveMessageStatus = "pending"
    HiveMessageDelivered HiveMessageStatus = "delivered"
    HiveMessageRead      HiveMessageStatus = "read"
    HiveMessageArchived  HiveMessageStatus = "archived"
)
```

**Routing Strategy:**
1. Messages addressed to specific agents go to `channels/inbox/{agent-name}/`
2. Messages addressed to projects go to `channels/inbox/{project-name}/`
3. Broadcast messages go to `channels/inbox/all/`
4. OpenClaw integration: symlink `~/.openclaw/channels/inbox` -> `{life-repo}/channels/inbox/`
5. Claude Code integration: poll inbox at session start via `adb hive messages check`

**Security:**
- Messages contain only references (task IDs, knowledge IDs, project names)
- No credentials, API keys, or PII in messages
- Agents query via MCP for full data after reading message references

**Integration Points:**
- **CLI:** `adb hive message send --to=agent --subject="..." --content="..."`
- **CLI:** `adb hive message list`, `adb hive message archive <id>`
- **Hook:** `adb-hook-session-end.sh` checks inbox and logs unread messages
- **OpenClaw:** Cron job `adb hive messages sync-openclaw` copies messages to/from OpenClaw channels

### 2.5 Hive Context Generator (`internal/hive/contextgenerator.go`)

**Purpose:** Generate project-aware and agent-aware context for AI sessions.

**Interface Definition:**
```go
package hive

// HiveContextGenerator produces enhanced context that includes cross-project knowledge.
type HiveContextGenerator interface {
    // GenerateForTask produces hive-aware context for a specific task.
    GenerateForTask(taskID string) (*models.HiveContext, error)

    // GenerateForAgent produces hive-aware context for an AI agent session.
    GenerateForAgent(agentName string) (*models.HiveContext, error)

    // GenerateForProject produces a project-level hive context summary.
    GenerateForProject(projectName string) (*models.HiveContext, error)
}
```

**Output Schema:** `pkg/models/hivecontext.go`
```go
type HiveContext struct {
    CurrentProject     models.Project
    RelatedProjects    []models.Project
    RelatedKnowledge   []HiveKnowledgeResult
    ActiveAgents       []models.Agent
    RecentMessages     []HiveMessage
    CrossRepoPatterns  []string
    Summary            string // Markdown summary
}
```

**Generation Logic:**
1. Identify current project and task
2. Query project registry for related projects (by tags, tech stack)
3. Query knowledge aggregator for related knowledge from related projects
4. Query agent registry for agents with relevant capabilities
5. Query message bus for recent relevant conversations
6. Synthesize into markdown summary

**Integration Points:**
- **CLI:** `adb hive context generate --for-task=<id>`, `adb hive context generate --for-agent=<name>`
- **MCP Server:** Exposed as MCP resource `hive://context/task/{id}`
- **Hook:** `adb-hook-pre-tool-use.sh` injects hive context into Claude Code sessions

## 3. MCP Server Design (Expanding ADR-0002 Phase 3)

### 3.1 Full Tool Catalog

| Tool | Parameters | Returns | Maps To |
|------|-----------|---------|---------|
| **Task Management (Existing)** |
| `get_task` | `task_id` | Task details | `TaskManager.GetTask()` |
| `list_tasks` | `status?`, `project?` | Task list | `BacklogStore.FilterTasks()` |
| `get_task_context` | `task_id` | Context markdown | `ContextStore.LoadContext()` |
| `update_task_status` | `task_id`, `status` | Success | `TaskManager.UpdateTaskStatus()` |
| `get_backlog` | `project?` | Backlog entries | `BacklogStore.GetAllTasks()` |
| **Hive Mind (New)** |
| `query_hive_knowledge` | `query`, `projects?`, `types?` | Knowledge results | `KnowledgeAggregator.SearchAcrossProjects()` |
| `get_related_knowledge` | `task_id` | Related knowledge | `KnowledgeAggregator.GetRelatedKnowledge()` |
| `list_projects` | `filter?` | Project list | `ProjectRegistry.List()` |
| `get_project` | `name_or_path` | Project details | `ProjectRegistry.Get()` |
| `list_agents` | `filter?` | Agent list | `AgentRegistry.List()` |
| `get_agent` | `name` | Agent details | `AgentRegistry.Get()` |
| `search_agents_by_capability` | `capability` | Agent list | `AgentRegistry.Query()` |
| `hive_message_send` | `to`, `subject`, `content`, `type` | Message ID | `MessageBus.Publish()` |
| `hive_message_list` | `recipient?` | Message list | `MessageBus.Subscribe()` |
| `get_conversation` | `conversation_id` | Message thread | `MessageBus.GetConversationHistory()` |
| `get_hive_context` | `task_id?`, `agent?`, `project?` | Hive context | `HiveContextGenerator.GenerateForTask()` |

### 3.2 Resources

MCP resources provide read-only access to hive data:

| Resource URI | Description | Maps To |
|--------------|-------------|---------|
| `hive://projects` | List of all projects | `ProjectRegistry.List()` |
| `hive://project/{name}` | Single project metadata | `ProjectRegistry.Get()` |
| `hive://agents` | List of all agents | `AgentRegistry.List()` |
| `hive://agent/{name}` | Single agent metadata | `AgentRegistry.Get()` |
| `hive://knowledge` | Hive knowledge index | `KnowledgeAggregator.Load()` |
| `hive://messages/inbox/{recipient}` | Agent's inbox | `MessageBus.Subscribe()` |
| `hive://context/task/{id}` | Hive context for task | `HiveContextGenerator.GenerateForTask()` |
| `hive://context/agent/{name}` | Hive context for agent | `HiveContextGenerator.GenerateForAgent()` |

### 3.3 Prompts

MCP prompts guide AI assistants on common hive workflows:

| Prompt | Arguments | Description |
|--------|-----------|-------------|
| `find_related_knowledge` | `topic`, `current_project` | Guides querying related knowledge from other projects |
| `consult_agent` | `capability_needed` | Guides finding and messaging an appropriate agent |
| `check_cross_project_patterns` | `problem_description` | Guides searching for similar patterns in other repos |
| `bootstrap_with_hive_context` | `task_id` | Guides starting a task with full hive awareness |

### 3.4 Server Configuration

**stdio transport for local usage:**
```json
{
  "mcpServers": {
    "adb-hive": {
      "command": "adb",
      "args": ["mcp", "serve"],
      "env": {
        "ADB_HOME": "/home/valter/Code/github.com/valter-silva-au/life",
        "ADB_MCP_MODE": "hive"
      }
    }
  }
}
```

**HTTP transport for remote usage (optional):**
```bash
adb mcp serve --http --port=8080 --token=$(cat ~/.adb-mcp-token)
```

### 3.5 OpenClaw Integration

OpenClaw bots consume the MCP server via Python MCP client:

**~/.openclaw/agents/{name}/agent/SKILLS.md** (auto-generated):
```markdown
## ADB Hive Mind Skills

You have access to the AI Dev Brain Hive Mind via MCP server. Use these tools to:
- Query knowledge from all 30+ projects
- Find related work and patterns
- Check agent capabilities and availability
- Send/receive messages from other agents

### Available Tools
- query_hive_knowledge(query, projects, types)
- get_related_knowledge(task_id)
- list_projects(filter)
- list_agents(filter)
- hive_message_send(to, subject, content, type)
- hive_message_list(recipient)
```

**OpenClaw MCP client setup:**
```python
# ~/.openclaw/mcp-config.json
{
  "servers": {
    "adb-hive": {
      "command": "adb",
      "args": ["mcp", "serve"],
      "env": {
        "ADB_HOME": "/home/valter/Code/github.com/valter-silva-au/life"
      }
    }
  }
}
```

## 4. Knowledge Architecture

### 4.1 Schema Design

**Unified Knowledge Store:** `hive-mind/knowledge/index.yaml`

The hive knowledge store is a flattened index of all project knowledge stores with project references:

```yaml
version: "1.0"
indexed_at: "2026-03-14T10:30:00Z"
index_size_mb: 2.4
entry_count: 342

# Project manifests
projects:
  - repo_path: "/home/valter/Code/repos/github.com/valter-silva-au/ai-dev-brain-v1"
    knowledge_path: "docs/knowledge/index.yaml"
    last_indexed: "2026-03-14T10:30:00Z"
    entry_count: 45

# Flattened knowledge entries with project namespacing
entries:
  - id: "ai-dev-brain-v1::K-00001"
    project: "ai-dev-brain-v1"
    project_path: "/home/valter/Code/repos/github.com/valter-silva-au/ai-dev-brain-v1"
    local_id: "K-00001"
    type: "decision"
    topic: "architecture"
    summary: "Use local interfaces to avoid import cycles"
    detail_path: "docs/knowledge/entries/K-00001.md"
    source_task: "TASK-00005"
    source_type: "task_archive"
    date: "2026-02-01"
    entities: ["core", "storage", "integration"]
    tags: ["go", "architecture", "patterns"]
    related: []

# Cross-project topic graph
topics:
  architecture:
    projects: ["ai-dev-brain-v1", "life", "openclaw"]
    entry_count: 23
    related_topics: ["patterns", "design-decisions"]

# Cross-project entity registry
entities:
  Claude-Opus:
    type: "system"
    projects: ["ai-dev-brain-v1", "life", "openclaw"]
    knowledge_count: 18
    roles: ["ai-assistant", "code-reviewer", "architect"]
```

### 4.2 Aggregation Strategy

**How knowledge flows in:**

1. **Task Completion Hook:**
   - `adb-hook-task-completed.sh` extracts knowledge
   - Knowledge saved to project's `docs/knowledge/`
   - Hook triggers `adb hive knowledge update --project=<current>`
   - Incremental update: only changed entries are re-indexed

2. **Manual Bulk Indexing:**
   - `adb hive knowledge index` scans all projects
   - Reads each project's `docs/knowledge/index.yaml`
   - Flattens into hive-level index with project namespacing
   - Merges topics and entities across projects

3. **Conflict Resolution:**
   - Knowledge IDs namespaced: `<project>::<local-id>`
   - Topics merged by lowercase name
   - Entities merged by lowercase name
   - Duplicate detection by content similarity (not implemented in Phase 1)

### 4.3 Query Strategy

**How knowledge flows out:**

1. **Fast Path (Hive Index):**
   - Query hive index first (in-memory after load)
   - Filter by project, type, tags, date range
   - Full-text search across summary and detail
   - Return results with project references

2. **Slow Path (Source Files):**
   - If hive index is stale (>24h), optionally refresh
   - Query source knowledge stores directly
   - Used for detailed queries needing full content

3. **MCP Tool Queries:**
   - `query_hive_knowledge` uses fast path
   - `get_related_knowledge` combines fast path + project-specific queries
   - Results include project name, repo path, and detail file path

### 4.4 Sync vs Real-Time Tradeoffs

**Sync (Chosen for Phase 1):**
- ✅ No background processes
- ✅ Consistent with file-based philosophy
- ✅ Predictable resource usage
- ✅ Git-friendly (index committed to life/ repo)
- ❌ Staleness up to 24h (acceptable for knowledge)

**Real-Time (Future Phase):**
- ✅ Always fresh
- ❌ Requires file watchers or background processes
- ❌ Higher resource usage
- ❌ Complexity for solo developer

## 5. Communication Architecture

### 5.1 Message Format

**File-based messages:** `channels/inbox/{recipient}/{message-id}.yaml`

```yaml
id: "MSG-00042"
conversation_id: "CONV-research-react-hooks"
from: "team-lead"
to: "research-owl"
subject: "Need React Hooks best practices research"
content: |
  Research Owl, I need comprehensive research on React Hooks best practices for TASK-00123.

  Specific focus areas:
  - useEffect dependency arrays
  - Custom hooks patterns
  - Performance optimization with useMemo/useCallback

  Please compile findings and update hive knowledge with your learnings.
type: "request"
priority: "P1"
date: "2026-03-14T10:30:00Z"
in_reply_to: ""
tags: ["research", "react", "hooks"]
metadata:
  related_task: "TASK-00123"
  expected_duration: "2h"
  format: "markdown-report"
status: "pending"
```

### 5.2 Routing

**Agent Routing Rules:**

1. **Direct addressing:** Messages with `to: "{agent-name}"` go to `channels/inbox/{agent-name}/`
2. **Capability-based routing:** Agent queries `adb hive agent search --capability=research` to find suitable agents
3. **Project routing:** Messages with `to: "{project-name}"` go to `channels/inbox/{project-name}/`
4. **Broadcast:** Messages with `to: "all"` go to `channels/inbox/all/`

**OpenClaw Integration:**
- Symlink `~/.openclaw/channels/inbox` -> `/home/valter/Code/github.com/valter-silva-au/life/channels/inbox/`
- Cron job (every 5 minutes): `adb hive messages sync-openclaw` copies outbox to OpenClaw delivery queue
- OpenClaw agents poll their inbox directories at session start

### 5.3 File-Based Pub/Sub Design

**Directory Structure:**
```
channels/
  inbox/
    team-lead/
      MSG-00001.yaml
      MSG-00002.yaml
    research-owl/
      MSG-00042.yaml
    code-panda/
    all/
      BROADCAST-00001.yaml
  outbox/
    team-lead/
      MSG-00043.yaml
    research-owl/
  archive/
    2026/
      03/
        MSG-00001.yaml
        MSG-00002.yaml
```

**Delivery Guarantees:**
- **At-least-once:** Messages persist until explicitly archived
- **No ordering guarantees:** Readers sort by timestamp
- **Idempotency:** Agents must handle duplicate messages (same `id`)

**Cleanup Strategy:**
- Messages archived after 30 days in inbox
- Archive retained for 1 year
- CLI command: `adb hive messages cleanup --before=30d`

### 5.4 Integration with OpenClaw Channels

**OpenClaw Current Design:**
- `~/.openclaw/channels/inbox` - Currently empty
- `~/.openclaw/channels/outbox` - Currently empty
- `~/.openclaw/delivery-queue/` - Message delivery tracking

**Hive Mind Integration:**

1. **Symlink Strategy:**
   ```bash
   ln -s /home/valter/Code/github.com/valter-silva-au/life/channels/inbox/code-panda ~/.openclaw/channels/inbox
   ln -s /home/valter/Code/github.com/valter-silva-au/life/channels/outbox/code-panda ~/.openclaw/channels/outbox
   ```

2. **Sync Command:**
   ```bash
   adb hive messages sync-openclaw
   ```
   - Reads OpenClaw `delivery-queue/` for sent messages
   - Copies to life/ repo `channels/outbox/{agent-name}/`
   - Moves delivered messages to archive
   - Polls inbox and notifies OpenClaw via Telegram (optional)

3. **OpenClaw Bot Awareness:**
   - Update SOUL.md with hive mind instructions
   - Add skill for checking inbox: `adb hive messages list --for=code-panda`
   - Add skill for sending messages: `adb hive message send --to=team-lead --subject="..." --content="..."`

## 6. Integration Points

### 6.1 OpenClaw Integration

**Discovery and Registration:**
```bash
adb hive openclaw discover
```
- Reads `~/.openclaw/openclaw.json` for agent list
- Parses each `agents/{name}/agent/SOUL.md` for capabilities
- Registers agents in `agents/index.yaml`

**MCP Server Configuration:**
```bash
adb hive openclaw configure-mcp
```
- Writes MCP config to `~/.openclaw/mcp-config.json`
- Adds skills to each agent's SKILLS.md
- Symlinks channel directories

**Message Sync:**
```bash
# Cron job: */5 * * * *
adb hive messages sync-openclaw
```

### 6.2 Claude Code Sessions Integration

**.mcp.json in life/ repo:**
```json
{
  "mcpServers": {
    "adb-hive": {
      "command": "adb",
      "args": ["mcp", "serve"],
      "env": {
        "ADB_HOME": "/home/valter/Code/github.com/valter-silva-au/life"
      }
    }
  }
}
```

**Task-level context injection:**
- `adb task create` triggers `adb hive context generate --for-task={id}`
- Hive context appended to `.claude/rules/task-context.md` in worktree
- AI assistant sees related knowledge from other projects immediately

**Session-start hook:**
```bash
# In worktree .claude/hooks/pre-tool-use.sh
adb hive messages check --for-task=$TASK_ID
# Prints unread messages relevant to current task
```

### 6.3 Existing ADB Hooks Integration

**PostToolUse:**
- Auto-register projects on first file edit in new repo
- Incremental knowledge index update on knowledge file changes

**TaskCompleted (Phase B):**
- Extract knowledge as usual
- Trigger `adb hive knowledge update --project=<current>`
- Generate cross-project knowledge summary

**SessionEnd:**
- Update agent status in registry (if agent name detected from session)
- Archive processed messages
- Generate session summary with hive context

### 6.4 life/ Repo as Central Hub

**Directory Structure:**
```
/home/valter/Code/github.com/valter-silva-au/life/
  .taskconfig                        # ADB base path root
  backlog.yaml                       # Central task registry
  tickets/                           # Per-task directories
  docs/
    knowledge/                       # life/ project knowledge
  projects/
    index.yaml                       # Project registry (NEW)
  agents/
    index.yaml                       # Agent registry (NEW)
  hive-mind/
    knowledge/
      index.yaml                     # Hive knowledge index (NEW)
    .last-index                      # Timestamp of last indexing (NEW)
  channels/                          # Message bus (NEW)
    inbox/
    outbox/
    archive/
  sessions/                          # Captured Claude Code sessions
  .adb_events.jsonl                  # Event log
  .knowledge_counter                 # Knowledge ID counter
  .session_counter                   # Session ID counter
  .hive_counter                      # Hive message ID counter (NEW)
```

**Why life/ as the hub:**
- Already the base path for ADB
- All tasks tracked here
- Single source of truth for backlog
- Git-friendly: all hive state version controlled
- Accessible to all Claude Code sessions via worktrees

## 7. Phased Implementation Plan

### Phase 1: Foundation (Minimum Viable Hive Mind)

**Goal:** Basic cross-project awareness without breaking existing functionality.

**Components:**
1. **Project Registry**
   - `internal/hive/projectregistry.go`
   - Storage: `projects/index.yaml`
   - CLI: `adb hive project register`, `adb hive project list`
   - Manual registration only (no auto-discovery)

2. **Knowledge Aggregator (Read-Only)**
   - `internal/hive/knowledgeaggregator.go`
   - Storage: `hive-mind/knowledge/index.yaml`
   - CLI: `adb hive knowledge index`, `adb hive knowledge search`
   - Manual indexing only (no hook integration)

3. **MCP Server Extension**
   - Add `query_hive_knowledge` tool
   - Add `list_projects` tool
   - Add `hive://knowledge` resource

**Acceptance Criteria:**
- Claude Code sessions can query knowledge from all registered projects
- Manual workflow: register projects, index knowledge, query via MCP
- No breaking changes to existing adb commands
- All Phase 1 code behind feature flag: `ADB_HIVE_ENABLED=1`

**Estimated Effort:** 3 days
- Day 1: Project registry + CLI
- Day 2: Knowledge aggregator + CLI
- Day 3: MCP server tools + integration tests

### Phase 2: Intelligence (Knowledge Aggregation & Querying)

**Goal:** Automated knowledge aggregation, agent registry, cross-project patterns.

**Components:**
1. **Agent Registry**
   - `internal/hive/agentregistry.go`
   - Storage: `agents/index.yaml`
   - CLI: `adb hive agent list`, `adb hive agent discover`
   - OpenClaw discovery

2. **Hive Context Generator**
   - `internal/hive/contextgenerator.go`
   - CLI: `adb hive context generate`
   - Hook integration: inject into task-context.md

3. **Hook-Driven Indexing**
   - PostToolUse: auto-register projects
   - TaskCompleted Phase B: trigger `adb hive knowledge update`
   - SessionEnd: update agent status

4. **MCP Server Expansion**
   - Add `list_agents`, `get_agent`, `search_agents_by_capability`
   - Add `get_related_knowledge`, `get_hive_context`
   - Add `hive://agents`, `hive://context/task/{id}` resources

**Acceptance Criteria:**
- Knowledge index updates automatically on task completion
- Agents registered automatically from OpenClaw and Claude Code
- Claude Code sessions start with hive context including related knowledge from other projects
- MCP tools cover all read-only hive queries

**Estimated Effort:** 4 days
- Day 1: Agent registry + OpenClaw discovery
- Day 2: Hive context generator
- Day 3: Hook integration (PostToolUse, TaskCompleted, SessionEnd)
- Day 4: MCP server expansion + end-to-end tests

### Phase 3: Interaction (Agent-to-Agent, Cross-Project)

**Goal:** Full bidirectional communication and collaboration.

**Components:**
1. **Message Bus**
   - `internal/hive/messagebus.go`
   - Storage: `channels/inbox/`, `channels/outbox/`, `channels/archive/`
   - CLI: `adb hive message send`, `adb hive message list`, `adb hive message archive`

2. **OpenClaw Message Sync**
   - CLI: `adb hive messages sync-openclaw`
   - Symlink strategy for channels
   - Cron job setup documentation

3. **MCP Server Write Tools**
   - Add `hive_message_send`, `hive_message_list`, `get_conversation`
   - Add `hive://messages/inbox/{recipient}` resource

4. **Agent Collaboration Workflows**
   - Documentation: "Consulting Another Agent"
   - Documentation: "Broadcasting Knowledge Updates"
   - Prompts: `consult_agent`, `find_related_knowledge`

**Acceptance Criteria:**
- Claude Code sessions can send messages to OpenClaw agents
- OpenClaw agents receive and respond to messages from Claude Code
- Message delivery confirmed via delivery queue
- End-to-end test: team-lead -> research-owl -> knowledge base update

**Estimated Effort:** 5 days
- Day 1: Message bus implementation
- Day 2: OpenClaw sync command + symlink setup
- Day 3: MCP write tools
- Day 4: OpenClaw integration testing
- Day 5: Documentation + collaboration workflows

### Phase 4 (Future): Advanced Intelligence

**Out of scope for initial implementation, but architecturally prepared:**

1. **Duplicate Detection:** Content similarity hashing for knowledge entries
2. **Knowledge Recommendations:** ML-based suggestions for related knowledge
3. **Agent Skill Learning:** Agents learn new capabilities from completed tasks
4. **Cross-Project Dependency Tracking:** Automatic detection of shared libraries/patterns
5. **Real-Time Indexing:** File watchers for instant knowledge updates
6. **HTTP MCP Server:** Remote access for distributed teams

## 8. Data Model Summary

**New Models in `pkg/models/`:**

```go
// project.go
type Project struct {
    Name                string
    RepoPath            string
    Purpose             string
    TechStack           []string
    Status              ProjectStatus
    LastUpdated         string
    Tags                []string
    DefaultAI           string
    BranchPattern       string
    RelatedProjects     []string
    KnowledgeEntryCount int
    DecisionCount       int
    ActiveTaskCount     int
    AgentCount          int
}

type ProjectStatus string
const (
    ProjectActive   ProjectStatus = "active"
    ProjectArchived ProjectStatus = "archived"
    ProjectPaused   ProjectStatus = "paused"
)

type ProjectFilter struct {
    Status   ProjectStatus
    Tags     []string
    TechStack []string
}

// agent.go
type Agent struct {
    Name          string
    Type          AgentType
    Model         string
    Capabilities  []string
    Role          string
    Status        AgentStatus
    LastSeen      string
    HomeProject   string
    ActiveTask    string
    SessionCount  int
    MemoryPath    string
    WorkspacePath string
}

type AgentType string
const (
    AgentClaudeCode AgentType = "claude-code"
    AgentOpenClaw   AgentType = "openclaw"
)

type AgentStatus string
const (
    AgentIdle    AgentStatus = "idle"
    AgentBusy    AgentStatus = "busy"
    AgentOffline AgentStatus = "offline"
)

type AgentFilter struct {
    Type         AgentType
    Capabilities []string
    Status       AgentStatus
}

// hivemessage.go
type HiveMessage struct {
    ID             string
    ConversationID string
    From           string
    To             string
    Subject        string
    Content        string
    Type           HiveMessageType
    Priority       Priority
    Date           string
    InReplyTo      string
    Tags           []string
    Metadata       map[string]string
    Status         HiveMessageStatus
}

// hiveknowledge.go
type HiveKnowledgeResult struct {
    ID          string
    Project     string
    ProjectPath string
    LocalID     string
    Entry       KnowledgeEntry
    DetailPath  string
}

// hivecontext.go
type HiveContext struct {
    CurrentProject    Project
    RelatedProjects   []Project
    RelatedKnowledge  []HiveKnowledgeResult
    ActiveAgents      []Agent
    RecentMessages    []HiveMessage
    CrossRepoPatterns []string
    Summary           string
}
```

## 9. Security Considerations

### 9.1 Data Isolation

**What's Shared:**
- Project metadata (name, purpose, tech stack, tags)
- Knowledge entries (decisions, learnings, patterns, gotchas)
- Agent capabilities and status
- Task metadata (title, status, tags, not content)
- Message content (plain text only)

**What's NOT Shared:**
- `.taskconfig` contents (may contain credentials)
- Task notes.md and design.md (may contain sensitive info)
- Session transcripts (may contain PII)
- Git commit messages (may reference internal systems)
- File contents (agents query via MCP, not direct file access)

### 9.2 Message Security

- Messages contain only references (task IDs, knowledge IDs)
- Full content retrieved via MCP tools with authentication check
- No credentials or API keys in messages
- Message archive pruned after 1 year

### 9.3 OpenClaw Isolation

- OpenClaw agents have read-only access to hive knowledge
- Write access limited to sending messages
- No direct filesystem access to other agents' workspaces
- MCP server validates recipient exists before delivering messages

## 10. Testing Strategy

### 10.1 Unit Tests

Each new component:
- `projectregistry_test.go` - CRUD operations, filtering
- `agentregistry_test.go` - Discovery, status updates
- `knowledgeaggregator_test.go` - Indexing, querying, merging
- `messagebus_test.go` - Publish, subscribe, routing
- `contextgenerator_test.go` - Context generation, markdown rendering

### 10.2 Property Tests

- `projectregistry_property_test.go` - Invariants: no duplicate projects, valid paths
- `knowledgeaggregator_property_test.go` - Invariants: no ID collisions, project references valid
- `messagebus_property_test.go` - Invariants: message delivery, no lost messages

### 10.3 Integration Tests

- `hive_integration_test.go` - End-to-end: register project -> index knowledge -> query via MCP
- `openclaw_integration_test.go` - OpenClaw discovery -> agent registry -> message send

### 10.4 End-to-End Scenarios

1. **Cross-Project Knowledge Query:**
   - Setup: 3 projects with knowledge entries
   - Execute: `adb hive knowledge search "authentication"`
   - Verify: Results from all 3 projects returned

2. **Agent-to-Agent Communication:**
   - Setup: Register team-lead and research-owl
   - Execute: team-lead sends message to research-owl
   - Verify: research-owl's inbox contains message

3. **Hive Context Generation:**
   - Setup: Task in project A with tags matching knowledge in project B
   - Execute: `adb hive context generate --for-task=TASK-00123`
   - Verify: Context includes related knowledge from project B

## 11. Performance Considerations

### 11.1 Indexing Performance

- **Cold index:** Scan all projects, ~30 repos × 50 entries = 1500 entries, ~2-3 seconds
- **Incremental update:** Single project, ~50 entries, <100ms
- **Query performance:** In-memory search, <50ms for most queries
- **Optimization:** Pre-filter by project tags before deep search

### 11.2 MCP Server Performance

- **Startup time:** Load all registries, ~500ms
- **Query latency:** <100ms for read-only tools
- **Write latency:** <200ms for message delivery (filesystem write)
- **Concurrent requests:** Single stdio process, sequential execution

### 11.3 Storage Growth

- **Hive knowledge index:** ~2-3 MB for 1500 entries
- **Message archive:** ~1 MB per 1000 messages
- **Project/agent registries:** <100 KB each
- **Total hive metadata:** <10 MB after 1 year

## 12. Migration Path

### 12.1 Zero-Disruption Rollout

Phase 1 is fully additive:
- New `internal/hive/` package
- New `projects/`, `agents/`, `hive-mind/` directories
- New `adb hive` command group
- Existing commands unchanged
- Feature flag: `ADB_HIVE_ENABLED=1`

### 12.2 Gradual Adoption

1. **Week 1:** Deploy Phase 1, manually register 5 key projects
2. **Week 2:** Index knowledge, validate MCP queries from Claude Code
3. **Week 3:** Deploy Phase 2, enable auto-discovery
4. **Week 4:** Deploy Phase 3, test agent messaging
5. **Week 5:** Full rollout with OpenClaw integration

### 12.3 Rollback Strategy

- All hive data in separate directories
- Delete `projects/`, `agents/`, `hive-mind/`, `channels/` to rollback
- Existing adb commands continue working without hive features
- MCP server falls back to Phase 2 tools (task management only)

## 13. Open Questions and Future Work

### 13.1 Open Questions

1. **Duplicate detection:** How to identify semantically duplicate knowledge across projects?
   - **Proposal:** Content similarity hashing (Phase 4)

2. **Message priority escalation:** How should urgent messages be surfaced to idle agents?
   - **Proposal:** Push notifications via Telegram for P0 messages

3. **Knowledge deprecation:** How to mark knowledge as outdated?
   - **Proposal:** Add `deprecated: true` field, filter from queries

4. **Agent authentication:** How to verify message sender identity?
   - **Proposal:** Message signing with agent private keys (Phase 4)

### 13.2 Future Enhancements

- **Knowledge graph visualization:** D3.js web UI for topic/entity relationships
- **Agent capability learning:** Agents auto-update capabilities based on completed tasks
- **Cross-project linting:** Detect pattern violations across all repos
- **Hive health dashboard:** TUI for monitoring message flow, index staleness, agent status
- **Real-time sync:** File watchers for instant knowledge updates (opt-in)

---

## Appendix A: File Paths Reference

All paths relative to `/home/valter/Code/github.com/valter-silva-au/life/`:

| Path | Purpose |
|------|---------|
| `projects/index.yaml` | Project registry |
| `agents/index.yaml` | Agent registry |
| `hive-mind/knowledge/index.yaml` | Hive knowledge index |
| `hive-mind/.last-index` | Last indexing timestamp |
| `channels/inbox/{recipient}/` | Incoming messages |
| `channels/outbox/{sender}/` | Outgoing messages |
| `channels/archive/{year}/{month}/` | Archived messages |
| `.hive_counter` | Hive message ID counter |

OpenClaw paths:
| Path | Purpose |
|------|---------|
| `~/.openclaw/openclaw.json` | OpenClaw config |
| `~/.openclaw/agents/{name}/agent/SOUL.md` | Agent identity |
| `~/.openclaw/memory/{name}.sqlite` | Agent memory |
| `~/.openclaw/channels/inbox` | Symlink to life/ inbox |
| `~/.openclaw/channels/outbox` | Symlink to life/ outbox |

## Appendix B: CLI Commands Reference

### Hive Command Group

```bash
# Project Management
adb hive project register <path>
adb hive project list [--status=active] [--tags=cli,go]
adb hive project update <name> --status=archived

# Agent Management
adb hive agent list [--type=openclaw] [--status=idle]
adb hive agent discover [--openclaw] [--claude-code]
adb hive agent status <name>

# Knowledge Operations
adb hive knowledge index [--force]
adb hive knowledge search <query> [--projects=proj1,proj2] [--types=decision,learning]
adb hive knowledge update --project=<name>

# Context Generation
adb hive context generate --for-task=<id>
adb hive context generate --for-agent=<name>
adb hive context generate --for-project=<name>

# Message Operations
adb hive message send --to=<agent> --subject="..." --content="..." [--type=request] [--priority=P1]
adb hive message list [--for=<agent>] [--unread]
adb hive message archive <message-id>
adb hive message show <message-id>
adb hive messages check --for-task=<id>
adb hive messages sync-openclaw
adb hive messages cleanup --before=30d

# OpenClaw Integration
adb hive openclaw discover
adb hive openclaw configure-mcp
```

## Appendix C: Adapter Wiring in app.go

```go
// Hive adapters
app.ProjectReg = hive.NewProjectRegistry(basePath)
app.AgentReg = hive.NewAgentRegistry(basePath)
app.KnowledgeAgg = hive.NewKnowledgeAggregator(basePath, app.ProjectReg)
app.MessageBus = hive.NewMessageBus(basePath)
app.HiveCtxGen = hive.NewHiveContextGenerator(
    basePath,
    app.ProjectReg,
    app.AgentReg,
    app.KnowledgeAgg,
    app.MessageBus,
)

// Wire to CLI
cli.ProjectReg = app.ProjectReg
cli.AgentReg = app.AgentReg
cli.KnowledgeAgg = app.KnowledgeAgg
cli.MessageBus = app.MessageBus
cli.HiveCtxGen = app.HiveCtxGen
```

## Appendix D: MCP Server Handler Example

```go
// internal/mcp/hivetools.go
package mcp

import (
    "github.com/valter-silva-au/ai-dev-brain/internal/hive"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type HiveToolHandlers struct {
    knowledgeAgg hive.KnowledgeAggregator
    projectReg   hive.ProjectRegistry
    agentReg     hive.AgentRegistry
    messageBus   hive.MessageBus
}

func (h *HiveToolHandlers) HandleQueryHiveKnowledge(req mcp.ToolRequest) mcp.ToolResponse {
    query := req.Params["query"].(string)
    projects := req.Params["projects"].([]string) // Optional
    types := req.Params["types"].([]string)       // Optional

    opts := hive.SearchOptions{
        ProjectFilter: projects,
        TypeFilter:    parseTypes(types),
        Limit:         100,
    }

    results, err := h.knowledgeAgg.SearchAcrossProjects(query, opts)
    if err != nil {
        return mcp.NewErrorResponse(req.ID, "query_failed", err.Error())
    }

    return mcp.NewSuccessResponse(req.ID, results)
}
```

---

**Document Status:** Ready for review
**Next Steps:** Review with stakeholder, refine based on feedback, begin Phase 1 implementation
