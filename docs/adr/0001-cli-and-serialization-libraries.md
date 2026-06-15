# ADR 0001: Choice of CLI and YAML Serialization Libraries

## Status
Accepted

## Context
For Dossier (codename *chainlink*), we need to implement a robust CLI command hierarchy (supporting subcommands, overridden flags, and global flags) and serialize/deserialize frontmatter metadata to local YAML-formatted text files. Because the domain core (`internal/core`) must remain completely pure of third-party dependencies, we must restrict these library imports strictly to driving and driven adapters (`internal/cli` and `internal/config`/`internal/store`).

## Decision
1. **CLI Framework:** We chose `github.com/spf13/cobra` as the CLI framework. Cobra is the industry standard for Go CLI applications, supporting robust subcommand hierarchies, automatic help generation, and POSIX-compliant flags.
2. **YAML Serializer:** We chose `gopkg.in/yaml.v3` for parsing and rendering YAML frontmatter. This library supports parsing YAML into structs, preserves comments, and provides flexible nodes APIs for more complex manual manipulation if needed in subsequent milestones.

## Consequences
- CLI flag definition, validation, and subcommand routing live exclusively inside `internal/cli`.
- YAML encoding and decoding is restricted to `internal/config` (for workspace configurations) and `internal/store` (for frontmatter persistence).
- The domain core (`internal/core`) remains completely free of Cobra and yaml.v3 library dependencies, importing only standard library packages and its own local subpackages.
