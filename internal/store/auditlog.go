package store

import (
	"bufio"
	"dossier/internal/core"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// AppendAuditLine writes a single AuditEvent line to the audit file in a thread-safe append-only manner.
func AppendAuditLine(path string, event core.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Open file with O_APPEND | O_CREATE | O_WRONLY
	// Write is atomic for lines < PIPE_BUF (typically 4KB) on POSIX
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit log entry: %w", err)
	}

	return nil
}

// ReadAuditEntries reads and parses the JSON Lines audit log file, sorting entries by timestamp.
func ReadAuditEntries(path string) ([]core.AuditEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []core.AuditEvent{}, nil
		}
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	var entries []core.AuditEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry core.AuditEvent
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip unparseable lines or return error?
			// Spec says: "doctor checks if audit log is parseable".
			// So we can return a warning/error or just skip for robust read.
			// Let's keep scanning but we can return error or collect them.
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan audit log: %w", err)
	}

	// Sort by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TS.Before(entries[j].TS)
	})

	return entries, nil
}
