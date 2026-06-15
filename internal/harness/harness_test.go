package harness

import (
	"dossier/internal/core"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCodeHarness(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Before creating config files, Detect should return empty capabilities
	h := NewClaudeCodeHarness("/tmp/dossier")
	caps, err := h.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps.SessionStartHook || caps.SessionEndHook {
		t.Errorf("expected empty capabilities when config doesn't exist, got %+v", caps)
	}

	// Create fake .claude.json
	claudeJSONPath := filepath.Join(tempHome, ".claude.json")
	initialConfig := map[string]any{
		"mcpServers": map[string]any{},
	}
	configBytes, _ := json.Marshal(initialConfig)
	if err := os.WriteFile(claudeJSONPath, configBytes, 0644); err != nil {
		t.Fatalf("failed to write fake config: %v", err)
	}

	// Detect should now return capabilities
	caps, err = h.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !caps.SessionStartHook || !caps.SessionEndHook || !caps.PreCompactionHook || !caps.TranscriptCapture {
		t.Errorf("expected full Tier 1 capabilities, got %+v", caps)
	}

	// Install hooks with YesToAll = true
	err = h.Install(core.InstallOpts{YesToAll: true})
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	// Check that config is updated
	updatedBytes, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		t.Fatalf("failed to read updated config: %v", err)
	}
	var updatedConfig map[string]any
	if err := json.Unmarshal(updatedBytes, &updatedConfig); err != nil {
		t.Fatalf("failed to unmarshal updated config: %v", err)
	}

	hooks, ok := updatedConfig["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks not found in config")
	}

	if !strings.Contains(hooks["SessionStart"].(string), "hook session-start") {
		t.Errorf("expected SessionStart hook configured, got %q", hooks["SessionStart"])
	}

	// Test idempotency: running Install again should not create new backup and should not change config
	files, _ := filepath.Glob(filepath.Join(tempHome, ".claude.json.*.bak"))
	initialBackupCount := len(files)

	err = h.Install(core.InstallOpts{YesToAll: true})
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	files, _ = filepath.Glob(filepath.Join(tempHome, ".claude.json.*.bak"))
	if len(files) != initialBackupCount {
		t.Errorf("expected no new backup on idempotent run, got %d backups (initially %d)", len(files), initialBackupCount)
	}
}

func TestCodexHarness(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Before creating config files, Detect should return empty capabilities
	h := NewCodexHarness("/tmp/dossier")
	caps, err := h.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps.SessionStartHook || caps.SessionEndHook {
		t.Errorf("expected empty capabilities when config doesn't exist, got %+v", caps)
	}

	// Create fake .codex directory and config.toml
	codexDir := filepath.Join(tempHome, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("failed to create fake .codex dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write fake config.toml: %v", err)
	}

	// Detect should now return capabilities
	caps, err = h.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !caps.SessionStartHook || !caps.SessionEndHook || caps.PreCompactionHook || caps.TranscriptCapture {
		t.Errorf("expected Tier 2 capabilities (start/end only), got %+v", caps)
	}

	// Install hooks with YesToAll = true
	err = h.Install(core.InstallOpts{YesToAll: true})
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	// Check that config is updated
	hooksPath := filepath.Join(codexDir, "hooks.json")
	updatedBytes, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("failed to read updated hooks.json: %v", err)
	}
	var updatedConfig map[string]any
	if err := json.Unmarshal(updatedBytes, &updatedConfig); err != nil {
		t.Fatalf("failed to unmarshal updated config: %v", err)
	}

	hooks, ok := updatedConfig["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks not found in config")
	}

	if !strings.Contains(hooks["SessionStart"].(string), "hook session-start") {
		t.Errorf("expected SessionStart hook configured, got %q", hooks["SessionStart"])
	}
	if !strings.Contains(hooks["Stop"].(string), "hook session-end") {
		t.Errorf("expected Stop hook configured, got %q", hooks["Stop"])
	}

	// Test idempotency: running Install again should not create new backup and should not change config
	files, _ := filepath.Glob(filepath.Join(codexDir, "hooks.json.*.bak"))
	initialBackupCount := len(files)

	err = h.Install(core.InstallOpts{YesToAll: true})
	if err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	files, _ = filepath.Glob(filepath.Join(codexDir, "hooks.json.*.bak"))
	if len(files) != initialBackupCount {
		t.Errorf("expected no new backup on idempotent run, got %d backups (initially %d)", len(files), initialBackupCount)
	}
}

func TestAntigravityHarness(t *testing.T) {
	h := NewAntigravityHarness("/tmp/dossier")
	if h.Name() != "antigravity" {
		t.Errorf("expected name to be antigravity, got %s", h.Name())
	}
	caps, err := h.Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !caps.MCP || caps.SessionStartHook || caps.SessionEndHook || caps.PreCompactionHook || caps.TranscriptCapture {
		t.Errorf("expected Tier 3 capabilities, got %+v", caps)
	}
	err = h.Install(core.InstallOpts{YesToAll: true})
	if err != nil {
		t.Errorf("expected nil error on install, got %v", err)
	}
}
