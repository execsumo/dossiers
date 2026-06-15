package store

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateID creates a new ULID prefixed with the given prefix.
func GenerateID(prefix string) (string, error) {
	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id, err := ulid.New(ulid.Timestamp(t), entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return prefix + strings.ToLower(id.String()), nil
}

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9]+`)
var repeatedHyphenRegex = regexp.MustCompile(`-+`)

// GenerateSlug normalizes a name into a URL-friendly slug.
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	// Replace non-alphanumeric with hyphens
	slug = nonAlphaNumRegex.ReplaceAllString(slug, "-")
	// Collapse repeated hyphens
	slug = repeatedHyphenRegex.ReplaceAllString(slug, "-")
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

// SlugWithSuffix appends the last 6 characters of the ULID Crockford base32 ID to resolve collisions.
func SlugWithSuffix(slug, id string) string {
	// Extract the suffix (last 6 chars of the ID, which is the raw ULID)
	// Example: dos_01jz8example000000000000000 -> last 6 chars are 000000
	suffix := ""
	if len(id) > 6 {
		suffix = id[len(id)-6:]
	} else {
		suffix = "suffix"
	}
	return fmt.Sprintf("%s-%s", slug, suffix)
}

// GenerateSessionID generates a standard session ID prefixed with sess_.
func GenerateSessionID() (string, error) {
	return GenerateID("sess_")
}

// GenerateConflictID generates a conflict ID prefixed with conf_.
func GenerateConflictID() (string, error) {
	return GenerateID("conf_")
}
