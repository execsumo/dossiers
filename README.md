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

There is no published binary yet, so you build it from source. **You do not need to keep working inside this repo** — once built, the `dossier` binary is self-contained. Install it to a stable location on your `PATH` so its path never changes (this matters for hooks — see [Known limitations](#known-limitations)).

```bash
git clone <repo-url> dossiers
cd dossiers
go build -o ~/.local/bin/dossier ./cmd/dossier   # or /usr/local/bin/dossier
which dossier                                      # confirm it's on PATH
```

If you only want to try it without installing, `go build ./cmd/dossier` drops a `./dossier` binary in the repo and you can run it as `./dossier`.

## Initialize

Run `init` **once, from anywhere** — it does not care about your current directory. It creates your workspace at `~/.dossier` (config, context templates, the Distillation Guide) and reports which harnesses it detected and at what capability tier.

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

### MCP (recommended, works today)

Register Dossier as an MCP server so your agent can read, search, promote, and update Dossiers from inside a session:

```bash
# Claude Code
claude mcp add dossier -- dossier mcp serve
```

For Codex/Antigravity, add an MCP server entry pointing at `dossier mcp serve` per that tool's MCP configuration.

### Session hooks (optional, see limitations)

`dossier init` will offer to install lifecycle hooks (`SessionStart`, `SessionEnd`, `PreCompact`) so your open Dossiers surface automatically when a session starts and state is saved at the right moments. Pass `-y` to accept non-interactively:

```bash
dossier init -y
```

You can also run a hook manually to see what it emits:

```bash
dossier hook session-start    # prints your Dossier library + capabilities
```

> ⚠️ Automatic hook installation uses absolute path bindings — see [Known limitations](#known-limitations) for caveats on rebuilding or moving the binary.

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

- **Hooks store an absolute path to the binary.** The installer registers the absolute path of the `dossier` binary in settings.json (for Claude Code) or hooks.json (for Codex). If you rebuild, rename, or move the binary, the configured hooks will dangle. To resolve this, simply re-run `dossier init` from the new binary location to update the hook bindings automatically and idempotently.
- **Token estimates are approximate.** Dossier uses one BPE tokenizer benchmarked against Opus 4.8 as a reference; it does not match every target model exactly. The 100k-token figure is a configurable warning threshold, not a hard limit — Dossier warns, never silently truncates.
- **Capabilities vary by harness.** Hooks and transcript capture depend on what each harness exposes. Dossier states what's available at install and at session start (e.g. Codex transcript capture is unavailable).

## License

> _TBD._
