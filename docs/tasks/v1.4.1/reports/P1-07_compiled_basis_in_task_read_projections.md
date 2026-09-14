# P1-07 Execution Report — Compiled basis in task/read projections

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** A0+P0 complete, P1-01…P1-06 DONE (progress 20/57 at start)
- **Defers-cleared:** the P1-04 risk entry "P1-07 落读模型 compiled:<hash> basis" is closed by this task

## 1. Seam choice (narrowest valid seam)

**The VerifiedBasis projection and the `compiled:<hash>` basis reference already exist in `internal/taskstate` (audited v1.4 baseline, commit `22d9996` — old v1.4 numbering "P1-08", PATH_OWNERSHIP gives P1-06/P1-07 joint taskstate ownership). What was missing was the production read path: nothing fed a compiled authority into `taskstate.Snapshot`, so the read model could never produce a verdict, level facts, or the compiled basis reference.** The taskstate layer is therefore zero-changed by this task; the landed change is entirely in the owned read surface (`v14_change.go` + template).

### Exact source/tests read (reference order per the L3 pack)

1. **Tests:** `internal/taskstate/basis_test.go` (asserts `VerdictVerified`/`VerdictVerifiedLimited` distinction, `CompiledHash`, `hasRef(res.Basis, "compiled:"+c.Hash())`, `ErrCompiledMismatch` cross-revision refusal); `internal/web/handlers/v14_task_test.go::TestV14TaskDetailShowsRepairChainBasisAndUncertainty`; `internal/web/handlers/v14_change_test.go`.
2. **Interface/types:** `taskstate.VerifiedBasis` / `taskstate.Verdict` / `EvidenceCoverage` / `PolicyCoverage`; `contract.Compiled.Hash()/Coverage()/Clauses()`; `artifacts.LoadBundle → (*ExecutionBundle).CompileAuthority() → CompiledAuthority{Compiled *contract.Compiled}` (P1-03).
3. **Core implementation:** `internal/web/handlers/v14_change.go` (read path under task), `internal/web/handlers/v13_changes.go` (`deriveV13Coverage` — the existing P1-19/P1-20 functional+independence coverage seams, `v13FunctionalRecords` evidence-row projection), `internal/taskstate/state.go` (Derive's basis wiring: `basis = append(basis, "compiled:"+s.Compiled.Hash())` when `Snapshot.Compiled != nil`), `web/templates/v14_change.tmpl`.
4. **Failure handling:** absent Compiled must render unknown/not available — the pre-existing not-wired dimension notes are preserved verbatim on the Compiled-absent path.

### Why this is the narrowest seam

- The projection is pure and already complete in `taskstate`; duplicating or extending it would widen surface without need (DL-18: extend-not-duplicate; the projection has no setter and Derive stays the only production path).
- The compiled authority must come from the same production seam as the worker (P1-03 `CompileAuthority`, PRD §16 "no bypass of the compiler"): the read path reuses `artifacts.LoadBundle(repo, <pinned HEAD hash>, taskID) → CompileAuthority()` at the exact same commit hash the page already loaded its artifact snapshot from (DL-04: one source, no re-resolution of a movable ref).
- Achieved assurance needs observed per-clause facts: `deriveV13Coverage` already derives a revision-bound (`§8.8.2`) `FunctionalCoverageReport` + `IndependenceCoverageReport` from the candidate's evidence rows via `verification.DeriveFunctionalCoverage` / `DeriveIndependenceCoverage`. Feeding those existing reports into the Snapshot (only when the compiled authority is present) makes achieved levels factual instead of default-L0.

### Read-path fact discovered and fixed (inside the owned file)

`v13env()` deliberately returns `project` zero ("由调用方在装载制品后填入"); `v14TaskPage` fills it (`env.project = core.ProjectID(snap.ProjectID)`), but the Change Detail path never did — so the task-scoped fact queries (`CodegenRunsForTask` / `ChangeSetsForTask` / `VerificationRunsForTask`) always returned empty rows, the derive never saw run/candidate/verification facts, state was pinned at READY ("门槛缺口 candidate: 没有当前候选 ChangeSet 事实"), and no VERIFIED-like verdict was reachable at all. Fixed with the same single-line pattern as the task page, right after the page's own snapshot load.

## 2. Changed files

- `internal/web/handlers/v14_change.go`:
  - `V14ChangePage` gains `HasVerdict`, `Verdict`, `VerdictLimited`, `VerdictOK`, `CompiledHash`, `CompiledNote` (projection-only fields; no state).
  - `v14ChangeBasis` now: fills `env.project` from the Git registry snapshot; loads the compiled authority from the same pinned HEAD commit via `artifacts.LoadBundle → CompileAuthority()` (worker's P1-03 seam); on success sets `Snapshot.Compiled` and — only when the existing revision-bound coverage derivation (`cov *v13Coverage`) is available — `Snapshot.Functional` / `Snapshot.Independence`; projects `res.VerifiedBasis` into the page; on typed failure sets an explicit `CompiledNote` reason and keeps every pre-existing not-wired display.
  - `v14BasisDims` gains a `*taskstate.VerifiedBasis` parameter: assurance dim renders `必需 <L?> · 达成 <L?>` facts (compiled-anchored note) when wired, keeps the exact v1.4 "未接入,本页不冒充档位" note otherwise; unmatched dim renders matched-policy count, unmatched risk signals verbatim (fail-safe L3 note) and unknown signals preserved (INV-V14-09) when wired, keeps the v1.4 note otherwise.
  - Header data-discipline comment updated to the new reality.
- `web/templates/v14_change.tmpl`: verdict line inside the basis section — `VERIFIED` (ok styling, "验证门全部满足且无未消化缺口"), `VERIFIED_LIMITED` (warn styling, "limited 结论 …… 不得显示成普通 VERIFIED"), `NOT_VERIFIED` (off styling) — visually distinct plus `结论依据已编译合同 compiled:<hash>`; new `CompiledNote` line ("展示缺席不放宽任何验证门;本页不冒充档位").
- `internal/web/handlers/v14_change_test.go`: intent-bearing repo fixture (`v14cIntentRepo` — v13Repo layout plus `.tally/intents/TASK-0001.yaml`, optional extra files), `v14cPassCandidate` seed helper, and four tests (below).

`internal/taskstate/**`: zero-changed (projection was already complete at the baseline; confirmed by its untouched test suite).

## 3. Semantic changes

- With the compiled authority available at the pinned HEAD commit, the Change Detail read model now shows the actual PRD §14 verification basis: the verdict distinguishes `VERIFIED` vs `VERIFIED_LIMITED` vs `NOT_VERIFIED` (VERIFIED_LIMITED names its gap classes and is never rendered as plain VERIFIED), the basis list carries `compiled:<hash>` (INV-STATE-02: same hash ⇒ same required/coverage view, verdict independently recomputable), assurance shows required/achieved (achieved honestly reported at the weakest clause), unmatched shows matched-policy count plus unmatched/unknown signals verbatim.
- `VERIFIED_LIMITED` arises exactly from the projection's trigger set (unmatched risk signals, waived requirements, unmet clauses) — with the fail-safe consequence visible: an unmatched signal shows "必需 L3" while evidence supports only L1.
- When the compiled authority is absent — repo without a FrozenIntent artifact, typed `LoadBundle`/`CompileAuthority` failure (e.g. `ErrPolicyRejections` on an invalid policy artifact), or a candidate that cannot be attributed to a task — the page keeps the exact v1.4 not-wired dimension notes, adds the typed failure reason, and never renders a verdict, level facts, or a compiled reference. Assurance is never claimed without Compiled (task requirement 2; basis entries stay real — nothing fabricated).
- Projection only: no new mutable state, no persistence, no setter; derive remains the sole production source of `VerifiedBasis`.
- The `env.project` fix makes the change page's derive actually consume run/candidate/verification facts (state can leave READY; "本候选即当前候选" can become true). Derive semantics themselves are untouched.

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `gofmt -l` on touched files (`v14_change.go`, `v14_change_test.go`) | clean (pre-existing unformatted neighbors untouched — baseline condition, e.g. `v13_changes.go`, `server.go`) |
| `go test ./internal/taskstate/... -count=1` | PASS (ok 0.695s) |
| `go test ./test/arch/... -count=1` | PASS (ok 1.115s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./internal/web/handlers/ -run TestV14Change -count=1` | PASS (4 new + 2 pre-existing v14 change tests) |
| `DATABASE_URL=… go test ./internal/web/handlers/ -count=1` (full package, DB-gated included) | PASS (ok 114.6s) |
| `DATABASE_URL=… go test ./internal/... -count=1` (full internal suite) | PASS (all packages ok; only `[no test files]` notices) |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/... -count=1` | PASS (ok 338.0s; v14_codex_trust ok) |
| `git diff --check` | clean |
| `make check` | NOT_RUN (no make binary in Git Bash; constituent families run individually as above) |

### New tests (all via real temp Git repo + real Postgres schema, SKIP-safe on DB absence)

- `TestV14ChangeCompiledBasisPlainVerified` — zero-policy, zero-signal intent: verdict renders plain `VERIFIED` (and never `VERIFIED_LIMITED`), basis list contains `compiled:sha256:…`, assurance dim shows `必需 L1 · 达成 L1`, unmatched dim shows "无未匹配风险信号", fact references still listed (INV-11).
- `TestV14ChangeCompiledBasisVerifiedLimitedOnUnmatchedRisk` — owner-confirmed `auth_symbol_change` risk hint with no matching policy: verdict is `VERIFIED_LIMITED` (warn, "不得显示成普通 VERIFIED"), signal shown verbatim, `必需 L3 · 达成 L1` (fail-safe uplift visible), `compiled:sha256:…` present.
- `TestV14ChangeCompiledAbsentDoesNotClaimAssurance` (negative) — v13Repo without any intent artifact: page shows `已编译合同未接入 <typed reason>` and the not-wired dims; banned strings asserted absent: verdict line, `必需 L`, `达成 L`, `compiled:sha256:` — no assurance claimed, no basis fabricated.
- `TestV14ChangeInvalidPolicyKeepsCompiledAbsent` (negative) — invalid `.tally/policies/policy-bad.yaml`: `CompileAuthority` fails typed (`ErrPolicyRejections` family); page stays not-wired with the reason; no verdict/level/compiled fabrication (P1-03 fail-safe continued into the read path).

## 5. Baseline failures observed

None. All pre-existing handler/taskstate/arch tests pass unchanged (the two pre-existing `TestV14Change*` full-chain tests keep their exact expectations because their fixture repo has no intent artifact, so the Compiled-absent path is byte-identical to v1.4 behavior).

## 6. Remaining risks

- Candidates whose verification runs bind an older frozen-contract revision (`c.matched=false`, §8.8.2) keep the coverage reports out of the derive and render `NOT_VERIFIED` with the compiled basis still listed — honest, but reviewers should expect NOT_VERIFIED on stale-revision candidates rather than a "legacy VERIFIED".
- Independence achieved-facts remain conservative in this read path (`v13IndependenceInput` zero implementation provenance, pre-existing): acceptances requiring source independence will surface as unmet (→ VERIFIED_LIMITED) even with pass evidence. This is fail-safe direction, now visible through the verdict instead of hidden.
- `v14_task.go` (closed-loop task detail page) intentionally untouched — outside this task's Allowed Paths — so the task page's basis line does not yet carry `compiled:<hash>`; extending it belongs to that page's owning lane.
- The compiled basis on this page always reflects HEAD artifacts (`c.git.headHash`), same as every other derive input on the page; a page request racing a freeze commit re-derives from the new HEAD deterministically.

## 7. Decision Locks

Not touched, not bypassed. Relevant readings honored: DL-04 (one pinned commit for all read inputs — the page's snapshot and the compiled authority come from the identical observed HEAD hash), DL-05 (risk stays derived; projection-only), DL-18 (existing domains extended in place; taskstate zero-changed), DL-19 (read/projection concern; no state-machine change), §3 trust invariants (every VERIFIED-like rendering carries a real basis incl. `compiled:<hash>`; nothing fabricated without Compiled). `hidden/` unread (DL-17).

## 8. Commit

- recorded in git log (`P1-07: …`)
