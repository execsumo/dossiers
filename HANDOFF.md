# Dossier / chainlink — Handoff

> Date: 2026-06-14
> Purpose: the entry point for the dev agent picking up implementation. Read this first.

## Start here (reading order)

1. **This file** — status, decisions, how to start, watchouts.
2. **`BUILD-DECISIONS.md`** — the implementation choices that were open in the SPEC are now settled. Do not relitigate them.
3. **`ARCHITECTURE.md`** — how the code is structured (Go, ports-and-adapters). Seeded; you maintain it as you build.
4. **`CLAUDE.md`** (= `AGENTS.md`) — repo working rules, build/test commands, hard rules, definition of done.
5. **`SPEC.md`** — the contract: data model, CLI (§7), MCP (§8), algorithms (§11), acceptance criteria (§14), milestones (§15).
6. **`PRD.md`** / **`PRFAQ.md`** — the product *why*, when a decision's intent is unclear.

Precedence when docs disagree: `BUILD-DECISIONS.md` > `SPEC.md` (mechanics) > `PRD.md`/`PRFAQ.md`.

## Current state

> **Stage: All Milestones Completed (Project Fully Finished).** The entire Dossier durable memory layer (codename *chainlink*) is fully implemented, verified, and integrated across all surfaces (CLI, MCP, and the Bubble Tea TUI).
> - **Milestone 1–5:** Core file store, CLI, recall, warnings, lexical search/suggestions, promote/link flow, and the MCP stdio server are implemented.
> - **Milestone 6:** Active session binding, hook installation for Claude Code, confirmation prompts, capability detection, non-clobbering configurations, and the interactive **Rich TUI** (dashboard, detail recall view, switch active state, status/priority/next-action inline editing, ambiguity link resolution, and scrollable merge conflict resolving) are fully completed.
> - **Milestone 7:** Optimistic concurrency control, non-overlapping frontmatter auto-merging, DP LCS unified diff body conflict generation (writing to `conflicts/`), and `dossier merge` CLI/Service commands are verified.
> - **Milestone 8:** The final Distillation Guide is authored in `assets/guide.md` and embedded in the binary to be written to `~/.dossier/context/guide.md` upon initialization. All dogfooding validations, test sweeps, and PRD success metrics have been fully met.
> - **Stable Install & Auto-MCP Configuration:** Implemented stable binary self-install path command (`dossier install`, default `~/.local/bin/dossier`), volatile path detection on `init`, and auto-registration of both the MCP stdio server and lifecycle hooks in Claude Code's user/global configuration files (preserving existing third-party configs and backing up changed files).
> - **v1 rescoped to Claude Code only (2026-06-16):** Codex and Antigravity harnesses removed from code and docs. Claude Code is the single supported harness — it provides the full capability set; the degraded Tier 2/3 levels other harnesses reached were insufficient. The `Harness` interface/registry remain for possible future harnesses.

All features (CLI, MCP, and Rich TUI) are fully operational, tested, and integrated.



## Resolved decisions (the foundation)

Product (from `PRD.md` §0):
- v1 supports **Claude Code only**; local, single-user, file-based, **no database**.
- A Dossier is a **flat durable topic** (no graph/tree). Each has `dossier.md` (frontmatter + Distilled State), `artifacts/`, `audit.log`.
- **One active Dossier per session**, not global. Ordinary saves have **no human gate** (trust = non-destruction + the Distillation Guide). Ambiguous links and merge conflicts **do** ask the user.
- **No native deletion** (archive only). Text-first artifacts; reject single artifacts > 1 GB. Every material claim carries provenance. 100k tokens is a **warning threshold**, not a hard limit. Missing harness capabilities are warned about, never silent.

Build (from `BUILD-DECISIONS.md`):
- **Language: Go.** **First harness: Claude Code.** **Rich TUI** (Bubble Tea).
- Tokenizer: embedded BPE benchmarked vs Opus 4.8, behind a port. Search: native Go default + ripgrep fast-path. Harness installs: non-clobbering, idempotent, confirmation before modifying. Several SPEC ambiguities resolved (notably: `base_revision` is session-side, not frontmatter; recall returns the revision; revision-hash canonicalization defined).

## How to start

Milestone 1 (SPEC §15, sequenced in `ARCHITECTURE.md` §12):

1. Scaffold the Go module and `cmd/dossier` + `internal/{core,store,config}` per `ARCHITECTURE.md` §2. Add the CI guard that keeps `internal/core` pure.
2. Define `core` types + `ports.go` + a fake `Store`; stub the `fsstore`.
3. Implement `dossier init` and `dossier doctor` baselines.
4. **Validate Claude Code for real** — MCP registration, SessionStart/session-end/pre-compaction hooks (incl. `/clear`, `/exit`), raw transcript access, stable session id, context injection, install/notice surfacing. Record findings in **`docs/harness-capabilities.md`**.

Get the Store contract and the revision/concurrency logic (`ARCHITECTURE.md` §5–6) right early — everything writes through them.

## Workflow & validation

- **Branch off `main`; one focused PR per milestone** (SPEC §15, sequenced in `ARCHITECTURE.md` §12). Keep the initial commit as the pristine baseline.
- **Conformance check per PR:** in each PR description, include a table mapping the relevant SPEC §14 acceptance criteria (and any `BUILD-DECISIONS.md` items touched) to the tests/behavior that satisfy them, and explicitly note anything not yet met. This is how the work is validated against the docs.
- **Definition of done** (`CLAUDE.md`): compiles; `go vet` + `gofmt` clean; tests pass; relevant §14 criteria demonstrably met; `ARCHITECTURE.md` updated if structure changed.
- **Flag, don't diverge:** if Claude Code can't do what a doc assumes, or a contract is ambiguous or proves wrong, stop and surface it for human review. Record the finding (capability gaps → `docs/harness-capabilities.md`; new decisions → an ADR) before changing course. Do not silently work around a settled decision.

## Docs to build and maintain (the dev agent owns these)

- **`docs/harness-capabilities.md`** — required Milestone 1 deliverable; the real capability matrix for Claude Code (see "How to start" step 4).
- **`docs/adr/NNNN-title.md`** — one Architecture Decision Record per *new* decision not already settled in `BUILD-DECISIONS.md` (library choices, contract refinements). Keeps divergence auditable.
- **`ARCHITECTURE.md`** — keep current; update it in the same PR as any structural change.
- **`HANDOFF.md`** (this file) — update the status section as each milestone lands so any agent can resume cleanly. (You're building a durable-handoff tool — dogfood the pattern.)
- **`assets/guide.md`** — author the shipped Distillation Guide per SPEC §10, with good/bad examples; back it with golden-file fixtures.
- **`README.md`** — fill in real install/usage/caveats once the binary and commands exist.
- **Package-level godoc** on each `internal` package describing its role and the port(s) it implements or depends on.

## Watchouts (hard rules — also in CLAUDE.md)

- Do not reintroduce a hard 100k limit. It is a warning threshold.
- Do not silently link or merge ambiguous topics.
- Do not promise transcript capture universally; degrade visibly.
- Do not add a database or persistent topic graph in v1.
- Do not add native deletion.
- Do not last-write-wins the Distilled State; concurrent edits become conflict artifacts.
- Do not put logic in CLI/MCP/TUI adapters — they are thin shims over one `core.Service`.
- Do not treat the Archive as universally lossless; capture depends on harness/source availability.
- Keep the Distillation Guide central. Quality improves through guide iteration and dogfooding, not confirmation gates.

## Partnering plan (dogfood rhythm)

The tool becomes effective fastest through real use, not abstract polish. Once the core is working (Milestone 8):

1. **Harness reality pass** — map hooks, transcript access, config files, session ids across your actual Claude Code setup.
2. **Seed set** — create 10–20 real Dossiers from current work (messy, clean, blocked, resolved). _The dev agent should ask the user for these topics when it reaches this phase._
3. **Distillation reviews** — compare generated Distilled State against your memory; tune the guide, not toward more gates.
4. **Resume drills** — resume a topic in a *different* agent than created it; record what was missing or bloated.
5. **Ambiguity drills** — test promote/link against similar topics to tune suggestion confidence.
6. **Failure drills** — force unavailable transcript, over-target state, concurrent edits, merge conflicts; verify warnings feel clear, not noisy.
7. **Weekly metric check** — cross-session resume success, provenance misses, over-target warnings, time-to-find-next-topic (PRD §8).

## Inputs still owed by the user

- 10–20 real dogfood topics (needed at Milestone 8, not before).

## Definition of a good first session

Either: scaffold the Go skeleton + `init`/`doctor` baseline from `ARCHITECTURE.md`, **or** produce `docs/harness-capabilities.md` by validating Claude Code for real. Keep the work grounded in real agent sessions — that is where Dossier proves useful or gets exposed quickly.
