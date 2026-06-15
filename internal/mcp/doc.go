// Package mcp provides the Model Context Protocol (MCP) stdio server implementation.
// It acts as a driving adapter over core.Service, translating JSON-RPC 2.0 requests
// from client harnesses (like Claude Code and Codex) to core operations, mapping errors
// to Spec-compliant error codes, and returning wrapped result envelopes.
package mcp
