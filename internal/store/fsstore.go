package store

import (
	"bytes"
	"dossier/assets"
	"dossier/internal/core"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

	guideContent, err := assets.FS.ReadFile("guide.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded guide: %w", err)
	}
	guidePath := filepath.Join(s.dossierHome, "context", "guide.md")
	if err := os.WriteFile(guidePath, guideContent, 0644); err != nil {
		return fmt.Errorf("failed to write guide.md: %w", err)
	}

	tmplContent, err := assets.FS.ReadFile("library.tmpl.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded library template: %w", err)
	}

	tmpl, err := template.New("library").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse library template: %w", err)
	}

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

// List scans the store for Dossier frontmatters.
func (s *FSStore) List(statusFilter string) ([]core.Frontmatter, error) {
	entries, err := os.ReadDir(s.dossierHome)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var list []core.Frontmatter
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "context" || name == "sessions" || strings.HasPrefix(name, ".") {
			continue
		}

		dirPath := filepath.Join(s.dossierHome, name)
		dossierPath := filepath.Join(dirPath, "dossier.md")
		data, err := os.ReadFile(dossierPath)
		if err != nil {
			continue
		}

		fm, _, err := ParseDossierFile(string(data))
		if err != nil {
			continue
		}

		if statusFilter == "all" || string(fm.Status) == statusFilter {
			list = append(list, *fm)
		}
	}
	return list, nil
}

// Read reads a Dossier and its current Revision.
func (s *FSStore) Read(slugOrID string) (*core.Dossier, core.Revision, error) {
	dossierDir, err := s.findDossierDir(slugOrID)
	if err != nil {
		return nil, "", err
	}

	dossierPath := filepath.Join(dossierDir, "dossier.md")
	data, err := os.ReadFile(dossierPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read dossier.md: %w", err)
	}

	fm, body, err := ParseDossierFile(string(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse dossier file: %w", err)
	}

	artifacts, _ := s.listArtifactsInternal(fm.ID, dossierDir)
	rev := core.CalculateRevision(*fm, body, artifacts)

	return &core.Dossier{
		Frontmatter:    *fm,
		DistilledState: core.DistilledState{Body: body},
	}, rev, nil
}

// Write writes a Dossier atomically checking concurrency.
func (s *FSStore) Write(d *core.Dossier, base core.Revision) (core.Revision, error) {
	slug := d.Frontmatter.Slug
	if slug == "" {
		slug = GenerateSlug(d.Frontmatter.Name)
		d.Frontmatter.Slug = slug
	}
	id := d.Frontmatter.ID
	if id == "" {
		var err error
		id, err = GenerateID("dos_")
		if err != nil {
			return "", err
		}
		d.Frontmatter.ID = id
	}

	dossierDir := filepath.Join(s.dossierHome, slug)

	// If editing an existing dossier, locate its path by ID first to handle renamed slugs
	var existingDir string
	if base != "" {
		existingDir, _ = s.findDossierDir(id)
	}

	if existingDir != "" {
		dossierDir = existingDir
	} else {
		// New dossier, check for slug collision
		if _, err := os.Stat(dossierDir); err == nil {
			slug = SlugWithSuffix(slug, id)
			d.Frontmatter.Slug = slug
			dossierDir = filepath.Join(s.dossierHome, slug)
		}
	}

	if err := os.MkdirAll(dossierDir, 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dossierDir, "artifacts"), 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dossierDir, "conflicts"), 0755); err != nil {
		return "", err
	}

	// Acquire Lock
	lock, err := Lock(filepath.Join(dossierDir, ".lock"))
	if err != nil {
		return "", fmt.Errorf("failed to acquire dossier lock: %w", err)
	}
	defer lock.Unlock()

	dossierPath := filepath.Join(dossierDir, "dossier.md")
	var currentRevision core.Revision
	var currentArtifacts []core.Artifact

	if _, err := os.Stat(dossierPath); err == nil {
		data, err := os.ReadFile(dossierPath)
		if err != nil {
			return "", fmt.Errorf("failed to read existing dossier: %w", err)
		}
		currFM, currBody, err := ParseDossierFile(string(data))
		if err != nil {
			return "", fmt.Errorf("failed to parse existing dossier: %w", err)
		}

		currentArtifacts, _ = s.listArtifactsInternal(id, dossierDir)
		currentRevision = core.CalculateRevision(*currFM, currBody, currentArtifacts)

		if base != "" && currentRevision != base {
			return "", core.NewError(core.ErrConcurrentEdit, fmt.Sprintf("concurrency mismatch: base is %q but current is %q", base, currentRevision))
		}
	}

	// Update dates
	d.Frontmatter.UpdatedAt = time.Now()
	if d.Frontmatter.CreatedAt.IsZero() {
		d.Frontmatter.CreatedAt = d.Frontmatter.UpdatedAt
	}
	if d.Frontmatter.LastTouchedAt.IsZero() {
		d.Frontmatter.LastTouchedAt = d.Frontmatter.UpdatedAt
	}

	if err := d.Frontmatter.Validate(); err != nil {
		return "", core.WrapError(core.ErrInvalidFrontmatter, "invalid frontmatter details", err)
	}

	// Write temp file
	tempFile, err := os.CreateTemp(dossierDir, "dossier.md.tmp.*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)

	serialized, err := FormatDossierFile(d.Frontmatter, d.DistilledState.Body)
	if err != nil {
		tempFile.Close()
		return "", err
	}

	if _, err := tempFile.WriteString(serialized); err != nil {
		tempFile.Close()
		return "", err
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return "", err
	}
	tempFile.Close()

	if err := os.Rename(tempName, dossierPath); err != nil {
		return "", fmt.Errorf("failed to atomically rename temp file: %w", err)
	}

	newRevision := core.CalculateRevision(d.Frontmatter, d.DistilledState.Body, currentArtifacts)
	return newRevision, nil
}

// WriteArtifact stores a source artifact file atomically.
func (s *FSStore) WriteArtifact(dossierID string, a *core.Artifact) error {
	dossierDir, err := s.findDossierDir(dossierID)
	if err != nil {
		return err
	}

	if a.ID == "" {
		a.ID, err = GenerateID("art_")
		if err != nil {
			return err
		}
	}
	a.DossierID = dossierID
	if a.CapturedAt.IsZero() {
		a.CapturedAt = time.Now()
	}
	if a.RefreshedAt.IsZero() {
		a.RefreshedAt = a.CapturedAt
	}
	a.SourceSizeBytes = int64(len(a.Content))

	if err := a.Validate(); err != nil {
		return core.WrapError(core.ErrInvalidFrontmatter, "invalid artifact details", err)
	}

	artifactsDir := filepath.Join(dossierDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return err
	}

	ext := "txt"
	if a.ContentFormat == core.ContentFormatMarkdown {
		ext = "md"
	} else if a.ContentFormat == core.ContentFormatJSON {
		ext = "json"
	}
	fileName := fmt.Sprintf("%s.%s", a.ID, ext)
	filePath := filepath.Join(artifactsDir, fileName)

	tempFile, err := os.CreateTemp(artifactsDir, fileName+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp artifact: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)

	serialized, err := formatArtifactFile(a)
	if err != nil {
		tempFile.Close()
		return err
	}

	if _, err := tempFile.WriteString(serialized); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return err
	}
	tempFile.Close()

	if err := os.Rename(tempName, filePath); err != nil {
		return fmt.Errorf("failed to rename temp artifact file: %w", err)
	}

	return nil
}

// ReadArtifact retrieves an artifact from store.
func (s *FSStore) ReadArtifact(dossierID string, artifactID string) (*core.Artifact, error) {
	dossierDir, err := s.findDossierDir(dossierID)
	if err != nil {
		return nil, err
	}

	artifactsDir := filepath.Join(dossierDir, "artifacts")
	exts := []string{"md", "json", "txt"}
	for _, ext := range exts {
		path := filepath.Join(artifactsDir, fmt.Sprintf("%s.%s", artifactID, ext))
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			return parseArtifactFile(string(data))
		}
	}

	return nil, core.NewError(core.ErrNotFound, fmt.Sprintf("artifact %q not found", artifactID))
}

// ListArtifacts lists all artifacts associated with a Dossier.
func (s *FSStore) ListArtifacts(dossierID string) ([]core.Artifact, error) {
	dossierDir, err := s.findDossierDir(dossierID)
	if err != nil {
		return nil, err
	}
	return s.listArtifactsInternal(dossierID, dossierDir)
}

func (s *FSStore) listArtifactsInternal(dossierID string, dossierDir string) ([]core.Artifact, error) {
	artifactsDir := filepath.Join(dossierDir, "artifacts")
	entries, err := os.ReadDir(artifactsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var list []core.Artifact
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(artifactsDir, entry.Name()))
		if err != nil {
			continue
		}

		art, err := parseArtifactFile(string(data))
		if err != nil {
			continue
		}
		list = append(list, *art)
	}
	return list, nil
}

// AppendAudit logs a JSONL event.
func (s *FSStore) AppendAudit(dossierID string, e core.AuditEvent) error {
	dossierDir, err := s.findDossierDir(dossierID)
	if err != nil {
		return err
	}
	return AppendAuditLine(filepath.Join(dossierDir, "audit.log"), e)
}

// ReadAuditLog reads the audit events log.
func (s *FSStore) ReadAuditLog(dossierID string) ([]core.AuditEvent, error) {
	dossierDir, err := s.findDossierDir(dossierID)
	if err != nil {
		return nil, err
	}
	return ReadAuditEntries(filepath.Join(dossierDir, "audit.log"))
}

// SaveSessionBinding saves session bindings.
func (s *FSStore) SaveSessionBinding(binding *core.SessionBinding) error {
	sessionsDir := filepath.Join(s.dossierHome, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", binding.SessionBindingID))
	data, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// GetSessionBinding retrieves session bindings.
func (s *FSStore) GetSessionBinding(sessionID string) (*core.SessionBinding, error) {
	filePath := filepath.Join(s.dossierHome, "sessions", fmt.Sprintf("%s.json", sessionID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, core.NewError(core.ErrNotFound, fmt.Sprintf("session %q not found", sessionID))
		}
		return nil, err
	}

	var binding core.SessionBinding
	if err := json.Unmarshal(data, &binding); err != nil {
		return nil, err
	}
	return &binding, nil
}

// ClearSessionBinding deletes session bindings.
func (s *FSStore) ClearSessionBinding(sessionID string) error {
	filePath := filepath.Join(s.dossierHome, "sessions", fmt.Sprintf("%s.json", sessionID))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WriteConflict logs conflict states.
func (s *FSStore) WriteConflict(conflict *core.Conflict) error {
	dossierDir, err := s.findDossierDir(conflict.DossierID)
	if err != nil {
		return err
	}

	if conflict.ID == "" {
		conflict.ID, err = GenerateID("conf_")
		if err != nil {
			return err
		}
	}

	conflictsDir := filepath.Join(dossierDir, "conflicts")
	if err := os.MkdirAll(conflictsDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(conflictsDir, fmt.Sprintf("%s.md", conflict.ID))
	serialized, err := formatConflictFile(conflict)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(serialized), 0644)
}

// ReadConflict retrieves conflict states.
func (s *FSStore) ReadConflict(conflictID string) (*core.Conflict, error) {
	entries, err := os.ReadDir(s.dossierHome)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "context" || name == "sessions" || strings.HasPrefix(name, ".") {
			continue
		}

		conflictPath := filepath.Join(s.dossierHome, name, "conflicts", fmt.Sprintf("%s.md", conflictID))
		if _, err := os.Stat(conflictPath); err == nil {
			data, err := os.ReadFile(conflictPath)
			if err != nil {
				return nil, err
			}
			return parseConflictFile(string(data))
		}
	}

	return nil, core.NewError(core.ErrNotFound, fmt.Sprintf("conflict %q not found", conflictID))
}

// ListConflicts lists active unresolved conflicts.
func (s *FSStore) ListConflicts() ([]core.Conflict, error) {
	entries, err := os.ReadDir(s.dossierHome)
	if err != nil {
		return nil, err
	}

	var list []core.Conflict
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "context" || name == "sessions" || strings.HasPrefix(name, ".") {
			continue
		}

		conflictsDir := filepath.Join(s.dossierHome, name, "conflicts")
		confEntries, err := os.ReadDir(conflictsDir)
		if err != nil {
			continue
		}

		for _, confEntry := range confEntries {
			if confEntry.IsDir() || strings.HasPrefix(confEntry.Name(), ".") {
				continue
			}

			filePath := filepath.Join(conflictsDir, confEntry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			c, err := parseConflictFile(string(data))
			if err != nil {
				continue
			}
			list = append(list, *c)
		}
	}

	return list, nil
}

// Private helper methods

func (s *FSStore) findDossierDir(slugOrID string) (string, error) {
	directPath := filepath.Join(s.dossierHome, slugOrID)
	if info, err := os.Stat(directPath); err == nil && info.IsDir() {
		if slugOrID != "context" && slugOrID != "sessions" {
			if _, err := os.Stat(filepath.Join(directPath, "dossier.md")); err == nil {
				return directPath, nil
			}
		}
	}

	entries, err := os.ReadDir(s.dossierHome)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "context" || name == "sessions" || strings.HasPrefix(name, ".") {
			continue
		}

		dirPath := filepath.Join(s.dossierHome, name)
		dossierPath := filepath.Join(dirPath, "dossier.md")
		data, err := os.ReadFile(dossierPath)
		if err != nil {
			continue
		}

		fm, _, err := ParseDossierFile(string(data))
		if err != nil {
			continue
		}

		if fm.ID == slugOrID || fm.Slug == slugOrID {
			return dirPath, nil
		}
	}

	return "", core.NewError(core.ErrNotFound, fmt.Sprintf("dossier %q not found", slugOrID))
}

// Public Parsing helpers exported for store integration tests

func ParseDossierFile(content string) (*core.Frontmatter, string, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, "", fmt.Errorf("missing frontmatter delimiters")
	}

	var fm core.Frontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, "", err
	}
	return &fm, strings.TrimPrefix(parts[2], "\n"), nil
}

func FormatDossierFile(fm core.Frontmatter, body string) (string, error) {
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\n%s---\n%s", string(yamlBytes), body), nil
}

func parseArtifactFile(content string) (*core.Artifact, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("missing artifact frontmatter delimiters")
	}

	var art core.Artifact
	if err := yaml.Unmarshal([]byte(parts[1]), &art); err != nil {
		return nil, err
	}
	art.Content = strings.TrimPrefix(parts[2], "\n")
	return &art, nil
}

func formatArtifactFile(a *core.Artifact) (string, error) {
	yamlBytes, err := yaml.Marshal(a)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\n%s---\n%s", string(yamlBytes), a.Content), nil
}

func formatConflictFile(c *core.Conflict) (string, error) {
	yamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(yamlBytes)
	sb.WriteString("---\n")
	sb.WriteString("## Rejected proposal\n")
	sb.WriteString(c.RejectedBody)
	sb.WriteString("\n\n## Diff against current\n")
	sb.WriteString(c.DiffAgainstCurrent)
	return sb.String(), nil
}

func parseConflictFile(content string) (*core.Conflict, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("missing conflict delimiters")
	}

	var c core.Conflict
	if err := yaml.Unmarshal([]byte(parts[1]), &c); err != nil {
		return nil, err
	}

	body := parts[2]
	subparts := strings.Split(body, "## Diff against current")
	if len(subparts) > 1 {
		c.DiffAgainstCurrent = strings.TrimSpace(subparts[1])
		proposalPart := subparts[0]
		proposalPart = strings.TrimPrefix(proposalPart, "## Rejected proposal\n")
		c.RejectedBody = strings.TrimSpace(proposalPart)
	} else {
		c.RejectedBody = strings.TrimSpace(body)
	}

	return &c, nil
}
