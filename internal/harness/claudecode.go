package harness

import (
	"dossier/internal/core"
	"os"
	"path/filepath"
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

	// Claude Code config resides in ~/.claude.json or ~/.claude/settings.json
	claudeJSONPath := filepath.Join(home, ".claude.json")
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	_, err1 := os.Stat(claudeJSONPath)
	_, err2 := os.Stat(settingsPath)

	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		// Return empty capabilities (inactive) if config is missing
		return core.Capabilities{}, nil
	}

	// If configuration exists, we confirm it supports Tier 1 capabilities
	return core.Capabilities{
		MCP:               true,
		SessionStartHook:  true,
		SessionEndHook:    true,
		PreCompactionHook: true,
		TranscriptCapture: true, // Supported via hook stdout injection / transcripts access
	}, nil
}

// Install stub for Milestone 6 hook configuration.
func (c *ClaudeCodeHarness) Install(opts core.InstallOpts) error {
	// To be implemented in Milestone 6
	return nil
}
