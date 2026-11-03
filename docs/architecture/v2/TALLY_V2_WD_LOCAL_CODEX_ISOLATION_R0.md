# Tally v2 W-D — Local Codex CLI and independent Hidden verifier isolation R0

**Owner:** [#90](https://github.com/kaicreator-mm/tally/issues/90). **State:** SOURCE_SECURITY_CONTRACT_ONLY / PHYSICAL_DENY_TESTS_NOT_RUN / AGENT_PROVIDER_NOT_ACTIVATED. Applies to native DomainHarness selected B/S/K/C/Host, never a standalone Node daemon masquerading as a Package. ADS target v4.18; Tally project adoption PR #85 pending CI profile/red-check disposition. DomainHarness R01 #1331 not yet qualified.

## Execution/principal topology

```text
Authenticated operator → Host/App Shell [session+Origin/CSRF]
  → Engineering B [repo/Task scope + agent role/delegation]
  → selected K [N=1 executing Attempt Admission/CAS/Journal/Effect]
    → C Local/Host [real Codex CLI under low-privilege Builder principal]
    → C GitHub [K-permitted publisher with separate native GitHub credential]
  → independent verifier OS principal [held-out fixtures after candidate frozen]
```

Host must **keep the model Provider authorization needed by Codex separate from GitHub native write authority**. Do not give Builder the operator's user home, gh config, Git Credential Manager, SSH agent/socket or token, noninteractive sudo/admin/backup privileges, or path to held-out tests. GitHub publish goes only K→C GitHub after real scope/expected SHA/provider grant/readback. Host is physical process executor, K is sole state/admission/effect owner.

## Typed gate semantics (must survive all layers)

| Error | Exact predicate | Safe UI behavior |
|---|---|---|
| `LOCAL_AGENT_UNAVAILABLE` | Real installed/authenticated Codex CLI absent, capability unavailable, only Fake/echo provider or provider cannot authenticate | No Task execution; authorized Project/requirements browse/edit stays usable |
| `BUILDER_CONFINEMENT_UNAVAILABLE` | Real Codex Provider available but least-privileged Builder OS account/GitHub write credential deny not witnessable | Deny CJ03/Task close/Release, preserve other permitted non-Task UI |
| `HIDDEN_VERIFIER_UNAVAILABLE` | Independent verifier principal, ACL fixture isolation or held-out runner genuinely unavailable | Hidden FAIL/HOLD/NOT_RUN; normal Product UI and safe non-Hidden work unaffected |
| `HIDDEN_NOT_QUALIFIED` | Builder can read fixture or probe via inherited agent/mount/sudo | Invalidate Hidden evidence and require independent cleanup/retest |
| `UNGOVERNED_EFFECT` | Builder manages to push/merge/direct-send outside K Permit→C GitHub | Block release/Task acceptance; preserve audit and external effect reconciliation |

## Strict P0 execution and candidate sequence

1. Read actual GitHub Task+L3 source SHA, select claim/current executor/provider and inspect operator repo rights without treating GitHub account as logical Agent identity.
2. Enforce **at most one `EXECUTING` Attempt globally per selected Tally v2 app P0**, even when DAG has multiple ready workfronts and separate worktrees. Second K Admission fails typed; non-executing planning can parallelize.
3. C Local creates clean disposable source-qualified Git worktree and scoped write allowlist, no shared workspace; capture base SHA/tree, process identity/UID/SID, actual CLI version and provider auth state without logging secrets.
4. Host launches actual Codex CLI under Builder least privilege; no GitHub write credential or hostile capability through GCM/SSH/sudo/admin/backup; require **real deny-read and deny-write probes** with negative exit/log before considering confinement satisfied.
5. K records source-bound real exit, process child cancellation/recovery/UNKNOWN, output artifacts; remote job acceptance is not terminal result. Freeze real candidate Git commit/tree before Hidden fixture materializes.
6. Independent verifier OS principal mounts actual held-out fixture AFTER frozen candidate; Builder cannot read current/future fixtures via OS principal, extra paths, delegated privilege, predictable socket/pipe or inherited mount. True verifier executes required tests with literal exit and writes source-qualified receipt; fake fixture/mock won't satisfy release.
7. Governed C GitHub publishes only with K permission/source match/provider grant, re-reads actual GitHub object; lost ACK is `UNKNOWN_EFFECT` and triggers native reconciliation, not duplicate write.

## Mandatory negative proof list

| ID | Real/proposed probe | Required result |
|---|---|---|
| LA-N01 | Real Codex binary absent or CLI login invalid | LOCAL_AGENT_UNAVAILABLE |
| LA-N02 | Fake Agent gives plausible success text | NOT_QUALIFIED, no Task/CJ PASS |
| LA-N03 | Builder environment inherits GH_TOKEN/GITHUB_TOKEN or config | confinement probe must deny access; unavailable if not |
| LA-N04 | Builder can call `gh auth token` / GCM helper / SSH agent | BUILDER_CONFINEMENT_UNAVAILABLE |
| LA-N05 | Builder can `sudo -n`/Windows admin/backup privileges | BUILDER_CONFINEMENT_UNAVAILABLE even if ordinary ACL deny |
| LA-N06 | Builder can push directly despite C-GitHub-only design | UNGOVERNED_EFFECT, actual remote event investigated |
| LA-N07 | Two Tasks start concurrently on separate worktrees | K second Admission NOT_ADMITTED |
| LA-N08 | Dirty base/stale source or wrong repo route | reject before any process launch |
| LA-N09 | Verifier fixture present before candidate freeze | HIDDEN_NOT_QUALIFIED if exposed; no PASS |
| LA-N10 | Builder read hidden files via symbolic link, mount, alternate path, direct IPC | denied read witness or INVALID Hidden |
| LA-N11 | Real independent verifier account missing | HIDDEN_VERIFIER_UNAVAILABLE, no fabricated skip PASS |
| LA-N12 | Child process receives cancel while external effect pending | bounded cancel+reconcile, UNKNOWN until native proof |
| LA-N13 | Process/Agent prints secret to stdout/stderr | Host redacts prior to persisted/uploaded evidence |
| LA-N14 | Same messageId replay same payload and intent | typed idempotent replay, no repeated remote effect |
| LA-N15 | Same messageId different intent/source input | typed Conflict, fail closed |
| LA-N16 | GitHub native ACK lost after successful write | re-read by idempotency identity before retry, not duplicate |
| LA-N17 | Provider auth accessible, but GitHub key stored in same accessible home | BUILDER_CONFINEMENT_UNAVAILABLE |
| LA-N18 | Task telemetry hook hung/unavailable | actual Agent/Verifier status unchanged; metrics UNKNOWN, bounded fail-open |

## Physical platform probes and review threshold

- **Windows workstation**: separate unprivileged Builder and verifier identities, NTFS ACL/Windows access token, logon SID / UAC noninteractive rights, Git/GCM/SSH agent and OS job/child process controls. No real GitHub secrets in test fixture or screenshots. Use benign credential-access probes and harmless test repo, no privileged live production push.
- **Ubuntu Host**: `id`, file owner/mode/ACL and `sudo -n` negative, no inherited agent socket/SSH keys, read-only fixture mount and actual separate verifier UID, proper process group cleanup. Never assume a container alone proves principal separation.
- **Evidence:** exact Tally source+DH package candidate SHA and package tarball digests, OS/runtime/CLI versions, process identity, denied OS operations with exit, real verifier results, K Effect/UNKNOWN receipt, independent reviewer. If actual Host/Builder install not present, `NOT_RUN`, not source PASS converted to physical PASS.

**Source-only next:** contract/negative fixture independent Review now; implement selected real DomainHarness Host/Local C only after qualified DH public R01, accepted Product/L2/ADS current source, native Issue/Claim and Task L3. `PHYSICAL_DENY_TESTS=NOT_RUN / IMPLEMENTATION_ADMISSION=NO`.
