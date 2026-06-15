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
			name: "Active, High Importance, High Urgency, No Due Date, No Staleness",
			fm: Frontmatter{
				Importance:    ImportanceHigh,
				Urgency:       UrgencyHigh,
				Status:        StatusActive,
				LastTouchedAt: now,
			},
			expectedScore: 60, // 30 + 30 + 0 + 0 + 0
		},
		{
			name: "Blocked, Medium Importance, Low Urgency, Overdue, Stale 10 days",
			fm: Frontmatter{
				Importance:    ImportanceMedium,
				Urgency:       UrgencyLow,
				Status:        StatusBlocked,
				DueDate:       "2026-06-10",
				LastTouchedAt: now.AddDate(0, 0, -10),
			},
			expectedScore: 80, // 15 + 5 + 10 + 40 + 10
		},
		{
			name: "Waiting, Low Importance, Medium Urgency, Due Today, Stale 20 days",
			fm: Frontmatter{
				Importance:    ImportanceLow,
				Urgency:       UrgencyMedium,
				Status:        StatusWaiting,
				DueDate:       "2026-06-14",
				LastTouchedAt: now.AddDate(0, 0, -20),
			},
			expectedScore: 74, // 5 + 15 + 5 + 35 + 14 (max staleness is 14)
		},
	}

	for _, tc := range tests {
		actual := CalculatePriorityScore(tc.fm, now)
		if actual != tc.expectedScore {
			t.Errorf("%s: expected score %d, but got %d", tc.name, tc.expectedScore, actual)
		}
	}
}
