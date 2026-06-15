/*
Package harness implements capability detection and hook integration for agent clients.

It provides implementations of core.Harness and core.HarnessRegistry for:
- Claude Code (Tier 1 capability target)
- Codex (Tier 2 capability target)
- Antigravity (Tier 3 capability target)

It validates integration options and config files (like ~/.claude.json or ~/.codex/config.toml)
to classify harness support into tiers and manage hook hooks installations.
*/
package harness
