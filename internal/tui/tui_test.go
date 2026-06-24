package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"dossier/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

type testClock struct{}

func (testClock) Now() time.Time {
	return time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
}

type testTokenizer struct{}

func (testTokenizer) Estimate(t string) int {
	return len(t)
}

type testSearcher struct{}

func (testSearcher) Search(ctx context.Context, q string, s core.SearchScope) ([]core.Hit, error) {
	return nil, nil
}

type testHarnessRegistry struct{}

func (testHarnessRegistry) All() []core.Harness {
	return nil
}

func (testHarnessRegistry) Get(name string) (core.Harness, error) {
	return nil, nil
}

type testStore struct {
	dossiers  map[string]*core.Dossier
	bindings  map[string]*core.SessionBinding
	conflicts map[string]*core.Conflict
	artifacts map[string][]core.Artifact
	auditLogs map[string][]core.AuditEvent
}

func newTestStore() *testStore {
	return &testStore{
		dossiers:  make(map[string]*core.Dossier),
		bindings:  make(map[string]*core.SessionBinding),
		conflicts: make(map[string]*core.Conflict),
		artifacts: make(map[string][]core.Artifact),
		auditLogs: make(map[string][]core.AuditEvent),
	}
}

func (s *testStore) Init() error { return nil }

func (s *testStore) Read(id string) (*core.Dossier, core.Revision, error) {
	d, ok := s.dossiers[id]
	if !ok {
		// Try searching by slug
		for _, dos := range s.dossiers {
			if dos.Frontmatter.Slug == id {
				return dos, "rev1", nil
			}
		}
		return nil, "", fmt.Errorf("not found")
	}
	return d, "rev1", nil
}

func (s *testStore) ReadRevision(id string, rev core.Revision) (*core.Dossier, error) {
	d, _, err := s.Read(id)
	return d, err
}

func (s *testStore) List(filter string) ([]core.Frontmatter, error) {
	var list []core.Frontmatter
	for _, d := range s.dossiers {
		list = append(list, d.Frontmatter)
	}
	return list, nil
}

func (s *testStore) Write(d *core.Dossier, base core.Revision) (core.Revision, error) {
	s.dossiers[d.Frontmatter.ID] = d
	return "rev_new", nil
}

func (s *testStore) WriteArtifact(dossierID string, a *core.Artifact) error {
	s.artifacts[dossierID] = append(s.artifacts[dossierID], *a)
	return nil
}

func (s *testStore) ReadArtifact(dossierID string, artifactID string) (*core.Artifact, error) {
	for _, a := range s.artifacts[dossierID] {
		if a.ID == artifactID {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (s *testStore) ListArtifacts(dossierID string) ([]core.Artifact, error) {
	return s.artifacts[dossierID], nil
}

func (s *testStore) AppendAudit(dossierID string, e core.AuditEvent) error {
	s.auditLogs[dossierID] = append(s.auditLogs[dossierID], e)
	return nil
}

func (s *testStore) ReadAuditLog(dossierID string) ([]core.AuditEvent, error) {
	return s.auditLogs[dossierID], nil
}

func (s *testStore) SaveSessionBinding(binding *core.SessionBinding) error {
	s.bindings[binding.SessionBindingID] = binding
	return nil
}

func (s *testStore) GetSessionBinding(sessionID string) (*core.SessionBinding, error) {
	b, ok := s.bindings[sessionID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return b, nil
}

func (s *testStore) ClearSessionBinding(sessionID string) error {
	delete(s.bindings, sessionID)
	return nil
}

func (s *testStore) WriteConflict(conflict *core.Conflict) error {
	s.conflicts[conflict.ID] = conflict
	return nil
}

func (s *testStore) ReadConflict(conflictID string) (*core.Conflict, error) {
	c, ok := s.conflicts[conflictID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (s *testStore) ListConflicts() ([]core.Conflict, error) {
	var list []core.Conflict
	for _, c := range s.conflicts {
		list = append(list, *c)
	}
	return list, nil
}

func (s *testStore) WriteLibraryContext(data core.LibraryData) error { return nil }

func setupTestService(store core.Store) *core.Service {
	return core.NewService(
		store,
		testSearcher{},
		testTokenizer{},
		testHarnessRegistry{},
		testClock{},
		core.Config{DossierHome: "/tmp/dossier_home", TokenTarget: 1000},
	)
}

func TestTUI_Dashboard(t *testing.T) {
	store := newTestStore()
	// Seed a dossier
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Project Alpha",
			Slug:          "project-alpha",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
	}
	svc := setupTestService(store)

	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Trigger Init cmd
	initCmd := m.Init()
	if initCmd == nil {
		t.Fatal("expected Init cmd to not be nil")
	}

	// Verify view rendering before loading items
	viewStr := m.View()
	if !strings.Contains(viewStr, "Loading dossiers") {
		t.Errorf("expected view to contain loading indicator, got:\n%s", viewStr)
	}

	// Perform the async load manually
	listMsg := m.listDossiersCmd()()

	// Update the model with results
	var newM tea.Model
	newM, _ = m.Update(listMsg)

	updatedModel := newM.(Model)
	if len(updatedModel.items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(updatedModel.items))
	}

	// Trigger a mock window resize to initialize columns and height
	newM, _ = updatedModel.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	resizedModel := newM.(Model)

	// Verify dossier name is rendered
	viewWithItems := resizedModel.View()
	if !strings.Contains(viewWithItems, "Project Alpha") {
		t.Errorf("expected view to contain 'Project Alpha', got:\n%s", viewWithItems)
	}
}

func TestTUI_Detail(t *testing.T) {
	store := newTestStore()
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Project Alpha",
			Slug:          "project-alpha",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
		DistilledState: core.DistilledState{
			Body: "This is the distilled state of Alpha",
		},
	}
	svc := setupTestService(store)
	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Load list items
	listMsg := m.listDossiersCmd()()
	newM, _ := m.Update(listMsg)
	m = newM.(Model)

	// Move cursor down to select the actual item, not the separator row
	m.table.MoveDown(1)

	// Dashboard: Enter to view detail
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected enter key to return a recall command")
	}

	// Run command
	recallMsg := cmd()
	newM, _ = m.Update(recallMsg)
	m = newM.(Model)

	if m.currentView != ViewDetail {
		t.Errorf("expected view to be ViewDetail, got %v", m.currentView)
	}

	viewStr := m.View()
	cleanView := stripANSI(viewStr)
	if !strings.Contains(cleanView, "This is the distilled state of Alpha") {
		t.Errorf("expected view to contain distilled state, got:\n%s", cleanView)
	}

	// Press esc to go back
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected view to be ViewDashboard after esc, got %v", m.currentView)
	}
}

func TestTUI_InlineEditing(t *testing.T) {
	store := newTestStore()
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Project Alpha",
			Slug:          "project-alpha",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
	}
	svc := setupTestService(store)
	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Load list items
	listMsg := m.listDossiersCmd()()
	newM, _ := m.Update(listMsg)
	m = newM.(Model)

	// Move cursor down to select actual item
	m.table.MoveDown(1)

	// 1. Test Status Editing (press 's')
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = newM.(Model)
	if m.currentView != ViewStatusPicker {
		t.Fatalf("expected view ViewStatusPicker, got %v", m.currentView)
	}

	// Press enter to confirm selection
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected status picker enter to return setStatus command")
	}
	mutMsg := cmd()
	newM, cmd = m.Update(mutMsg)
	m = newM.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected to return to ViewDashboard after status update, got %v", m.currentView)
	}

	// 2. Test Next Action Editing (press 'n')
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = newM.(Model)
	if m.currentView != ViewNextActionEditor {
		t.Fatalf("expected view ViewNextActionEditor, got %v", m.currentView)
	}
	m.nextActionInput.SetValue("New Next Action")
	// Press enter
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected next action enter to return save command")
	}
	mutMsg = cmd()
	newM, cmd = m.Update(mutMsg)
	m = newM.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected to return to ViewDashboard, got %v", m.currentView)
	}

	// 3. Test Priority Editing (press 'p')
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = newM.(Model)
	if m.currentView != ViewPriorityEditor {
		t.Fatalf("expected view ViewPriorityEditor, got %v", m.currentView)
	}
	// Focus is initially 0 (Importance). Press down until Save button (focus 3) is focused
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // focus 1 (Urgency)
	m = newM.(Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // focus 2 (Due Date)
	m = newM.(Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // focus 3 (Save button)
	m = newM.(Model)
	if m.priorityFocus != 3 {
		t.Fatalf("expected focus to be 3 (Save), got %d", m.priorityFocus)
	}

	// Press enter on Save button
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected save enter to return save command")
	}
	mutMsg = cmd()
	newM, cmd = m.Update(mutMsg)
	m = newM.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected to return to ViewDashboard after priority save, got %v", m.currentView)
	}
}

// TestTUI_NoActiveBinding asserts the TUI exposes no per-session "active"
// affordance: pressing 'a' is a no-op, and the dashboard has no ★ marker. The
// per-session active binding (Switch) is intentionally not reachable from the
// TUI — see ADR 0004 and BUILD-DECISIONS B9.
func TestTUI_NoActiveBinding(t *testing.T) {
	store := newTestStore()
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Project Alpha",
			Slug:          "project-alpha",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
	}
	svc := setupTestService(store)
	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Load list items
	listMsg := m.listDossiersCmd()()
	newM, _ := m.Update(listMsg)
	m = newM.(Model)

	// Press 'a': must be a no-op (no command, view unchanged).
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = newM.(Model)
	if cmd != nil {
		t.Error("expected 'a' key to be a no-op, but it returned a command")
	}
	if m.currentView != ViewDashboard {
		t.Errorf("expected to remain on ViewDashboard, got %v", m.currentView)
	}

	// The dashboard must not render an active-dossier star marker.
	viewStr := m.View()
	if strings.Contains(viewStr, "★") {
		t.Errorf("expected no active dossier star marker, got:\n%s", viewStr)
	}
}

func TestTUI_Link(t *testing.T) {
	store := newTestStore()
	// Seed two dossiers matching "Alpha"
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Alpha project",
			Slug:          "alpha-proj",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
	}
	store.dossiers["dos2"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos2",
			Name:          "Alpha team",
			Slug:          "alpha-team",
			Status:        core.StatusActive,
			LastTouchedAt: testClock{}.Now(),
		},
	}
	svc := setupTestService(store)
	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Load list items
	listMsg := m.listDossiersCmd()()
	newM, _ := m.Update(listMsg)
	m = newM.(Model)

	// Press 'l' key to link
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = newM.(Model)
	if m.currentView != ViewLinkInput {
		t.Fatalf("expected view ViewLinkInput, got %v", m.currentView)
	}

	m.linkTextInput.SetValue("Alpha content")
	// Press enter to link
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected enter key to return link analyze command")
	}

	// Run first link cmd which detects ambiguity
	resMsg := cmd()
	newM, cmd = m.Update(resMsg)
	m = newM.(Model)

	if m.currentView != ViewLinkSelector {
		t.Fatalf("expected view ViewLinkSelector, got %v", m.currentView)
	}
	if len(m.linkSuggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(m.linkSuggestions))
	}

	// Select first suggestion and press enter
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected confirm link command")
	}

	confirmMsg := cmd()
	newM, cmd = m.Update(confirmMsg)
	m = newM.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected view to return to ViewDashboard, got %v", m.currentView)
	}
}

func TestTUI_Merge(t *testing.T) {
	store := newTestStore()
	// Seed two dossiers with incompatible statuses to force merge conflict
	store.dossiers["dos1"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos1",
			Name:          "Source Dossier",
			Slug:          "source-dossier",
			Status:        core.StatusActive,
			NextAction:    "Action A",
			LastTouchedAt: testClock{}.Now(),
		},
		DistilledState: core.DistilledState{
			Body: "Distilled A",
		},
	}
	store.dossiers["dos2"] = &core.Dossier{
		Frontmatter: core.Frontmatter{
			ID:            "dos2",
			Name:          "Target Dossier",
			Slug:          "target-dossier",
			Status:        core.StatusBlocked,
			NextAction:    "Action B",
			LastTouchedAt: testClock{}.Now(),
		},
		DistilledState: core.DistilledState{
			Body: "Distilled B",
		},
	}
	svc := setupTestService(store)
	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	// Load list items
	listMsg := m.listDossiersCmd()()
	newM, _ := m.Update(listMsg)
	m = newM.(Model)

	// Move cursor down to select actual item
	m.table.MoveDown(1)

	// Press 'm' to merge Source Dossier
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m = newM.(Model)

	if m.currentView != ViewMergeSelector {
		t.Fatalf("expected view ViewMergeSelector, got %v", m.currentView)
	}
	if len(m.mergeTargets) != 1 {
		t.Fatalf("expected 1 merge target, got %d", len(m.mergeTargets))
	}

	// Press enter to merge into Target Dossier
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected merge command")
	}

	// Run command which will fail with a conflict
	resMsg := cmd()
	newM, cmd = m.Update(resMsg)
	m = newM.(Model)

	if m.currentView != ViewMergeConflictResolver {
		t.Fatalf("expected ViewMergeConflictResolver, got %v", m.currentView)
	}
	if m.mergeConflict == nil {
		t.Fatal("expected mergeConflict details to be populated")
	}

	// Select Resolve (focus 0) and press Enter
	m.conflictResolverCursor = 0
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(Model)
	if cmd == nil {
		t.Fatal("expected resolve merge command")
	}

	resolveMsg := cmd()
	newM, cmd = m.Update(resolveMsg)
	m = newM.(Model)

	if m.currentView != ViewDashboard {
		t.Errorf("expected view to return to ViewDashboard, got %v", m.currentView)
	}
}

// TestHeaderHasNoSession asserts the TUI carries no session identity: the header
// shows only the app title, with no "Session:" or "Active:" field and no
// standalone-session warning footer. See ADR 0004.
func TestHeaderHasNoSession(t *testing.T) {
	store := newTestStore()
	svc := setupTestService(store)

	m := NewModel(svc)
	m.width = 100
	m.height = 40
	m.recalculateTableLayout()

	view := m.View()
	if !strings.Contains(view, "DOSSIER TUI") {
		t.Errorf("expected view to contain the 'DOSSIER TUI' title, got:\n%s", view)
	}
	for _, forbidden := range []string{"Session:", "Active:", "No active Claude session"} {
		if strings.Contains(view, forbidden) {
			t.Errorf("expected view NOT to contain %q, got:\n%s", forbidden, view)
		}
	}
}
