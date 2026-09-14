# Applying the Tally v1.4 Documentation Refresh

This package is an overlay for the repository root.

## Replace / add

Copy:

```text
README.md
AGENTS.md
AGENT.md
agent.md
CLAUDE.md
CONTRIBUTING.md
APPLY.md
docs/
tasks/README.md
```

## What this refresh does

- promotes `Tally_PRD_v1.4_Final.md` to current implementation authority;
- keeps `Tally_Product_Direction.md` explicitly non-binding;
- rewrites current source/implementation status from the actual `tally.zip` baseline;
- marks v1.3 task docs as historical implementation evidence;
- adds the v1.4 Task DAG, L2 architecture evidence and L3 Batch 1 Reference Packs;
- archives the old v1.3 PRD/target under `docs/archive/v1.3/`;
- keeps compatibility pointer paths for old `docs/0x-*.md` references;
- archives repository-root legacy task files under `docs/archive/pre-v1.3/root-tasks/`.

## Legacy root tasks

The original `tasks/T*.md` and related backlog are pre-v1.3 history.

After applying this package, `tasks/README.md` marks them historical. They may be deleted from the root after confirming the archive exists.

## Line endings

The supplied checkout reports many modified paths caused only by CRLF/LF conversion. Consider adding a repository `.gitattributes` in a separate, explicit change after reviewing team platform requirements. This documentation package intentionally does not change Git line-ending policy.
