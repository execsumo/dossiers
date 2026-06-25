---
name: Dossier Operations
description: Mandatory operating instructions for agents working within a Dossier context.
---

# Dossier Operations

You are operating within a Dossier. You MUST follow these rules to maintain state and context integrity.

## 1. Resume (Start of Session)
- **Load Context:** Read `dossier.md` to establish baseline state.
- **Poll Monitors:** Check `## Active Monitors`. Fetch updates for any listed external streams (Slack, Jira) using available tools.
- **Sync:** Distill monitor findings into `## Findings` or `## Decisions`. Update `(Last polled: date)`. Remove resolved monitors.

## 2. Execute & Distill (During Work)
- **Archive First:** Do not summarize long transcripts or logs from memory. Save raw text to `artifacts/` first.
- **Cite Sources:** Append `[src:art_<id>]` to every material claim, decision, or metric.
- **Telegraphic Phrasing:** Write punchy, noun-heavy declarations. Strip conversational fluff (e.g., use "Test failed" instead of "I ran the test and it failed").
- **Record Negative Space:** Log abandoned paths and failed experiments as `[Rejected]` findings to prevent repeated mistakes.

## 3. Handoff (End of Work)
- Keep `## Next Steps` immediately actionable.
- Update `dossier.md` incrementally as tasks complete so the next agent can resume instantly if interrupted.
