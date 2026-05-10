package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valter-silva-au/ai-dev-brain/internal/hooks"
)

func TestHookEngine_PreventRecursion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)

	t.Run("No recursion without flag", func(t *testing.T) {
		os.Unsetenv("ADB_HOOK_ACTIVE")
		if engine.PreventRecursion() {
			t.Errorf("PreventRecursion() = true, want false")
		}
	})

	t.Run("Recursion detected with flag", func(t *testing.T) {
		os.Setenv("ADB_HOOK_ACTIVE", "1")
		defer os.Unsetenv("ADB_HOOK_ACTIVE")

		if !engine.PreventRecursion() {
			t.Errorf("PreventRecursion() = false, want true")
		}
	})
}

func TestHookEngine_ProcessPreToolUse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Allow normal file edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/main.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil", err)
		}
	})

	t.Run("Block vendor/ edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/vendor/package/file.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error")
		}
	})

	t.Run("Block go.sum edit", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/go.sum",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error")
		}
	})

	t.Run("Allow other tools", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Read",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/vendor/package/file.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil", err)
		}
	})

	t.Run("Block vendor/ edit with Windows backslash path", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": `C:\Users\dev\repo\vendor\package\file.go`,
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error for Windows-style vendor path")
		}
	})

	t.Run("Block go.sum edit with Windows backslash path", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": `C:\Users\dev\repo\go.sum`,
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error for Windows-style go.sum path")
		}
	})

	t.Run("Block bare go.sum path", func(t *testing.T) {
		event := &hooks.PreToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": "go.sum",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err == nil {
			t.Errorf("ProcessPreToolUse() error = nil, want error for bare go.sum")
		}
	})

	t.Run("Allow file with vendor in name but not in path", func(t *testing.T) {
		// Guard against false positives like "vendorlist.go"
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/vendorlist.go",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil for file named vendorlist.go", err)
		}
	})

	t.Run("Allow file named something.go.sum", func(t *testing.T) {
		// Previously the check matched any HasSuffix "go.sum" including
		// "notes.go.sum" — tighten it to real go.sum files only.
		event := &hooks.PreToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": "/path/to/notes.go.sum",
			},
		}

		err := engine.ProcessPreToolUse(event)
		if err != nil {
			t.Errorf("ProcessPreToolUse() error = %v, want nil for notes.go.sum", err)
		}
	})
}

func TestHookEngine_EvidenceGate_DisabledByDefault(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-default-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngine(tmp) // zero options -> gate off
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// A write that would otherwise be gated must pass when the gate is off.
	event := &hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "test-results.json",
		},
	}
	if err := engine.ProcessPreToolUse(event); err != nil {
		t.Errorf("ProcessPreToolUse() with gate disabled should not block, got %v", err)
	}
}

func TestHookEngine_EvidenceGate_BlocksWriteWithoutRead(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-block-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Evidence: EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   []string{"test-results.json"},
			ReadPatterns: []string{"*.png"},
		},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	event := &hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "test-results.json",
		},
	}
	err = engine.ProcessPreToolUse(event)
	if err == nil {
		t.Fatalf("ProcessPreToolUse() expected block, got nil")
	}
	if !strings.Contains(err.Error(), "evidence-gate") {
		t.Errorf("expected error to mention evidence-gate, got %v", err)
	}
}

func TestHookEngine_EvidenceGate_AllowsWriteAfterMatchingRead(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-pass-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Evidence: EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   []string{"test-results.json"},
			ReadPatterns: []string{"*.png"},
		},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	readEvt := &hooks.PreToolUseEvent{
		ToolName: "Read",
		Parameters: map[string]interface{}{
			"file_path": "screenshots/feature-1.png",
		},
	}
	if err := engine.ProcessPreToolUse(readEvt); err != nil {
		t.Fatalf("Read event should not error, got %v", err)
	}

	writeEvt := &hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "test-results.json",
		},
	}
	if err := engine.ProcessPreToolUse(writeEvt); err != nil {
		t.Errorf("Write after matching Read should pass, got %v", err)
	}
}

func TestHookEngine_EvidenceGate_UnrelatedWritesUnaffected(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-unrelated-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Evidence: EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   []string{"test-results.json"},
			ReadPatterns: []string{"*.png"},
		},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Writing an unrelated file (not in WritePaths) must pass regardless
	// of evidence state.
	event := &hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "main.go",
		},
	}
	if err := engine.ProcessPreToolUse(event); err != nil {
		t.Errorf("unrelated Write should pass, got %v", err)
	}
}

func TestHookEngine_EvidenceGate_WindowsBackslashPathsMatch(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-win-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Evidence: EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   []string{"test-results.json"},
			ReadPatterns: []string{"*.png"},
		},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Claude Code on Windows can pass backslash file_path values.
	readEvt := &hooks.PreToolUseEvent{
		ToolName: "Read",
		Parameters: map[string]interface{}{
			"file_path": `screenshots\feature-1.png`,
		},
	}
	if err := engine.ProcessPreToolUse(readEvt); err != nil {
		t.Fatalf("Read with backslash path should not error, got %v", err)
	}

	writeEvt := &hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "test-results.json",
		},
	}
	if err := engine.ProcessPreToolUse(writeEvt); err != nil {
		t.Errorf("Write after backslash-path Read should pass, got %v", err)
	}
}

func TestHookEngine_KillSwitch_BlocksWhenSentinelPresent(t *testing.T) {
	tmp, err := os.MkdirTemp("", "killswitch-on-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Operator: OperatorConfig{KillSwitchEnabled: true},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Without the sentinel, calls pass.
	ok := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	if err := engine.ProcessPreToolUse(ok); err != nil {
		t.Fatalf("pre-sentinel ProcessPreToolUse should pass, got %v", err)
	}

	// Touch AGENT_STOP: any call blocks.
	sentinel := filepath.Join(tmp, "AGENT_STOP")
	if err := os.WriteFile(sentinel, []byte{}, 0o644); err != nil {
		t.Fatalf("writing sentinel: %v", err)
	}
	if err := engine.ProcessPreToolUse(ok); err == nil {
		t.Errorf("sentinel present: expected block, got nil")
	} else if !strings.Contains(err.Error(), "kill-switch") {
		t.Errorf("expected kill-switch message, got %v", err)
	}

	// Remove AGENT_STOP: calls pass again.
	if err := os.Remove(sentinel); err != nil {
		t.Fatalf("removing sentinel: %v", err)
	}
	if err := engine.ProcessPreToolUse(ok); err != nil {
		t.Errorf("after sentinel removal: expected pass, got %v", err)
	}
}

func TestHookEngine_KillSwitch_BeatsRecursionGuard(t *testing.T) {
	tmp, err := os.MkdirTemp("", "killswitch-recursion-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Operator: OperatorConfig{KillSwitchEnabled: true},
	})

	// Simulate we are inside a recursive hook invocation.
	os.Setenv("ADB_HOOK_ACTIVE", "1")
	defer os.Unsetenv("ADB_HOOK_ACTIVE")

	// Sentinel present: kill-switch must fire even though recursion
	// would normally cause ProcessPreToolUse to return nil early.
	sentinel := filepath.Join(tmp, "AGENT_STOP")
	if err := os.WriteFile(sentinel, []byte{}, 0o644); err != nil {
		t.Fatalf("writing sentinel: %v", err)
	}
	evt := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	if err := engine.ProcessPreToolUse(evt); err == nil {
		t.Errorf("expected kill-switch to block despite recursion guard, got nil")
	}
}

func TestHookEngine_KillSwitch_DisabledByDefault(t *testing.T) {
	tmp, err := os.MkdirTemp("", "killswitch-off-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngine(tmp) // zero options: kill-switch off
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Even with the file present, a caller that didn't opt in is
	// unaffected — the file could be anything to them.
	sentinel := filepath.Join(tmp, "AGENT_STOP")
	if err := os.WriteFile(sentinel, []byte{}, 0o644); err != nil {
		t.Fatalf("writing sentinel: %v", err)
	}
	evt := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	if err := engine.ProcessPreToolUse(evt); err != nil {
		t.Errorf("disabled kill-switch must ignore sentinel file, got %v", err)
	}
}

func TestHookEngine_Steer_ConsumesFileAndPrintsOnce(t *testing.T) {
	tmp, err := os.MkdirTemp("", "steer-once-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	// Redirect stderr to a pipe so we can assert on what the engine prints.
	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Operator: OperatorConfig{SteerEnabled: true},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Write STEER.md — first call should consume it.
	steer := filepath.Join(tmp, "STEER.md")
	if err := os.WriteFile(steer, []byte("pivot: focus on auth refactor\r\n"), 0o644); err != nil {
		t.Fatalf("writing STEER.md: %v", err)
	}

	evt := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	if err := engine.ProcessPreToolUse(evt); err != nil {
		t.Fatalf("first call error = %v", err)
	}
	// Second call with no STEER.md must not print again.
	if err := engine.ProcessPreToolUse(evt); err != nil {
		t.Fatalf("second call error = %v", err)
	}

	w.Close()
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "OPERATOR STEERING: pivot: focus on auth refactor") {
		t.Errorf("expected steering message in stderr, got %q", output)
	}
	// Count occurrences: exactly 1.
	if strings.Count(output, "OPERATOR STEERING:") != 1 {
		t.Errorf("steering message should appear once, got %d in %q", strings.Count(output, "OPERATOR STEERING:"), output)
	}

	// STEER.md is gone, STEER.md.consumed exists.
	if _, err := os.Stat(steer); !os.IsNotExist(err) {
		t.Errorf("STEER.md should be gone after consume, err = %v", err)
	}
	if _, err := os.Stat(steer + ".consumed"); err != nil {
		t.Errorf("STEER.md.consumed should exist, err = %v", err)
	}
}

func TestHookEngine_Steer_DisabledByDefault(t *testing.T) {
	tmp, err := os.MkdirTemp("", "steer-off-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngine(tmp) // steer off
	os.Unsetenv("ADB_HOOK_ACTIVE")

	steer := filepath.Join(tmp, "STEER.md")
	if err := os.WriteFile(steer, []byte("should not be touched"), 0o644); err != nil {
		t.Fatalf("writing STEER.md: %v", err)
	}

	evt := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	if err := engine.ProcessPreToolUse(evt); err != nil {
		t.Fatalf("call error = %v", err)
	}
	// STEER.md must remain untouched when steer is disabled.
	if _, err := os.Stat(steer); err != nil {
		t.Errorf("STEER.md should still exist with steer disabled, err = %v", err)
	}
}

func TestHookEngine_Steer_EmptyFileSkipsPrint(t *testing.T) {
	tmp, err := os.MkdirTemp("", "steer-empty-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	r, w, _ := os.Pipe()
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Operator: OperatorConfig{SteerEnabled: true},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	steer := filepath.Join(tmp, "STEER.md")
	if err := os.WriteFile(steer, []byte("\r\n  \t"), 0o644); err != nil {
		t.Fatalf("writing empty-ish STEER.md: %v", err)
	}

	evt := &hooks.PreToolUseEvent{ToolName: "Read", Parameters: map[string]interface{}{"file_path": "foo.go"}}
	_ = engine.ProcessPreToolUse(evt)

	w.Close()
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	if strings.Contains(output, "OPERATOR STEERING:") {
		t.Errorf("empty-ish file should not produce a steering print, got %q", output)
	}
}

func TestHookEngine_EvidenceGate_ClearedOnSessionEnd(t *testing.T) {
	tmp, err := os.MkdirTemp("", "evidence-cleared-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmp)

	engine := NewHookEngineWithOptions(tmp, HookEngineOptions{
		Evidence: EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   []string{"test-results.json"},
			ReadPatterns: []string{"*.png"},
		},
	})
	os.Unsetenv("ADB_HOOK_ACTIVE")

	// Record a read in session 1.
	_ = engine.ProcessPreToolUse(&hooks.PreToolUseEvent{
		ToolName: "Read",
		Parameters: map[string]interface{}{
			"file_path": "screenshots/a.png",
		},
	})

	// End the session.
	if err := engine.ProcessSessionEnd(&hooks.SessionEndEvent{SessionID: "S1"}); err != nil {
		t.Fatalf("ProcessSessionEnd error = %v", err)
	}

	// The next session must re-block until a fresh Read arrives.
	err = engine.ProcessPreToolUse(&hooks.PreToolUseEvent{
		ToolName: "Write",
		Parameters: map[string]interface{}{
			"file_path": "test-results.json",
		},
	})
	if err == nil {
		t.Errorf("Write in new session without fresh Read should block, got nil")
	}
}

func TestNormalisePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"forward slashes preserved", "/path/to/file.go", "/path/to/file.go"},
		{"windows backslashes converted", `C:\Users\dev\repo\file.go`, "C:/Users/dev/repo/file.go"},
		{"mixed separators", `C:\Users/dev\repo/file.go`, "C:/Users/dev/repo/file.go"},
		{"dot segments cleaned", "/path/./to/../to/file.go", "/path/to/file.go"},
		{"trailing slash cleaned", "/path/to/dir/", "/path/to/dir"},
		{"bare filename", "go.sum", "go.sum"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalisePath(tc.in)
			if got != tc.want {
				t.Errorf("normalisePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestHookEngine_ProcessPostToolUse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Track file change on Edit", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		event := &hooks.PostToolUseEvent{
			ToolName: "Edit",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		err := engine.ProcessPostToolUse(event)
		if err != nil {
			t.Errorf("ProcessPostToolUse() error = %v", err)
		}

		// Verify change was tracked
		changes, err := engine.tracker.GetChanges()
		if err != nil {
			t.Fatalf("Failed to get changes: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 {
			if changes[0].FilePath != testFile {
				t.Errorf("Change FilePath = %v, want %v", changes[0].FilePath, testFile)
			}
			if changes[0].Operation != "modified" {
				t.Errorf("Change Operation = %v, want %v", changes[0].Operation, "modified")
			}
		}
	})

	t.Run("Track file change on Write", func(t *testing.T) {
		// Clear previous changes
		engine.tracker.Clear()

		testFile := filepath.Join(tmpDir, "new.go")
		if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		event := &hooks.PostToolUseEvent{
			ToolName: "Write",
			Parameters: map[string]interface{}{
				"file_path": testFile,
			},
		}

		err := engine.ProcessPostToolUse(event)
		if err != nil {
			t.Errorf("ProcessPostToolUse() error = %v", err)
		}

		// Verify change was tracked
		changes, err := engine.tracker.GetChanges()
		if err != nil {
			t.Fatalf("Failed to get changes: %v", err)
		}

		if len(changes) != 1 {
			t.Errorf("Expected 1 change, got %d", len(changes))
		}

		if len(changes) > 0 {
			if changes[0].Operation != "created" {
				t.Errorf("Change Operation = %v, want %v", changes[0].Operation, "created")
			}
		}
	})
}

func TestHookEngine_ProcessStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process stop without errors", func(t *testing.T) {
		// ProcessStop should not error even if checks fail (advisory only)
		err := engine.ProcessStop()
		if err != nil {
			t.Errorf("ProcessStop() error = %v, want nil (advisory only)", err)
		}
	})
}

func TestHookEngine_ProcessTaskCompleted(t *testing.T) {
	// This test requires a valid Go project structure, so we'll test the error cases
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process task completed", func(t *testing.T) {
		// Create task directory
		taskDir := filepath.Join(tmpDir, "tickets", "TASK-001")
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("Failed to create task dir: %v", err)
		}

		// Track some changes
		engine.tracker.TrackChange("file1.go", "modified")
		engine.tracker.TrackChange("file2.go", "created")

		event := &hooks.TaskCompletedEvent{
			TaskID:    "TASK-001",
			Status:    "done",
			Timestamp: "2024-01-01T00:00:00Z",
		}

		// This will fail quality gates (no valid Go project), but we test it runs
		err := engine.ProcessTaskCompleted(event)
		// We expect an error from quality gates since this isn't a real Go project
		if err == nil {
			t.Logf("ProcessTaskCompleted() succeeded (unexpected in test env)")
		}
	})
}

func TestHookEngine_ProcessSessionEnd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create tickets directory
	ticketsDir := filepath.Join(tmpDir, "tickets")
	if err := os.MkdirAll(ticketsDir, 0o755); err != nil {
		t.Fatalf("Failed to create tickets dir: %v", err)
	}

	engine := NewHookEngine(tmpDir)
	os.Unsetenv("ADB_HOOK_ACTIVE")

	t.Run("Process session end", func(t *testing.T) {
		event := &hooks.SessionEndEvent{
			SessionID: "sess-123",
			Timestamp: "2024-01-01T00:00:00Z",
			Duration:  120.5,
			Metadata: map[string]interface{}{
				"transcript": "Test transcript content",
			},
		}

		err := engine.ProcessSessionEnd(event)
		if err != nil {
			t.Errorf("ProcessSessionEnd() error = %v", err)
		}
	})
}

func TestHookEngine_GetCurrentTaskID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hookengine-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewHookEngine(tmpDir)

	t.Run("Get from environment", func(t *testing.T) {
		os.Setenv("ADB_TASK_ID", "TASK-001")
		defer os.Unsetenv("ADB_TASK_ID")

		taskID := engine.getCurrentTaskID()
		if taskID != "TASK-001" {
			t.Errorf("getCurrentTaskID() = %v, want %v", taskID, "TASK-001")
		}
	})

	t.Run("Empty when not set", func(t *testing.T) {
		os.Unsetenv("ADB_TASK_ID")
		// Note: getCurrentTaskID will try to get from git branch, which may or may not exist
		// in the test environment, so we just ensure it doesn't panic
		_ = engine.getCurrentTaskID()
	})
}
