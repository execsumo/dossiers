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

	// Create fake .claude.json with a mix of styles to test migration and preservation
	claudeJSONPath := filepath.Join(tempHome, ".claude.json")
	initialConfig := map[string]any{
		"mcpServers": map[string]any{},
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"matcher": "*",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "echo 'unrelated'",
						},
					},
				},
			},
			"PreCompact": "old-style-string-hook",
		},
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

	// Assert the exact structure for SessionStart (Claude Code schema: array of matchers)
	startVal, ok := hooks["SessionStart"].([]any)
	if !ok {
		t.Fatalf("expected SessionStart to be a slice, got %T", hooks["SessionStart"])
	}
	if len(startVal) != 1 {
		t.Fatalf("expected SessionStart slice to have 1 matcher, got %d", len(startVal))
	}
	matcherObj, ok := startVal[0].(map[string]any)
	if !ok {
		t.Fatalf("expected matcher object, got %T", startVal[0])
	}
	if matcherObj["matcher"] != "*" {
		t.Errorf("expected matcher to be '*', got %v", matcherObj["matcher"])
	}
	hooksList, ok := matcherObj["hooks"].([]any)
	if !ok {
		t.Fatalf("expected hooks list to be slice, got %T", matcherObj["hooks"])
	}
	if len(hooksList) != 1 {
		t.Fatalf("expected hooks list to have 1 item, got %d", len(hooksList))
	}
	hookEntry, ok := hooksList[0].(map[string]any)
	if !ok {
		t.Fatalf("expected hook entry map, got %T", hooksList[0])
	}
	if hookEntry["type"] != "command" {
		t.Errorf("expected type to be 'command', got %v", hookEntry["type"])
	}
	if !strings.Contains(hookEntry["command"].(string), "hook session-start") {
		t.Errorf("expected command to contain 'hook session-start', got %v", hookEntry["command"])
	}

	// Assert PreCompact was converted from string to array
	preCompactVal, ok := hooks["PreCompact"].([]any)
	if !ok {
		t.Fatalf("expected PreCompact to be converted to slice, got %T", hooks["PreCompact"])
	}
	if len(preCompactVal) != 1 {
		t.Fatalf("expected PreCompact to have 1 matcher, got %d", len(preCompactVal))
	}

	// Assert unrelated hook UserPromptSubmit was preserved
	unrelatedVal, ok := hooks["UserPromptSubmit"].([]any)
	if !ok {
		t.Fatalf("expected UserPromptSubmit to be slice, got %T", hooks["UserPromptSubmit"])
	}
	if len(unrelatedVal) != 1 {
		t.Fatalf("expected UserPromptSubmit to have 1 item, got %d", len(unrelatedVal))
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

	// Also write a hooks.json with a mix of styles to test migration and preservation
	hooksPath := filepath.Join(codexDir, "hooks.json")
	initialHooks := map[string]any{
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "echo 'unrelated'",
						},
					},
				},
			},
			"Stop": "old-style-string-hook",
		},
	}
	hooksBytes, _ := json.Marshal(initialHooks)
	if err := os.WriteFile(hooksPath, hooksBytes, 0644); err != nil {
		t.Fatalf("failed to write fake hooks.json: %v", err)
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

	// Assert SessionStart is a slice of maps (without matcher)
	startVal, ok := hooks["SessionStart"].([]any)
	if !ok {
		t.Fatalf("expected SessionStart to be slice, got %T", hooks["SessionStart"])
	}
	if len(startVal) != 1 {
		t.Fatalf("expected SessionStart to have 1 entry, got %d", len(startVal))
	}
	matcherObj, ok := startVal[0].(map[string]any)
	if !ok {
		t.Fatalf("expected matcher object, got %T", startVal[0])
	}
	if _, hasMatcher := matcherObj["matcher"]; hasMatcher {
		t.Errorf("expected Codex matcher object to NOT contain 'matcher' key, but got it")
	}
	hooksList, ok := matcherObj["hooks"].([]any)
	if !ok {
		t.Fatalf("expected hooks list, got %T", matcherObj["hooks"])
	}
	if len(hooksList) != 1 {
		t.Fatalf("expected 1 hook entry, got %d", len(hooksList))
	}
	hookEntry, ok := hooksList[0].(map[string]any)
	if !ok {
		t.Fatalf("expected hook entry map, got %T", hooksList[0])
	}
	if !strings.Contains(hookEntry["command"].(string), "hook session-start") {
		t.Errorf("expected command to contain 'hook session-start', got %v", hookEntry["command"])
	}

	// Assert Stop was converted from string to array
	_, ok = hooks["Stop"].([]any)
	if !ok {
		t.Fatalf("expected Stop to be converted to slice, got %T", hooks["Stop"])
	}

	// Assert UserPromptSubmit was preserved
	unrelatedVal, ok := hooks["UserPromptSubmit"].([]any)
	if !ok {
		t.Fatalf("expected UserPromptSubmit to be slice, got %T", hooks["UserPromptSubmit"])
	}
	if len(unrelatedVal) != 1 {
		t.Fatalf("expected UserPromptSubmit to have 1 item, got %d", len(unrelatedVal))
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
