# Tally v1.4 Source Baseline

- Source archive: `tally.zip`
- SHA-256: `93577bdd56db181e8a81bd36c0f49b991f5dea599767000a74a8c0f66ca820ba`
- Git HEAD: `18193d5ff9ff9b6699a4d9b16b6429e2e084094e`
- PRD: `Tally_PRD_v1.4_Final.md`
- PRD SHA-256: `35474f42422222d339214a69c3153547e4a4c8d616f573f1d50af9c371b38907`
- L2 Prompt SHA-256: `8285fd567bc1ad4198fc8b25ab3c7724f866321e82f74adbc6ebe4efbc4c5115`

## A0-01 recheck (2026-09-08)

Re-verified against the live working tree:

- `git rev-parse HEAD` = `18193d5ff9ff9b6699a4d9b16b6429e2e084094e` — matches baseline.
- `docs/product/Tally_PRD_v1.4_Final.md` SHA-256 recomputed = `35474f42422222d339214a69c3153547e4a4c8d616f573f1d50af9c371b38907` — matches recorded PRD hash.

Additional authority-input hashes (first recorded here):

| Input | SHA-256 |
|---|---|
| `docs/product/Tally_Product_Direction.md` | `d38bcd867b7515ea16659c89a02b64e77459b1be00df6397ab2f5fa22b9ccb13` |
| `docs/architecture/V1.4_TARGET.md` | `2bc13fdb2d7b94bca1b94a43236601aa75dadeb057d25bd0a1052b56828cc3a1` |
| `docs/architecture/CURRENT_CODEBASE.md` | `8ae84a94a9f8fae17cf49f7d34c1942faa770e8dcd3b353be0c9b9e785a91c49` |
| `docs/tasks/v1.4/DECISION_LOCKS.md` | `82c08a6ae0040c0cdc3d5f0c57fcb0ee39a22b1cd44483d8950c6dface1a328a` |
| `docs/tasks/v1.4/TASK_MANIFEST.json` | `67e35615590d6c83e8b099f90a5473bed5e422477f96509859cea536e8ccc2a9` |
| `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md` | `a8f03832a3cf80c69ec32af6069e657af56299f451b2a7e2334abb3ab79fd2c0` |

## Working tree observation

The original archive-extraction observation (229 Git-dirty paths, `git diff --ignore-cr-at-eol` clean, line-ending normalization only, `.gitattributes` absent) described the tree **at L2 review time**.

As of the A0-01 recheck (2026-09-08) the working tree is **not** line-ending-only dirty:

- `git diff --ignore-cr-at-eol --quiet` exits `1`: the 28 modified tracked paths contain real content changes from the documentation refresh (e.g. `docs/STATUS.md` rewritten to v1.4 status, `docs/product/Tally_PRD_v1.3_Review_Candidate.md` reduced to an archive pointer, `docs/architecture/CURRENT_CODEBASE.md` refreshed).
- `git diff --check` exits `2` on pre-existing trailing-whitespace lines in `docs/STATUS.md` and `docs/architecture/CURRENT_CODEBASE.md` from that refresh (lines end with two-space markdown line breaks).
- The entire v1.4 task/documentation tree (`docs/tasks/v1.4/**`, `docs/implementation/**`, `docs/product/Tally_PRD_v1.4_Final.md`, etc.) is untracked and not yet committed.
- Git still warns `LF will be replaced by CRLF` on the modified files; `.gitattributes` remains absent. EOL hygiene is owned by task `P0-12`.

## Current validation observation

Fresh Go tests/build could not be completed in the analysis container because required Go modules were not cached and outbound access to `proxy.golang.org` was unavailable. Treat current validation as:

```text
NOT_RUN(dependency fetch/network unavailable)
```

Historical evidence exists in `docs/tasks/v1.3/reports/P1_TRUST_GATE.md`; it must not be relabeled as a fresh v1.4 pass.

## Historical gates vs current validation

| Evidence | Type | Status for v1.4 |
|---|---|---|
| `docs/tasks/v1.3/reports/P0_GATE.md` | historical PASS (v1.3 program) | evidence of record only |
| `docs/tasks/v1.3/reports/P1_TRUST_GATE.md` | historical PASS (v1.3 P1 kernel) | evidence of record only; not proof of real Codex held-out production delivery |
| `docs/tasks/v1.3/reports/P1_METRIC_BASELINE.md` | historical report | baseline input for measurement tasks |
| Fresh `go test` / `make check` | current validation | `NOT_RUN(dependency fetch/network unavailable)` at baseline; must be rerun by implementation tasks where required |

## P0-12 EOL policy record (2026-09-09)

Task `P0-12` added the repository's first `.gitattributes`:

```text
* text=auto eol=lf
*.bat/*.cmd/*.ps1   text eol=crlf
*.png/*.jpg/*.jpeg/*.gif/*.ico/*.zip/*.gz/*.woff/*.woff2   binary
*.min.js            -text
```

Facts at introduction:

- Index EOL distribution: 673 `i/lf`, 1 `i/none` (`web/static/htmx.min.js`); zero tracked blobs stored CRLF, so the policy creates no renormalization diff.
- Working tree at introduction: 454 `w/lf` / 219 `w/crlf`; system git config had `core.autocrlf=true` with no repo-level override.
- No nested `.gitattributes` existed anywhere in the tree.
- No tracked `.bat`/`.cmd`/`.ps1` files existed; the untracked v1.4 doc trees were already LF-only.

Verified effect (same session, before any commit):

- No tracked file became newly dirty from the policy; the modified set remained the 28 doc-refresh paths. (A later 29th modified path, `internal/measurement/report.go`, came from a concurrent task's edit under `internal/**`, not from the policy.)
- The `LF will be replaced by CRLF` warnings git previously emitted for all 28 modified doc files stopped immediately.
- No renormalization (`git add --renormalize`) was performed; committing the policy file and any eventual normalization belongs to the coordinator.

Validation observed at introduction: `git diff --check` exits `2` on pre-existing trailing-whitespace markdown line breaks in `docs/STATUS.md` (lines 3-5) and `docs/architecture/CURRENT_CODEBASE.md` (lines 3-4) — paths outside P0-12 Allowed paths, owned by the documentation refresh. `git diff --ignore-cr-at-eol --check` reports the same five findings; none is CR/LF-related.
