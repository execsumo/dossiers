package harness

import (
	"dossier/internal/core"
	"fmt"
)

// Registry manages the set of supported client harnesses.
type Registry struct {
	harnesses []core.Harness
}

// NewRegistry instantiates the harness registry.
func NewRegistry(dossierHome string) *Registry {
	return &Registry{
		harnesses: []core.Harness{
			NewClaudeCodeHarness(dossierHome),
			NewCodexHarness(dossierHome),
			NewAntigravityHarness(dossierHome),
		},
	}
}

// All returns all registered harnesses.
func (r *Registry) All() []core.Harness {
	return r.harnesses
}

// Get retrieves a harness by name.
func (r *Registry) Get(name string) (core.Harness, error) {
	for _, h := range r.harnesses {
		if h.Name() == name {
			return h, nil
		}
	}
	return nil, fmt.Errorf("harness %q not found", name)
}
