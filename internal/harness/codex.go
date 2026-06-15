package harness

import (
	"dossier/internal/core"
	"os"
	"path/filepath"
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

// Install stub for Milestone 6 hook configuration.
func (c *CodexHarness) Install(opts core.InstallOpts) error {
	// To be implemented in Milestone 6
	return nil
}
