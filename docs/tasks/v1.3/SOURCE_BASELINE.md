# Tally v1.3 Source Baseline

This task pack is prepared against:

- `tally.zip` SHA-256: `6581c27893a9a7ff800bb81da3de10cf98abe6c4880f16e5d8ca101a9306f75f`
- `docs/product/Tally_PRD_v1.3_Review_Candidate.md` SHA-256: `bed4d8f69b387d1e5ab3dc8878498e94679fd308d314ff23cd601311265952b0`
- Go module: `tally`, Go `1.23`
- Source size at audit: 343 files, 216 Go files, 86 `_test.go` files, about 29.9k Go LOC.

## Authority

The copied v1.3 PRD is the product baseline for this pack. Existing `docs/00-PRD.md`, old task files and old `CLAUDE.md` describe the legacy product and MUST NOT override v1.3 semantics.

## Brownfield strategy

**Keep proven infrastructure; repair P0; build the v1.3 core as new bounded modules; replace legacy paths only after the new vertical slice is proven.** Do not rewrite the repository from scratch and do not preserve legacy abstractions merely for compatibility.

## Audited source facts that control ordering

1. `internal/web/handlers/tasks.go` / `taskstatus.go` treat one project as one “task” and derive columns from build/deploy/clause coverage. This is legacy execution summary, not the v1.3 TaskContract domain.
2. `migrations/0010_jobs.sql` and `internal/jobs/jobs.go` have no lease owner/heartbeat/stale-running recovery.
3. `internal/store/queries/trace.sql::DeleteStaleTraceEdges` compares columns independently instead of complete tuples; trace identity is not consistently project-namespaced.
4. `internal/llm/llm.go` hardcodes a requirement-interview system prompt used by requirement/failure/harvest callers.
5. `cmd/worker/main.go::failureRunner` writes candidate files directly into the supplied workdir and executes `strings.Fields(command)`.
6. `runner.Command` is already structured (`Argv`, `Dir`, `Env`) and should be reused rather than replaced.
7. `failure_event` currently stores `workdir`, `repro TEXT`, `other_targets`; structured command persistence is required.
8. Git writes bypass serialization in `internal/web/handlers/content.go`, `internal/requirement/sign.go`, `internal/harvest/catalog.go`, `internal/knowledge/store.go`.
9. `failure.Handle` supports `SpecDefectSink`, but production worker wiring leaves it nil.
10. `jobs.KindDeploy` exists and operations docs describe deploy, but `exec.Handlers` does not register a deploy handler; UI exposes rollback only.
11. Existing failure gates are valuable deterministic assets: protected test/source scope, repeated repro, repeated green and other-target checks. Preserve them while migrating command/workspace semantics.
12. Existing append-only fact triggers and architecture tests are valuable and must extend to new P1 fact tables.
13. The repository already has rich unit/integration/architecture tests. Avoid a directory-only “monorepo reorganization” before functionality.
14. sqlc-generated files (`internal/store/*.sql.go`, `models.go`, `querier.go`) are shared conflict hotspots. Leaf SQL tasks must not edit them; dedicated integration tasks run `sqlc generate`.
15. `cmd/worker/main.go` and `internal/web/handlers/server.go` are serialized integration hotspots.

## Environment note from pack preparation

The preparation container had no internet and no cached Go module downloads, so a full `go test ./...` could not be executed there. This is not recorded as a source test failure. Codex tasks must run the requested validations in their own environment and report exact blockers.
