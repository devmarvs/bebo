# Deprecation Policy

## Marking Deprecations
- Use the standard Go doc format: `// Deprecated: ...`.
- Explain the replacement and expected removal version.

## Support Window
- Deprecated APIs remain for at least two MINOR releases in v0.x.
- After 1.0, deprecated APIs remain for at least one MINOR release.

## Removal
- Removals happen only in a breaking release.
- `CHANGELOG.md` must document the removal.
