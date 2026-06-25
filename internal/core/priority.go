package core

import (
	"time"
)

// CalculatePriorityScore computes the numerical priority score for a Dossier.
// It maps the importance and urgency dimensions to a 1-4 Eisenhower matrix scale:
// - 1: High Importance, High Urgency ("1. Do")
// - 2: High Importance, Low Urgency ("2. Plan")
// - 3: Low Importance, High Urgency ("3. Delegate")
// - 4: Low Importance, Low Urgency ("4. Delete")
func CalculatePriorityScore(fm Frontmatter, now time.Time) int {
	// fm is passed by value; normalize the copy so legacy/invalid priority
	// fields (e.g. a removed "medium") score toward attention rather than
	// silently collapsing to low — matching how they will heal on next write.
	fm.Normalize()

	isHighImportance := fm.Importance == ImportanceHigh
	isHighUrgency := fm.Urgency == UrgencyHigh

	if isHighImportance && isHighUrgency {
		return 1
	} else if isHighImportance && !isHighUrgency {
		return 2
	} else if !isHighImportance && isHighUrgency {
		return 3
	} else {
		return 4
	}
}
