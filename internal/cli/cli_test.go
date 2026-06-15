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

func TestCLIMilestone3(t *testing.T) {
	tempHome, err := os.MkdirTemp("", "dossier-cli-m3-test-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tempHome)

	// Initialize the store directories (which also writes context/library.md)
	svc, err := wire(tempHome)
	if err != nil {
		t.Fatalf("failed to wire: %v", err)
	}

	_, err = svc.Init(context.Background(), true)
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// 1. Create a dossier using svc.Save
	saveReq := core.SaveReq{
		DistilledStateMarkdown: "# Product Specifications\n\nWe need to build a single Go binary.",
		FrontmatterUpdates: map[string]any{
			"name":        "Chainlink core engine",
			"status":      "active",
			"importance":  "high",
			"urgency":     "high",
			"next_action": "Implement the MCP server",
		},
	}
	res, err := svc.Save(context.Background(), saveReq)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// Let's list dossiers to get the actual ID
	listRes, err := svc.List(context.Background(), core.ListReq{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	items := listRes.Data.([]core.ListItem)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	realID := items[0].ID

	// Write an artifact to the dossier
	art := core.Artifact{
		ID:            "art_test_m3",
		DossierID:     realID,
		Type:          core.ArtifactTypeSourceSnapshot,
		Title:         "System design requirements document",
		ContentFormat: core.ContentFormatText,
		Content:       "Make sure it compiles into a single binary called dossier.",
	}
	// Save the dossier again with this artifact
	saveReq2 := core.SaveReq{
		ID:           realID,
		BaseRevision: res.Data.(core.Revision),
		Artifacts:    []core.Artifact{art},
	}
	_, err = svc.Save(context.Background(), saveReq2)
	if err != nil {
		t.Fatalf("Save with artifact failed: %v", err)
	}

	// 2. Perform global search
	searchRes, err := svc.Search(context.Background(), core.SearchReq{
		Query: "single",
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	hits := searchRes.Data.([]core.Hit)
	if len(hits) != 2 {
		t.Errorf("expected 2 hits (dossier body and artifact), got %d", len(hits))
	}

	// 3. Perform scoped search to this dossier
	scopedRes, err := svc.Search(context.Background(), core.SearchReq{
		Query: "single",
		Scope: core.SearchScope{DossierID: realID},
	})
	if err != nil {
		t.Fatalf("Scoped search failed: %v", err)
	}
	scopedHits := scopedRes.Data.([]core.Hit)
	if len(scopedHits) != 2 {
		t.Errorf("expected 2 scoped hits, got %d", len(scopedHits))
	}

	// 4. Perform scoped search to a different/non-existent dossier
	_, err = svc.Search(context.Background(), core.SearchReq{
		Query: "single",
		Scope: core.SearchScope{DossierID: "dos_nonexistent"},
	})
	if err == nil {
		t.Errorf("expected error for nonexistent dossier scope, got nil")
	}

	// 5. Run context refresh
	refreshRes, err := svc.ContextRefresh(context.Background())
	if err != nil {
		t.Fatalf("ContextRefresh failed: %v", err)
	}
	if !refreshRes.OK {
		t.Fatalf("ContextRefresh returned not OK")
	}

	// Read generated library.md
	libBytes, err := os.ReadFile(filepath.Join(tempHome, "context", "library.md"))
	if err != nil {
		t.Fatalf("failed to read library.md: %v", err)
	}
	libContent := string(libBytes)

	if !strings.Contains(libContent, "Harness:") {
		t.Errorf("expected Harness: header in library.md, got:\n%s", libContent)
	}
	if !strings.Contains(libContent, "Chainlink core engine") {
		t.Errorf("expected 'Chainlink core engine' in library.md, got:\n%s", libContent)
	}
	if !strings.Contains(libContent, "Implement the MCP server") {
		t.Errorf("expected next action 'Implement the MCP server' in library.md, got:\n%s", libContent)
	}
}
