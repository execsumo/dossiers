package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFSStoreInit(t *testing.T) {
	// Create a temporary home directory for testing
	tempHome, err := os.MkdirTemp("", "dossier-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	store := NewFSStore(tempHome)
	if err := store.Init(); err != nil {
		t.Fatalf("FSStore.Init() failed: %v", err)
	}

	// Verify directories are created
	dirs := []string{
		tempHome,
		filepath.Join(tempHome, "context"),
		filepath.Join(tempHome, "sessions"),
	}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %s to exist, but got err: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory, but it is not", dir)
		}
	}

	// Verify guide.md is written and contains signature content
	guidePath := filepath.Join(tempHome, "context", "guide.md")
	guideBytes, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatalf("failed to read guide.md: %v", err)
	}
	guideContent := string(guideBytes)
	if !strings.Contains(guideContent, "Dossier Distillation Guide") {
		t.Errorf("expected guide.md to contain Distillation Guide title, but got:\n%s", guideContent)
	}

	// Verify library.md is written and contains correct placeholders
	libraryPath := filepath.Join(tempHome, "context", "library.md")
	libraryBytes, err := os.ReadFile(libraryPath)
	if err != nil {
		t.Fatalf("failed to read library.md: %v", err)
	}
	libraryContent := string(libraryBytes)
	if !strings.Contains(libraryContent, "Dossier Library") {
		t.Errorf("expected library.md to contain header, but got:\n%s", libraryContent)
	}
}
