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

// ClaudeCodeHarness implements capability detection and installation for Claude Code.
type ClaudeCodeHarness struct {
	dossierHome string
}

// NewClaudeCodeHarness instantiates the Claude Code harness.
func NewClaudeCodeHarness(dossierHome string) *ClaudeCodeHarness {
	return &ClaudeCodeHarness{
		dossierHome: dossierHome,
	}
}

// Name returns the identifier of the harness.
func (c *ClaudeCodeHarness) Name() string {
	return "claude-code"
}

// Detect checks if Claude Code is installed and returns its capabilities.
func (c *ClaudeCodeHarness) Detect() (core.Capabilities, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return core.Capabilities{}, err
	}

	claudeJSONPath := filepath.Join(home, ".claude.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	_, err1 := os.Stat(claudeJSONPath)
	_, err2 := os.Stat(settingsPath)

	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		return core.Capabilities{}, nil
	}

	return core.Capabilities{
		MCP:               true,
		SessionStartHook:  true,
		SessionEndHook:    true,
		PreCompactionHook: true,
		TranscriptCapture: true,
	}, nil
}

// Install configures lifecycle hooks in settings.json or .claude.json.
func (c *ClaudeCodeHarness) Install(opts core.InstallOpts) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	claudeJSONPath := filepath.Join(home, ".claude.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var configPath string
	if _, err := os.Stat(settingsPath); err == nil {
		configPath = settingsPath
	} else if _, err := os.Stat(claudeJSONPath); err == nil {
		configPath = claudeJSONPath
	} else {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read Claude Code config: %w", err)
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
	endCmd := execCmd + " session-end"
	preCompactCmd := execCmd + " pre-compaction"

	if isHookConfigured(hooksMap["SessionStart"], startCmd) &&
		isHookConfigured(hooksMap["SessionEnd"], endCmd) &&
		isHookConfigured(hooksMap["PreCompact"], preCompactCmd) {
		return nil
	}

	if !opts.YesToAll {
		fmt.Printf("Configure Claude Code session hooks? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return nil
		}
	}

	backupPath := fmt.Sprintf("%s.%d.bak", configPath, time.Now().Unix())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create config backup: %w", err)
	}

	hooksMap["SessionStart"] = updateHookArray(hooksMap["SessionStart"], startCmd, "hook session-start", true)
	hooksMap["SessionEnd"] = updateHookArray(hooksMap["SessionEnd"], endCmd, "hook session-end", true)
	hooksMap["PreCompact"] = updateHookArray(hooksMap["PreCompact"], preCompactCmd, "hook pre-compaction", true)
	configMap["hooks"] = hooksMap

	newData, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
