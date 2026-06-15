package harness

import (
	"dossier/internal/core"
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

// Install stub.
func (a *AntigravityHarness) Install(opts core.InstallOpts) error {
	return nil
}
