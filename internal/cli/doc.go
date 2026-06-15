/*
Package cli implements the Cobra-based CLI driver adapter for Dossier.

It parses command-line arguments and flags, instantiates and wires the core.Service
with its required driven adapters (fsstore, harnesses, search, tokenizers),
invokes the appropriate Service methods, and formats results as formatted
text or JSON envelopes.
*/
package cli
