package harness

import (
	"dossier/internal/core"
	"fmt"
)

// AntigravityHarness implements capability detection for Antigravity.
type AntigravityHarness struct {
	dossierHome string
}

// NewAntigravityHarness instantiates the Antigravity harness.
func NewAntigravityHarness(dossierHome string) *AntigravityHarness {
	return &AntigravityHarness{
		dossierHome: dossierHome,
	}
}

// Name returns the identifier of the harness.
func (a *AntigravityHarness) Name() string {
	return "antigravity"
}

// Detect returns Tier 3 capabilities (context / MCP fallback only).
func (a *AntigravityHarness) Detect() (core.Capabilities, error) {
	// Antigravity is the current running agent harness, always present in Tier 3 capability.
	return core.Capabilities{
		MCP:               true,
		SessionStartHook:  false,
		SessionEndHook:    false,
		PreCompactionHook: false,
		TranscriptCapture: false,
	}, nil
}

// Install prints a warning since Antigravity does not support auto-registration.
func (a *AntigravityHarness) Install(opts core.InstallOpts) error {
	fmt.Printf("Warning: Antigravity MCP auto-registration is not supported. Please configure the MCP server manually in your client settings.\n")
	fmt.Printf("  Command: %s\n", opts.StableBinaryPath)
	fmt.Printf("  Args:    [mcp serve]\n")
	return nil
}
