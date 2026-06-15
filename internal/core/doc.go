/*
Package core implements the pure Hexagonal Architecture domain logic for Dossier.

This package contains all the business entities (Dossiers, Artifacts, Sessions,
Conflicts), the domain errors, the unified result envelope, and the swappable
Port interfaces (Store, Searcher, Tokenizer, HarnessRegistry, Harness, Clock).

It maintains a strict dependency rule: it imports only standard library packages.
It does not import any sibling packages or third-party libraries.
*/
package core
