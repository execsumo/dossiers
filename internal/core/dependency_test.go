package core

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// TestCorePackageIsPure enforces the Hexagonal Architecture constraint:
// internal/core MUST NOT import any sibling packages or third-party libraries.
// It is only allowed to import the Go standard library and its own subpackages (if any).
func TestCorePackageIsPure(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", "dossier/internal/core")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run go list: %v (stderr: %q)", err, stderr.String())
	}

	imports := strings.Split(stdout.String(), "\n")
	for _, imp := range imports {
		imp = strings.TrimSpace(imp)
		if imp == "" {
			continue
		}

		// Allow Go standard library packages (which do not contain a dot ".")
		if !strings.Contains(imp, ".") {
			continue
		}

		// Allow self-imports of internal/core subdirectories (if any, e.g. dossier/internal/core/foo)
		if strings.HasPrefix(imp, "dossier/internal/core") {
			continue
		}

		// Any other import containing a dot or referring to internal sibling packages is forbidden.
		t.Errorf("FORBIDDEN IMPORT IN CORE: %s. internal/core must remain pure.", imp)
	}
}
