package core

import "time"

// Conflict represents a rejected concurrent edit or merge conflict preserved for human resolution.
type Conflict struct {
	ID                 string    `yaml:"id"`
	DossierID          string    `yaml:"dossier_id"`
	Kind               string    `yaml:"kind"`
	BaseRevision       string    `yaml:"base_revision"`
	AttemptedRevision  string    `yaml:"attempted_revision"`
	Session            string    `yaml:"session,omitempty"`
	TS                 time.Time `yaml:"ts"`
	RejectedBody       string    `yaml:"-"`
	DiffAgainstCurrent string    `yaml:"-"`
}
