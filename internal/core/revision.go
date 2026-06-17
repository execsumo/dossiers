package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// CanonicalFrontmatter serializes a Frontmatter struct into a deterministic format.
// It uses json.Marshal to automatically include all fields without manual updates.
func CanonicalFrontmatter(fm Frontmatter) string {
	b, _ := json.Marshal(fm)
	return string(b)
}

// NormalizeNewlines converts CRLF to LF, trims trailing whitespace on each line, and ensures a single trailing LF.
func NormalizeNewlines(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	var trimmedLines []string
	for _, line := range lines {
		trimmedLines = append(trimmedLines, strings.TrimRight(line, " \t"))
	}
	content := strings.Join(trimmedLines, "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return "\n"
	}
	return content + "\n"
}

// CanonicalArtifacts generates a sorted representation of the artifacts and their hashes.
func CanonicalArtifacts(artifacts []Artifact) string {
	var artLines []string
	for _, art := range artifacts {
		h := sha256.Sum256([]byte(art.Content))
		artLines = append(artLines, fmt.Sprintf("%s:%s", art.ID, hex.EncodeToString(h[:])))
	}
	sort.Strings(artLines)
	return strings.Join(artLines, "\n")
}

// CalculateRevision computes the canonical SHA-256 hash (truncated to 32 chars) prefixed with "rev_".
func CalculateRevision(fm Frontmatter, body string, artifacts []Artifact) Revision {
	var sb strings.Builder
	sb.WriteString(CanonicalFrontmatter(fm))
	sb.WriteString("\n---\n")
	sb.WriteString(NormalizeNewlines(body))
	sb.WriteString("\n---artifacts---\n")
	sb.WriteString(CanonicalArtifacts(artifacts))

	h := sha256.Sum256([]byte(sb.String()))
	hexHash := hex.EncodeToString(h[:])
	if len(hexHash) > 32 {
		hexHash = hexHash[:32]
	}
	return Revision("rev_" + hexHash)
}
