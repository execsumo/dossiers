# Dossier

> A local, single-user durable memory layer for agent-driven work — across Claude Code, Codex, and Antigravity.

Dossier keeps your long-running agent work alive across sessions and across tools — without the bloat. You promote any agent session into a **Dossier**: the critical state of a topic (situation, decisions and who made them, open questions, next action) with the noise stripped out, backed by an **Archive** of the raw material that supports it. Every material claim cites its source. You resume with exactly the distilled context you need — with the full archive one search away.

Local and single-user. No database — your data is plain Markdown you can open in any reader (e.g. Obsidian).

> **Naming:** *Dossier* (singular) is the tool. *A Dossier* is one durable topic; *Dossiers* are many. The repository directory is `dossiers`, but the project is **Dossier**.

## Status

All milestones (1–8) are implemented and committed to `main`: core domain, file store, recall/search, MCP server, session hooks, promote/link/merge, concurrency handling, and the shipped Distillation Guide. The CLI, MCP, and hook surfaces are functional. See `HANDOFF.md` for the per-milestone status and `SPEC.md` for the full contract.

See [Known limitations](#known-limitations) before relying on automatic session hooks.

## Requirements

- **Go 1.22+** to build (there is no prebuilt release yet).
- macOS or Linux. One or more supported harnesses installed: Claude Code, Codex, or Antigravity.

## Install

There is no published binary yet, so you build it from source. **You do not need to keep working inside this repo** — once built, the `dossier` binary is self-contained. Install it to a stable location on your `PATH` using the `install` command so its path never changes:

```bash
git clone <repo-url> dossiers
cd dossiers
go build ./cmd/dossier
./dossier install                                  # copies the binary to stable ~/.local/bin/dossier
which dossier                                      # confirm it's on PATH
```

If you only want to try it without installing, `./dossier` can be run directly from the repo directory.

## Initialize

Run `init` **once, from anywhere**. If Dossier detects it is running from a volatile path (such as a temporary folder or repository build folder), it will offer to self-install itself to a stable path first.

It then creates your workspace at `~/.dossier` (config, context templates, the Distillation Guide) and automatically registers the Dossier MCP server and lifecycle hooks in your active harnesses.

```bash
dossier init
```

Example output:

```
Dossier initialized at /Users/you/.dossier

Harness support:
- Claude Code: Tier 1 (MCP, hooks, transcript capture detected)
- Codex:       Tier 2 (MCP/hooks detected, transcript capture unavailable)
- Antigravity: Tier 3 (context/MCP fallback only)
```

Verify the workspace any time:

```bash
dossier doctor      # "Dossier workspace is healthy!"
```

Your data lives in `~/.dossier/` as plain files and **persists across restarts**. Dossier is not a daemon — there is no background process; the binary is invoked on demand by you, by hooks, or by the MCP server.

## Connect it to your agent

`dossier init` automatically detects supported harnesses and configures both the MCP server and session hooks (`SessionStart`, `SessionEnd`, `PreCompact`) in their user/global configuration files. 

It does this after prompting for confirmation per harness (skipped if `-y` is passed). The installation is fully idempotent, non-clobbering (preserving other MCP servers and hooks), and backs up config files before editing.

### Automatic configuration
Simply run:
```bash
dossier init
```

If it detects Claude Code or Codex on your machine, it will offer to integrate itself:
```
Configure Claude Code integration (hooks + MCP server)? [y/N]: y
```

### Manual configuration (if needed)

If a harness cannot be automatically configured (e.g. Antigravity), Dossier will warn you during `init` and print manual setup instructions.

#### MCP
To register Dossier manually:
- **Claude Code:** Registered in `~/.claude.json` under `"mcpServers"` (or locally via `claude mcp add dossier -- dossier mcp serve`).
- **Codex:** Registered in `~/.codex/config.toml` under `[mcp_servers.dossier]`.
- **Antigravity:** Add a stdio MCP server pointing to the stable binary path with arguments `["mcp", "serve"]`.

#### Session hooks
Run hooks manually to see what they emit:
```bash
dossier hook session-start    # prints your Dossier library + capabilities
```

> ⚠️ Automatic hook/MCP configuration uses stable path bindings. If you relocate the binary, run `dossier install` and `dossier init` again to re-bind paths.

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

> _TBD._
