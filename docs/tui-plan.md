# TUI Implementation Plan (for agy)

> Scope: build the Rich TUI for Dossier — the one remaining unbuilt v1 surface.
> Settled decision: `BUILD-DECISIONS.md` **B3** (Rich TUI, Bubble Tea) and **B9** (every op reachable via CLI *and* TUI). Planned home: `internal/tui/` (`ARCHITECTURE.md` §2, §3). Listed in Milestone 6.
> Author: claude (sibling pane). Owner: agy.

## Prime directive — read these first, do not relitigate

1. `CLAUDE.md` (repo working rules + hard rules), `HANDOFF.md`, `BUILD-DECISIONS.md` (B3, B9), `ARCHITECTURE.md` (§2 layout, §3 Service facade, §3.1 surfaces, the cardinal rule).
2. The TUI is a **thin adapter**. Hard rule (CLAUDE.md): *"Do not put logic in CLI/MCP/TUI adapters — they are thin shims over one `core.Service`."* The TUI may ONLY:
   - call `core.Service` methods,
   - render `core.Result` (`data`, `warnings`, `next_actions`),
   - manage view/keyboard state.
   It must NOT: parse frontmatter, touch the filesystem, re-sort priority, re-implement warnings, or re-derive anything core already returns. If you find yourself needing logic that isn't on `core.Service`, **stop and flag it** — do not add it in the adapter.
3. Behavior must be **identical** to CLI/MCP because all three call the same `core.Service`. Warnings (over-token-target, transcript-unavailable, etc.) are produced once in core and must be surfaced verbatim, never re-created.

## Dependency direction (CI-guarded)

`core` imports no siblings. Allowed: `cli → tui → core`. The `tui` package may import `internal/core` (types + Service) and the Bubble Tea libs. It must **not** import `store`, `search`, `harness`, `config`, `cli`, or `mcp`. Service construction stays in `cli` (see Wiring below) and is passed into the TUI.

## Libraries to add

- `github.com/charmbracelet/bubbletea` (Elm-style model/update/view)
- `github.com/charmbracelet/lipgloss` (styling)
- `github.com/charmbracelet/bubbles` (list, table, viewport, textinput, key, help)

Run `go get` for each; commit the resulting `go.mod`/`go.sum`. Keep versions current/compatible.

## Integration seams (already in the code — reuse, don't reinvent)

- `core.Service` methods you will call: `List`, `Recall`, `Switch`, `Active`, `SetStatus`, `SetPriority`, `SetNextAction`, `SetOpenQuestions`, `Archive`, `Link`, `Merge`, `Search`, `Doctor`. (Signatures: `func(ctx, XxxReq) (core.Result, error)`; `SessionStart/SessionEnd` differ — ignore for TUI.)
- Wiring: `internal/cli/cli.go` has unexported `wire(dossierHome) (*core.Service, error)` and `resolveSessionID()` (flag → `DOSSIER_SESSION` env → `sess_default`). The bare `dossier` root command currently has **no** `Run`.
  - **Launch path:** add a `RunE` to `rootCmd` (in `cli.go`) so bare `dossier` (no subcommand) wires the service + resolves the session id and calls `tui.Run(ctx, svc, sessionID)`. Also add an explicit `dossier tui` subcommand that does the same (so it's discoverable). This satisfies `ARCHITECTURE.md` §241 ("`--tui` or the bare `dossier` ... can launch the TUI").
  - Do **not** export `wire`/`resolveSessionID` into other packages; call them from within `cli` and pass results into `tui`. This keeps the dependency direction clean.
- `core.Result.Warnings` / `.NextActions` — render these in a status/footer area on every view.

## What to build (scope = B3)

A full-screen Bubble Tea app. Build in this order so the tree always compiles and each step is demoable:

### Step 1 — Skeleton + dashboard (open-work list)
- `internal/tui/tui.go`: `Run(ctx, svc *core.Service, sessionID string) error` — sets up the program, root model, alt-screen.
- Root model holds: the Service, sessionID, current view, window size, a shared footer (warnings + key help).
- **Dashboard view:** priority-sorted open-work list from `svc.List`. Columns: name, status, lead, priority score, next action, last touched. (Columns conditionally drop on smaller terminal widths). Use `bubbles/table`.
- Keys: `↑/↓` move, `q` quit, `?` toggle help, `enter` open detail.
- Footer surfaces `Result.Warnings`.

### Step 2 — Detail view (recall)
- `enter` on a row → call `svc.Recall` for that dossier → show Distilled State in a `bubbles/viewport` (scrollable). Show token estimate + over-target warning from the Result. `esc` back to dashboard.

### Step 3 — Inline editing (status / priority / next action)
- From detail (or dashboard) `s` → status picker (active/waiting/blocked/resolved/archived) → `svc.SetStatus`.
- `p` → priority editor → `svc.SetPriority`.
- `n` → next-action `textinput` → `svc.SetNextAction`.
- After each mutation, re-fetch via `svc.List`/`svc.Recall` and re-render. Show any returned warnings; on error show the typed domain error message (don't crash).

### Step 4 — Switch (bind active dossier to session) — REMOVED (see ADR 0004)
> This step was built, then removed on 2026-06-17: the TUI no longer exposes `Switch`/`Active`. Kept for history.
- `enter`/`a` action "make active" → `svc.Switch{ID, SessionID}` → reflect new active binding (call `svc.Active` to confirm). Mark the active dossier in the dashboard.

### Step 5 — Link & merge conflict resolution (the interactive payoff)
- **Link:** a view that calls `svc.Link` with no id to get candidates (`requires_selection`), lists them with confidence/reason, lets the user pick (or cancel). Never auto-pick low confidence — honor the hard rule "no silent link of ambiguous targets."
- **Merge:** pick source + target, call `svc.Merge`; if `conflict_detected`, render the conflicts and let the user resolve, then re-call with `resolved_conflicts`. Surface that source files are retained, not deleted.
- These two are the reason the TUI exists (B3); give them real care. If the Service surface can't express a resolution step the UI needs, **flag it** rather than working around it in the adapter.

## Open question to resolve (decide + document, don't guess silently)

- **Session identity for an interactive local TUI.** CLI uses `resolveSessionID()` → `sess_default` by default. Decide whether the TUI should: (a) reuse `DOSSIER_SESSION`/`sess_default` like the CLI (simplest, consistent — recommended), or (b) mint a per-launch session id. Pick (a) for v1 unless you find a concrete reason; record the choice in a one-line ADR (`docs/adr/NNNN-tui-session-id.md`) per HANDOFF's ADR rule. **Resolved:** option (a) — see ADR 0002, now updated by ADR 0003 (the shared resolver adds `CLAUDE_CODE_SESSION_ID` ahead of `DOSSIER_SESSION`).

---

## Catch-up after the MCP session-id fix (2026-06-16, ADR 0003) — OBSOLETE

> **Superseded by [ADR 0004](adr/0004-tui-no-session.md) (2026-06-17).** The TUI no longer
> resolves or carries any session identity and no longer exposes the per-session active
> binding (`Switch`/`Active`). The `a` key, the `★` marker, the `Session:`/`Active:` header
> fields, and the standalone-session footer warning were all removed — so items 1–3 below
> (honest session header, honest active, "don't fix the divergence") are **moot, not
> pending**. The section is kept only for history; do not action it.

> Context: the MCP `dossier_switch`/`dossier_active` gap was fixed by `harness.ResolveSessionID`
> (precedence `explicit → CLAUDE_CODE_SESSION_ID → DOSSIER_SESSION → sess_default`). The TUI
> takes its `sessionID` from `cli.resolveSessionID()`, which now routes through that resolver, so
> the TUI **already** resolves `CLAUDE_CODE_SESSION_ID` automatically — no change is needed for
> *resolution*. What is stale is the TUI's session/active **presentation and honesty**, plus tests.
> These are follow-ups; nothing agent-facing depends on them.

**Dependency rule still binds:** the TUI must NOT import `harness`/`config`. All session resolution
stays in `cli`; pass results into `tui.Run`. Items 1–2 below require `cli` to compute and hand the
TUI a little more than the bare session id.

1. **Honest session-header display.** Today the header always prints `Session: <id>`, which in
   standalone use is the noise string `sess_default`. Have `cli` pass `tui.Run` both the resolved
   session id **and** whether it came from a real harness source (`CLAUDE_CODE_SESSION_ID`/
   `DOSSIER_SESSION`/`--session`) vs the `sess_default` fallback. The TUI then renders the
   `Session:` line only for a real session; for the default bucket, hide it or label it
   `(local default — no active Claude session)`.

2. **Make `active` honest in standalone mode (degrade visibly).** The `a` action and the `★`
   marker still work on `sess_default`, but that binding only affects the local default bucket, not
   any live Claude session — exactly the original "why does this do nothing for me?" confusion. When
   on the default bucket, surface a one-line footer note saying so (consistent with the hard rule
   "degrade visibly"). When a real session id is present, `active` genuinely controls that session —
   no note needed.

3. **Document the intentional MCP-vs-TUI degrade divergence — do not "fix" it.** MCP calls
   `ResolveSessionID(allowDefault=false)` and errors when no session resolves; the CLI/TUI call it
   with `allowDefault=true` and fall back to `sess_default` (an interactive default is acceptable
   and expected for a local tool). This asymmetry is deliberate. A future dev must not make the TUI
   error like MCP. (Recorded here and in ADR 0003.)

4. **Tests.** Add a TUI model test asserting header rendering for the two inputs — a real session id
   (shows `Session: <uuid>`, no footer note) vs the default fallback (session line hidden/labelled,
   footer note shown). Drive it headlessly like the existing TUI tests.

**Out of scope for catch-up** (separate, optional UX ideas, not required by the fix): making `active`
do something *locally* useful (e.g. default the link/merge target to it). Decide separately if ever
desired; not part of getting the TUI "caught up."

## Hard rules to honor (from CLAUDE.md — non-negotiable)

- No logic in the adapter; thin shim over `core.Service`.
- Non-destructive always; no delete path in the TUI (archive only).
- No silent link/merge of ambiguous targets — always ask.
- No silent truncation of Distilled State; surface the over-target warning.
- Degrade visibly — surface every `Result.Warning`.
- Identical behavior to CLI/MCP.

## Tests

- Bubble Tea models are testable headlessly: drive `Update(msg)` with `tea.KeyMsg`/custom msgs and assert resulting model state + that the right `core.Service` method was invoked. Use a fake/stub `core.Service` collaborator (the core tests already use an in-memory `Store` fake — mirror that approach; you can construct a real `*core.Service` over the in-memory fakes from `internal/core`'s test helpers if exported, otherwise add minimal seams).
- At minimum: dashboard renders a list from a stubbed `List` result; selecting a row triggers `Recall`; a status edit triggers `SetStatus`; link with ambiguous candidates never auto-selects.

## Definition of done (CLAUDE.md)

- `go build ./...`, `go vet ./...`, `gofmt -l .` all clean; `go test ./...` passes.
- B3/B9 satisfied: dashboard, status/priority/next-action editing, list+switch, link + merge conflict resolution all reachable in the TUI.
- `ARCHITECTURE.md` kept current (the `internal/tui/` entry already exists; update it if your structure differs; note the launch routing).
- Update `HANDOFF.md`: the TUI was the outstanding piece of Milestone 6 — mark it done when it is, and correct the "project finished" overstatement.
- One focused PR off `main`. PR description includes a conformance table mapping B3/B9 + SPEC §3.1 to the views/tests that satisfy them, and notes anything deferred.

## Workflow

1. Branch off `main` (e.g. `feat/tui`).
2. Build steps 1→5 incrementally; keep it compiling at every step.
3. Manually run `dossier` (bare) against a throwaway store: `DOSSIER_HOME=$(mktemp -d) ./dossier init` then `DOSSIER_HOME=... ./dossier` to launch the TUI. Seed a couple of dossiers via `promote` to have list content.
4. When blocked or when reality contradicts a doc, **stop and flag it** (record capability gaps in `docs/harness-capabilities.md`, new decisions as an ADR). Do not silently work around a settled decision.

**Run through all steps (1→5) to completion in one pass — do NOT stop or wait for confirmation between steps.** Keep going until the full Definition of Done is met (build/vet/gofmt/tests green, ARCHITECTURE.md + HANDOFF.md updated, PR-ready). Only pause for a genuine blocker per step 4 above. Report once, at the end.
