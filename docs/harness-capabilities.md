# Harness Capability Matrix

This document defines the integration capabilities and resulting tier for each supported agent harness, verified through analysis of their local configuration files and command-line interfaces.

## 1. Capability Matrix

| Feature | Claude Code | Codex | Antigravity |
|:---|:---|:---|:---|
| **Config File Path** | `~/.claude.json`<br>`~/.claude/settings.json` | `~/.codex/config.toml`<br>`~/.codex/hooks.json` | N/A |
| **MCP Registration Path** | `~/.claude.json` -> `mcpServers` | `~/.codex/config.toml` -> `[mcp_servers]` | System MCP registry / Dynamic |
| **Hook Configuration** | `"hooks"` in `settings.json` | `"hooks"` in `hooks.json` | N/A |
| **Hook Payload Format** | JSON on `stdin` (includes `session_id`, `hook_event_name`) | JSON on `stdin` (includes `session_id`) | N/A |
| **SessionStart Hook** | Yes (`SessionStart`) | Yes (`SessionStart`) | No |
| **SessionEnd Hook** | Yes (`SessionEnd`) | Partial (uses `Stop` event) | No |
| **Pre-Compaction Hook** | Yes (`PreCompact`) | No | No |
| **Raw Transcript Access** | Yes (via session UUID matching) | No | No |
| **Stable Session ID** | Yes (UUID string in payload) | Yes (UUID string in payload) | No |
| **Context Injection** | Yes (Stdout from `SessionStart` hook) | No | No |
| **Install/Notice Surfacing** | Yes (During init & session start) | Yes (Warnings during init) | Yes (Warnings in MCP responses) |
| **Resulting Tier** | **Tier 1** | **Tier 2** | **Tier 3** |

---

## 2. Harness Integration Details

### Claude Code (Tier 1)
- **MCP Path:** Stdio-based server registered globally in `~/.claude.json` under `"mcpServers"` or locally in a project's `.mcp.json`.
- **Hooks:** Lifecycle hooks trigger commands. The standard output of the `SessionStart` hook is directly injected into Claude Code's active context window. The `PreCompact` hook triggers just before history truncation, enabling a final `Save` of the session's active Dossier context.
- **Session ID:** A stable UUID is passed in the JSON payload on `stdin` to any hook handler.

### Codex (Tier 2)
- **MCP Path:** Stdio-based server registered in `~/.codex/config.toml` under `[mcp_servers.<name>]`.
- **Hooks:** Configured in `~/.codex/hooks.json`. Supports `SessionStart` and `Stop` (acting as session end). No `PreCompact` hook is available.
- **Session ID:** Available as a stable string in the hook payload on `stdin`.
- **Degradation:** Lacks direct context injection from hook stdout and transcript capture capabilities. Install and start notices must visibly warn the user.

### Antigravity (Tier 3)
- **MCP Path:** Relies on standard client registration.
- **Hooks:** No hooks are supported.
- **Session ID:** No stable session identifier is exposed.
- **Degradation:** Relies entirely on manual CLI/TUI switching or MCP tool calls. Capability warnings must degrade visibly by appending warning structures to MCP responses.
