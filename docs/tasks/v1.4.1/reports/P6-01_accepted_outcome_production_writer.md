# P6-01 Execution Report — Accepted Outcome production writer

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** HEAD 33197fe on master (audited baseline 979a55a); progress 28/57 at start
- **Execution model:** HIGH

## 1. Seam adjudication (chosen before code, per task contract)

**Ruling: the Outcome writer attaches inside the Closed Loop Engine's accept transition, after the P1-06 AcceptGuard pass and *before* `step(PhaseVerifying, EventVerified)`; when the sink is bound, `PhaseAccepted` exists only after the `kind=accepted` outcome event is durably appended.**

### Source facts the ruling is based on

1. **The Engine's accept transition is the only acceptance authority (DL-01; P1-06 report §1).** The only place a verified candidate becomes Accepted is the `VerdictVerified` branch of `internal/closedloop/engine.go`. The P1-06 guard and the P2-06 exposure guard already live at exactly this spot — an Outcome writer placed anywhere else would make the durable acceptance seam ambiguous.
2. **There is no other durable acceptance record to attach to.** Post-acceptance, `v14Handler` only logs; the verification_run facts are verification facts, not acceptance. The accepted `outcome_event` row (migration 0023, PRD §23, DL-12/DL-14) is therefore the *first* durable record of acceptance — which is precisely why it must be written before the Accepted terminal state is returned, not after (fire-and-forget after return would allow a job crash between accept and append to silently lose an uncollectable-after-the-fact §23 fact).
3. **Rejected alternative "write in `v14Handler` after `res.Phase == Accepted`":** attaches after the acceptance authority has already spoken and returned; a crash or append failure there leaves an Accepted result with no accepted outcome fact, and the handler has no typed fail-safe vocabulary to convert it into (only job-error retry, which re-runs the entire loop and mints a *new* candidate/changeset — two acceptances in the event stream for one logical acceptance).
4. **Rejected alternative "second authority inside the verifier/adapter":** would violate "no two acceptance authorities". The chosen seam keeps the writer an *observer*: verdict + AcceptGuard still solely authorize; the sink only records the completed ruling.

### Transaction / order semantics (task requirement 1)

Order inside the accept branch: `VerdictVerified` (basis present) → exposure guard → **P1-06 AcceptGuard** → **AcceptSink.RecordAccepted** → `EventVerified` → `PhaseAccepted` returned. The sink is invoked with `context.WithoutCancel(ctx)` (same fact-persistence discipline as `v14CodegenRunner`), so queue cancellation cannot fabricate an append failure; cancellation *before* the sink call is checked first and remains a first-class Cancelled terminal (acceptance never finalized, no outcome, no verdict credited). An append that succeeds is an acceptance that is durably complete — cancellation arriving after the write no longer rewrites the terminal state. `AcceptFact.RecordedAt` comes from the Engine clock (`l.e.now()`), never sink-side "when we got around to it".

### Retry / idempotency (task requirement 2)

`closedloop.StoreAcceptSink` is idempotent: before appending it reads the task's existing outcome stream and returns success without writing if an `accepted` event already exists for the same `(task_id, changeset_id)`; a different changeset for the same task is a genuinely new acceptance fact and appends. Migration 0023 has no unique constraint and migrations are not in this task's paths (P3-02 reserves numbering) — check-then-append atomicity rests on the closed loop's single-writer discipline (one job lease → one one-shot Engine → at most one sink call per acceptance), documented in `outcome.go`.

### Failure policy (task requirement 3, explicit fail-safe)

Sink failure (ctx alive — cancel cannot cause one, see above) → typed escalation `EscalationOutcomeUnavailable = "outcome_unavailable"`: no Basis, no Accepted terminal, no Router consultation, no repair budget consumed (the acceptance ruling passed; this is not a candidate defect). Read failure during the idempotency check is treated identically to write failure — "cannot read" is never folded into "nothing there". Nothing is ever silently dropped.

## 2. Changed files

- `internal/closedloop/outcome.go` (new): `AcceptFact` (+`Validate`), `AcceptSink` port (append-only, idempotent contract), `StoreAcceptSink` idempotent persistent adapter over `internal/outcome` (`outcomeEvents` narrow face), `ErrInvalidAcceptSink`.
- `internal/closedloop/engine.go`: `Config.AcceptSink` field + Engine wiring; sink invocation at the accept seam (order semantics above).
- `internal/closedloop/result.go`: `EscalationOutcomeUnavailable = "outcome_unavailable"` (new typed reason; Outcome *event* kind set untouched per DL-12/BI-02).
- `internal/closedloop/wiring.go`: `LoopConfig.AcceptSink` passthrough into `Engine.Config`.
- `cmd/worker/v14.go`: production composition — `v14Deps.accept` injection seam; default binds `closedloop.NewStoreAcceptSink(outcome.New(deps.queries))` whenever the fact store is present (always in production); `LoopConfig.AcceptSink` set.
- `internal/closedloop/outcome_test.go` (new): engine seam tests (fact coordinates = Basis, repair-final-candidate binding, failure escalation, cancel-after-write semantics) + StoreAcceptSink idempotency/failure unit tests (in-memory events face).
- `cmd/worker/v14_outcome_test.go` (new): DB-backed production tests (exactly-one accepted event bound to task/changeset; real-DB replay idempotency; injected sink failure → `outcome_unavailable` escalation with zero outcome rows).

## 3. Semantic changes

- With the sink bound (production worker since it always has a fact store), a durable acceptance now always yields exactly one `accepted` outcome event carrying the task and accepted-changeset coordinates, with `RecordedAt` = the acceptance moment.
- An acceptance whose outcome fact cannot be recorded escalates `outcome_unavailable` instead of being silently accepted.
- Sink unbound (nil) keeps exact v1.4 behavior for test compositions — no new mandatory dependency inside `Engine`/`NewLoop`.
- P1-06 guard and verdict semantics untouched (writer observes, never authorizes). Outcome event kinds unchanged (five-kind closed enum).

## 4. Tests / commands executed (exact)

| Command | Result |
|---|---|
| `go build ./...` | PASS |
| `go vet ./internal/closedloop/... ./internal/outcome/... ./cmd/worker/...` | PASS |
| `go test ./internal/closedloop/... ./internal/outcome/...` | PASS (ok 10.4s / ok 0.2s) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./cmd/worker/...` | PASS (ok 68.8s; DB-backed tests ran, not skipped) |
| `go test ./test/arch/...` | PASS (ok 1.4s — P1-08 no-bypass guard and DL-12 Outcome kind-set guard stay green) |
| `gofmt -l` on all touched files | clean (`exposure_test.go`, `main.go`, `v13.go` etc. are pre-existing unformatted neighbors, untouched) |
| `git diff --check` | clean for my paths (CRLF warnings observed only on P6-05's `internal/verification/*` files) |

## 5. Baseline failures observed

None in the focused suites; the only working-tree noise originates in other lanes' paths (`internal/gitintegration/*`, `internal/gitwriter/*`, `internal/verification/*` — P5-03/P6-05).

## 6. Negative / failure-path tests added

- `TestEngineAcceptSinkFailureEscalates` (engine: sink error → `outcome_unavailable`, no Basis, no Router/budget consumption).
- `TestStoreAcceptSinkFailuresSurface` (read failure, write failure, invalid fact, nil-store constructor).
- `TestV14AcceptSinkFailureEscalates` (production wiring with injected failing sink → escalation, zero outcome rows).
- `TestEngineCancelAfterAcceptWriteKeepsAccepted` (cancel-after-write ordering pin).

## 7. Decision Locks

- No Decision Lock touched or bypassed. DL-12 (five Outcome kinds) preserved — no new `outcome.Kind`; the only vocabulary addition is a closedloop-side `EscalationReason`. No migration needed (0023 surface sufficient; no new DDL required).

## 8. Remaining risks / notes

- Check-then-append idempotency is not database-enforced (0023 has no unique index; migrations outside this task's paths). If a future task wants a DB-level guarantee, a partial unique index `ON outcome_event (task_id, changeset_id) WHERE event_kind='accepted'` is the exact DDL to adjudicate — not required for correctness under the single-writer job-lease discipline.
- `docs/architecture/CURRENT_CODEBASE.md` / `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md` updates are owned by P6-06 (docs lane); this report is the implementation fact record.
- Not committed (orchestrator owns commits); no commit hash.
