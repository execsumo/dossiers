package core

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
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

		// Install the hook if supported
		if err == nil && (caps.SessionStartHook || caps.SessionEndHook) {
			installErr := h.Install(InstallOpts{
				Interactive: !yesToAll,
				YesToAll:    yesToAll,
			})
			if installErr != nil {
				warnings = append(warnings, Warning(fmt.Sprintf("Failed to install hooks for %s: %v", h.Name(), installErr)))
			}
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
	Content                string
	Force                  bool
}

func (s *Service) Promote(ctx context.Context, req PromoteReq) (Result, error) {
	if req.Name == "" {
		return Result{}, NewError(ErrInvalidFrontmatter, "dossier name is required")
	}

	now := s.clock.Now()

	if !req.Force {
		fms, err := s.store.List("all")
		if err == nil {
			var candidates []Suggestion
			for _, fm := range fms {
				d, _, err := s.store.Read(fm.ID)
				if err != nil {
					continue
				}
				sug := ScoreDossier(req.Name, d, now)
				if sug.Confidence == "high" || sug.Confidence == "medium" {
					candidates = append(candidates, sug)
				}
			}

			if len(candidates) > 0 {
				sort.Slice(candidates, func(i, j int) bool {
					return candidates[i].Score > candidates[j].Score
				})
				return Result{
					OK:   false,
					Data: candidates,
				}, NewError(ErrAmbiguousTarget, "Multiple likely Dossiers match this promote request.")
			}
		}
	}

	saveRes, err := s.Save(ctx, SaveReq{
		DistilledStateMarkdown: req.DistilledStateMarkdown,
		FrontmatterUpdates: map[string]any{
			"name": req.Name,
		},
	})
	if err != nil {
		return Result{}, err
	}

	newRevision := saveRes.Data.(Revision)
	fms, err := s.store.List("all")
	var newID string
	if err == nil {
		for _, fm := range fms {
			if fm.Name == req.Name {
				newID = fm.ID
				break
			}
		}
	}

	var warnings []Warning
	if req.Content != "" && newID != "" {
		art := Artifact{
			ID:            "art_transcript",
			DossierID:     newID,
			Type:          ArtifactTypeTranscript,
			Title:         "Captured Session Transcript",
			ContentFormat: ContentFormatText,
			Content:       req.Content,
			CapturedAt:    now,
			RefreshedAt:   now,
		}
		_ = s.store.WriteArtifact(newID, &art)

		_ = s.store.AppendAudit(newID, AuditEvent{
			TS:             now,
			Event:          AuditEventSave,
			DossierID:      newID,
			BeforeRevision: string(newRevision),
			AfterRevision:  string(newRevision),
			ArtifactsAdded: []string{art.ID},
		})
	}

	harnesses := s.hreg.All()
	var activeHarness Harness
	for _, h := range harnesses {
		caps, err := h.Detect()
		if err == nil && (caps.MCP || caps.SessionStartHook || caps.SessionEndHook || caps.PreCompactionHook || caps.TranscriptCapture) {
			activeHarness = h
			if !caps.TranscriptCapture {
				warnings = append(warnings, Warning("Transcript archive is unavailable in this session."))
			}
			break
		}
	}
	if activeHarness == nil {
		warnings = append(warnings, Warning("Transcript archive is unavailable in this session."))
	}

	return Result{
		OK:       true,
		Data:     newID,
		Warnings: warnings,
	}, nil
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
	Content      string
	Title        string
}

func (s *Service) Link(ctx context.Context, req LinkReq) (Result, error) {
	now := s.clock.Now()

	if req.ID == "" {
		fms, err := s.store.List("all")
		if err != nil {
			return Result{}, err
		}

		var suggestions []Suggestion
		for _, fm := range fms {
			d, _, err := s.store.Read(fm.ID)
			if err != nil {
				continue
			}
			sug := ScoreDossier(req.Content, d, now)
			suggestions = append(suggestions, sug)
		}

		sort.Slice(suggestions, func(i, j int) bool {
			return suggestions[i].Score > suggestions[j].Score
		})

		limit := 3
		if len(suggestions) < limit {
			limit = len(suggestions)
		}
		top := suggestions[:limit]

		return Result{
			OK:   false,
			Data: top,
		}, NewError(ErrAmbiguousTarget, "Multiple likely Dossiers match this link request.")
	}

	d, baseRev, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

	d.Frontmatter.LastTouchedAt = now
	newRev, err := s.store.Write(d, baseRev)
	if err != nil {
		return Result{}, err
	}

	title := req.Title
	if title == "" {
		title = "Linked Session Content"
	}

	art := Artifact{
		DossierID:     d.Frontmatter.ID,
		Type:          ArtifactTypeSourceSnapshot,
		Title:         title,
		ContentFormat: ContentFormatText,
		Content:       req.Content,
		CapturedAt:    now,
		RefreshedAt:   now,
	}

	if err := s.store.WriteArtifact(d.Frontmatter.ID, &art); err != nil {
		return Result{}, err
	}

	_ = s.store.AppendAudit(d.Frontmatter.ID, AuditEvent{
		TS:             now,
		Event:          AuditEventSave,
		DossierID:      d.Frontmatter.ID,
		BeforeRevision: string(baseRev),
		AfterRevision:  string(newRev),
		ArtifactsAdded: []string{art.ID},
	})

	return Result{
		OK:   true,
		Data: newRev,
	}, nil
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
	if req.SessionID == "" {
		return Result{}, NewError(ErrInternal, "session_id is required for switch")
	}

	oldBinding, err := s.store.GetSessionBinding(req.SessionID)
	if err == nil && oldBinding != nil && oldBinding.DossierID != "" {
		oldD, oldRev, err := s.store.Read(oldBinding.DossierID)
		if err == nil {
			oldD.Frontmatter.LastTouchedAt = s.clock.Now()
			_, _ = s.store.Write(oldD, oldRev)
		}
		_ = s.store.ClearSessionBinding(req.SessionID)
	}

	d, rev, err := s.store.Read(req.ID)
	if err != nil {
		return Result{}, err
	}

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
	if activeHarness != nil {
		harnessName = activeHarness.Name()
	}

	binding := &SessionBinding{
		SessionBindingID: req.SessionID,
		Harness:          harnessName,
		DossierID:        d.Frontmatter.ID,
		BoundAt:          s.clock.Now(),
		LastSeenRevision: string(rev),
		Capabilities:     activeCaps,
	}
	if err := s.store.SaveSessionBinding(binding); err != nil {
		return Result{}, WrapError(ErrInternal, "failed to save session binding", err)
	}

	return s.Recall(ctx, RecallReq{ID: d.Frontmatter.ID})
}

type ActiveReq struct {
	SessionID string
}

func (s *Service) Active(ctx context.Context, req ActiveReq) (Result, error) {
	if req.SessionID == "" {
		return Result{}, NewError(ErrInternal, "session_id is required")
	}

	binding, err := s.store.GetSessionBinding(req.SessionID)
	if err != nil {
		return Result{}, err
	}

	return Result{
		OK:   true,
		Data: binding,
	}, nil
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

// SessionStart returns the injected context payload for a harness session.
func (s *Service) SessionStart(ctx context.Context, sessionID string) (string, error) {
	binding, err := s.store.GetSessionBinding(sessionID)
	var activeDossierID string
	if err == nil && binding != nil {
		activeDossierID = binding.DossierID
	}

	// Fetch open dossiers
	fms, err := s.store.List("all")
	if err != nil {
		return "", err
	}

	now := s.clock.Now()
	type scoredMeta struct {
		fm    Frontmatter
		score int
	}
	var scored []scoredMeta
	for _, fm := range fms {
		if fm.Status != StatusArchived {
			score := CalculatePriorityScore(fm, now)
			scored = append(scored, scoredMeta{fm: fm, score: score})
		}
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

	var openLines []string
	for _, sItem := range scored {
		openLines = append(openLines, fmt.Sprintf("- **%s** (status: %s, slug: %s, priority: %d)", sItem.fm.Name, sItem.fm.Status, sItem.fm.Slug, sItem.score))
	}
	openStr := strings.Join(openLines, "\n")
	if openStr == "" {
		openStr = "(No open dossiers)"
	}

	// Detect capabilities
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
	}

	mcpAvail := "unavailable"
	startAvail := "unavailable"
	endAvail := "unavailable"
	compactionAvail := "unavailable"
	transcriptAvail := "unavailable"

	if activeCaps.MCP {
		mcpAvail = "available"
	}
	if activeCaps.SessionStartHook {
		startAvail = "available"
	}
	if activeCaps.SessionEndHook {
		endAvail = "available"
	}
	if activeCaps.PreCompactionHook {
		compactionAvail = "available"
	}
	if activeCaps.TranscriptCapture {
		transcriptAvail = "available"
	}

	var sb strings.Builder
	sb.WriteString("# Dossier Library\n\n")
	sb.WriteString(fmt.Sprintf("Harness: %s\n", harnessName))
	sb.WriteString("Capabilities:\n")
	sb.WriteString(fmt.Sprintf("- MCP: %s\n", mcpAvail))
	sb.WriteString(fmt.Sprintf("- Session-start hook: %s\n", startAvail))
	sb.WriteString(fmt.Sprintf("- Session-end save hook: %s\n", endAvail))
	sb.WriteString(fmt.Sprintf("- Pre-compaction save hook: %s\n", compactionAvail))
	sb.WriteString(fmt.Sprintf("- Transcript capture: %s\n\n", transcriptAvail))

	if harnessName == "Codex" && !activeCaps.TranscriptCapture {
		sb.WriteString("Warning: Transcript archive is unavailable in this session.\n\n")
	}

	sb.WriteString("Open Dossiers:\n")
	sb.WriteString(openStr)
	sb.WriteString("\n\n")

	sb.WriteString("Distillation Guide:\n")
	sb.WriteString("See: ~/.dossier/context/guide.md\n\n")

	if activeDossierID != "" {
		recallRes, err := s.Recall(ctx, RecallReq{ID: activeDossierID})
		if err == nil {
			recData := recallRes.Data.(RecallResult)
			sb.WriteString("Active Dossier:\n")
			sb.WriteString(fmt.Sprintf("ID: %s\n", recData.Frontmatter.ID))
			sb.WriteString(fmt.Sprintf("Name: %s\n", recData.Frontmatter.Name))
			sb.WriteString(fmt.Sprintf("Revision: %s\n\n", recData.Revision))
			sb.WriteString("Distilled State:\n")
			sb.WriteString(recData.DistilledState)
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("No active dossier bound to this session. Please select an existing dossier to continue or create a new one.\n")
	}

	return sb.String(), nil
}

// SessionEnd saves state and appends the transcript artifact on session completion.
func (s *Service) SessionEnd(ctx context.Context, sessionID string, distilledState string, transcript string) error {
	binding, err := s.store.GetSessionBinding(sessionID)
	if err != nil {
		return nil
	}

	now := s.clock.Now()

	if distilledState != "" {
		_, err = s.Save(ctx, SaveReq{
			ID:                     binding.DossierID,
			BaseRevision:           Revision(binding.LastSeenRevision),
			DistilledStateMarkdown: distilledState,
		})
		if err != nil {
			return err
		}
	}

	if transcript != "" {
		art := Artifact{
			ID:            "art_transcript",
			DossierID:     binding.DossierID,
			Type:          ArtifactTypeTranscript,
			Title:         "Session End Transcript",
			ContentFormat: ContentFormatText,
			Content:       transcript,
			CapturedAt:    now,
			RefreshedAt:   now,
		}
		if err := s.store.WriteArtifact(binding.DossierID, &art); err != nil {
			return err
		}
		_ = s.store.AppendAudit(binding.DossierID, AuditEvent{
			TS:             now,
			Event:          AuditEventSave,
			DossierID:      binding.DossierID,
			BeforeRevision: binding.LastSeenRevision,
			AfterRevision:  binding.LastSeenRevision,
			ArtifactsAdded: []string{art.ID},
		})
	}

	return nil
}
