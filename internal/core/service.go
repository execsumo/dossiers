package core

import (
	"context"
	"fmt"
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
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type LinkReq struct {
	ID           string
	FromFilePath string
}

func (s *Service) Link(ctx context.Context, req LinkReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type MergeReq struct {
	SourceID string
	TargetID string
}

func (s *Service) Merge(ctx context.Context, req MergeReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type RecallReq struct {
	ID string
}

func (s *Service) Recall(ctx context.Context, req RecallReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type ListReq struct {
	Status string
}

func (s *Service) List(ctx context.Context, req ListReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type SearchReq struct {
	Query string
	Scope SearchScope
}

func (s *Service) Search(ctx context.Context, req SearchReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type SwitchReq struct {
	ID        string
	SessionID string
}

func (s *Service) Switch(ctx context.Context, req SwitchReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type ActiveReq struct {
	SessionID string
}

func (s *Service) Active(ctx context.Context, req ActiveReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type ArchiveReq struct {
	ID string
}

func (s *Service) Archive(ctx context.Context, req ArchiveReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type PathReq struct {
	ID string
}

func (s *Service) Path(ctx context.Context, req PathReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}

type SetStatusReq struct {
	ID     string
	Status Status
}

func (s *Service) SetStatus(ctx context.Context, req SetStatusReq) (Result, error) {
	return Result{}, NewError(ErrInternal, "unimplemented in Milestone 1")
}
