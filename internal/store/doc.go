/*
Package store implements the core.Store port using a local filesystem layout.

It is responsible for:
- Initializing the ~/.dossier storage directories.
- Storing and reading Dossier frontmatter YAML and distilled Markdown.
- Writing captured source artifacts atomically.
- Managing append-only JSON Lines audit logging.
- Persisting active session bindings and conflict resolution artifacts.

This package depends on the core package and uses third-party serialization
libraries (like gopkg.in/yaml.v3) to handle YAML serialization.
*/
package store
