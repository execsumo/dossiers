package core

import "time"

// Capabilities defines the integration extension points supported by a harness.
type Capabilities struct {
	MCP               bool `json:"mcp"`
	SessionStartHook  bool `json:"session_start_hook"`
	SessionEndHook    bool `json:"session_end_hook"`
	PreCompactionHook bool `json:"pre_compaction_hook"`
	TranscriptCapture bool `json:"transcript_capture"`
}

// SessionBinding records which Dossier is currently active in a specific harness session.
type SessionBinding struct {
	SessionBindingID string       `json:"session_binding_id"`
	Harness          string       `json:"harness"`
	DossierID        string       `json:"dossier_id"`
	BoundAt          time.Time    `json:"bound_at"`
	LastSeenRevision string       `json:"last_seen_revision"`
	Capabilities     Capabilities `json:"capabilities"`
}
