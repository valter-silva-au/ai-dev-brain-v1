package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LiveProcess represents a detected AI process running on the system
type LiveProcess struct {
	PID     int
	Name    string // claude, agent-loops, openclaw-gateway, adb
	Type    string // claude-code, agent-loops, openclaw, adb-mcp
	CWD     string // working directory
	Project string // extracted project name from CWD
	CPU     string
	TTY     string
	Started string
	CmdLine string
}

// ScanLiveProcesses detects all running AI-related processes
func ScanLiveProcesses() []LiveProcess {
	var procs []LiveProcess

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return procs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}

		cmd := string(cmdline)
		cmd = strings.ReplaceAll(cmd, "\x00", " ")

		proc := classifyProcess(pid, cmd)
		if proc == nil {
			continue
		}

		// Read CWD
		cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
		if err == nil {
			proc.CWD = cwd
			proc.Project = extractProjectName(cwd)
		}

		// Read stat for CPU/TTY info
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err == nil {
			fields := strings.Fields(string(stat))
			if len(fields) > 6 {
				ttyNr, _ := strconv.Atoi(fields[6])
				if ttyNr > 0 {
					proc.TTY = fmt.Sprintf("pts/%d", ttyNr&0xFF)
				}
			}
		}

		procs = append(procs, *proc)
	}

	return procs
}

// classifyProcess determines if a process is an AI-related process
func classifyProcess(pid int, cmdline string) *LiveProcess {
	lower := strings.ToLower(cmdline)

	// Claude Code interactive sessions
	if strings.Contains(lower, "claude") && strings.Contains(lower, "--dangerously-skip-permissions") {
		return &LiveProcess{
			PID:  pid,
			Name: "Claude Code",
			Type: "claude-code",
		}
	}

	// Claude Agent SDK (spawned by Agent Loops)
	if strings.Contains(lower, "claude_agent_sdk") || (strings.Contains(lower, "claude") && strings.Contains(lower, "bypasspermissions") && strings.Contains(lower, "stream-json")) {
		return &LiveProcess{
			PID:  pid,
			Name: "Agent SDK",
			Type: "agent-loops",
		}
	}

	// Agent Loops engine
	if strings.Contains(lower, "agent_loops") && strings.Contains(lower, "loopengine") {
		return &LiveProcess{
			PID:  pid,
			Name: "Agent Loops",
			Type: "agent-loops",
		}
	}

	// OpenClaw gateway
	if strings.Contains(lower, "openclaw-gateway") {
		return &LiveProcess{
			PID:  pid,
			Name: "OpenClaw Gateway",
			Type: "openclaw",
		}
	}

	// ADB MCP server
	if strings.Contains(lower, "adb") && strings.Contains(lower, "mcp serve") {
		return &LiveProcess{
			PID:  pid,
			Name: "ADB MCP",
			Type: "adb-mcp",
		}
	}

	return nil
}

// extractProjectName gets the repo name from a CWD path
func extractProjectName(cwd string) string {
	// Pattern: .../repos/github.com/valter-silva-au/<project>/...
	// or: /home/valter/Code/repos/github.com/valter-silva-au/<project>
	parts := strings.Split(cwd, "/")
	for i, part := range parts {
		if part == "valter-silva-au" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback: last meaningful directory
	return filepath.Base(cwd)
}

// ProcessSummary aggregates live processes into a dashboard-friendly view
type ProcessSummary struct {
	ClaudeCodeSessions []LiveProcess
	AgentLoopsRuns     []LiveProcess
	OpenClawGateway    *LiveProcess
	ADBServices        []LiveProcess
	TotalActive        int
}

// SummarizeProcesses groups live processes by type
func SummarizeProcesses(procs []LiveProcess) ProcessSummary {
	var summary ProcessSummary
	for _, p := range procs {
		switch p.Type {
		case "claude-code":
			summary.ClaudeCodeSessions = append(summary.ClaudeCodeSessions, p)
		case "agent-loops":
			summary.AgentLoopsRuns = append(summary.AgentLoopsRuns, p)
		case "openclaw":
			summary.OpenClawGateway = &p
		case "adb-mcp":
			summary.ADBServices = append(summary.ADBServices, p)
		}
	}
	summary.TotalActive = len(summary.ClaudeCodeSessions) + len(summary.AgentLoopsRuns)
	if summary.OpenClawGateway != nil {
		summary.TotalActive++
	}
	return summary
}
