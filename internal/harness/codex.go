package harness

import (
	"dossier/internal/core"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodexHarness implements capability detection and installation for Codex.
type CodexHarness struct {
	dossierHome string
}

// NewCodexHarness instantiates the Codex harness.
func NewCodexHarness(dossierHome string) *CodexHarness {
	return &CodexHarness{
		dossierHome: dossierHome,
	}
}

// Name returns the identifier of the harness.
func (c *CodexHarness) Name() string {
	return "codex"
}

// Detect checks if Codex is installed and returns its capabilities.
func (c *CodexHarness) Detect() (core.Capabilities, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return core.Capabilities{}, err
	}

	// Codex config resides in ~/.codex/config.toml or ~/.codex/hooks.json
	configPath := filepath.Join(home, ".codex", "config.toml")
	hooksPath := filepath.Join(home, ".codex", "hooks.json")

	_, err1 := os.Stat(configPath)
	_, err2 := os.Stat(hooksPath)

	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		// Return empty capabilities (inactive) if config is missing
		return core.Capabilities{}, nil
	}

	// Codex is Tier 2: hooks + MCP, but no transcript capture
	return core.Capabilities{
		MCP:               true,
		SessionStartHook:  true,
		SessionEndHook:    true,
		PreCompactionHook: false,
		TranscriptCapture: false,
	}, nil
}

// Install configures lifecycle hooks in hooks.json.
func (c *CodexHarness) Install(opts core.InstallOpts) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	hooksPath := filepath.Join(home, ".codex", "hooks.json")

	_, err1 := os.Stat(configPath)
	_, err2 := os.Stat(hooksPath)

	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		return nil // Codex not detected, skip
	}

	// Ensure .codex directory exists
	if err := os.MkdirAll(filepath.Dir(hooksPath), 0755); err != nil {
		return fmt.Errorf("failed to create .codex directory: %w", err)
	}

	var data []byte
	if _, err := os.Stat(hooksPath); err == nil {
		data, err = os.ReadFile(hooksPath)
		if err != nil {
			return fmt.Errorf("failed to read Codex hooks: %w", err)
		}
	}

	var configMap map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &configMap); err != nil {
			configMap = make(map[string]any)
		}
	} else {
		configMap = make(map[string]any)
	}

	executable, err := os.Executable()
	if err != nil {
		executable = "dossier"
	}
	execCmd := fmt.Sprintf("%s hook", executable)

	hooksVal, ok := configMap["hooks"]
	var hooksMap map[string]any
	if ok {
		hooksMap, _ = hooksVal.(map[string]any)
	}
	if hooksMap == nil {
		hooksMap = make(map[string]any)
	}

	startCmd := execCmd + " session-start"
	stopCmd := execCmd + " session-end"

	if isHookConfigured(hooksMap["SessionStart"], startCmd) &&
		isHookConfigured(hooksMap["Stop"], stopCmd) {
		return nil // Already installed
	}

	if !opts.YesToAll {
		fmt.Printf("Configure Codex session hooks? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return nil
		}
	}

	// Backup existing hooksPath if it exists and has content
	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.%d.bak", hooksPath, time.Now().Unix())
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("failed to create config backup: %w", err)
		}
	}

	hooksMap["SessionStart"] = updateHookArray(hooksMap["SessionStart"], startCmd, "hook session-start", false)
	hooksMap["Stop"] = updateHookArray(hooksMap["Stop"], stopCmd, "hook session-end", false)
	configMap["hooks"] = hooksMap

	newData, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(hooksPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
