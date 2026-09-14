# P5-04 Execution Report — Protected-branch and Git permission policy L0-L3

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** 29/57 (P5-01 + P5-02 + P5-03 landed); working tree shared with parallel lanes P6-01 / P6-05 / P6-06 / P4-04 / P2-02 (their files untouched).

## 1. Seam adjudication — why this is the narrowest valid seam

The task builds on the seams P5-01/P5-02/P5-03 froze: the same package (`internal/gitintegration`), the same validation pipelines, zero new Git authority, no Decision Lock change.

**Ruling: the Git permission layer is one new file in the integration package — a typed ladder (`Level` L0-L3, PRD §18.2), a caller taxonomy (`CallerKind`: Tally domain vs agent runtime), a pure deterministic decision point (`Decider.Evaluate`/`Authorize`), and an unforgeable authorization token (`Authorization`, unexported fields) that the two L3 operations (`EnsureIntegrationBranch`, `Merge`) now require in their request structs. The decision is made in Tally domain/composition and structurally cannot be made (or bypassed) by the runtime.**

### Evidence read (in the mandated order)

1. Task contract `docs/tasks/v1.4.1/tasks/P5/P5-04_protected_branch_and_git_permission_policy_l0_l3.md` including its L3 pack; then `DECISION_LOCKS.md` (DL-03, DL-08/BI-01, DL-13; runtime-boundary guard rows), `PRD_AMENDMENTS.md` (A-04: EngineeringPolicy schema stays narrow), `SOURCE_BASELINE.md` (no Git-permission guidance — semantics defined by this task's contract + PRD §18).
2. PRD `docs/product/Tally_PRD_v1.4.1_Final.md`: §3.6 and §18.2 define the ladder L0 Read / L1 Workspace / L2 Commit / L3 Controlled Integration, with L4 Remote (adapter/experimental) and L5 Destructive (human approval) explicitly outside the enforced scope; §18.1 principle "Agent produces code changes. Tally owns durable repository state."; §23 initial safety boundaries ("disposable workspace; no direct protected push; no owner-worktree mutation").
3. Current source: `internal/gitintegration/service.go` + `branch.go` + `merge.go` (frozen seams: validation pipelines, deterministic `tally/` branch naming, CAS ref discipline, `refs/heads/tally/` audit-namespace refusal as merge target) and their reports; `internal/agentruntime/protocol.go` + `capability.go` (the runtime protocol's entire vocabulary is `Instruction`/claims/capability names — there is **no Git operation vocabulary**; the frozen capability set contains no Git capability, so "adding a tool/capability" cannot express an L3 request at the protocol level); `test/arch/v14_runtime_boundary_test.go` (existing machine guards; `test/**` is owned by P6-05, so the new boundary guard lives inside this package instead); `internal/contract/**` + `internal/engineeringpolicy/**` (the L0-L3 **assurance** vocabulary consumed there is a different axis — consumed by reading, not redefined or conflated).
4. External reference pack: **EXT-GIT-UPDATE-REF / EXT-GIT-WORKTREE** — already materialized in P5-02/P5-03's CAS ref primitives; no new external reference was needed for this task (the permission layer adds no Git plumbing at all).

### Why no narrower seam exists

1. **The decision must be a Tally-domain fact, not an agentruntime feature.** Per the reference pack ("Git permission decisions belong to Tally policy/compiled contract, not agentruntime") the ladder and decision live beside the operations they gate — the integration layer — as pure values and pure functions. No runtime protocol change (read-only), no new package, no parallel authority.
2. **An enum alone is not enforcement; a token is.** A caller-kind flag on a request could be self-asserted by any future code path. The Go-idiomatic structural boundary is an `Authorization` value with unexported fields that only `Decider.Authorize` mints: packages outside `gitintegration` (which includes everything under `runtimes/**` and `internal/agentruntime`) can only produce the zero value, and `requireAuthorization` rejects the zero value (`ErrPermissionRequired`) before any repository state is touched. A misbound token (different target) is `ErrPermissionDenied` — tokens are not reusable across targets.
3. **Protected branches are enforced at the only ref-write entry points.** `Merge` is the single operation that can advance a user-owned ref; it now refuses without a target-bound token, and a protected target can only be authorized with a matching `Approval{TargetRef, Basis, ApprovedAt}` — "policy-approved local integration" is structural, and every allow carries a basis (trust invariant).
4. **L4/L5 are out of scope by type, not by convention.** `Level` has exactly four reserved values; `Level(4)`/`Level(5)`/anything else is unrepresentable vocabulary and `Evaluate` fails safe with `ErrInvalidPermissionRequest`. No remote, force, history-rewrite, or deletion path exists anywhere in the package (inherited from P5-01..03, unchanged).
5. **EngineeringPolicy schema is not touched (DL-08/BI-01).** The protected-branch set is a typed composition input (`NewDecider(protectedRefs...)`), and the approval is a typed fact consumed like `AcceptanceFact` (shape + identity validated, provenance owned by composition/governance). Adding trigger axes or fields to `engineeringpolicy.Policy` YAML/struct would have violated the frozen runtime schema and the AC3 import boundary (stdlib + yaml + risk only). **Schema-affecting decisions: none — this is the explicit adjudication.**
6. **Rejected alternative A — gate L3 through `contract.Compiled`/assurance levels.** The L0-L3 assurance ladder (PRD §13) is a verification-strength axis consumed by `contract`; conflating it with the Git permission axis would redefine frozen gate semantics. The two axes are documented as distinct and no conversion exists.
7. **Rejected alternative B — keep `Merge`/`EnsureIntegrationBranch` signatures unchanged and only add a `Decider` beside them.** That would make the permission layer advisory (callers could skip it). The task AC requires structural impossibility, so the L3 operations themselves refuse unauthorized requests. `Integrate` (L2) is deliberately untouched: the AC boundary is L3, and `Integrate` remains gated by its acceptance-fact pipeline (a runtime cannot produce a valid `AcceptanceFact`).
8. **The runtime-side boundary is machine-pinned where I own tests.** `test/arch/**` is P6-05's; the same guard discipline lives in `internal/gitintegration/permission_test.go::TestRuntimePackagesNeverImportGitIntegration`, which parses all non-test Go files under `internal/agentruntime` and `runtimes/**` and fails if any imports `tally/internal/gitintegration` (and asserts the reverse: the integration layer imports no runtime-side package).

### Resulting shape

- `Level` (L0 Read / L1 Workspace / L2 Commit / L3 Integration) + `IsValid`/`String`; release scope closed at L3 (DL-13).
- `CallerKind` (`CallerTallyDomain` / `CallerAgentRuntime`) + `TallyCaller`/`RuntimeCaller` constructors (descriptor is provenance only, never credentials).
- `Decider`: `NewDecider(protectedRefs []string)` (strict validation: full `refs/heads/<name>`, no option injection/charset violations/duplicates — reusing the shared `validHeadRef` extracted from `validateTargetRef`); `IsProtected` (exact membership, no globs); `Evaluate(caller, request, approval) (Decision, error)` — the deterministic decision projection; `Authorize(...)` — mints the L3 token on allow only.
- Decision matrix (deterministic, table-tested):
  - unknown caller kind / invalid level → `ErrInvalidPermissionRequest` (fail-safe);
  - runtime + L2 → `ErrRuntimeAuthorityExceeded` (DL-03: runtimes do not own Tally Git durability);
  - runtime + L3 → `ErrRuntimeAuthorityExceeded` **always** — no `Approval` can override (the structural AC);
  - Tally domain + L0-L2 → allow (`level_within_caller_ceiling`);
  - Tally domain + L3 unprotected target → allow (`controlled_integration_flow`; the acceptance+CAS gates live at the operation);
  - Tally domain + L3 protected target → allow only with a complete, target-matching `Approval` (`protected_branch_approved:<basis>`), else `ErrProtectedTargetDenied`.
- `Authorization` token (unexported `targetRef`/`basis`; zero value invalid) required by `EnsureIntegrationBranch` (bound to the deterministic `refs/heads/tally/integration/<hex>` ref) and `Merge` (bound to the caller-declared target ref); checked at the operation entry **before any repository interaction**. Authorization ordering in both pipelines: identity + target identified → authority → data validity (acceptance, expected values).
- `refs/heads/tally/**` audit namespace remains unwritable as a merge target (existing `validateTargetRef`, unchanged behavior); as a *declaration* in the protected set it is legal (defense in depth, tested).

## 2. Changed files

- `internal/gitintegration/permission.go` (new): permission sentinels, `Level` L0-L3, `CallerKind`/`Caller`, `Request`, `Approval`, `Decision`, `Authorization`, `Decider` (`NewDecider`/`IsProtected`/`Evaluate`/`Authorize`), `requireAuthorization`.
- `internal/gitintegration/permission_test.go` (new): 9 tests — deterministic permission matrix (25 rows), structural runtime refusal, token lifecycle (mint/zero/misbind), decider construction validation + nil fail-safe, three real-repo negative paths, protected-branch end-to-end, import-scan boundary guard.
- `internal/gitintegration/branch.go`: `IntegrationBranchRequest.Authorization` field; `EnsureIntegrationBranch` derives the deterministic ref immediately after candidate identity and enforces `requireAuthorization` before acceptance/repository work; doc updates only otherwise.
- `internal/gitintegration/merge.go`: `MergeRequest.Authorization` field; `Merge` enforces `requireAuthorization` after target-ref validation and before acceptance/expected-value checks; `validHeadRef` extracted from `validateTargetRef` (shared with the permission layer; merge target keeps the tally/-namespace refusal); step renumbering and doc updates only otherwise.
- `internal/gitintegration/branch_test.go`, `internal/gitintegration/merge_test.go`: existing call sites mint target-bound tokens via the `testAuth` helper; validation tables updated where the new authorization check precedes the asserted sentinel (all asserted sentinels unchanged); three new negative tests added in `permission_test.go` cover the missing/misbound/unauthorized paths.
- `internal/engineeringpolicy/**`, `internal/contract/**`, `internal/agentruntime/**`, `runtimes/**`, `test/**`: **unchanged** (consumed/inspected only; no schema change, no protocol change).

## 3. Semantic changes

Git permission is now a first-class Tally-domain decision at the integration layer: L0-L3 are typed; the agent runtime caller class is structurally capped at L1 (L2 durable writes and L3 integration are refused, approval-proof); the two L3 operations are unreachable without a `Decider.Authorize` token bound to their exact target; protected branches are only integrable through a policy approval carrying a basis; L4/L5 have no representable vocabulary and any out-of-range request fails safe. Acceptance criterion met: an agent runtime cannot request an L3 operation through the runtime protocol — the protocol carries no Git operation vocabulary (existing fact, now machine-pinned), and even a hypothetical runtime-shaped caller holding a `*Service` and a fabricated request is refused by the permission layer with no token to present.

## 4. Tests / commands executed

- `go test ./internal/gitintegration/ -count=1` — **PASS** (41 tests = 13 P5-01 + 7 P5-02 + 12 P5-03 + 9 new)
- `go test ./internal/gitintegration/ -run "Permission|Protected|Authorization|RuntimeCaller|RuntimePackages" -v` — **PASS** (all 8 named new tests individually: TestPermissionMatrix, TestRuntimeCallerNeverObtainsIntegrationAuthority, TestAuthorizationTokenLifecycle, TestMergeRefusesMissingAuthorization, TestMergeRefusesMisboundAuthorization, TestProtectedBranchMergeRequiresApproval, TestEnsureIntegrationBranchRefusesMissingAuthorization, TestRuntimePackagesNeverImportGitIntegration)
- `go test ./internal/gitwriter/... ./internal/changeset/... ./internal/engineeringpolicy/... ./internal/contract/...` — **PASS** (181 tests; neighbors and untouched owned packages, no regressions)
- `go test ./test/arch/...` — **PASS** (100 tests; dependency policy and boundary guards intact)
- `go vet ./internal/gitintegration/` — **PASS**
- `gofmt -l internal/gitintegration internal/engineeringpolicy internal/contract` — clean for all touched files (pre-existing baseline gofmt deltas in untouched `internal/engineeringpolicy/{policy,hash_test,parse_test,policy_test}.go` were observed and deliberately not "cleaned up" — out of scope, not modified by this task)
- `git diff --check -- internal/gitintegration internal/engineeringpolicy internal/contract` — clean (exit 0)
- CR-byte check on both new files — 0 CR (LF per `.gitattributes`)

Negative/failure-path tests added: runtime L3 refusal across protected/unprotected targets with and without a valid approval (asserts `ErrRuntimeAuthorityExceeded`, no token minted, zero token rejected at operation entry); runtime L2 refusal; zero-token `Merge` and `EnsureIntegrationBranch` over real repositories (assert refusal before any repository interaction: refs/branch/HEAD untouched); misbound-token `Merge` (`ErrPermissionDenied`); protected merge without/incomplete/mismatched approval (`ErrProtectedTargetDenied`); invalid decider constructions (7 cases); nil-decider fail-safe; out-of-range levels incl. L4/L5 shapes (`Level(4)`, `Level(5)`, `Level(-1)`, `Level(100)`); invalid L3 targets. The matrix asserts exact decision basis strings and re-evaluates allow rows for determinism.

## 5. Baseline failures observed

None in owned or neighbor packages. Cross-lane noise: none encountered during validation runs (parallel lanes' files were not touched and did not affect these suites). Pre-existing gofmt deltas in untouched `internal/engineeringpolicy` files noted above — not failures, not owned.

## 6. Remaining risks

- Production wiring is a later integration task (per the P5-01/02/03 pattern): the composition root must construct the `Decider` (protected set), call `Authorize` with the correct caller descriptor and approval provenance, and pass the token into `EnsureIntegrationBranch`/`Merge`. The permission layer cannot force composition to *use* protected sets it never declares; default posture with `NewDecider(nil)` equals the P5-03 baseline plus the mandatory token gate.
- `Approval` provenance is validated for shape and target identity only; its authenticity is owned by the composition/governance layer (same discipline as `AcceptanceFact`).
- Same-package code could bypass `requireAuthorization` by calling unexported helpers directly — within one Go package that is trusted domain code by definition; the machine-pinned import boundary keeps runtime-side code out of the package entirely.
- The import-scan guard walks `internal/agentruntime` and `runtimes/**` at test time; a transient parse failure in a file being written concurrently by another lane would surface as a one-off test failure (transient cross-lane noise, retryable).
- Caller descriptors are free text for audit; no format enforcement (deliberate — provenance, not identity).

## 7. Decision Locks touched

None edited, weakened, or routed around. DL-03 respected (runtime ceiling: L2/L3 refused for the runtime caller class; no second Git authority created). DL-08/BI-01 respected (`engineeringpolicy.Policy` YAML/struct untouched — the protected set and approval are typed values in `gitintegration`, documented above as the explicit no-schema-change adjudication). DL-13 respected (level vocabulary closed at L3; L4 Remote / L5 Destructive unrepresentable and unimplemented; no remote/force/history-rewrite path introduced). DL-02 respected (no runtime protocol change; `agentruntime` untouched). Trust invariants respected: unknown callers/levels fail safe; every allow carries a basis; cancellation and git errors are not fabricated into permission outcomes; the owner primary worktree guard is unchanged.

## 8. Commit hash

Not committed — orchestrator owns commits per task protocol.

---

### L3 completion evidence (reference-pack requirement)

- **Exact source/tests read:** task file + L3 pack; `DECISION_LOCKS.md`, `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md`, `docs/product/Tally_PRD_v1.4.1_Final.md` (§3.6, §18, §18.1, §18.2, §23, §34 Phase 6); `internal/gitintegration/{service,branch,merge}.go` and their tests/reports; `internal/agentruntime/{protocol,capability}.go`; `test/arch/v14_runtime_boundary_test.go`; `internal/contract/{contract,compile}.go`, `internal/engineeringpolicy/{policy,match,gate}.go` (consume-only).
- **External references actually used:** none new (EXT-GIT-UPDATE-REF / EXT-GIT-WORKTREE were consumed conceptually as already-materialized CAS primitives from P5-02/P5-03).
- **Why this is the narrowest valid seam:** §1 above — decision machinery + enforcement token in the package that owns the L3 operations; no new package, no protocol change, no policy-schema change, no new dependency.
- **Changed files:** §2 above.
- **Exact test commands/results:** §4 above.
- **Negative tests added:** §4 above.
- **Known uncertainty / version sensitivity:** none new (no new git invocation; `merge-tree` ≥ 2.38 sensitivity inherited unchanged from P5-03).
- **Decision Locks not bypassed:** §7 above.
