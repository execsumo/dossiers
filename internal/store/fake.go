package store

import (
	"dossier/internal/core"
	"fmt"
)

// FakeStore implements core.Store in-memory for core unit tests.
type FakeStore struct {
	Dossiers  map[string]*core.Dossier
	Revisions map[string]core.Revision
	Artifacts map[string][]core.Artifact
	Audits    map[string][]core.AuditEvent
	Sessions  map[string]*core.SessionBinding
	Conflicts map[string]*core.Conflict
}

// NewFakeStore instantiates an in-memory FakeStore.
func NewFakeStore() *FakeStore {
	return &FakeStore{
		Dossiers:  make(map[string]*core.Dossier),
		Revisions: make(map[string]core.Revision),
		Artifacts: make(map[string][]core.Artifact),
		Audits:    make(map[string][]core.AuditEvent),
		Sessions:  make(map[string]*core.SessionBinding),
		Conflicts: make(map[string]*core.Conflict),
	}
}

func (f *FakeStore) Init() error {
	return nil
}

func (f *FakeStore) Read(slugOrID string) (*core.Dossier, core.Revision, error) {
	d, ok := f.Dossiers[slugOrID]
	if !ok {
		// Try by ID or Slug check
		for _, dos := range f.Dossiers {
			if dos.Frontmatter.ID == slugOrID || dos.Frontmatter.Slug == slugOrID {
				return dos, f.Revisions[dos.Frontmatter.ID], nil
			}
		}
		return nil, "", core.NewError(core.ErrNotFound, fmt.Sprintf("dossier %q not found in fake store", slugOrID))
	}
	return d, f.Revisions[d.Frontmatter.ID], nil
}

func (f *FakeStore) List(statusFilter string) ([]core.Frontmatter, error) {
	list := []core.Frontmatter{}
	for _, d := range f.Dossiers {
		if statusFilter == "all" || string(d.Frontmatter.Status) == statusFilter {
			list = append(list, d.Frontmatter)
		}
	}
	return list, nil
}

func (f *FakeStore) Write(d *core.Dossier, base core.Revision) (core.Revision, error) {
	id := d.Frontmatter.ID
	currentRev := f.Revisions[id]
	if currentRev != base {
		return "", core.NewError(core.ErrConcurrentEdit, "concurrent edit detected")
	}

	f.Dossiers[id] = d
	newRev := core.Revision(fmt.Sprintf("rev_fake_%d", len(f.Revisions)+1))
	f.Revisions[id] = newRev
	return newRev, nil
}

func (f *FakeStore) WriteArtifact(dossierID string, a *core.Artifact) error {
	f.Artifacts[dossierID] = append(f.Artifacts[dossierID], *a)
	return nil
}

func (f *FakeStore) ReadArtifact(dossierID string, artifactID string) (*core.Artifact, error) {
	for _, a := range f.Artifacts[dossierID] {
		if a.ID == artifactID {
			return &a, nil
		}
	}
	return nil, core.NewError(core.ErrNotFound, "artifact not found")
}

func (f *FakeStore) ListArtifacts(dossierID string) ([]core.Artifact, error) {
	return f.Artifacts[dossierID], nil
}

func (f *FakeStore) AppendAudit(dossierID string, e core.AuditEvent) error {
	f.Audits[dossierID] = append(f.Audits[dossierID], e)
	return nil
}

func (f *FakeStore) ReadAuditLog(dossierID string) ([]core.AuditEvent, error) {
	return f.Audits[dossierID], nil
}

func (f *FakeStore) SaveSessionBinding(binding *core.SessionBinding) error {
	f.Sessions[binding.SessionBindingID] = binding
	return nil
}

func (f *FakeStore) GetSessionBinding(sessionID string) (*core.SessionBinding, error) {
	binding, ok := f.Sessions[sessionID]
	if !ok {
		return nil, core.NewError(core.ErrNotFound, "session binding not found")
	}
	return binding, nil
}

func (f *FakeStore) ClearSessionBinding(sessionID string) error {
	delete(f.Sessions, sessionID)
	return nil
}

func (f *FakeStore) WriteConflict(conflict *core.Conflict) error {
	f.Conflicts[conflict.ID] = conflict
	return nil
}

func (f *FakeStore) ReadConflict(conflictID string) (*core.Conflict, error) {
	conflict, ok := f.Conflicts[conflictID]
	if !ok {
		return nil, core.NewError(core.ErrNotFound, "conflict not found")
	}
	return conflict, nil
}

func (f *FakeStore) ListConflicts() ([]core.Conflict, error) {
	list := []core.Conflict{}
	for _, c := range f.Conflicts {
		list = append(list, *c)
	}
	return list, nil
}

func (f *FakeStore) WriteLibraryContext(data core.LibraryData) error {
	return nil
}
