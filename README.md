# Dossier

> Codename: *chainlink* · A local, single-user durable memory layer for agent-driven work.
> Status: **Milestone 1 completed** — skeletons, harnesses verified, and init/doctor ready. See `HANDOFF.md`.

Dossier keeps your long-running agent work alive across sessions and across tools — without the bloat. You promote any agent session into a **Dossier**: the critical state of a topic (situation, decisions and who made them, open questions, next action) with the noise stripped out, backed by an **Archive** of the raw material that supports it. Every material claim cites its source. Your open Dossiers surface automatically when you start a supported agent session, and you resume with exactly the distilled context you need — with the full archive one search away.

v1 supports **Claude Code, Codex, and Antigravity** with visible capability tiers per harness. Local and single-user. No database — your data is plain Markdown you can open in any reader (e.g. Obsidian).

## Status

Milestone 1 is fully implemented. The Go module is scaffolded, CLI/core/store/harness packages are built, capabilities of harnesses verified, and `init` and `doctor` command baselines are functional and tested.

## Install & Build

Build the single self-contained binary:

```bash
go build ./cmd/dossier
```

Initialize your Dossier workspace (creates `~/.dossier`, configs, context templates, and registers capabilities):

```
./dossier init
```

Then open a supported agent session — it surfaces your Dossier library, tells you the capabilities available in that harness, and helps you continue an existing topic or promote the current conversation into a new one.

## How it works

- **Distilled State** (one Markdown file per Dossier) + **Archive** (captured source artifacts) + append-only `audit.log`, all under `~/.dossier/<slug>/`.
- One **Go** binary serving CLI, an **MCP** server over stdio, session **hooks**, and a **TUI**.
- A shipped **Distillation Guide** steers what the agent keeps; deterministic hooks steer when it saves. No confirmation gate — trust comes from non-destruction, the audit log, and provenance.

See `SPEC.md` for the full command and tool surface.

## Caveats

- **Token estimates are approximate.** Dossier uses one BPE tokenizer benchmarked against Opus 4.8 as a reference; it does not match every target model exactly. The 100k-token figure is a configurable warning threshold, not a hard limit.
- **Capabilities vary by harness.** Hooks and transcript capture depend on what each harness exposes; Dossier states what's available at install and at session start.

## License

> _TBD._
