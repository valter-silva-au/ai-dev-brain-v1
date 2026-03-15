package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	pythonPath = "/home/valter/Code/repos/github.com/Comfy-Org/ComfyUI/venv/bin/python"
	scriptPath = "/home/valter/.claude/scripts/comfyui-generate.py"
)

// ImageGenerator continuously generates images reflecting the system state
type ImageGenerator struct {
	hub       *WSHub
	outputDir string
	templates *Templates
	getState  func() string // returns current state description for prompt synthesis
	latest    string        // path to latest generated image
}

// NewImageGenerator creates a new image generation loop
func NewImageGenerator(hub *WSHub, outputDir string, templates *Templates, getState func() string) *ImageGenerator {
	os.MkdirAll(outputDir, 0o755)
	return &ImageGenerator{
		hub:       hub,
		outputDir: outputDir,
		templates: templates,
		getState:  getState,
	}
}

// Run starts the continuous image generation loop
func (ig *ImageGenerator) Run(ctx context.Context) {
	// Check if python and script exist
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		log.Printf("imagegen: python not found at %s — image generation disabled", pythonPath)
		return
	}
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Printf("imagegen: script not found at %s — image generation disabled", scriptPath)
		return
	}

	log.Printf("imagegen: starting continuous generation loop (output: %s)", ig.outputDir)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			ig.generateOne()
			// Brief pause between generations to not hammer the GPU
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

// generateOne creates a single image based on current system state
func (ig *ImageGenerator) generateOne() {
	state := ig.getState()
	prompt := synthesizePrompt(state)

	filename := fmt.Sprintf("adb-mind-%d", time.Now().Unix())
	outputPath := filepath.Join(ig.outputDir, filename+".png")

	log.Printf("imagegen: generating — %s", truncate(prompt, 80))

	cmd := exec.Command(pythonPath, scriptPath,
		"--prompt", prompt,
		"--output", ig.outputDir,
		"--filename", filename,
		"--width", "1024",
		"--height", "576",
		"--steps", "4",
		"--guidance-scale", "4.0",
		"--seed", fmt.Sprintf("%d", rand.Intn(999999)),
	)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		log.Printf("imagegen: failed (%v): %s", err, truncate(string(output), 200))
		return
	}

	log.Printf("imagegen: generated in %s — %s", duration.Round(time.Second), outputPath)
	ig.latest = "/images/generated/" + filename + ".png"

	// Push to dashboard
	html := fmt.Sprintf(`<div id="mindimage" hx-swap-oob="innerHTML">
  <img src="%s" alt="ADB Mind" class="w-full rounded-xl border border-slate-700 shadow-lg shadow-brand-500/10" />
  <p class="text-xs text-slate-500 mt-2">Generated %s ago — %s</p>
</div>`, ig.latest, duration.Round(time.Second), truncate(prompt, 100))

	ig.hub.Broadcast(html)
}

// synthesizePrompt creates a visual prompt from the system state description
func synthesizePrompt(state string) string {
	styles := []string{
		"cinematic sci-fi digital art, neon glow, dark space station command center",
		"cyberpunk control room, holographic displays, blue and purple neon",
		"futuristic AI neural network visualization, glowing nodes and connections",
		"high-tech mission control with floating holographic screens, dark ambient",
		"digital dreamscape with data streams and crystalline AI structures",
	}
	style := styles[rand.Intn(len(styles))]

	// Count active elements from state
	claudeCount := strings.Count(state, "Claude Code")
	agentLoopsCount := strings.Count(state, "Agent Loops")
	taskCount := strings.Count(state, "task")

	// Build scene elements
	var elements []string
	if claudeCount > 0 {
		elements = append(elements, fmt.Sprintf("%d glowing robotic engineers working at holographic terminals", claudeCount))
	}
	if agentLoopsCount > 0 {
		elements = append(elements, fmt.Sprintf("%d autonomous assembly lines with robotic arms building code", agentLoopsCount))
	}
	if taskCount > 0 {
		elements = append(elements, "floating task cards with progress bars")
	}

	// Always include ADB brain
	elements = append(elements, "a luminous central brain overseeing everything")

	scene := strings.Join(elements, ", ")
	return fmt.Sprintf("%s, %s, ultra detailed, 8k quality", scene, style)
}

// truncate cuts a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
