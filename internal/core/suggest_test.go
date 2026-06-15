package core

import (
	"testing"
	"time"
)

func TestScoreDossier(t *testing.T) {
	now := time.Now()
	d := &Dossier{
		Frontmatter: Frontmatter{
			ID:            "dos_1",
			Name:          "Pricing Model Refresh",
			Slug:          "pricing-model-refresh",
			Status:        StatusActive,
			NextAction:    "Compare revised sales scenarios",
			OpenQuestions: []string{"Does Sales prefer account-tier or usage-tier packaging?"},
			LastTouchedAt: now,
		},
		DistilledState: DistilledState{
			Body: "This dossier tracks the core pricing restructure project.",
		},
	}

	// Test case 1: Exact Name/Slug match
	s1 := ScoreDossier("Pricing Model Refresh", d, now)
	if s1.Confidence != "high" {
		t.Errorf("expected high confidence for exact name match, got %s (score: %f)", s1.Confidence, s1.Score)
	}

	// Test case 2: Overlap in next_action / questions
	s2 := ScoreDossier("sales packaging restructure", d, now)
	if s2.Confidence != "medium" {
		t.Errorf("expected medium confidence for partial overlap, got %s (score: %f)", s2.Confidence, s2.Score)
	}

	// Test case 3: Weak / no overlap
	s3 := ScoreDossier("unrelated gardening advice", d, now)
	if s3.Confidence != "low" {
		t.Errorf("expected low confidence for unrelated query, got %s (score: %f)", s3.Confidence, s3.Score)
	}
}
