package store

import (
	"bytes"
	"dossier/assets"
	"dossier/internal/core"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// FSStore implements core.Store port interface using the local filesystem.
type FSStore struct {
	dossierHome string
}

// NewFSStore instantiates a filesystem-backed store.
func NewFSStore(dossierHome string) *FSStore {
	return &FSStore{
		dossierHome: dossierHome,
	}
}

// Init creates storage directories, writes the guide, and generates the library context.
func (s *FSStore) Init() error {
	// 1. Create root, context, and sessions directories
	dirs := []string{
		s.dossierHome,
		filepath.Join(s.dossierHome, "context"),
		filepath.Join(s.dossierHome, "sessions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 2. Extract and write guide.md from embedded assets
	guideContent, err := assets.FS.ReadFile("guide.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded guide: %w", err)
	}
	guidePath := filepath.Join(s.dossierHome, "context", "guide.md")
	if err := os.WriteFile(guidePath, guideContent, 0644); err != nil {
		return fmt.Errorf("failed to write guide.md: %w", err)
	}

	// 3. Render and write library.md fallback context
	tmplContent, err := assets.FS.ReadFile("library.tmpl.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded library template: %w", err)
	}

	tmpl, err := template.New("library").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse library template: %w", err)
	}

	// For the initial library.md, write empty capabilities and lists
	var rendered bytes.Buffer
	data := map[string]any{
		"Harness": "CLI",
		"Capabilities": map[string]bool{
			"MCP":               false,
			"SessionStartHook":  false,
			"SessionEndHook":    false,
			"PreCompactionHook": false,
			"TranscriptCapture": false,
		},
		"Warnings":     []string{"No harness session active. Run from within a supported client harness for full integration."},
		"OpenDossiers": []any{},
	}
	if err := tmpl.Execute(&rendered, data); err != nil {
		return fmt.Errorf("failed to execute library template: %w", err)
	}

	libraryPath := filepath.Join(s.dossierHome, "context", "library.md")
	if err := os.WriteFile(libraryPath, rendered.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write library.md: %w", err)
	}

	return nil
}

// List scans the store for Dossier frontmatters. (Baseline for doctor/init checks)
func (s *FSStore) List(statusFilter string) ([]core.Frontmatter, error) {
	// For Milestone 1 skeleton, we return an empty list.
	// In M2 we will implement full recursive scanning.
	return []core.Frontmatter{}, nil
}

// Read reads a Dossier and its current Revision.
func (s *FSStore) Read(slugOrID string) (*core.Dossier, core.Revision, error) {
	return nil, "", core.NewError(core.ErrNotFound, "read unimplemented in Milestone 1")
}

// Write writes a Dossier atomically checking concurrency.
func (s *FSStore) Write(d *core.Dossier, base core.Revision) (core.Revision, error) {
	return "", core.NewError(core.ErrInternal, "write unimplemented in Milestone 1")
}

// WriteArtifact stores a source artifact file atomically.
func (s *FSStore) WriteArtifact(dossierID string, a *core.Artifact) error {
	return core.NewError(core.ErrInternal, "write artifact unimplemented in Milestone 1")
}

// ReadArtifact retrieves an artifact from store.
func (s *FSStore) ReadArtifact(dossierID string, artifactID string) (*core.Artifact, error) {
	return nil, core.NewError(core.ErrNotFound, "read artifact unimplemented in Milestone 1")
}

// ListArtifacts lists all artifacts associated with a Dossier.
func (s *FSStore) ListArtifacts(dossierID string) ([]core.Artifact, error) {
	return nil, nil
}

// AppendAudit logs a JSONL event.
func (s *FSStore) AppendAudit(dossierID string, e core.AuditEvent) error {
	return core.NewError(core.ErrInternal, "append audit unimplemented in Milestone 1")
}

// ReadAuditLog reads the audit events log.
func (s *FSStore) ReadAuditLog(dossierID string) ([]core.AuditEvent, error) {
	return nil, nil
}

// SaveSessionBinding saves session bindings.
func (s *FSStore) SaveSessionBinding(binding *core.SessionBinding) error {
	return core.NewError(core.ErrInternal, "save session unimplemented in Milestone 1")
}

// GetSessionBinding retrieves session bindings.
func (s *FSStore) GetSessionBinding(sessionID string) (*core.SessionBinding, error) {
	return nil, core.NewError(core.ErrNotFound, "get session unimplemented in Milestone 1")
}

// ClearSessionBinding deletes session bindings.
func (s *FSStore) ClearSessionBinding(sessionID string) error {
	return core.NewError(core.ErrInternal, "clear session unimplemented in Milestone 1")
}

// WriteConflict logs conflict states.
func (s *FSStore) WriteConflict(conflict *core.Conflict) error {
	return core.NewError(core.ErrInternal, "write conflict unimplemented in Milestone 1")
}

// ReadConflict retrieves conflict states.
func (s *FSStore) ReadConflict(conflictID string) (*core.Conflict, error) {
	return nil, core.NewError(core.ErrNotFound, "read conflict unimplemented in Milestone 1")
}

// ListConflicts lists active unresolved conflicts.
func (s *FSStore) ListConflicts() ([]core.Conflict, error) {
	return nil, nil
}
