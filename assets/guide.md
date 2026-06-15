# Dossier Distillation Guide

This guide describes how to maintain the Distilled State of a Dossier.
Follow these rules strictly to keep topic contexts accurate, compact, and highly useful.

## 1. Core Principles

- **Keep context current:** Maintain the session's active Dossier using a best-effort approach each turn.
- **Save on lifecycle events:** Save context at session end, on `/clear`, on `/exit`, and before compaction where possible.
- **No small talk or conversational noise:** Trim greetings, pleasantries, dead-end paths, tool-call mechanics, and verbose restatements.
- **Durable State only:** The Distilled State must represent the current, clean, consolidated truth of the topic.
- **Provenance is required:** Every material claim and decision MUST refer to the source artifact or line range. Example: `[src:art_01jz8salesfeedback#L42-L68]` or `[src:art_01jz8launchthread]`.
- **Archive first, distill second:** Save transcripts, code snapshots, and thread contents as source artifacts in the Archive, then reference them in the Distilled State.
- **Degrade visibly:** If a harness cannot capture transcripts or hooks, warn the user. Never silently ignore missing integrations.
- **Never silently truncate:** Never silently truncate the Distilled State to satisfy a token target. If approaching the target, warn the user.
- **Optimistic Concurrency & Disambiguation:** Concurrent edits will create conflict files. Ask the user for ambiguous link targets and merge conflict resolution. Never last-write-wins.

## 2. Structure of the Distilled State

Every `dossier.md` body must follow this exact section structure:

```markdown
# <Dossier Name>

## Situation
What is the core problem, goal, or topic? Briefly outline the initial state and context.

## Decisions
What has been decided? List who decided, the rationale, date, and provenance references.
- [Decided on YYYY-MM-DD] <Decision description> by <Attribution>. Rationale: <Why>. [src:art_<id>]

## Findings
What has been learned or validated? Key insights, experiment results, or data.
- <Finding description> [src:art_<id>#L<a>-L<b>]

## Current State
Where does the work stand right now? What are the active files, blockers, or configurations?

## Next Steps
What must be done next? Align this section with the `next_action` and `open_questions` in the metadata.
```

## 3. Distillation Examples

### BAD DISTILLATION (Verbose, no provenance, conversational)
> Hey there! So I started looking into the pricing bug. I ran the test script `go test ./...` and it failed on line 12. Then I talked to Herwin and he said we should use usage-tier instead. I tried fixing it by changing the condition and it passed. Next step is to clean up.

### GOOD DISTILLATION (Concise, structured, carries provenance)
> - **Situation:** Enforcing usage-tier billing calculation under high concurrency. [src:art_01jz8initial_bug]
> - **Decisions:**
>   - [Decided on 2026-06-14] Switched billing model from flat-tier to usage-tier. Approved by Herwin. Rationale: reduces billing leakage on concurrent actions. [src:art_01jz8pm_alignment#L12-L28]
> - **Findings:**
>   - Concurrency tests fail if lock timeout is set below 200ms. [src:art_01jz8test_results#L102]
> - **Current State:** Lock timeout increased to 500ms; tests passing locally.
> - **Next Steps:** Merge pricing change and monitor production logs.
