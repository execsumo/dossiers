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

> **Stage: All Milestones Completed (Project Fully Finished) + Lead Tracking + TUI Enhancements.** The entire Dossier durable memory layer (codename *chainlink*) is fully implemented, verified, and integrated across all surfaces (CLI, MCP, and the Bubble Tea TUI).
> - **Milestone 1–5:** Core file store, CLI, recall, warnings, lexical search/suggestions, promote/link flow, and the MCP stdio server are implemented.
> - **Milestone 6:** Active session binding, hook installation for Claude Code, confirmation prompts, capability detection, non-clobbering configurations, and the interactive **Rich TUI** (dashboard, detail recall view with native markdown rendering and fsnotify hot-refreshing, status/priority/next-action inline editing, ambiguity link resolution, and syntax-highlighted merge conflict resolving) are fully completed.
> - **Lead Tracking & Accountability:** A newly added `Lead` field allows assigning team members to specific dossiers. Supported end-to-end through `dossier promote --lead`, `dossier lead`, `dossier_update` MCP tool, and visually grouped in the TUI with an inline `a` (assign) editor.
> - **Milestone 7:** Optimistic concurrency control, non-overlapping frontmatter auto-merging, DP LCS unified diff body conflict generation (writing to `conflicts/`), and `dossier merge` CLI/Service commands are verified.
> - **Milestone 8:** The final Distillation Guide is authored in `assets/guide.md` and embedded in the binary to be written to `~/.dossier/context/guide.md` upon initialization. It has been upgraded to employ rigorous linguistic compression (syntactic pruning, lexical density, and negative space framing). All dogfooding validations, test sweeps, and PRD success metrics have been fully met.
> - **Stable Install & Auto-MCP Configuration:** Implemented stable binary self-install path command (`dossier install`, default `~/.local/bin/dossier`), volatile path detection on `init`, and auto-registration of both the MCP stdio server and lifecycle hooks in Claude Code's user/global configuration files (preserving existing third-party configs and backing up changed files).
> - **v1 rescoped to Claude Code only (2026-06-16):** Codex and Antigravity harnesses removed from code and docs. Claude Code is the single supported harness — it provides the full capability set; the degraded Tier 2/3 levels other harnesses reached were insufficient. The `Harness` interface/registry remain for possible future harnesses.
> - **In-session Dossier switching fixed (2026-06-16, ADR 0003):** Previously the MCP `dossier_session` tool (consolidating switch/active) required a `session_id` the agent had no way to obtain, so an agent could not change (or even read) its active Dossier from inside Claude Code. Root cause + fix: Claude Code sets `CLAUDE_CODE_SESSION_ID` in the MCP server's env (verified identical to the transcript UUID and hook stdin `session_id`). A shared `harness.ResolveSessionID` now resolves the session at the adapter edge (precedence: explicit → `CLAUDE_CODE_SESSION_ID` → `DOSSIER_SESSION` → `sess_default`); MCP degrades visibly instead of falling back to the shared bucket. An agent can now call `dossier_session` with just a slug; per-session isolation is preserved. See `docs/harness-capabilities.md` (§ MCP Session Identity) and ADR 0003.
> - **Hardening (2026-06-17):** Addressed critical system stability and performance risks. Prevented OOM crashes by streaming artifact frontmatter instead of loading large bodies. Fixed severe O(N) performance degradation by bypassing full YAML parsing with fast-path string lookups during directory scans. Ensured strict history preservation, bounded LCS diff matrices for massive conflict files, resolved symlink-destructive writes in the harness adapter, and fortified non-interactive stdin blocking. Enhanced `Doctor` with comprehensive provenance validation, artifact origin checks, and unresolved conflict reporting. Improved `SessionEnd` safety by adding explicit audit logging for missing payload scenarios.
> - **TUI de-sessioned (2026-06-17, ADR 0004):** The TUI no longer resolves or carries a session identity and no longer exposes the per-session active binding. Removed the `a` ("make active") key, the `★` active marker/column, the `Session:`/`Active:` header fields, and the standalone-session footer warning — these only ever acted on the `sess_default` bucket that no live agent session reads (the "make-active does nothing for me" / "fixed Session value" confusion). The TUI is now purely a reactive browse/edit viewer (list, markdown recall with live hot-refreshing, status/priority/next-action inline editing, link, syntax-highlighted merge). `Service.Switch`/`Active` are unchanged and remain the CLI/MCP surface for per-session binding. Narrows B9 (see ADR 0004); supersedes ADR 0002 and obsoletes `docs/tui-plan.md`'s catch-up section.

> - **Cleanup & hardening (2026-06-24):** Code-review follow-ups. (1) **Unified write path** — removed `Service.SetStatus`/`SetLead`; CLI, MCP, and TUI now route all metadata edits (status, lead, …) through `Save`, eliminating an adapter fork that violated the "thin adapters over one core.Service" rule. `Save`'s audit entry now records a field-level `field "old"→"new"` summary, and still emits a `status_changed` event (not a generic `save`) when the lifecycle status changes, preserving the agent-facing provenance the dedicated commands had and satisfying SPEC §300. (2) **Quality gate wired up** — added `.github/workflows/ci.yml` (gofmt + `go vet` + `go test` + the long-documented `internal/core` dependency-direction guard); removed a stray `test_glamour.go` scratch file and a gofmt-dirty commit that the gate now catches. (3) **TUI polish** — markdown renderer cached (rebuilt only on width change); hot-refresh now watches every dossier directory so the dashboard live-updates, not just the detail view; `store.Lock` simplified to a single blocking `flock.Lock`. (4) Renaming via `dossier_update` keeps the slug stable (documented in SPEC §8.1).
> - **Simplification of Priority and Dashboard (2026-06-25):** Dropped the cumulative points-based priority scoring in favor of a clean 2x2 Eisenhower matrix (1-4: Do, Plan, Delegate, Delete) strictly mapping importance (high/low) and urgency (high/low). In the TUI dashboard, replaced the Staleness column with a 'Due' column formatted as MM/DD, and expanded the Priority column width to 12 characters to display the full Eisenhower matrix labels.
> - **Programmatic Context Injection (2026-06-25, expanded 2026-06-26):** Migrated from passive injection (`guide.md` printed in `session-start` or requiring an explicit file read) to active interception via the `dossier_session` MCP tool payload. When an LLM calls `dossier_session` to bind or retrieve a dossier, the MCP handler intercepts the `Service.Switch`/`Active` response and wraps it in a struct containing the state AND the full string contents of `guide.md`. This zero-tax, seamless programmatic pattern guarantees strict distillation adherence without polluting global Claude Code prompts (`skill.md`) with a 1500-token overhead on generic tasks. On 2026-06-26, this interception pattern was extended to include runtime operating instructions (`instructions.md`), vastly shrinking the global `skill.md` context bloat down to just a trigger instruction.

> - **TUI Lead landing screen (2026-06-29):** The TUI now opens on a `ViewLeadSelector` landing screen for meeting prep: pick a Lead to scope the dashboard to, with "All" and "Unassigned" pinned first and search-as-you-type narrowing the list (each row shows its dossier count). Selecting a lead drops into the dashboard filtered to that owner; `f` reopens the selector, and the dashboard subtitle shows the active scope. Implementation notes: (1) the filter is a typed `leadFilter{kind, name}` value (not a sentinel string), so a lead literally named "All"/"Unassigned" can't collide with the pinned modes; (2) a single `Model.visibleItems` (the full `items` narrowed by the filter) is the source of truth every table-cursor lookup indexes into — eliminating the row↔item desync that would otherwise hit `enter`/`m`/`getTargetDossier`; (3) lead enumeration/filtering are pure, table-tested helpers (`deriveLeadOptions`, `filterLeadOptions`, `leadFilter.matches`). The dashboard list now fetches **all** statuses (a lead's resolved/archived dossiers must be on hand for prep), and a `statusTier` sort keeps live work (active/waiting/blocked) above terminal work (resolved/archived) so the all-status fetch never buries open work under a high-priority archived item. All logic stays in `internal/tui`; `core`/CLI/MCP are untouched. The selector list is **windowed** (`leadWindow`/`leadVisibleRows`): only a height-bounded slice around the cursor renders, with `↑ N more above` / `↓ N more below` indicators, so a long lead list scrolls instead of overflowing the screen.

> - **Homebrew distribution (2026-07-08):** Added `Formula/dossier.rb` to a shared personal tap, `execsumo/homebrew-tap` (renamed from the Heard-only `homebrew-heard`, so it can host formulas/casks for any of the user's projects going forward). `dossier install` via `~/.local/bin` still works and takes PATH precedence if present, but `brew tap execsumo/tap && brew install dossier` is now the recommended path for keeping multiple devices current. `.github/workflows/release.yml` gained an `update-tap` job that runs after every tagged release: it reads that release's `checksums.txt` and regenerates `Formula/dossier.rb` (version + per-platform `sha256`/`url` for darwin/linux × arm64/amd64) and pushes it to the tap via a `GH_PAT` repo secret (fine-grained PAT, contents:read/write scoped to `homebrew-tap`). From then on `brew upgrade dossier` tracks latest with no manual step. README's Quickstart now leads with this option.

> - **SessionStart slimmed to a one-line nudge (2026-07-15):** Dogfooding found the unbound-session branch of `Service.SessionStart` — a full open-dossier bulletlist (name/status/slug/priority per line) plus a 3-step "check before creating" instruction block — was steering the agent toward thinking about Dossier on every single session, including ones with nothing to do with it, since the hook fires unconditionally on session start. Replaced it with a single-line nudge (`N open dossier(s): <names>. Use dossier_list ... dossier_session ... dossier_promote ...`). This isn't a capability loss: `dossier_promote` already runs its own similarity check server-side and returns `next_actions` guiding the agent through ambiguous matches (see `Service.Promote`), so the old instructional prose was purely redundant with behavior the tool already has. The **bound**-session branch (active dossier's full Distilled State + Distillation Guide inlined) is unchanged — a session with an explicit binding has earned the heavier payload. This is the same "active interception, zero-tax" principle documented under **Programmatic Context Injection** above, now applied to the hook's default-empty path too.

All features (CLI, MCP, and Rich TUI) are fully operational, tested, and integrated.

## Current initiative: Team Sync (started 2026-07-15, branch `team-sync`) — IN PROGRESS

**What/why:** v1's "single-user" premise is amended (owner decision, 2026-07-15): multiple colleagues' sessions will contribute to one shared store, with session captures stashed per-dossier so nothing is lost. Transport is a **private GitHub repo fully hidden behind the binary** (embedded go-git; colleagues never see git). Chosen over a shared-drive live store because drive sync clients last-write-wins behind our backs, violating two hard rules. Decision record: **[ADR 0005](docs/adr/0005-team-sync-via-github.md)** + **B12** in `BUILD-DECISIONS.md`.

**Plan:** `docs/team-sync-plan.md` — 4 phases, strictly ordered, one PR each off the `team-sync` integration branch (merge to `main` only when the whole feature dogfoods clean):
1. **Phase 1 — Identity & store layout v2** (no networking): `author` config field; author-stamped audit events; per-author audit shards `<slug>/audit/<author>.log` (legacy `audit.log` stays readable, never rewritten); per-dossier session stash `<slug>/sessions/<author>/<session-id>.md`; schema bump + additive `Migrate`; SPEC §3.2 + ARCHITECTURE updates in the same PR.
2. **Phase 2 — `Syncer` port + go-git adapter + manual `dossier sync`**: pull→resolve→commit→push; `dossier.md` sync conflicts route into existing `WriteConflict` (`kind: sync_concurrent_edit`, remote wins working tree); PAT auth `0600`; >100 MB artifacts excluded from sync with visible warning; tested against local bare repos.
3. **Phase 3 — Onboarding + auto-sync**: `dossier team create|join`; sync piggybacks on Recall/Save/session hooks (async/debounced, never blocks); TUI/doctor/MCP surfacing. No daemon.
4. **Phase 4 — Docs, dogfood, hardening**: README for the least technical teammate, PRD/SPEC amendments, failure drills, two-colleague pilot.

**Status:** Plan docs committed (this entry, ADR 0005, B12, plan doc, CLAUDE.md premise update). **Phase 1 (Identity & store layout v2) implemented, reviewed, and merged into `team-sync` (2026-07-15)** — delivered by an antigravity delegate, verified against the DoD (build/vet/gofmt clean, 94 tests incl. two-author isolation + legacy-audit-untouched; transcript discovery lives in `harness.ResolveTranscript`, preferring the hook payload's `transcript_path`).

**Spec/docs alignment merged (2026-07-15):** PRD/PRFAQ/SPEC single-user premise amended (dated, history preserved); SPEC §7 gained the `dossier sync` / `dossier team create|join` command surface (marked specified-ahead-of-implementation); SPEC §14.10 Team Sync acceptance criteria added; §4.4 corrected to per-author audit shards; guide.md gained a Multi-author Dossiers etiquette note. Delivered by an antigravity delegate, DoD-verified, merged.

**Phase 2 de-risk spike (go-git sync engine) — DONE & reviewed (2026-07-16).** Delivered by a cline delegate on `delegate/cline-gitsync-spike` (commit `e461156`), scope-fenced to `internal/sync/**` + `go.mod`/`go.sum` + `docs/spikes/gitsync-findings.md`. Prototypes pull→resolve→commit→push against local bare repos, remote-wins conflict capture on `dossier.md` (`ConflictRecord`, zero git merge markers), >100 MB exclusion, store-wide `.sync.lock` — NOT wired into `core.Service`. **Reviewed & independently verified** (build/vet/gofmt clean, 8/8 tests). Findings: `docs/spikes/gitsync-findings.md` (excellent — honestly names its own gaps and gives an 11-point Phase 2 recommendation list). Key go-git finding: `Worktree.Pull` has no per-file remote-wins policy and writes merge markers on divergence, so the spike bypasses it with a custom 3-way remote-wins merge (`DiffTree` + `MergeBase` + per-file blob checkout + a manual 2-parent merge commit).

**Spike reconciled onto a Phase 2 base (2026-07-16):** the spike branched 9 commits behind `team-sync` (base `8b84978`, before Phase 1 + docs-alignment landed). Rebased the single spike commit onto current `team-sync` → branch **`feat/teamsync-p2`** (spike commit `970baaa`). Rebase was conflict-free (Phase 1 never touched `go.mod`/`go.sum`, so the go-git dep diff replayed cleanly). Verified the reconciled base: full-repo `go build`/`go vet`/`gofmt` clean, **102 tests pass**. `feat/teamsync-p2` is the known-good base the Phase 2 wiring delegate builds on.

### Decisions log — owner away 2026-07-16 (decision authority delegated; discuss on return)

- **D1 — Do NOT merge the spike into `team-sync` standalone.** An unwired `internal/sync` package on the integration branch is off-plan ("one PR per phase, each a wired milestone"). Instead the Phase 2 PR lands `internal/sync` *already wired* via `core.Syncer`. Base = `feat/teamsync-p2` (spike rebased onto team-sync).
- **D2 — Phase 2 stays one coherent PR/one delegate; do not fragment it** across agy+cline (fragmenting a single phase creates intra-phase merge seams — the thing that causes rework). agy owns all of Phase 2 including PAT auth.
- **D3 — Delegate routing:** Phase 2 wiring → **agy** (owner's stated default for sequential work). Parallel non-colliding track → **cline** (reserved for genuine parallelism). See memory `prefer-agy-delegate-default`.
- **D4 — cline's parallel track = docs-only, brand-new files** (the only guaranteed zero-collision surface while Phase 2 rewrites code): Phase 4 teammate-facing onboarding + failure-drill runbook as new `docs/` files, describing the already-SPEC'd §7 command surface. Light post-Phase-2 reconciliation is cheap; code collision is impossible. **[DONE — integrated `b114ede`.]**
- **D5 — Post-integration dogfood of `dossier sync` (advisor-prompted) caught a real bug the tests missed; fixed inline** (`bee5507`). Driving the actual cobra→`Service.Sync` path against two throwaway local stores showed the normal onboarding flow (each colleague `dossier init`s their own store → no shared git ancestor) flagged the identical, managed `.gitignore` as a phantom sync conflict. Fixed in `remoteWinsMerge`: only capture a conflict for `<slug>/dossier.md` paths with genuinely differing content; managed/single-writer files take remote-wins silently. Regression test added. I fixed this myself rather than round-tripping a delegate — it was a ~10-line, precisely-root-caused change (delegate-skill rule: don't delegate a quick edit you'd finish before writing the spec). Genuine dossier.md conflicts unaffected (e2e test still green).
- **D6 — Phase 3 split by risk profile; run agy solo, no forced cline parallelism** (advisor-aligned). 3a = `dossier team create|join` (clean/testable → delegate now). 3b = auto-sync on hooks ("async/debounced/never-blocks", behavioral) → spec only after 3a's surface is real, with concrete non-blocking/debounce/failure checks. This is *sequencing two risk profiles*, not fragmenting one phase (respects D2). **No cline parallel track this phase:** Phase 3a spans CLI + config + `internal/sync` (Clone/create), leaving no collision-free surface for a second writer — forcing cline on would manufacture collisions or docs-about-unbuilt-commands (churn). Honest call over busywork; cline resumes at 3a-land with README/onboarding polish against the *real* command surface. (Owner may override if throughput is preferred over churn-avoidance.)

**Phase 2 wiring: DONE & INTEGRATED into `team-sync` (`6d3e559`, 2026-07-16)** — delivered by an agy (antigravity, Gemini 3.1 Pro) delegate, verified + integrated by the orchestrator. `core.Syncer` port in `internal/core/ports.go` (core stays pure — dependency-direction guard passes), core-owned `SyncReport`/`SyncConflict`/`SyncStatus`/`SyncExcluded` DTOs, `internal/sync/adapter.go` mapping `GitSync`→port, `Service.Sync`/`SyncStatus` orchestration, thin `dossier sync` CLI (`--status`/`--json`), sync-conflict routing into the existing `WriteConflict` machinery (`kind: sync_concurrent_edit`, one mechanism/two triggers), `internal/sync/credentials.go` (PAT `~/.dossier/credentials` exact-0600 + `gh auth token` fallback, injectable runner), `team.remote`/`team.branch` config, gitignore machine-local set (`/config.yaml`, root `/sessions/`, `/context/`). End-to-end test `internal/cli/cli_sync_test.go` (two clones converge; concurrent `dossier.md` edits → distinct `conflicts/*.md` remote-wins, zero markers; >100 MB excluded; nil-syncer errors cleanly). **Build/vet/gofmt clean, 105 tests, purity guard passes.** ARCHITECTURE.md §2/§4 document the port. Review caught + fixed: a `confID` collision (time-only ID overwrote a 2nd same-sync conflict → now slug+index unique, with a 2-conflict test); a gofmt nit; agy's missed ARCHITECTURE update (added by orchestrator). Auth is unit-tested only (real-GitHub PAT flow is a Phase 4 dogfood item — local bare repos need no auth). The now-stale `feat/teamsync-p2` base branch and `delegate/*` branches can be pruned.

**Phase 2 is now genuinely closed** — validated end-to-end through the real binary (build → `dossier init`/`promote`/`sync` on two local stores → convergence, remote-wins, zero phantom conflicts), plus the `bee5507` fix + regression test above. `dossier sync` HEAD on `team-sync` is `bee5507`.

**Phase 3a (`dossier team create|join`): DONE & INTEGRATED into `team-sync` (`aa815dd`, 2026-07-16)** — delivered by an agy delegate, verified + integrated by the orchestrator. `team create <url>` inits the store as a git repo (HEAD `main`) + origin remote + pushes existing content; `team join <url>` shallow-clones (depth 50) the remote into the store then runs `Service.Init` (harness/layout), guarding a non-empty target. `core.Syncer` port extended with `Create(ctx)` + `Clone(ctx,url,dir,depth)` (core stays pure); thin CLI (`team create`/`join`) loads→sets→saves `config.Team`→wires→calls `Service.TeamCreate`/`TeamJoin`. **Verified via real-binary dogfood** (create pushes A's store; join clones into B with zero phantom conflicts; join-into-non-empty and unreachable-remote both error cleanly, no panic) + a committed round-trip integration test (`internal/cli/cli_team_test.go`, written by the orchestrator — agy had only run a manual script). Build/vet/gofmt clean, tests pass, purity guard passes. ARCHITECTURE §3/§4 updated.

**Phase 3b (auto-sync on hooks): NEXT.** Sync piggybacks on Recall/Save/session hooks — async/debounced, never blocks a save; no daemon; plus TUI/doctor/MCP sync-status surfacing. Behavioral DoD (per advisor): a fake `Syncer` that blocks on a channel proves `Save`/`Recall` return before sync completes; N rapid saves coalesce into fewer sync calls; a sync failure surfaces as a warning and never fails the `Save`. Spec against the REAL 3a surface (`Service.Sync`, `config.Team`, hooks in `internal/hooks`).

**Remaining Phase 4** (after 3b): README Quickstart pointer to `docs/team-sync-onboarding.md`; final PRD/SPEC amendments; **real-GitHub PAT dogfood + two-colleague pilot — these NEED the owner** (a real private GitHub repo, a PAT, and actual colleagues), so the autonomous run stops at that boundary.

**Resume here if interrupted:** current `team-sync` HEAD carries Phase 2 + 3a (`aa815dd`). All local-bare-repo flows are green + dogfooded. Check `git worktree list`/panes for in-flight delegates; read `docs/spikes/gitsync-findings.md`, `docs/team-sync-plan.md`, and the D1–D6 decisions log above. The plan's "Non-negotiables" + CLAUDE.md's Team Sync hard rule bind all phases. Stale branches to prune: `feat/teamsync-p2`, `delegate/agy-teamsync-p2`, `delegate/cline-*`, `delegate/cline-gitsync-spike`.

> **Active Monitors & Agent Skill (Completed):** Added `## Active Monitors` to the Dossier schema (`assets/guide.md`) for tracking live, mutable context (e.g., Slack threads, Jira tickets) separately from static archived references. We have also created a "Resumption Protocol" Skill (`assets/skill.md`) that is embedded and written out during `dossier init` to `~/.dossier/context/skill.md`. `init` also configures Claude Code's `customInstructions` to point to this skill so agents automatically know to poll these monitors upon resuming a dossier.
> **TUI catch-up to the session-id fix (2026-06-16) — later superseded:** The TUI's session/active presentation was updated to reflect real-session vs fallback (honest header banner + standalone footer warning). This was **superseded on 2026-06-17 by ADR 0004**, which removed the session/active concept from the TUI entirely (see the de-sessioned entry above). The honest-banner/footer work no longer exists.
> **Frontmatter backward compatibility (2026-06-25):** A store written by an older build (e.g. a three-level `importance: medium` from before the binary high/low schema) no longer hard-fails on load or edit. The invariant is *map toward attention, not away*: `Frontmatter.Normalize` (`internal/core/dossier.go`) coerces any value that is no longer a valid enum member, or a field absent from an older file, to its highest-attention valid value (`importance`/`urgency` → `high`, `status` → `active`). It is wired at three layers: (1) **read** — `CalculatePriorityScore` normalizes its copy so legacy values sort toward attention immediately; (2) **write** — `Service.Save` heals + persists + emits a `Warning` for every coercion; (3) **startup** — a version-gated one-time sweep (`Service.Migrate`, `config.schema_version` vs `core.CurrentSchemaVersion`) runs from `wire`, rewriting every stale Dossier once and logging to **stderr** (never stdout, so `mcp serve` stays clean). `doctor` reports legacy values as healable rather than fatal. The write-side source of `medium` was also closed (MCP tool schema, CLI `--importance/--urgency` mapping, SPEC §357). To extend for a future field: add the field's `Normalize` + one block in `Frontmatter.Normalize`, and bump `CurrentSchemaVersion`.



## Planned: `dossier-delegate` skill (design captured 2026-06-30 — NOT wired up)

**Not part of the shipped product yet.** A real, working `SKILL.md` exists at
`~/.claude/skills/dossier-delegate/SKILL.md` (global, personal, testable via
`/dossier-delegate` today) but is deliberately **not** bundled into
`assets/`, `dossier init`'s embed, or the harness install path — that
integration is an explicit follow-up, not assumed.

**Problem:** the user delegates work captured in Dossiers to an overseas team
on an ~8-hour offset. An undefined success criterion or escalation path
doesn't cost a Slack reply there — it costs a full lost day. The user wants
best-practice delegation structure (clear output, validation criteria,
escalation path, decision rights) *available* on a Dossier, but explicitly
**not enforced** — it must never become required overhead on the
zero-friction "dossiers start organically and quickly" flow that's working
today.

**Design (see the skill file for full detail):**
- **Pull-only, explicit invocation** — a slash command / trigger phrases
  ("help me define this exercise," "let's clarify so we can delegate"),
  never triggered by dossier state (no auto-fire off a thin `next_action` or
  a `lead` being set).
- **Method is a stall-simulation, not a fixed rubric:** "read this as the
  teammate, waking up with no way to reach the sender for hours — where do
  they stall?" Biases toward **Decision Rights** and **Escalation** first —
  the two categories that actually cause a full-day loss on an offset team —
  before Objective/Context/Constraints, which are usually already implicit.
  Output is binary/qualitative (name the missing fact, or say "ready") —
  **never a completeness score**, which would recreate the enforcement feel
  the user is avoiding.
- **Persisted contract vs. rendered note (resolves a real tension):** the
  outbound note is verbose by design (frontloading is the point); the
  Distilled State is terse by design (the Distillation Guide fights bloat).
  Resolution: persist a *compressed* delegation contract (Objective /
  Success Criteria / Validation / Constraints / Decision Rights / Escalation)
  into the Dossier body via the existing `Save` path — so it's
  optimistic-concurrency protected and every change lands in the audit log
  with a field-level diff. That answers the user's "how do I check success
  criteria was met without moving the goalpost on my team" concern: the
  goalpost can still move, but never silently — a later re-render must flag
  drift between the persisted contract and current dossier state rather than
  silently rendering the new state as original. The paste-able Slack/Jira
  note is then a pure rendering of that persisted contract, regenerated on
  demand, never inventing new criteria at render time.
- **Return contract:** the teammate has no MCP access (human, async, Slack/
  Jira) — the note defines the exact shape of their expected reply so it can
  be pasted back into the Dossier later, rather than assuming they can write
  to the store themselves.
- Reuses existing frontmatter fields (`next_action`, `open_questions`,
  `lead`) as the pre-delegation gap-tracking home wherever possible — no
  core/SPEC schema change for v1.

**Explicitly deferred (named, not forgotten):**
- Bundling into `dossier init`'s installed asset set.
- A new `delegation_note` artifact type — the existing SPEC §4.3 artifact
  taxonomy (transcript/source_snapshot/file_snapshot/link/query/
  decision_evidence) is entirely about *captured* source material, not
  synthesized outbound documents, so a generated note doesn't cleanly fit
  today's artifact model. Revisit only if rendered-note-only proves
  insufficient in practice.
- Per-teammate formatting/escalation preferences.
- Scoping a contract to a single Next Step vs. the whole Dossier (multiple
  concurrent delegated threads on one topic).

**Grounded in:** Anthropic's agent-design/context-engineering writing,
Karpathy's autoresearch loop, and this environment's own `delegate` skill's
7-block Spec Contract (`~/.claude/skills/delegate/SKILL.md`) — adapted here
for a human, async, offset delegate rather than a supervised agent.

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
