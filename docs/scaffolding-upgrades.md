# Scaffolding Upgrades

The `bebo new` generator writes `bebo.manifest.json` into every scaffolded project.
Use this file to track which template version created a project and to guide upgrades.

## Manifest fields
- `schema_version`: manifest schema for tooling.
- `template_version`: generator template version (matches the bebo version used).
- `bebo_version`: framework version used at generation time.
- `kind`: `api`, `web`, or `desktop`.
- `generated_at`: UTC timestamp.
- `module`: module path for the project.

## Upgrade workflow
1. Check the current `bebo.manifest.json` in your project.
2. Review `CHANGELOG.md` and `DEPRECATION.md` for breaking changes.
3. Generate a fresh scaffold with the same flags:
   - `bebo new ./tmp -module <module> -api|-web|-desktop`
4. Diff the new scaffold against your project and apply the deltas.
5. Update `bebo.manifest.json` after merging changes.

## Notes
- Template upgrades are manual by design to keep diffs reviewable.
- If you use config profiles, keep `config/base.json` aligned with new defaults.
