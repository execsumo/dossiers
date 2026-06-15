package cli

import (
	"bytes"
	"context"
	"dossier/internal/core"
	"dossier/internal/store"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLICommands(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "dossier-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	// Pre-populate with a dossier
	dossierDir := filepath.Join(tempHome, "pricing-model-refresh")
	if err := os.MkdirAll(dossierDir, 0755); err != nil {
		t.Fatalf("failed to create dossier dir: %v", err)
	}

	fm := core.Frontmatter{
		ID:            "dos_test123",
		Name:          "Pricing model refresh",
		Slug:          "pricing-model-refresh",
		CreatedAt:     time.Now().Truncate(time.Second),
		UpdatedAt:     time.Now().Truncate(time.Second),
		LastTouchedAt: time.Now().Truncate(time.Second),
		Status:        core.StatusActive,
		Importance:    core.ImportanceHigh,
		Urgency:       core.UrgencyMedium,
		NextAction:    "Compare revised scenarios",
		OpenQuestions: []string{"Sales feedback?"},
	}
	body := "# Pricing model refresh\n\n## Situation\nWorking draft."

	serialized, err := store.FormatDossierFile(fm, body)
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dossierDir, "dossier.md"), []byte(serialized), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Lock file is needed for writes
	if err := os.WriteFile(filepath.Join(dossierDir, ".lock"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to write lock: %v", err)
	}

	// 1. Run wire to get service
	svc, err := wire(tempHome)
	if err != nil {
		t.Fatalf("failed to wire: %v", err)
	}

	// Test List
	res, err := svc.List(context.Background(), core.ListReq{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	items, ok := res.Data.([]core.ListItem)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Slug != "pricing-model-refresh" {
		t.Errorf("expected slug pricing-model-refresh, got %s", items[0].Slug)
	}

	// Test Recall
	recallRes, err := svc.Recall(context.Background(), core.RecallReq{ID: "dos_test123"})
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}
	recallData, ok := recallRes.Data.(core.RecallResult)
	if !ok {
		t.Fatalf("unexpected type for RecallResult: %T", recallRes.Data)
	}
	if recallData.Frontmatter.ID != "dos_test123" {
		t.Errorf("expected ID dos_test123, got %s", recallData.Frontmatter.ID)
	}

	// Test Path
	pathRes, err := svc.Path(context.Background(), core.PathReq{ID: "dos_test123"})
	if err != nil {
		t.Fatalf("Path failed: %v", err)
	}
	pathStr, ok := pathRes.Data.(string)
	if !ok || !strings.HasSuffix(pathStr, "pricing-model-refresh") {
		t.Errorf("expected suffix pricing-model-refresh, got %q", pathStr)
	}

	// Test Archive
	archiveRes, err := svc.Archive(context.Background(), core.ArchiveReq{ID: "dos_test123"})
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}
	if !archiveRes.OK {
		t.Errorf("expected Archive result to be OK")
	}

	// Verify archived
	recallRes2, err := svc.Recall(context.Background(), core.RecallReq{ID: "dos_test123"})
	if err != nil {
		t.Fatalf("failed to recall archived dossier: %v", err)
	}
	recallData2 := recallRes2.Data.(core.RecallResult)
	if recallData2.Frontmatter.Status != core.StatusArchived {
		t.Errorf("expected status to be archived, got %s", recallData2.Frontmatter.Status)
	}
}

// TestCLIOutputFormat prints and verifies printed output strings
func TestCLIOutputFormat(t *testing.T) {
	// Simple validation of printJSON
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printJSON(map[string]string{"foo": "bar"})

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	output := strings.TrimSpace(buf.String())
	expected := "{\n  \"foo\": \"bar\"\n}"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}
