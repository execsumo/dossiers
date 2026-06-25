---
name: Dossier Operations
description: Mandatory operating instructions for agents working within a Dossier context.
---

# Dossier Operations

You are operating within a Dossier context. You are equipped with a Dossier MCP server. **You MUST use the MCP tools for all Dossier operations.** Do NOT read or edit `dossier.md` or `artifacts/` directly via the file system.

## 1. Resume (Start of Session)
- **Load Context:** ALWAYS use the `dossier_session` tool to identify your current active dossier. Then, use `dossier_recall` to load the distilled state, metadata, and the `base_revision`.
- **Poll Monitors:** Check `## Active Monitors` in the recalled state. **CRITICAL: Evaluate the `(Last polled: date)` timestamp. If you have already polled it recently in this session or if the timestamp is very recent, DO NOT poll it again.** Fetch updates only if necessary.
- **Sync:** Distill monitor findings into `## Findings` or `## Decisions`. Update `(Last polled: date)`. Remove resolved monitors.

## 2. Execute & Distill (During Work)
- **Eager Saves:** Call `dossier_save` immediately after you reach a material decision, discovery, or milestone. DO NOT wait until the end of the session, to prevent data loss if the session is abruptly killed via Ctrl+C.
- **Use the MCP:** When saving updates, you MUST use `dossier_save`. 
- **Concurrency:** You must pass the `base_revision` you received from `dossier_recall` into `dossier_save` to prevent clobbering concurrent edits from the user's TUI.
- **Artifacts:** Do not summarize long transcripts or logs from memory. Pass raw text as structured artifacts in your `dossier_save` payload rather than writing to the file system directly.
- **Cite Sources:** Append `[src:art_<id>]` to every material claim, decision, or metric.
- **Telegraphic Phrasing:** Write punchy, noun-heavy declarations. Strip conversational fluff (e.g., use "Test failed" instead of "I ran the test and it failed").
- **Record Negative Space:** Log abandoned paths and failed experiments as `[Rejected]` findings to prevent repeated mistakes.

## 3. Handoff (End of Work)
- Keep `## Next Steps` immediately actionable.
- Use `dossier_save` to commit your final distilled state and use `dossier_update` if you need to update isolated metadata fields (like status or lead) before finishing.
