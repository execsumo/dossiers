# Dossier Operating Instructions

- **Poll Monitors:** Evaluate `(Last polled: date)` in `## Active Monitors`. Fetch updates solely if outdated. Distill findings; update timestamp.
- **Eager Saves:** Execute `dossier_save` immediately upon material decisions or milestones. End-of-session batching: [Rejected].
- **Concurrency:** Inject `base_revision` into `dossier_save`. Mitigates concurrent TUI overwrite conflicts.
- **Artifacts:** Pass raw logs/transcripts as structured artifacts via `dossier_save`. Direct filesystem writes: [Rejected].
- **Handoff:** Commit final state via `dossier_save`. Maintain actionable `## Next Steps`. Use `dossier_update` for isolated metadata mutations.
