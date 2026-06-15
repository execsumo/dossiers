package store

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id, err := GenerateID("dos_")
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}

	if !strings.HasPrefix(id, "dos_") {
		t.Errorf("expected ID to start with dos_, but got: %s", id)
	}

	// ULID is 26 chars, prefix is 4 chars, total should be 30
	if len(id) != 30 {
		t.Errorf("expected ID length to be 30, but got: %d", len(id))
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Pricing model refresh", "pricing-model-refresh"},
		{"  Hello,   World!!!  ", "hello-world"},
		{"test--slug--collapsing", "test-slug-collapsing"},
		{"-trim-ends-", "trim-ends"},
		{"Alpha123 Beta", "alpha123-beta"},
	}

	for _, tc := range tests {
		actual := GenerateSlug(tc.input)
		if actual != tc.expected {
			t.Errorf("GenerateSlug(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestSlugWithSuffix(t *testing.T) {
	id := "dos_01jz8example000000abc123"
	slug := "pricing-model-refresh"
	expected := "pricing-model-refresh-abc123"

	actual := SlugWithSuffix(slug, id)
	if actual != expected {
		t.Errorf("SlugWithSuffix(%q, %q) = %q; expected %q", slug, id, actual, expected)
	}
}
