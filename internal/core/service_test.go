package core

import (
	"context"
	"testing"
	"time"
)

type mockTokenizer struct{}

func (m *mockTokenizer) Estimate(text string) int {
	return len(text) / 4
}

type mockSearcher struct{}

func (m *mockSearcher) Search(ctx context.Context, query string, scope SearchScope) ([]Hit, error) {
	return nil, nil
}

type mockHarnessRegistry struct{}

func (m *mockHarnessRegistry) All() []Harness {
	return nil
}
func (m *mockHarnessRegistry) Get(name string) (Harness, error) {
	return nil, nil
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

type localFakeStore struct {
	dossiers  map[string]*Dossier
	revisions map[string]Revision
	audits    map[string][]AuditEvent
	history   map[Revision]*Dossier
}

func newLocalFakeStore() *localFakeStore {
	return &localFakeStore{
		dossiers:  make(map[string]*Dossier),
		revisions: make(map[string]Revision),
		audits:    make(map[string][]AuditEvent),
		history:   make(map[Revision]*Dossier),
	}
}

func (f *localFakeStore) Init() error { return nil }
func (f *localFakeStore) Read(id string) (*Dossier, Revision, error) {
	d, ok := f.dossiers[id]
	if !ok {
		return nil, "", NewError(ErrNotFound, "not found")
	}
	cp := *d
	return &cp, f.revisions[id], nil
}
func (f *localFakeStore) ReadRevision(id string, rev Revision) (*Dossier, error) {
	d, ok := f.history[rev]
	if !ok {
		currRev := f.revisions[id]
		if currRev == rev {
			cp := *f.dossiers[id]
			return &cp, nil
		}
		return nil, NewError(ErrNotFound, "revision not found")
	}
	cp := *d
	return &cp, nil
}
func (f *localFakeStore) List(filter string) ([]Frontmatter, error) {
	var list []Frontmatter
	for _, d := range f.dossiers {
		list = append(list, d.Frontmatter)
	}
	return list, nil
}
func (f *localFakeStore) Write(d *Dossier, base Revision) (Revision, error) {
	if d.Frontmatter.ID == "" {
		d.Frontmatter.ID = "dos_fake_id"
	}
	if d.Frontmatter.Slug == "" {
		d.Frontmatter.Slug = "fake-slug"
	}

	// Check concurrency
	if base != "" {
		if currRev, ok := f.revisions[d.Frontmatter.ID]; ok && currRev != base {
			return "", NewError(ErrConcurrentEdit, "concurrency mismatch")
		}
	}

	// Save existing to history before overwriting
	if currentRev, ok := f.revisions[d.Frontmatter.ID]; ok {
		if existing, ok := f.dossiers[d.Frontmatter.ID]; ok {
			cp := *existing
			f.history[currentRev] = &cp
		}
	}

	f.dossiers[d.Frontmatter.ID] = d
	rev := CalculateRevision(d.Frontmatter, d.DistilledState.Body, nil)
	f.revisions[d.Frontmatter.ID] = rev
	return rev, nil
}
func (f *localFakeStore) WriteArtifact(id string, a *Artifact) error              { return nil }
func (f *localFakeStore) ReadArtifact(id string, artID string) (*Artifact, error) { return nil, nil }
func (f *localFakeStore) ListArtifacts(id string) ([]Artifact, error)             { return nil, nil }
func (f *localFakeStore) AppendAudit(id string, e AuditEvent) error {
	f.audits[id] = append(f.audits[id], e)
	return nil
}
func (f *localFakeStore) ReadAuditLog(id string) ([]AuditEvent, error)         { return f.audits[id], nil }
func (f *localFakeStore) SaveSessionBinding(b *SessionBinding) error           { return nil }
func (f *localFakeStore) GetSessionBinding(id string) (*SessionBinding, error) { return nil, nil }
func (f *localFakeStore) ClearSessionBinding(id string) error                  { return nil }
func (f *localFakeStore) WriteConflict(c *Conflict) error                      { return nil }
func (f *localFakeStore) ReadConflict(id string) (*Conflict, error)            { return nil, nil }
func (f *localFakeStore) ListConflicts() ([]Conflict, error)                   { return nil, nil }
func (f *localFakeStore) WriteLibraryContext(data LibraryData) error           { return nil }

func TestServiceListAndRecall(t *testing.T) {
	fakeStore := newLocalFakeStore()
	tok := &mockTokenizer{}
	srch := &mockSearcher{}
	hreg := &mockHarnessRegistry{}
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	clk := &mockClock{now: now}
	cfg := Config{DossierHome: "/tmp/dossier-test", TokenTarget: 100}

	svc := NewService(fakeStore, srch, tok, hreg, clk, cfg)

	ctx := context.Background()
	saveReq := SaveReq{
		DistilledStateMarkdown: "# Test\n\n## Situation\nWorking fine.",
		FrontmatterUpdates: map[string]any{
			"name":       "Pricing model refresh",
			"status":     "active",
			"importance": "high",
			"urgency":    "medium",
		},
	}

	res, err := svc.Save(ctx, saveReq)
	if err != nil {
		t.Fatalf("Service.Save failed: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected Service.Save result to be OK")
	}

	listRes, err := svc.List(ctx, ListReq{})
	if err != nil {
		t.Fatalf("Service.List failed: %v", err)
	}
	if !listRes.OK {
		t.Errorf("expected list response to be OK")
	}

	// Recall
	var dossierID string
	for id := range fakeStore.dossiers {
		dossierID = id
		break
	}

	recallRes, err := svc.Recall(ctx, RecallReq{ID: dossierID})
	if err != nil {
		t.Fatalf("Service.Recall failed: %v", err)
	}
	if !recallRes.OK {
		t.Fatalf("expected Recall to be OK")
	}

	// Set status to archived
	archiveRes, err := svc.Archive(ctx, ArchiveReq{ID: dossierID})
	if err != nil {
		t.Fatalf("Service.Archive failed: %v", err)
	}
	if !archiveRes.OK {
		t.Fatalf("expected Archive to be OK")
	}

	// Read back and verify status is archived
	d, _, err := fakeStore.Read(dossierID)
	if err != nil {
		t.Fatalf("failed to read from store: %v", err)
	}
	if d.Frontmatter.Status != StatusArchived {
		t.Errorf("expected status to be archived, got %q", d.Frontmatter.Status)
	}
}
