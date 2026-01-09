# Versioning Policy

bebo follows Semantic Versioning (SemVer): MAJOR.MINOR.PATCH.

## Pre-1.0 (current)
- MINOR releases may include breaking changes.
- PATCH releases include bug fixes and documentation updates only.

## 1.0 and later
- MAJOR releases may include breaking changes.
- MINOR releases add backwards-compatible functionality.
- PATCH releases include backwards-compatible bug fixes.

## Compatibility Guarantees
- Public APIs are documented in Go docs and README examples.
- The compatibility test suite (`compat/`) exercises common public APIs.
- Breaking changes are documented in `CHANGELOG.md`.
