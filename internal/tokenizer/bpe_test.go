package tokenizer

import (
	"testing"
)

func TestBPETokenizerEstimate(t *testing.T) {
	tok, err := NewBPETokenizer()
	if err != nil {
		t.Fatalf("failed to create BPETokenizer: %v", err)
	}

	text := "Hello, world! This is a simple test for our BPE tokenizer."
	count := tok.Estimate(text)

	// "Hello, world! This is a simple test for our BPE tokenizer." is ~13 tokens
	if count < 10 || count > 20 {
		t.Errorf("unexpected token estimate: got %d, expected roughly 13-15", count)
	}

	emptyCount := tok.Estimate("")
	if emptyCount != 0 {
		t.Errorf("expected 0 tokens for empty text, got %d", emptyCount)
	}
}
