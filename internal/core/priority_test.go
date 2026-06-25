package core

import (
	"testing"
	"time"
)

func TestCalculatePriorityScore(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		fm            Frontmatter
		expectedScore int
	}{
		{
			name: "High Importance, High Urgency",
			fm: Frontmatter{
				Importance: ImportanceHigh,
				Urgency:    UrgencyHigh,
			},
			expectedScore: 1,
		},
		{
			name: "High Importance, Low Urgency",
			fm: Frontmatter{
				Importance: ImportanceHigh,
				Urgency:    UrgencyLow,
			},
			expectedScore: 2,
		},
		{
			name: "Low Importance, High Urgency",
			fm: Frontmatter{
				Importance: ImportanceLow,
				Urgency:    UrgencyHigh,
			},
			expectedScore: 3,
		},
		{
			name: "Low Importance, Low Urgency",
			fm: Frontmatter{
				Importance: ImportanceLow,
				Urgency:    UrgencyLow,
			},
			expectedScore: 4,
		},
	}

	for _, tc := range tests {
		actual := CalculatePriorityScore(tc.fm, now)
		if actual != tc.expectedScore {
			t.Errorf("%s: expected score %d, but got %d", tc.name, tc.expectedScore, actual)
		}
	}
}
