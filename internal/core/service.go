package core

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
)

// Config holds the service-level configurations used by the core logic.
type Config struct {
	DossierHome string
	TokenTarget int
}

// Service orchestrates Dossier domain use-cases over the port interfaces.
// It contains zero business logic leakages to driving adapters (CLI/MCP/TUI).
type Service struct {
	store  Store
	search Searcher
	tok    Tokenizer
	hreg   HarnessRegistry
	clock  Clock
	cfg    Config
}

// RecallResult carries the output fields for dossier recall queries.
type RecallResult struct {
	DistilledState string      `json:"distilled_state"`
	Frontmatter    Frontmatter `json:"frontmatter"`
	Revision       Revision    `json:"revision"`
	TokenEstimate  int         `json:"token_estimate"`
}

// ListItem represents a single summary item for dossier listings.
type ListItem struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Status        string   `json:"status"`
	NextAction    string   `json:"next_action"`
	OpenQuestions []string `json:"open_questions"`
	Importance    string   `json:"importance"`
	Urgency       string   `json:"urgency"`
	DueDate       string   `json:"due_date,omitempty"`
	StalenessDays int      `json:"staleness_days"`
	PriorityScore int      `json:"priority_score"`
}

// NewService instantiates the core orchestration service.
func NewService(store Store, search Searcher, tok Tokenizer, hreg HarnessRegistry, clock Clock, cfg Config) *Service {
	return &Service{
		store:  store,
		search: search,
		tok:    tok,
		hreg:   hreg,
		clock:  clock,
		cfg:    cfg,
	}
}

// Init initializes the store directories, writes default configs and guide.
func (s *Service) Init(ctx context.Context, yesToAll bool) (Result, error) {
	// For Milestone 1 baseline, we delegate to the store's Init method.
	if err := s.store.Init(); err != nil {
		return Result{OK: false}, WrapError(ErrInternal, "failed to initialize local store", err)
	}

	warnings := []Warning{}
	data := make(map[string]any)

	// Detect harnesses and construct the capability tiers details
	harnesses := s.hreg.All()
	harnessTiers := make(map[string]string)
	for _, h := range harnesses {
		caps, err := h.Detect()
		tier := "Tier 3"
		if err == nil {
			if caps.SessionStartHook && caps.SessionEndHook && caps.TranscriptCapture {
				tier = "Tier 1"
			} else if caps.SessionStartHook && caps.SessionEndHook {
				tier = "Tier 2"
			}
		}
		harnessTiers[h.Name()] = tier
		if h.Name() == "codex" && !caps.TranscriptCapture {
			warnings = append(warnings, Warning("Codex transcript archive unavailable. Dossier will say this at session start."))
		}
	}
	data["harness_tiers"] = harnessTiers

	return Result{
		OK:       true,
		Data:     data,
		Warnings: warnings,
	}, nil
}

// Doctor validates store integrity and configuration correctness.
func (s *Service) Doctor(ctx context.Context) (Result, error) {
	// For Milestone 1 baseline, we run check checks
	warnings := []Warning{}

	// Try reading config or checking directories
	if s.store == nil {
		return Result{OK: false}, NewError(ErrInternal, "store not configured")
	}

	// For baseline, list dossiers to make sure it doesn't fail
	_, err := s.store.List("all")
	if err != nil {
		warnings = append(warnings, Warning(fmt.Sprintf("Failed to list dossiers: %v", err)))
	}

	return Result{
		OK:       err == nil,
		Warnings: warnings,
	}, nil
}

// Stubs for future milestones

type PromoteReq struct {
	Name                   string
	DistilledStateMarkdown string
	FromFilePath           string
}

func (s *Service) Promote(ctx context.Context, req PromoteReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type SaveReq struct {
	ID                     string
	BaseRevision           Revision
	DistilledStateMarkdown string
	FrontmatterUpdates     map[string]any
	Artifacts              []Artifact
}

func (s *Service) Save(ctx context.Context, req SaveReq) (Result, error) {
	var d *Dossier
	var baseRev Revision
	var err error

	isNew := req.ID == ""

	if isNew {
		d = &Dossier{
			Frontmatter: Frontmatter{
				Status:     StatusActive,
				Importance: ImportanceMedium,
				Urgency:    UrgencyMedium,
			},
		}
	} else {
		d, baseRev, err = s.store.Read(req.ID)
		if err != nil {
			return Result{}, err
		}

		if req.BaseRevision != "" && baseRev != req.BaseRevision {
			return Result{}, NewError(ErrConcurrentEdit, fmt.Sprintf("concurrency mismatch: base is %q, current is %q", req.BaseRevision, baseRev))
		}
	}

	if req.FrontmatterUpdates != nil {
		if val, ok := req.FrontmatterUpdates["name"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.Name = strVal
			}
		}
		if val, ok := req.FrontmatterUpdates["status"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.Status = Status(strVal)
			}
		}
		if val, ok := req.FrontmatterUpdates["next_action"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.NextAction = strVal
			}
		}
		if val, ok := req.FrontmatterUpdates["importance"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.Importance = Importance(strVal)
			}
		}
		if val, ok := req.FrontmatterUpdates["urgency"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.Urgency = Urgency(strVal)
			}
		}
		if val, ok := req.FrontmatterUpdates["due_date"]; ok {
			if strVal, ok := val.(string); ok {
				d.Frontmatter.DueDate = strVal
			}
		}
		if val, ok := req.FrontmatterUpdates["token_target"]; ok {
			if intVal, ok := val.(int); ok {
				d.Frontmatter.TokenTarget = intVal
			} else if floatVal, ok := val.(float64); ok {
				d.Frontmatter.TokenTarget = int(floatVal)
			}
		}
		if val, ok := req.FrontmatterUpdates["open_questions"]; ok {
			if listVal, ok := val.([]string); ok {
				d.Frontmatter.OpenQuestions = listVal
			} else if anyListVal, ok := val.([]any); ok {
				var questions []string
				for _, q := range anyListVal {
					if qStr, ok := q.(string); ok {
						questions = append(questions, qStr)
					}
				}
				d.Frontmatter.OpenQuestions = questions
			}
		}
	}

	if req.DistilledStateMarkdown != "" {
		d.DistilledState.Body = req.DistilledStateMarkdown
	}

	d.Frontmatter.LastTouchedAt = s.clock.Now()

	newRev, err := s.store.Write(d, baseRev)
	if err != nil {
		return Result{}, err
	}

	var addedArtifactIDs []string
	for _, art := range req.Artifacts {
		art.DossierID = d.Frontmatter.ID
		err := s.store.WriteArtifact(d.Frontmatter.ID, &art)
		if err == nil {
			addedArtifactIDs = append(addedArtifactIDs, art.ID)
		}
	}

	event := AuditEvent{
		TS:             s.clock.Now(),
		DossierID:      d.Frontmatter.ID,
		BeforeRevision: string(baseRev),
		AfterRevision:  string(newRev),
		ArtifactsAdded: addedArtifactIDs,
		TokenEstimate:  s.tok.Estimate(d.DistilledState.Body),
	}
	if isNew {
		event.Event = AuditEventCreate
	} else {
		event.Event = AuditEventSave
	}
	_ = s.store.AppendAudit(d.Frontmatter.ID, event)

	return Result{
		OK:   true,
		Data: newRev,
	}, nil
}

type LinkReq struct {
	ID           string
	FromFilePath string
}

func (s *Service) Link(ctx context.Context, req LinkReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 2")
}

type MergeReq struct {
	SourceID string
	TargetID string
}

func (s *Service) Merge(ctx context.Context, req MergeReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 2")
}

type RecallReq struct {
	ID string
}

func (s *Service) Recall(ctx context.Context, req RecallReq) (Result, error) {
	d, rev, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

	tokens := s.tok.Estimate(d.DistilledState.Body)

	var warnings []Warning
	target := d.Frontmatter.TokenTarget
	if target == 0 {
		target = s.cfg.TokenTarget
	}
	if target == 0 {
		target = 100000
	}
	if tokens > target {
		warnings = append(warnings, Warning(fmt.Sprintf("Distilled State exceeds token target (%d > %d tokens). Consider condensing.", tokens, target)))
	}

	return Result{
		OK:       true,
		Data:     RecallResult{DistilledState: d.DistilledState.Body, Frontmatter: d.Frontmatter, Revision: rev, TokenEstimate: tokens},
		Warnings: warnings,
	}, nil
}

type ListReq struct {
	Status string
}

func (s *Service) List(ctx context.Context, req ListReq) (Result, error) {
	fms, err := s.store.List("all")
	if err != nil {
		return Result{OK: false}, WrapError(ErrInternal, "failed to list dossiers", err)
	}

	var filtered []Frontmatter
	for _, fm := range fms {
		if req.Status == "" {
			if fm.Status == StatusActive || fm.Status == StatusWaiting || fm.Status == StatusBlocked {
				filtered = append(filtered, fm)
			}
		} else if req.Status == "all" || string(fm.Status) == req.Status {
			filtered = append(filtered, fm)
		}
	}

	type scoredMeta struct {
		fm    Frontmatter
		score int
	}

	now := s.clock.Now()
	var scored []scoredMeta
	for _, fm := range filtered {
		score := CalculatePriorityScore(fm, now)
		scored = append(scored, scoredMeta{fm: fm, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].fm.LastTouchedAt.Equal(scored[j].fm.LastTouchedAt) {
			return scored[i].fm.LastTouchedAt.Before(scored[j].fm.LastTouchedAt)
		}
		return scored[i].fm.UpdatedAt.Before(scored[j].fm.UpdatedAt)
	})

	var items []ListItem
	for _, sItem := range scored {
		daysSinceTouched := int(now.Sub(sItem.fm.LastTouchedAt).Hours() / 24)
		if daysSinceTouched < 0 {
			daysSinceTouched = 0
		}

		items = append(items, ListItem{
			ID:            sItem.fm.ID,
			Name:          sItem.fm.Name,
			Slug:          sItem.fm.Slug,
			Status:        string(sItem.fm.Status),
			NextAction:    sItem.fm.NextAction,
			OpenQuestions: sItem.fm.OpenQuestions,
			Importance:    string(sItem.fm.Importance),
			Urgency:       string(sItem.fm.Urgency),
			DueDate:       sItem.fm.DueDate,
			StalenessDays: daysSinceTouched,
			PriorityScore: sItem.score,
		})
	}

	return Result{
		OK:   true,
		Data: items,
	}, nil
}

type SearchReq struct {
	Query string
	Scope SearchScope
}

func (s *Service) Search(ctx context.Context, req SearchReq) (Result, error) {
	if req.Scope.DossierID != "" {
		d, _, err := s.store.Read(req.Scope.DossierID)
		if err != nil {
			return Result{}, err
		}
		req.Scope.DossierID = d.Frontmatter.ID
	}

	hits, err := s.search.Search(ctx, req.Query, req.Scope)
	if err != nil {
		return Result{}, WrapError(ErrInternal, "search failed", err)
	}

	return Result{
		OK:   true,
		Data: hits,
	}, nil
}

func (s *Service) ContextRefresh(ctx context.Context) (Result, error) {
	fms, err := s.store.List("all")
	if err != nil {
		return Result{OK: false}, WrapError(ErrInternal, "failed to list dossiers for context refresh", err)
	}

	// Filter and score open dossiers (non-archived)
	type scoredMeta struct {
		fm    Frontmatter
		score int
	}

	now := s.clock.Now()
	var scored []scoredMeta
	for _, fm := range fms {
		if fm.Status != StatusArchived {
			score := CalculatePriorityScore(fm, now)
			scored = append(scored, scoredMeta{fm: fm, score: score})
		}
	}

	// Sort open dossiers by priority score descending
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].fm.LastTouchedAt.Equal(scored[j].fm.LastTouchedAt) {
			return scored[i].fm.LastTouchedAt.Before(scored[j].fm.LastTouchedAt)
		}
		return scored[i].fm.UpdatedAt.Before(scored[j].fm.UpdatedAt)
	})

	var openDossiers []LibraryDossier
	for _, sItem := range scored {
		openDossiers = append(openDossiers, LibraryDossier{
			Name:          sItem.fm.Name,
			Status:        string(sItem.fm.Status),
			Slug:          sItem.fm.Slug,
			NextAction:    sItem.fm.NextAction,
			PriorityScore: sItem.score,
		})
	}

	// Detect harnesses and capabilities
	harnesses := s.hreg.All()
	var activeHarness Harness
	var activeCaps Capabilities

	for _, h := range harnesses {
		caps, err := h.Detect()
		if err == nil && (caps.MCP || caps.SessionStartHook || caps.SessionEndHook || caps.PreCompactionHook || caps.TranscriptCapture) {
			activeHarness = h
			activeCaps = caps
			break
		}
	}

	harnessName := "CLI"
	harnessCaps := map[string]bool{
		"MCP":               false,
		"SessionStartHook":  false,
		"SessionEndHook":    false,
		"PreCompactionHook": false,
		"TranscriptCapture": false,
	}
	var warnings []string

	if activeHarness != nil {
		harnessName = activeHarness.Name()
		switch harnessName {
		case "claude-code":
			harnessName = "Claude Code"
		case "codex":
			harnessName = "Codex"
		case "antigravity":
			harnessName = "Antigravity"
		}

		harnessCaps["MCP"] = activeCaps.MCP
		harnessCaps["SessionStartHook"] = activeCaps.SessionStartHook
		harnessCaps["SessionEndHook"] = activeCaps.SessionEndHook
		harnessCaps["PreCompactionHook"] = activeCaps.PreCompactionHook
		harnessCaps["TranscriptCapture"] = activeCaps.TranscriptCapture

		if harnessName == "Codex" && !activeCaps.TranscriptCapture {
			warnings = append(warnings, "Transcript archive is unavailable in this session.")
		}
	} else {
		warnings = append(warnings, "No harness session active. Run from within a supported client harness for full integration.")
	}

	libData := LibraryData{
		Harness:      harnessName,
		Capabilities: harnessCaps,
		Warnings:     warnings,
		OpenDossiers: openDossiers,
	}

	if err := s.store.WriteLibraryContext(libData); err != nil {
		return Result{OK: false}, WrapError(ErrInternal, "failed to write library context", err)
	}

	return Result{
		OK: true,
	}, nil
}

type SwitchReq struct {
	ID        string
	SessionID string
}

func (s *Service) Switch(ctx context.Context, req SwitchReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 2")
}

type ActiveReq struct {
	SessionID string
}

func (s *Service) Active(ctx context.Context, req ActiveReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 2")
}

type ArchiveReq struct {
	ID string
}

func (s *Service) Archive(ctx context.Context, req ArchiveReq) (Result, error) {
	d, rev, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

	d.Frontmatter.Status = StatusArchived
	d.Frontmatter.LastTouchedAt = s.clock.Now()

	newRev, err := s.store.Write(d, rev)
	if err != nil {
		return Result{}, err
	}

	_ = s.store.AppendAudit(d.Frontmatter.ID, AuditEvent{
		TS:             s.clock.Now(),
		Event:          AuditEventArchived,
		DossierID:      d.Frontmatter.ID,
		BeforeRevision: string(rev),
		AfterRevision:  string(newRev),
	})

	return Result{
		OK:   true,
		Data: newRev,
	}, nil
}

type PathReq struct {
	ID string
}

func (s *Service) Path(ctx context.Context, req PathReq) (Result, error) {
	d, _, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

	dossierPath := filepath.Join(s.cfg.DossierHome, d.Frontmatter.Slug)
	return Result{
		OK:   true,
		Data: dossierPath,
	}, nil
}

type SetStatusReq struct {
	ID     string
	Status Status
}

func (s *Service) SetStatus(ctx context.Context, req SetStatusReq) (Result, error) {
	d, rev, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

	oldStatus := d.Frontmatter.Status
	d.Frontmatter.Status = req.Status
	d.Frontmatter.LastTouchedAt = s.clock.Now()

	newRev, err := s.store.Write(d, rev)
	if err != nil {
		return Result{}, err
	}

	_ = s.store.AppendAudit(d.Frontmatter.ID, AuditEvent{
		TS:             s.clock.Now(),
		Event:          AuditEventStatusChanged,
		DossierID:      d.Frontmatter.ID,
		BeforeRevision: string(rev),
		AfterRevision:  string(newRev),
		Message:        fmt.Sprintf("status changed from %s to %s", oldStatus, req.Status),
	})

	return Result{
		OK:   true,
		Data: newRev,
	}, nil
}
