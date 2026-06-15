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

// updateCodexTOML parses and updates config.toml with the dossier MCP server config.
// Returns the updated content, whether it was changed, and any error.
func updateCodexTOML(content string, stablePath string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	startIndex := -1
	endIndex := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[mcp_servers.dossier]" {
			startIndex = i
			// Find the end of this section
			for j := i + 1; j < len(lines); j++ {
				trimmedNext := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmedNext, "[") && strings.HasSuffix(trimmedNext, "]") {
					endIndex = j
					break
				}
			}
			if endIndex == -1 {
				endIndex = len(lines)
			}
			break
		}
	}

	newBlock := []string{
		"[mcp_servers.dossier]",
		fmt.Sprintf("command = %q", stablePath),
		"args = [",
		"    \"mcp\",",
		"    \"serve\",",
		"]",
	}

	// Check if already correct
	if startIndex != -1 {
		blockLines := lines[startIndex:endIndex]
		hasCommand := false
		hasArgs := false
		for _, bl := range blockLines {
			tbl := strings.TrimSpace(bl)
			if strings.HasPrefix(tbl, "command") && strings.Contains(tbl, stablePath) {
				hasCommand = true
			}
			if strings.HasPrefix(tbl, "args") {
				hasArgs = true
			}
		}
		if hasCommand && hasArgs {
			return content, false, nil
		}

		// Replace the block
		var newLines []string
		newLines = append(newLines, lines[:startIndex]...)
		newLines = append(newLines, newBlock...)
		newLines = append(newLines, lines[endIndex:]...)
		return strings.Join(newLines, "\n"), true, nil
	}

	// Append to the end
	var newLines []string
	newLines = append(newLines, lines...)
	if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
		newLines = append(newLines, "")
	}
	newLines = append(newLines, newBlock...)
	return strings.Join(newLines, "\n"), true, nil
}

// Install configures lifecycle hooks in hooks.json and MCP registration in config.toml.
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

	stablePath := opts.StableBinaryPath
	if stablePath == "" {
		executable, err := os.Executable()
		if err != nil {
			executable = "dossier"
		}
		stablePath = executable
	}
	execCmd := fmt.Sprintf("%s hook", stablePath)

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

	hooksOk := isHookConfigured(hooksMap["SessionStart"], startCmd) &&
		isHookConfigured(hooksMap["Stop"], stopCmd)

	var tomlContent string
	var tomlExists bool
	if _, err := os.Stat(configPath); err == nil {
		tomlExists = true
		tomlBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read Codex config.toml: %w", err)
		}
		tomlContent = string(tomlBytes)
	}

	updatedTOML, tomlChanged, err := updateCodexTOML(tomlContent, stablePath)
	if err != nil {
		return fmt.Errorf("failed to update Codex config.toml: %w", err)
	}

	if hooksOk && !tomlChanged {
		return nil // Already installed
	}

	if !opts.YesToAll {
		fmt.Printf("Configure Codex integration (hooks + MCP server)? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return nil
		}
	}

	timestamp := time.Now().Unix()
	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.%d.bak", hooksPath, timestamp)
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("failed to create config backup: %w", err)
		}
	}
	if tomlExists && tomlChanged {
		backupTOMLPath := fmt.Sprintf("%s.%d.bak", configPath, timestamp)
		if err := os.WriteFile(backupTOMLPath, []byte(tomlContent), 0644); err != nil {
			return fmt.Errorf("failed to create config TOML backup: %w", err)
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

	if tomlChanged {
		if err := os.WriteFile(configPath, []byte(updatedTOML), 0644); err != nil {
			return fmt.Errorf("failed to write Codex config.toml: %w", err)
		}
	}

	return nil
}
