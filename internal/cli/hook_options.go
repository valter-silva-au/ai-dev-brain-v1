package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valter-silva-au/ai-dev-brain/internal/core"
	"github.com/valter-silva-au/ai-dev-brain/internal/memory"
	"github.com/valter-silva-au/ai-dev-brain/pkg/models"
)

// hookOptionsFromConfig reads MergedConfig.Global.Hooks on App and turns it
// into a core.HookEngineOptions value suitable for NewHookEngineWithOptions.
// The three advanced features (EvidenceGate, OperatorControls, Memory) are
// wired from .taskconfig here — that's the bridge that was missing in PR #51
// and #53 so their features were dormant until the user called the Go API
// directly.
//
// Returns an empty options value (zero-valued, all features off) and a
// nil error when Hooks config is absent, so legacy callers keep working.
// Logs warnings on sub-feature setup failures (memory store open, etc.)
// and returns the caller a partially-configured Options value so the rest
// of hook processing still runs.
func hookOptionsFromConfig() core.HookEngineOptions {
	opts := core.HookEngineOptions{}
	if App == nil || App.MergedConfig == nil {
		return opts
	}
	cfg := resolvedHookConfig(App.MergedConfig)

	// --- Evidence gate
	if cfg.EvidenceGate.Enabled {
		opts.Evidence = core.EvidenceGateConfig{
			Enabled:      true,
			WritePaths:   append([]string(nil), cfg.EvidenceGate.WritePaths...),
			ReadPatterns: append([]string(nil), cfg.EvidenceGate.ReadPatterns...),
		}
	}

	// --- Operator controls (kill-switch + steer)
	if cfg.OperatorControls.KillSwitchEnabled || cfg.OperatorControls.SteerEnabled {
		opts.Operator = core.OperatorConfig{
			KillSwitchEnabled: cfg.OperatorControls.KillSwitchEnabled,
			KillSwitchFile:    cfg.OperatorControls.KillSwitchFile,
			SteerEnabled:      cfg.OperatorControls.SteerEnabled,
			SteerFile:         cfg.OperatorControls.SteerFile,
		}
	}

	// --- Memory
	if cfg.Memory.Enabled {
		indexer, err := openMemoryIndexerFromConfig(cfg.Memory)
		if err != nil {
			// Non-fatal: the rest of the hook pipeline still runs.
			fmt.Fprintf(os.Stderr, "Warning: memory hook indexer disabled: %v\n", err)
		} else {
			opts.Memory = core.MemoryHookConfig{Enabled: true, Indexer: indexer}
		}
	}

	return opts
}

// resolvedHookConfig merges repo-level hooks over global hooks. Repo
// values that are zero/empty fall back to the global value; repo values
// that are set win. This is a shallow merge per top-level sub-struct:
// if RepoConfig.Hooks.Memory.Enabled is true, the entire Memory block
// comes from the repo; otherwise the global Memory block is used.
// Simpler than a deep merge and matches user intent (a repo that
// toggles a feature usually wants to provide the whole sub-block).
func resolvedHookConfig(mc *models.MergedConfig) models.HookConfig {
	var global, repo models.HookConfig
	if mc.Global != nil {
		global = mc.Global.Hooks
	}
	if mc.Repo != nil {
		repo = mc.Repo.Hooks
	}
	result := global
	if repo.Enabled {
		result = repo
	}
	if repo.EvidenceGate.Enabled {
		result.EvidenceGate = repo.EvidenceGate
	}
	if repo.OperatorControls.KillSwitchEnabled || repo.OperatorControls.SteerEnabled {
		result.OperatorControls = repo.OperatorControls
	}
	if repo.Memory.Enabled {
		result.Memory = repo.Memory
	}
	return result
}

// openMemoryIndexerFromConfig constructs a memory.SQLiteStore from the
// MemoryHookConfig schema. The embedder is resolved by
// buildEmbedderFromConfig so .taskconfig can pick fake / openai /
// ollama without special-casing in each hook command.
func openMemoryIndexerFromConfig(mc models.MemoryHookConfig) (core.MemoryIndexer, error) {
	emb, err := buildEmbedderFromConfig(mc.Embedder)
	if err != nil {
		return nil, fmt.Errorf("build embedder: %w", err)
	}
	dbPath := mc.DBPath
	if dbPath == "" {
		dbPath = filepath.Join(App.BasePath, ".adb_memory.sqlite")
	}
	// Short-lived ctx just for Open; hook bodies pass their own.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := memory.OpenSQLiteStore(ctx, dbPath, emb)
	if err != nil {
		return nil, fmt.Errorf("open memory store at %q: %w", dbPath, err)
	}
	return store, nil
}

// buildEmbedderFromConfig translates the embedder block in .taskconfig
// into a memory.EmbeddingProvider. Shared with the `adb memory …` CLI
// via memory.go's buildEmbedder() which uses the flag-derived form;
// this sibling is the config-derived form.
func buildEmbedderFromConfig(ec models.MemoryEmbedderConf) (memory.EmbeddingProvider, error) {
	dim := ec.Dim
	if dim <= 0 {
		dim = 64 // sensible default for the fake provider
	}
	apiKey := ec.APIKey
	if strings.HasPrefix(apiKey, "$") {
		apiKey = os.Getenv(strings.TrimPrefix(apiKey, "$"))
	}
	switch strings.ToLower(ec.Provider) {
	case "", "fake":
		return memory.NewFakeEmbedder(dim), nil
	case "openai":
		endpoint := ec.Endpoint
		if endpoint == "" {
			endpoint = "https://api.openai.com/v1/embeddings"
		}
		return &memory.OpenAIEmbedder{
			Endpoint: endpoint,
			APIKey:   apiKey,
			Model:    ec.Model,
			Dim:      dim,
			Client:   &http.Client{Timeout: 30 * time.Second},
		}, nil
	case "ollama":
		endpoint := ec.Endpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		return &memory.OllamaEmbedder{
			Endpoint: endpoint,
			Model:    ec.Model,
			Dim:      dim,
			Client:   &http.Client{Timeout: 60 * time.Second},
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q (valid: fake, openai, ollama)", ec.Provider)
	}
}
