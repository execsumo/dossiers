package core

import (
	"time"
)

// CalculatePriorityScore computes the numerical priority score for a Dossier.
func CalculatePriorityScore(fm Frontmatter, now time.Time) int {
	score := 0

	// 1. Importance (high=30, medium=15, low=5)
	switch fm.Importance {
	case ImportanceHigh:
		score += 30
	case ImportanceMedium:
		score += 15
	case ImportanceLow:
		score += 5
	default:
		score += 15 // Default fallback
	}

	// 2. Urgency (high=30, medium=15, low=5)
	switch fm.Urgency {
	case UrgencyHigh:
		score += 30
	case UrgencyMedium:
		score += 15
	case UrgencyLow:
		score += 5
	default:
		score += 15 // Default fallback
	}

	// 3. Status (blocked=10, waiting=5, active=0)
	switch fm.Status {
	case StatusBlocked:
		score += 10
	case StatusWaiting:
		score += 5
	case StatusActive:
		score += 0
	}

	// 4. Due Date
	if fm.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02", fm.DueDate)
		if err == nil {
			// Compare dates ignoring times
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			due := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, time.UTC)

			daysUntilDue := int(due.Sub(today).Hours() / 24)
			switch {
			case daysUntilDue < 0:
				score += 40 // overdue
			case daysUntilDue == 0:
				score += 35 // due_today
			case daysUntilDue > 0 && daysUntilDue <= 3:
				score += 25 // due_soon_3_days
			case daysUntilDue > 3 && daysUntilDue <= 7:
				score += 15 // due_soon_7_days
			}
		}
	}

	// 5. Staleness (min(days_since_last_touched, 14))
	daysSinceTouched := int(now.Sub(fm.LastTouchedAt).Hours() / 24)
	if daysSinceTouched < 0 {
		daysSinceTouched = 0
	}
	stalenessPoints := daysSinceTouched
	if stalenessPoints > 14 {
		stalenessPoints = 14
	}
	score += stalenessPoints

	return score
}
