package sync

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSync_FastForwardPreservesMachineLocal is the regression guard for the bug
// where a fast-forward pull (go-git Force checkout) deleted gitignored
// machine-local files — notably config.yaml, which silently un-teamed a joined
// colleague on their first pull of a teammate's change. Two clones of a seeded
// remote share an ancestor, so when only A advances, B's pull is a pure
// fast-forward that exercises ffToRemote.
func TestSync_FastForwardPreservesMachineLocal(t *testing.T) {
	bareDir, storeA, storeB := setupPair(t)

	// B holds machine-local files that must never be touched by sync.
	writeFile(t, storeB, "config.yaml", "author: bob\nteam:\n  remote: r\n")
	writeFile(t, storeB, "context/guide.md", "local guide\n")

	// A creates a dossier and pushes it (remote advances beyond the shared base).
	writeFile(t, storeA, "topic/dossier.md", "# Topic\nfrom A\n")
	mustSync(t, newSyncer(storeA, bareDir, "alice"))

	// B pulls — a fast-forward (B's HEAD is a strict ancestor of the remote).
	rep := mustSync(t, newSyncer(storeB, bareDir, "bob"))
	if !rep.Pulled {
		t.Fatalf("expected B's sync to pull the remote advance")
	}

	// B's machine-local files must survive the fast-forward checkout.
	assertFile(t, storeB, "config.yaml", "author: bob\nteam:\n  remote: r\n")
	assertFile(t, storeB, "context/guide.md", "local guide\n")

	// And B must have actually received A's dossier.
	if _, err := os.Stat(filepath.Join(storeB, "topic", "dossier.md")); err != nil {
		t.Fatalf("B did not receive A's dossier on fast-forward: %v", err)
	}
}
