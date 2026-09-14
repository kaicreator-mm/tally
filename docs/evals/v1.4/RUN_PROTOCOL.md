# v1.4 Product Value Experiment — Run Protocol (P4-02)

**Status:** Procedure for executing the two arms on the FROZEN task set.
**Authority:** `docs/evals/v1.4/PRE_REGISTRATION.md` (A0-05), `docs/evals/v1.4/TASK_SET.md` (P4-01 freeze), PRD §30–33. This document is procedure only; it does not amend metrics, thresholds, tasks, tiers, or exclusions. Task owner: `docs/tasks/v1.4/P4/P4-02.md`; arm B execution owns `docs/tasks/v1.4/P4/P4-03.md`.

Frozen manifest SHA-256: `c4050efe1fe1d6017bcf310333698f2dcdf8a7ee76dd86561dad774c9c9f7420` (pinned in `test/eval/v14_product/manifest_test.go` and in the bare runner; any byte change invalidates the trial per PRE_REGISTRATION §5).

## 0. Preconditions — before any arm runs

1. `go test ./test/eval/v14_product/...` passes (manifest schema, freeze hash, fixture hashes, oracle coherence, arm neutrality).
2. `go run ./scripts/v14_trial/bare self-check` passes every line EXCEPT any line whose FAIL names a **frozen-fixture defect**; such a line blocks the TRIAL until the freeze owner adjudicates it (see §7; the set itself is never edited — TASK_SET §8).

## 1. Environment identity (both arms, every result row)

Recorded per PRE_REGISTRATION §6 and Decision Lock 15 (actual ≠ desired):

| Fact | Capture | Both arms |
| --- | --- | --- |
| host OS / arch | runner `env.host_os` / `env.host_arch` | identical command |
| Go version | runner `env.go_version` | identical command |
| runtime/CLI identity | `codex --version` self-report (parsed only in the `codex-cli <version>` form; raw line kept otherwise) | identical surface |
| model | **observed only** — see §5; requested `-m` recorded separately | identical requested id |
| DB version | `SHOW server_version` via the live evidence path (migration tasks); recorded only when it executes | identical surface |
| date | runner `env.trial_date` | identical command |

Requested configuration (model id, sandbox mode, provider endpoint fields, timeout) is recorded in every attempt record; it is never reported as an observed fact.

## 2. Arm A — bare (same coding agent, no Tally loop)

The bare arm is the SAME codex CLI on the SAME task inputs, driven directly by a human operator. No contract freeze pipeline, no policy matching, no automatic verification/repair; the only retries are what the agent does unprompted plus the human's own actions, each journalled.

Per task, first attempt:

```bash
go run ./scripts/v14_trial/bare run --task <task_id> \
    --model <MODEL_ID> \
    --sandbox workspace-write \
    --timeout 30m \
    [--codex-home <owner codex home>] \
    [--provider-id <id> --provider-name <name> --provider-base-url <url> \
     --provider-wire-api responses|chat --provider-requires-openai-auth]
```

What `run` guarantees:

- **Workspace**: attempt 1 provisions a fresh disposable workspace under the runs root: every agent-facing pinned fixture, byte-identical, at its repo-relative layout (TASK_SET §4). The frozen `acceptance.yaml`, `oracle/**`, and the reference repair NEVER enter a workspace. Later attempts (`--attempt N`, N>1) REUSE the same workspace unchanged — the human continues where they left off.
- **Instruction**: exactly the verbatim `intent.md` bytes, sha256-checked against the manifest at send time; nothing is prepended or appended.
- **Isolation**: one fresh `CODEX_HOME` per attempt; the only cross-home transfer is `auth.json` (Decision Lock 22); custom-provider deployments get the ADR-0003 minimal `config.toml` seeding (typed fields only, no secret value ever written or recorded). The home is deleted after the attempt (retention is recorded, never hidden).
- **Argv**: mirrors the production adapter's probe-recorded shape (`exec --json --skip-git-repo-check -C <ws> -s <mode> [-m <model>] -o <last-message> <instruction>`); recorded verbatim in the attempt record.
- **Budget**: per-attempt wall-clock ceiling (`--timeout`, equal in both arms); a ceiling hit is recorded as `timeout`, never as a candidate failure.
- **No loop**: the runner never imports or wires any Tally loop package; verification output is never fed back to the agent.

Human orchestration between/after attempts — every discrete action is journalled:

```bash
go run ./scripts/v14_trial/bare event --task <task_id> --attempt <n> \
    --type <clarification|manual_repair|manual_rerun|waiver|tradeoff_decision|\
forced_retry|direct_code_correction|failure_copy_back|prompt_reissue|verification_restart> \
    [--note "<what was done>"]
```

Diff viewing MAY be journalled with `--type diff_viewed`; it is logged only and is NOT an intervention (PRD §31.1). `events` prints the journal with the frozen classifier applied.

Judging (external, AFTER the last attempt — identical command for both arms):

```bash
go run ./scripts/v14_trial/bare judge --task <task_id> [--attempt <n>] \
    [--pg <maintenance-uri-without-db-name>]
```

- **bug_fix**: the delivered package is compiled in a judge sandbox (temp module outside the owner worktree) and every frozen oracle case is driven through the documented entry point; `regression_check_passes_after` requires all cases to pass. `bug_repro_failed_before` is judged from frozen fixture identity + the P4-01 registry proof.
- **database_migration**: live-DB evidence per PRD §24.2 — the delivered chain must apply cleanly on a fresh database AND on the seeded N-1 upgrade path, then the frozen per-task data-integrity assertions must hold. Without a PostgreSQL client/connection the keys are recorded `NOT_RUN(reason)` — never PASS (Decision Lock 20).
- Every ACCEPTED verdict carries a basis (gate key statuses + per-case/assertion records) in `judge-result.json`. An incomplete gate is never an acceptance.

## 3. Arm B — Tally (P4-03)

Same frozen bytes; Tally derives the Contract from the same intent; the human receives escalations only. Arm B is judged by THE SAME judge command as arm A (§2) — the judge is arm-neutral by construction: it reads the frozen acceptance + oracle from the repository and the candidate bytes from a workspace, and is wired to no arm. Its home under `scripts/v14_trial/bare/` is a P4-02 path-ownership fact, not an arm label.

## 4. Parity contract (differences limited to the Tally loop itself)

| Dimension | Guarantee |
| --- | --- |
| Task assets | byte-identical: all agent-facing fixtures hash-pinned in the manifest; verified at every provisioning and every load |
| Workspace provisioning | same agent-facing subset, same relative layout, same disposable-workspace discipline |
| Context | verbatim `intent.md` bytes to every participant; arm B's Contract is generated from the same bytes |
| Agent/runtime | same codex binary, same CLI version surface, same sandbox mode (`workspace-write`), same requested model id |
| Budgets | equal per-attempt wall-clock ceiling; token/cost computed later from the recorded JSONL only (transparency metric in trial 1) |
| Judge | the same command, the same frozen acceptance/oracle, the same code path, for both arms |
| Records | same result-row shape: timing, environment identity, observed identity, basis |

What legitimately differs: arm B runs the Tally loop (Contract → Execute → Verify → bounded Repair → Accept/Escalate) and therefore produces its own verification/repair lineage; arm A's human produces journal events instead. That difference IS the experiment; nothing else may differ.

## 5. Honest accounting — what an arm cannot capture is recorded as absent

- **Observed model**: the codex exec JSONL surface does not self-report the model (the production adapter records the same omission, Decision Lock 15). The requested `-m` value is recorded separately and never backfilled into observed facts.
- **Adapter version**: arm A does not run Tally's codex adapter; an adapter version is not an observable fact of the bare arm and is recorded as such. The CLI version IS observed (`--version` self-report).
- **Agent event stream / CodegenRun / assurance basis**: not applicable to arm A; absence is recorded, never fabricated into a Tally-shaped record.
- **DB version**: recorded only when the live-DB judge executes; otherwise the row says NOT_RUN with the reason.
- **Token/cost**: computed by P4-04 from recorded event logs only (PRE_REGISTRATION §7); the runner deliberately does not interpret unrecorded JSONL event shapes (DECISION_LOCKS #13 discipline).

## 6. Exclusions and replacement policy

Verbatim PRE_REGISTRATION §5: only (1) a fixture that cannot materialize deterministically (excluded from BOTH arms, with evidence) and (2) a mid-task environment outage >30 min (task voided in BOTH arms, logged, never silently re-run). No exclusion based on outcome, difficulty, or condition. Tasks are never replaced, re-tiered, or added after the first condition starts; a manifest byte change invalidates the trial unless BOTH arms re-run on the new set with first results preserved.

## 7. Standing findings recorded at the freeze of this protocol

- **bf-01-token-bucket-offbyone — frozen-fixture coherence defect (TRIAL-BLOCKING until adjudicated).** The pinned baseline snapshot (`test/eval/v14_product/testdata/tasks/bf-01-token-bucket-offbyone/baseline/token_bucket.go`) compiles standalone and its `Allow` behaves as the FIXED boundary on all six frozen oracle cases (`available >= 0` ⟺ allow iff `requestIndex ≤ capacity+refilled`), so it fails `{}` where the frozen `expected.baseline_failures` is `{fourth-immediate, fifth-after-refill, capacity-two-third}` (the registry's buggy mirror allows `requestIndex ≤ capacity+refilled+1`). Consequence: a do-nothing candidate would pass bf-01's `regression_check_passes_after` in BOTH arms. `bare self-check` reproduces this mechanically. Per TASK_SET §8 the set is never edited; the freeze owner must either re-freeze a corrected set (new preregistration + new pinned hash, both arms on the new set) or explicitly accept the documented limitation BEFORE arm A starts.

## 8. Records layout (runs root, never committed)

```text
<runs-root>/<task_id>/
  events.jsonl                       # journal: agent attempts, human events, verifications
  attempt-<N>/
    ws/                              # disposable candidate workspace (attempt 1; reused later)
    attempt.json                     # argv, timing, exit classification, requested+observed identity
    codex-stdout.jsonl               # raw agent stdout stream (evidence; parsed conservatively)
    last-message.txt                 # completion artifact (-o)
    judge-result.json                # external judge verdict + basis (arm-neutral)
    codex-home/                      # isolated CODEX_HOME, deleted after the attempt
```

Default runs root: `<os temp>/tally-v14-trial/bare` (override with `--runs-root`). Records are local trial artifacts; nothing under a runs root is committed, and no credential material is ever written into a record.
