# ADR 0003: MCP Session ID Resolved from CLAUDE_CODE_SESSION_ID

## Status
Accepted

## Context
The Dossier MCP tools `dossier_switch` and `dossier_active` require a `session_id` parameter to operate. However, the calling agent is never told its own session ID, causing both tools to fail with a `session_id is required` error during live MCP calls. 

We verified that Claude Code sets the `CLAUDE_CODE_SESSION_ID` environment variable in the environment of the MCP server process (`dossier mcp serve`). This UUID is the canonical session identity and is identical across all surfaces (the MCP server environment variable, the transcript filename under `~/.claude/projects/`, and the `~/.claude/session-env/` entry). We need to resolve this session ID automatically in the MCP server context to allow switching or querying the active Dossier without requiring the agent to supply its own session ID.

## Decision
We choose to resolve the session ID in adapters using a strict precedence ladder. The core domain code remains pure and continues to take an explicit `SessionID` (or equivalent type), while the adapter handles environment resolution.

The precedence ladder for resolving the session ID is:
1. Explicit `session_id` parameter (if passed directly by the caller).
2. `CLAUDE_CODE_SESSION_ID` environment variable (the per-session key set by Claude Code).
3. `DOSSIER_SESSION` environment variable (manual or power-user override).
4. **CLI/TUI only:** Fall back to `"sess_default"`.

Crucially, the MCP server must **not** fall back to `"sess_default"`. If options 1–3 are empty, the MCP adapter must return a clear error and degrade visibly. Silently binding to a shared default bucket would cross-contaminate concurrent sessions.

## Consequences
- Switching and querying the active dossier now works seamlessly from inside a session using only the dossier slug (since the session ID is automatically resolved).
- Strict per-session isolation is preserved, preventing cross-contamination between concurrent sessions.
- Older versions of Claude Code (or other harness environments) that lack the `CLAUDE_CODE_SESSION_ID` environment variable will degrade to a visible error rather than silently sharing a default session bucket.
- A single Claude Code session may spawn two concurrent `dossier mcp serve` processes. Because both inherit the identical `CLAUDE_CODE_SESSION_ID`, environment-based resolution remains unambiguous and safe.
