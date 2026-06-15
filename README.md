# Dossier

> A local, single-user durable memory layer for agent-driven work — across Claude Code, Codex, and Antigravity.

Dossier keeps your long-running agent work alive across sessions and across tools — without the bloat. You promote any agent session into a **Dossier**: the critical state of a topic (situation, decisions and who made them, open questions, next action) with the noise stripped out, backed by an **Archive** of the raw material that supports it. Every material claim cites its source. You resume with exactly the distilled context you need — with the full archive one search away.

Local and single-user. No database — your data is plain Markdown you can open in any reader (e.g. Obsidian).

> **Naming:** *Dossier* (singular) is the tool. *A Dossier* is one durable topic; *Dossiers* are many. The repository directory is `dossiers`, but the project is **Dossier**.

## Status

All milestones (1–8) are implemented and committed to `main`: core domain, file store, recall/search, MCP server, session hooks, promote/link/merge, concurrency handling, and the shipped Distillation Guide. The CLI, MCP, and hook surfaces are functional. See `HANDOFF.md` for the per-milestone status and `SPEC.md` for the full contract.

See [Known limitations](#known-limitations) before relying on automatic session hooks.

## Requirements

- **Go 1.26+** to build (there is no prebuilt release yet).
- macOS or Linux. One or more supported harnesses installed: Claude Code, Codex, or Antigravity.

## Install & set up

There is no published binary yet, so you build from source. After that, **`dossier init` is the only command you need** — it sets everything up in one pass:

```bash
git clone https://github.com/execsumo/dossiers.git
cd dossiers
go build ./cmd/dossier
./dossier init        # self-installs to a stable PATH, then configures everything
```

Running `init` from the build directory triggers a self-install: because the binary is on a volatile path, `init` offers to copy itself to a stable location on your `PATH` (`~/.local/bin/dossier`) so its path never changes, then continues. (Pass `-y` to accept without prompting; once installed, `dossier init` from anywhere skips this step.)

`init` then, in the same run:

- creates your workspace at `~/.dossier` (config, context templates, the Distillation Guide), and
- detects your supported harnesses and registers the Dossier **MCP server** and lifecycle **hooks** (`SessionStart`, `SessionEnd`, `PreCompact`) in their user/global config.

Harness configuration is confirmed per harness (skipped with `-y`), idempotent, non-clobbering (your other MCP servers and hooks are preserved), and backs up each config file before editing:

```
Configure Claude Code integration (hooks + MCP server)? [y/N]: y
```

Example output:

```
Dossier initialized at /Users/you/.dossier

Harness support:
- Claude Code: Tier 1 (MCP, hooks, transcript capture detected)
- Codex:       Tier 2 (MCP/hooks detected, transcript capture unavailable)
- Antigravity: Tier 3 (context/MCP fallback only)
```

Verify any time:

```bash
which dossier       # confirm it's on PATH
dossier doctor      # "Dossier workspace is healthy!"
```

Your data lives in `~/.dossier/` as plain files and **persists across restarts**. Dossier is not a daemon — there is no background process; the binary is invoked on demand by you, by hooks, or by the MCP server.

> ⚠️ The hook/MCP config stores the absolute path of the stable binary. If you later rebuild, rename, or move it, re-run `dossier install` (re-copies to the stable path) then `dossier init` (re-binds config) to fix the paths idempotently.

### Manual configuration

If a harness can't be auto-configured (e.g. Antigravity), `init` warns and prints setup instructions. To wire it up by hand:

**MCP server**
- **Claude Code:** under `"mcpServers"` in `~/.claude.json` (or `claude mcp add dossier -- dossier mcp serve`).
- **Codex:** under `[mcp_servers.dossier]` in `~/.codex/config.toml`.
- **Antigravity:** add a stdio MCP server pointing to the stable binary path with args `["mcp", "serve"]`.

**Session hooks** — run a hook directly to see what it emits:
```bash
dossier hook session-start    # prints your Dossier library + capabilities
```

## Everyday use

```bash
# Create a Dossier from a topic (optionally seed its distilled state)
dossier promote "payments-migration" --distilled "## Situation
Migrating billing off the legacy gateway.
## Next action
Confirm webhook signing keys with the vendor."

dossier ls                        # open Dossiers, sorted by priority
dossier show payments-migration   # full distilled state + metadata
dossier search "webhook"          # search distilled state + archives
dossier next payments-migration "Write the cutover runbook"
dossier priority payments-migration --importance high --urgency high
dossier link payments-migration --from-file ./notes.md   # attach source to the archive
dossier merge old-slug payments-migration                # fold one Dossier into another
dossier archive payments-migration                       # archive (never deletes)
```

Full command reference: `dossier --help` and `SPEC.md`.

## How it works

- **Distilled State** (one Markdown file per Dossier) + **Archive** (captured source artifacts) + an append-only `audit.log`, all under `~/.dossier/<slug>/`.
- One **Go** binary serving the CLI, an **MCP** server over stdio, session **hooks**, and a **TUI**.
- A shipped **Distillation Guide** steers *what* the agent keeps; deterministic hooks steer *when* it saves. There is no confirmation gate — trust comes from non-destruction, the audit log, and source provenance on every claim.
- **Non-destructive always:** superseded content moves to the Archive and audit log; nothing is deleted.

## Known limitations

- **Config is split across files for Claude Code.** Hooks are written to `~/.claude/settings.json`; the MCP server is registered in `~/.claude.json` (the only place Claude Code reads user-scope MCP servers). Codex keeps both in `~/.codex/config.toml`. Each entry stores the absolute path of the stable `dossier` binary — if you rebuild, rename, or move it, re-run `dossier install` then `dossier init` to re-bind paths idempotently. `init` also strips any stale `dossier` MCP entry an older build mistakenly wrote into `settings.json`.
- **Token estimates are approximate.** Dossier uses one BPE tokenizer benchmarked against Opus 4.8 as a reference; it does not match every target model exactly. The 100k-token figure is a configurable warning threshold, not a hard limit — Dossier warns, never silently truncates.
- **Capabilities vary by harness.** Hooks and transcript capture depend on what each harness exposes. Dossier states what's available at install and at session start (e.g. Codex transcript capture is unavailable).

## License

[MIT](LICENSE) © 2026 Herwin Gill
