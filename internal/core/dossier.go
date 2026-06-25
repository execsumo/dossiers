package core

import (
	"fmt"
	"time"
)

// Status represents the lifecycle state of a Dossier.
type Status string

const (
	StatusActive   Status = "active"
	StatusWaiting  Status = "waiting"
	StatusBlocked  Status = "blocked"
	StatusResolved Status = "resolved"
	StatusArchived Status = "archived"
)

// IsValid validates if the status is one of the allowed enums.
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusWaiting, StatusBlocked, StatusResolved, StatusArchived:
		return true
	}
	return false
}

// Importance represents the priority dimension of importance.
type Importance string

const (
	ImportanceHigh Importance = "high"
	ImportanceLow  Importance = "low"
)

// IsValid validates if importance is one of the allowed enums.
func (i Importance) IsValid() bool {
	switch i {
	case ImportanceHigh, ImportanceLow:
		return true
	}
	return false
}

// Urgency represents the priority dimension of urgency.
type Urgency string

const (
	UrgencyHigh Urgency = "high"
	UrgencyLow  Urgency = "low"
)

// IsValid validates if urgency is one of the allowed enums.
func (u Urgency) IsValid() bool {
	switch u {
	case UrgencyHigh, UrgencyLow:
		return true
	}
	return false
}

// Frontmatter represents the parsed metadata block of a Dossier.
// In conformance with BUILD-DECISIONS, base_revision is session-side, not in frontmatter.
type Frontmatter struct {
	ID            string     `yaml:"id"`
	Name          string     `yaml:"name"`
	Slug          string     `yaml:"slug"`
	CreatedAt     time.Time  `yaml:"created_at"`
	UpdatedAt     time.Time  `yaml:"updated_at"`
	LastTouchedAt time.Time  `yaml:"last_touched_at"`
	Status        Status     `yaml:"status"`
	Lead          string     `yaml:"lead,omitempty"`
	NextAction    string     `yaml:"next_action"`
	OpenQuestions []string   `yaml:"open_questions"`
	Importance    Importance `yaml:"importance"`
	Urgency       Urgency    `yaml:"urgency"`
	DueDate       string     `yaml:"due_date,omitempty"`
	TokenTarget   int        `yaml:"token_target,omitempty"`
}

// Validate ensures that all required fields are present and valid.
func (f *Frontmatter) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("id is required")
	}
	if f.Name == "" {
		return fmt.Errorf("name is required")
	}
	if f.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if f.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if f.UpdatedAt.IsZero() {
		return fmt.Errorf("updated_at is required")
	}
	if f.LastTouchedAt.IsZero() {
		return fmt.Errorf("last_touched_at is required")
	}
	if !f.Status.IsValid() {
		return fmt.Errorf("invalid status: %q", f.Status)
	}
	if !f.Importance.IsValid() {
		return fmt.Errorf("invalid importance: %q", f.Importance)
	}
	if !f.Urgency.IsValid() {
		return fmt.Errorf("invalid urgency: %q", f.Urgency)
	}
	return nil
}

// DistilledState contains the curated markdown representation of the topic.
type DistilledState struct {
	Body string
}

// Dossier represents the combined domain entity.
type Dossier struct {
	Frontmatter    Frontmatter
	DistilledState DistilledState
}
