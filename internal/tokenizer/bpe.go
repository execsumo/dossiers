package tokenizer

import (
	"github.com/tiktoken-go/tokenizer"
)

// BPETokenizer implements core.Tokenizer using cl100k_base vocab.
type BPETokenizer struct {
	codec tokenizer.Codec
}

// NewBPETokenizer instantiates a new BPE-based tokenizer.
func NewBPETokenizer() (*BPETokenizer, error) {
	codec, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, err
	}
	return &BPETokenizer{codec: codec}, nil
}

// Estimate counts tokens in the input text using cl100k_base encoding.
func (b *BPETokenizer) Estimate(text string) int {
	ids, _, err := b.codec.Encode(text)
	if err != nil {
		// Fallback to average characters per token approximation
		return len(text) / 4
	}
	return len(ids)
}
