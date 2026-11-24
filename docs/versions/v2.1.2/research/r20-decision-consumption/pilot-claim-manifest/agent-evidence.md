# T3 pilot — builder agent evidence: Claim provider manifest framing

Output label: `TALLY_V212_R20_T3_REAL_CLAIM_MANIFEST_KNOWLEDGE_CONSUMPTION`
Parent: #561 (version #521). Reference: #562 @ `976f46dcc377da2a54d7147e6fc370ecf7773225`.
Base / exact head: `3dbbb7d6c1ddbf82aad652011fab69b4375f686a` (tree
`6d181e5dfb4fc9e49e6d83e723bde6ada9a6fc10`), branch `research/v212-r20-t3-claim-manifest`.
Role: independent software engineering agent (builder). Mode: finite real-code knowledge
consumption. Source of truth: GitHub exact SHA.

This file has two parts. **Part A (sections 0–2) was written and committed before any
knowledge was materialized.** Part B (section 3 onward) records what happened after the
selector ran. Nothing in Part A was edited afterwards.

## Identity and environment (as of Part A)

- `actor_role` builder; `operator_kind` other (execution surface: ZCode desktop agent,
  model GLM-5.4 — the ADS enum has no value for this surface); `operator_id`
  `other:zcode-nilar-r20t3-builder`; `session_ref` `zcode-r20-t3-claim-manifest-20261125`;
  `transport_actor` `github:kaicreator-mm`.
- Host: NILAR, Windows 10.0.26300, win32 x64, Git Bash. Node v22.16.0 selected through fnm
  with the version directory PATH-prefixed (the shell default is v24.21.0).
- Standard in force: tally `.dev-standard/VERSION` pins ADS v4.18.0 at
  `2c395c55569fe2edf3f4135f12d3d069e526d0b7`.

## 0. Reading log before any knowledge

Read: `TASK.md`; `claim-target.mjs`; `claim-target.test.mjs`;
`claim-admission-shim.mjs`; `pilot-contract.test.mjs` (the four frozen blob IDs were
checked against the values in TASK.md and all match at the exact head); repo
`AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `.dev-standard/*`, `.woodpecker/*`,
`docs/research/v2/GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md`; ADS v4.18.0 `AGENTS.md`,
`standards/CODEX_ROLE.md`, `standards/VALIDATION_STANDARD.md`,
`templates/agent-event-comment.md`; `selector.mjs` in full (112 lines: CLI contract,
route table, budget logic).

Deliberately not read: `catalog.json`, `README.md`, `example.json` and
`selector.test.mjs` in `r20-decision-consumption/`, the body and comments of #561, and
every other r20 document. The reason is the pilot rule against pre-reading the knowledge
corpus or coordinator recommendations; the cost is that the meaning of each selector risk
label was assessed from the label alone.

Other-consumer check (read-only): the frame has exactly one writer and one reader outside
this pilot directory, `experiments/v2/selected-business-build/bundle-claim-reader.mjs`
(write) and `producer.mjs` (read). No second independent parser of the marker exists, and
no original path is touched by this pilot.

## 1. Baseline reproduction (unmodified exact head)

Recorded at `2026-11-24T20:23:51Z`, HEAD `3dbbb7d6`, clean tree, Node v22.16.0, win32 x64.

| Check | Result |
| --- | --- |
| `node --check claim-target.mjs` | exit 0 |
| Original suite `node --test claim-target.test.mjs` | 5/5 PASS, exit 0 |
| Public probe `node --test pilot-contract.test.mjs` | 4/8 PASS, exit 1 |

Failing probes and their observed behavior, which matches the coordinator's H0 exactly:

- Probe 2 (long admitted target): the valid frame is rejected with
  `T06L_PROVIDER_TARGET_MANIFEST_INVALID`.
- Probe 4 (long wrong target): `T06L_PROVIDER_TARGET_MANIFEST_INVALID` where
  `T06L_PROVIDER_TARGET_MISMATCH` is required.
- Probe 6 (marker without LF): no exception is thrown.
- Probe 7 (oversize): the writer does not throw.

Frame sizes measured with a read-only probe against the baseline module: short example
(owner 12 / repository 20) 246 bytes excluding LF and accepted; long admitted example
(owner 39 / repository 100) 534 bytes excluding LF and rejected; owner 2000 / repository
2000 gives a 10830-byte header that the writer emits without objection.

Root cause, from reading the source: `assertClaimProviderTarget` derives the first line
from `bytes.toString('utf8',0,512)`, so any valid header longer than 512 bytes is cut
before validation, and an unterminated header shorter than 512 bytes is accepted because
`.split('\n')[0]` never asks whether a terminator existed. The writer has no size check.

## 2. Part A — engineering decision recorded before knowledge

Recorded at `2026-11-24T20:26Z` and committed before the selector was invoked.

### 2.1 Risks identified by the builder

- **RK1 frame-bound precision.** The limit is 4096 bytes including the LF. A wrong bound
  silently accepts an over-limit frame or rejects a valid one. Specific traps: the window
  edge (LF at index 4095 against 4096), and character index against byte index when the
  window is a decoded string.
- **RK2 writer/reader agreement.** Both must use one bound, and the writer must measure
  exactly the bytes it emits (prefix, base64 and LF).
- **RK3 error classification.** Structural faults (no LF, bad encoding, unparseable
  metadata, over-limit) are `T06L_PROVIDER_TARGET_MANIFEST_INVALID`; a valid frame for
  another target is `T06L_PROVIDER_TARGET_MISMATCH`, including long frames; a selected
  target with no marker is `T06L_PROVIDER_TARGET_MANIFEST_MISSING`; markerless input with
  the historical default target is accepted.
- **RK4 bounded inspection.** The whole provider bundle is the input, and the contract
  forbids decoding or scanning it for header validation. Valid-input tests cannot observe
  a violation, because body bytes are ignored either way.
- **RK5 legacy and regression.** `resolveClaimBuildTarget`, `sealClaimShimTarget`, the
  markerless path and the frozen original suite must not change.
- **RK6 evidence integrity.** Frozen blobs stay byte-identical, negative controls must
  actually discriminate, and self-report must stay separate from observed behavior.
- **RK7 write-set compliance.** Only `claim-target.mjs`, `builder.test.mjs` and this file.

### 2.2 Decision: invoke the selector

Yes. The reasons, as I actually weighed them:

- The contract settles behavior, so I expect to finish correctly without knowledge. That
  is not the question. The question is whether any decision I still own could be made
  better or shown wrong by a relevant lesson.
- Two such decisions exist. The representation of the frame (decoded-string window or
  byte slice) and its organization are explicitly left to me, so a project convention
  would legitimately settle a choice that is otherwise a coin flip. And my preferred
  representation depends on an argument — character index equals byte index for any frame
  that can be accepted — that is exactly the kind of reasoning a prior lesson could
  falsify.
- The cost is bounded by the selector (at most 3 pieces, 6144 bytes).
- I am not invoking for the sake of participating. Had RK1 and the representation choice
  been fixed by the contract, the honest input would have been no applicable risk.

Honest limits of this input: I classified the stage as L3 (implementation) because the
decision is how to implement a given contract inside one module, although the pilot is
named a source-maintenance pilot. Each risk label was judged by its name alone.

### 2.3 Risk vocabulary mapping

Selector context: `_tmp/tally-i561-r20t3-zcode/selector-context.json` in the kAiCreator
workspace, SHA-256 `9b03f81394911b2559e04bd9df93122671becf2332d13b1d65a611caedd20f4f`.
Stage L3, contract CONFIRMED, default budget.

| Selector risk | Applicability | Basis |
| --- | --- | --- |
| precision | YES | RK1 |
| abstraction | YES | representation and organization freedom; RK2 |
| effects | YES | RK4 |
| boundary | NO | the contract forbids any permission or claim-semantic change |
| alias | NO | no new name or identity mapping; comparison semantics unchanged |
| oracle | NO | the acceptance oracle is the frozen public probe |
| maintenance | NO | legacy behavior is fixed by the frozen task and original suite |
| cost | NO | the bound and test set were chosen by the coordinator |

### 2.4 Design decision before knowledge (D1)

Stated so that ALREADY_ALIGNED and APPLIED can be told apart honestly afterwards.

- **D1.1** One module-level constant `MANIFEST_FRAME_MAX = 4096` (bytes, including the
  terminating LF), used by both writer and reader.
- **D1.2** Writer: build the header string, throw `TypeError('T06L_PROVIDER_TARGET_INVALID')`
  when its byte length exceeds the constant, otherwise return header plus code.
- **D1.3** Reader: decode only `bytes.toString('utf8', 0, MANIFEST_FRAME_MAX)`, keep the
  existing prefix check, require an LF in that window, validate the text between prefix
  and LF with the existing canonical-base64 check and `JSON.parse`, all inside the existing
  `try` block so every structural fault maps to `T06L_PROVIDER_TARGET_MANIFEST_INVALID`.
  Minimal diff; the existing control flow and the comparison are untouched.
- **D1.4** Classification kept as it is today: metadata that parses but is not this
  target's manifest is MISMATCH. That includes the exotic case of base64 that decodes to
  `null`. I read "invalid metadata" as unparseable metadata, because the contract also
  says existing identity and mismatch semantics are retained. This is my interpretation,
  not a certainty, and both codes fail closed.
- **D1.5** Builder tests: both sides of the limit for writer and reader, LF position,
  unterminated variants, non-canonical base64, non-ASCII header bytes, legacy markerless
  input, long-frame mismatch, and an instrumented buffer that fails if the reader asks for
  more than the first 4096 bytes.

Points I expected knowledge could move: decoded-string window against byte slice (D1.3),
`utf8` against a byte-exact decoding, where the LF check lives, and the visibility and
name of the constant.

## 3. Part B — after the selector

Recorded `2026-11-24T20:38Z`. Part A above was committed as
`8e52ea56c66216654c22f78df54a22c086d3484f` at `2026-11-24T20:26:54Z` and has not
been edited since. The knowledge outcome in one line: three pieces were delivered
and read, and none of them changed the decision (`ALREADY_ALIGNED`).

### 3.1 Selector invocation

| Item | Value |
| --- | --- |
| Time | `2026-11-24T20:27:04Z`, ten seconds after the Part A commit |
| Worktree HEAD at invocation | `8e52ea56c66216654c22f78df54a22c086d3484f` |
| Command | `node selector.mjs selector-context.json`, run from `r20-decision-consumption/` |
| Node | v22.16.0, win32 x64 |
| Context SHA-256 | `9b03f81394911b2559e04bd9df93122671becf2332d13b1d65a611caedd20f4f`, the file hashed in Part A section 2.3 |
| `selector.mjs` blob | `e79114b54c4e1d26f6d7dee9d30b40d2089ca107`, unchanged |
| `catalog.json` blob | `28c76328d2b590154f73708c3d830e92318271f2`, unchanged |
| Exit code | 0 |
| Receipt status | `DELIVERED`, reason `CALLER_ASSESSED_APPLICABILITY` |
| Budget | 6144 bytes and 3 pieces allowed; 3324 bytes and 3 pieces used |
| Receipt SHA-256 | `e51a0f1f6c99a4a1b5c9a814a4cce14ff2e4ab285d8e10ec3171aa44779ddabe` |
| Delivery meaning | `MATERIALIZED_NOT_READ`, qualification `ADVISORY_CANDIDATE` |
| Rejected by the selector | boundary, alias, oracle, maintenance, cost, each `NOT_APPLICABLE` |

### 3.2 Exact materialized text

Copied mechanically from the saved receipt, not retyped. Each block is the `text` field of one delivered piece.

#### Piece `precision`

- path `K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md`, blob `a1e640efb9866f1981a80f43d64c661635d52c64`, lines 52–54
- excerpt SHA-256 `34a351c9b466c0a8c16a40d36ba0f27ee8776da2d515970eed3caefa4e032809`, 1116 bytes

````text
[R19-F2 研究反馈](../professional-abstraction-empirical-feedback-r19f2.md) 记录了一个跨多个实现的 JS 时间边界反例：输入均为安全整数，并不保证窗口端点计算、下一周期、重试倒计时的**中间运算**不会产生舍入误差。对有明确高精度合法域的任务，应设计接近整数/时间边界的可证伪测试，核对完整表达式范围；根据现行合同选择安全的差值/余数推导、范围限制或 BigInt。缺少精度要求时先向 Source Owner 确认，而不是机械要求更复杂的架构。

这一条属于**行为正确性/输入语义**，不因出现数字 bug 就自动计成 UNDER、WRONG 或 OVER 抽象；若要判为抽象错误，Reviewer 应证明失效与具体抽象决策（例如领域表示、语义合并、共享机制、Owner 边界或抽象粒度）的因果联系。例如局部领域类型把精确整数与近似测量值统一成舍入表示，导致合法精确值不可表达，即使没有跨模块共享也可能是 WRONG；若只是公式或实现 bug，则记录行为缺陷，不自动贴抽象标签。
````

#### Piece `abstraction`

- path `K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md`, blob `a1e640efb9866f1981a80f43d64c661635d52c64`, lines 19–24
- excerpt SHA-256 `30af3c68a79af7b8f34efba3b3740ef10529f723d76f10cd4378759127b642bb`, 1107 bytes

````text
| **保持内联** | 逻辑局部、短且无真实复用或独立变化，额外接口不降低认知负担 | 最短行为路径，无隐藏 indirection | 大量散落的同一行为导致修改遗漏 |
| **局部 private helper** | 同一个 Owner 内多个点共享已证明相同机制，不需要跨包消费者 | 统一算法/校验，保留局部阅读 | 把不同错误/权限隐藏成一个布尔开关 |
| **领域模块/文件内聚** | 一组不变量、状态/资源生命周期和变化原因由同一 Owner 管理 | 隔离变化，把责任、测试和失败收口 | 仅按动词拆模块导致共享状态失控 |
| **窄公共接口/契约** | 存在真实异构消费者、稳定替换边界或独立发布/兼容要求 | 隐藏实现细节、减少非必要传播 | 因为一个调用者就引入接口、Adapter、Registry 或额外版本承诺 |
| **Strategy/可插拔 Provider** | 被授权的动态扩展/替换、多个实现共享同等合同而非只是同名 | 显式合法变化轴，可受控替换 | 两种实现有根本不同的错误/事务/信任语义，被强行统一 |

````

#### Piece `effects`

- path `professional-risk-gated-knowledge-selection-r15.md`, blob `b5281db44c2c222f295404bffd7850dd016ef017`, lines 21–23
- excerpt SHA-256 `808d7fc4a43120d53a34a01b1f75c73f1f249903e7aeddaad1c0b0f23c29c6d1`, 1101 bytes

````text
| **并发写入、可恢复的本地持久状态** | L2：K02 状态 Owner / 崩溃切点；选定方案后 L3：K03 文件/DB 原语；验证：K05 不变量负例 | 实际可持久观察的状态序列、提交点、竞争处理、真实 OS/FS 前提和独立测试 | 条件创建 `wx` ≠ 业务事务；`rename` ≠ 一切介质上的掉电耐久保证 |
| **外部 Effect / 发送成功但 ACK 丢失** | K02 业务与外部 Effect 的提交边界；K03 有条件回读/恢复；K05 作用数、冲突/UNKNOWN 的反例；需要时 K10 | 明确“权威状态读回”的所有者和事务外副作用表；不能确定的记 UNKNOWN | 本地存储 Commit 不能代表外部接收方 Effect 已提交；超时不能直接报告未发生 |
| **版本前置条件、跨 Writer 冲突** | K01 真正错误合同 → K02 原子性范围 → K03 并发/回读实现 → K05 顺序旧版本 + 竞争者两类独立 Oracle | `CONFLICT`、`CORRUPT`、`UNKNOWN` 的有权判别和不可变事实 | 文件已存在或被锁占用 ≠ 一定损坏；竞态冲突测试不能替代顺序 `--expect` 测试 |
````

### 3.3 Reading and usage, stated separately

Read: yes. The selector command printed all three texts above into my context
and I read each in full at `20:27Z`. Nothing else from the catalog was read. The
catalog file was parsed only by `selector.mjs` and, once more, by my `record.mjs`
wrapper in order to call `recordApplication`; no text from it other than these
three pieces was printed.

Usage, piece by piece:

| Piece | What it says | Applies to this decision? | Outcome |
| --- | --- | --- | --- |
| precision | A JS counterexample where safe-integer inputs still round in intermediate arithmetic. It asks for falsifiable tests near the boundary, a check of the full expression range, and a safe derivation chosen under the current contract. | The discipline applies; the hazard does not. Every value here is a small integer below 4097 and there is no float arithmetic. Boundary tests on both sides were already D1.5. | ALREADY_ALIGNED |
| abstraction | A table for choosing an abstraction level: keep logic inline when it is local and short; a private helper only for one proven mechanism shared inside one Owner; a narrow public interface only for real heterogeneous consumers. | Yes. It settles the point Part A left open, constant visibility, in favour of a private shared constant with no new export, which is what D1.1 already said. | ALREADY_ALIGNED |
| effects | External-effect and commit-boundary semantics: local persistence against business transactions, send-succeeded-but-ACK-lost, cross-writer conflict and readback. | No. The reader is a pure function and the writer returns a string; there is no external effect, persistence or concurrency. The YES I declared for this risk was judged from the label alone and was wrong. | REJECTED |

Overall: decision unchanged. `recordApplication`, the selector's own library
export with the selector unmodified, accepted the record with the same
`receiptSha256`:

```json
{
 "decisionId": "r20-t3-claim-manifest-frame-v1",
 "outcome": "ALREADY_ALIGNED",
 "qualification": "SELF_REPORTED_NOT_VERIFIED",
 "qualityBenefit": "NOT_ESTABLISHED"
}
```

That qualification is accurate. The alignment is my report. The behaviour in
section 3.6 is observed. That the knowledge caused none of it is not something a
test can show.

### 3.4 Decision after knowledge (D2)

Identical to D1, point by point against section 2.4:

- **D2.1 = D1.1.** One private module constant `MANIFEST_FRAME_MAX = 4096`, bytes
  including the LF, used by writer and reader. Not exported.
- **D2.2 = D1.2.** The writer builds `prefix + base64 + '\n'`, throws
  `TypeError('T06L_PROVIDER_TARGET_INVALID')` when its byte length exceeds the
  constant, and returns header plus code.
- **D2.3 = D1.3.** The reader decodes only `bytes.toString('utf8', 0, MANIFEST_FRAME_MAX)`,
  keeps the prefix check, requires an LF in that window as the first statement of
  the existing `try` block, and validates the text between prefix and LF with the
  unchanged canonical-base64 check and `JSON.parse`.
- **D2.4 = D1.4.** Metadata that parses but is not this target stays `MISMATCH`.
- **D2.5 = D1.5.** Builder tests as listed in Part A.

The four points Part A expected knowledge might move all stayed where they were:
decoded-string window rather than byte slice, `utf8` rather than a byte-exact
decoding, the LF check inside the existing `try` block, and a private constant.

### 3.5 Resulting source diff

Write set respected: `claim-target.mjs` modified, `builder.test.mjs` added, this
file. Nothing else.

| File | Blob before | Blob after | Change |
| --- | --- | --- | --- |
| `claim-target.mjs` | `ee60d8b25420e766087e259fa7d586b913f988f0` | `ee3955ebf9fe2af12300b5c8e0743f851380b292` | 15 lines added, 4 removed |
| `builder.test.mjs` | absent | new, 177 lines, 10 tests | added |

The candidate blob has been `ee3955eb…` since the first edit and was identical
before and after every control run. The frozen files keep their blob IDs:
`claim-target.test.mjs` `1388ae09…`, `claim-admission-shim.mjs` `729fb2f4…`,
`pilot-contract.test.mjs` `81e8fce1…`, `TASK.md` `d245d30f…`, `selector.mjs`
`e79114b5…`, `catalog.json` `28c76328…`.

### 3.6 Verification before the first code commit

All runs are against the working tree at base `3dbbb7d6` plus the candidate,
Node v22.16.0 on win32 x64 unless stated.

| Check | Baseline | Candidate |
| --- | --- | --- |
| `node --check` on the module and the builder test | exit 0 | exit 0 |
| Original suite `claim-target.test.mjs` | 5/5, exit 0 | 5/5, exit 0 |
| Public probe `pilot-contract.test.mjs` | 4/8, exit 1 | 8/8, exit 0 |
| Builder tests `builder.test.mjs` | 4/10, exit 1 | 10/10, exit 0 |
| All three files in one `node --test` run | not run | 23/23, exit 0 |
| Same on Node v24.21.0 | not run | 23/23, exit 0 |

**Differential against the frozen baseline.** A corpus of 131 cases (writer
output for 20 targets, valid frames read as same, other and historical target,
markerless bodies, structural variants) was recorded from the baseline module
before any edit and again from the candidate. Result: 108 unchanged, 23 changed,
no unexpected change, no expected change missing, no change in a wrong direction.
Every change is one the contract mandates:

- 20 long valid frames: `T06L_PROVIDER_TARGET_MANIFEST_INVALID` at the baseline
  (truncated) became `ACCEPT` or `T06L_PROVIDER_TARGET_MISMATCH`.
- The unterminated short header: `ACCEPT` (selected target) or `MISMATCH`
  (historical target) became `T06L_PROVIDER_TARGET_MANIFEST_INVALID`.
- The 2000/2000 target: emitted at the baseline, now refused by the writer with
  `T06L_PROVIDER_TARGET_INVALID`.

All 19 non-oversize writer outputs are byte-identical, all 28 markerless verdicts
are unchanged, and metadata that parses but differs (`null`, an array, changed key
order, an extra key) keeps `MISMATCH` as it had at the baseline.

**Mutation controls.** Each mutation was applied to a disposable copy of the
module, with the three test files copied beside it; the candidate was never
modified. 19 controls, all matching their stated expectation:

| Class | Count | Result |
| --- | --- | --- |
| Unmodified baseline | 1 | killed: 10 of 23 tests fail (6 builder, 4 public probe) |
| Contract fault | 9 | all killed |
| Reads past the 4096-byte frame but same behaviour | 3 | each killed by the bounded-inspection test alone |
| Equivalent mutant | 6 | all survive, as analysed |

The nine contract faults and the tests that killed them:

| Mutation | Killed by |
| --- | --- |
| Reader window 4099, accepts an over-limit frame | reader edge test, bounded-inspection test |
| Reader window 4094, rejects the largest valid frame | reader edge test, agreement sweep |
| LF check removed | unterminated-frame test |
| Unterminated frame read to the window end | unterminated-frame test, public probe 6 |
| Whole buffer decoded | reader edge test, bounded-inspection test |
| Writer limit removed | writer edge test, agreement sweep, public probe 7 |
| Writer rejects the largest valid frame | writer edge test, reader edge test, agreement sweep |
| Writer accepts the first oversize frame | writer edge test, agreement sweep |
| Truncation at 512 reintroduced | 4 builder tests, public probes 2 and 4 |

The three out-of-frame reads (window 4097, window 4098, whole buffer decoded then
sliced to 4096) give exactly the same verdicts as the candidate for every input.
Only the instrumented-buffer test distinguishes them, which is the reason that
test exists.

### 3.7 Mistakes I made and corrected

None of these touched the candidate after its first edit. They are listed because
they are the real cost of this task, and because two of them would have produced
false confidence if left in place.

1. My first size probe searched a string with `indexOf(10)`, which looks for the
   text `10`, not a line feed. Fixed and re-run before any record was written.
2. I counted the prefix as 29 bytes; it is 30. Frame sizes are therefore 31 + 4k
   and the edges are 4095 and 4099, not 4094 and 4098. Caught while writing the
   edge tests; no test or code ever carried the wrong numbers.
3. Builder test 5 first injected its invalid byte at offset 600 of a 534-byte
   header, which is the body. The test failed on the candidate and the candidate
   was right. Fixed by using a header that ends at index 906 and asserting the
   offset is inside it.
4. Builder test 8 first subclassed `Buffer`. `new Watched(bytes)` returns a plain
   Buffer, so nothing was recorded. It failed on its own assertion that something
   had been recorded. Replaced with a `Proxy`.
5. The builder tests lacked the non-canonical base64 cases that D1.5 had listed.
   Added.
6. My differential oracle first demanded a change for an unterminated header
   followed by code. The baseline already rejects that input, for an unrelated
   reason. The oracle was wrong, not the candidate; corrected.
7. My first mutation list carried five wrong expectations. Three of those
   mutants reassign a `const` or throw inside the `try` block, which makes them
   identical to the candidate; the other two sit inside an interval that no input
   can distinguish. The list was rewritten with the classes shown above.

### 3.8 What this does not prove

- **Platform.** Everything in section 3.6 ran on win32 x64. The coordinator's H0
  was Linux. Results at the committed SHA on other hosts are in the Issue receipt,
  not here.
- **Remote CI: NOT_RUN.** `.woodpecker/ci.yaml` triggers only on `main` and
  `version/*`, and its sparse checkout excludes `docs/**`.
- **Independent review: NOT_RUN.** Nothing here is a review verdict.
- **Production.** Only the pilot mirror changed.
  `experiments/v2/selected-business-build/claim-target.mjs` is untouched. The
  4096 limit is a pilot contract and is not a statement about the previous
  production contract.
- **Bounded inspection is observed only for the access paths the reader uses.**
  The instrumented buffer sees `toString`, `subarray`, `slice`, `indexOf`,
  `lastIndexOf`, `includes` and numeric element reads. An implementation that
  reached the body through some other Buffer method would not be caught.
- **Some thresholds cannot be observed.** No writer-reachable frame has 4096,
  4097 or 4098 bytes, so a writer comparison anywhere in that interval and a
  reader window of 4095 are indistinguishable from the candidate (E01, E02, E03).
- **Interpretation.** D1.4, that "invalid metadata" means unparseable metadata,
  is my reading of the contract. It preserves the baseline behaviour and both
  codes fail closed, but it is an interpretation.
- `OUTPUT_LEDGER.md` was not updated. The repository protocol asks for it, but it
  is outside the write set this task authorises, so the durable record is the
  Issue #561 receipt.

### 3.9 Cost

- Wall clock: about 9 minutes from worktree creation to the Part A commit, 10
  seconds from that commit to the selector run, and about 11 minutes from the
  selector run to this record.
- Knowledge: 3324 bytes delivered and read, against a 6144-byte budget.
- Model tokens and money: NOT_RECONSTRUCTABLE. No usage meter is exposed to me.
- The first valid code commit SHA is the commit that contains this file, so it is
  recorded in the Issue receipt.

### 3.10 Local evidence index

Kept in the kAiCreator workspace at `_evidence-private/tally-i561-r20t3-zcode/` (working copy in `_tmp/tally-i561-r20t3-zcode/`, which is auto-cleaned). Not pushed. SHA-256 of each file at the time of this commit:

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `00-env-baseline.txt` | 197 | `1136bc744aca3d648fb5693c74b6ecfd408e9b4eb7e59a5f9394ad54b9a969bb` |
| `01-baseline-syntax.log` | 14 | `143805bdc3e0864ab8cddca6310e511a8bb6d71975fc868cd895ee5ee1bbc8a5` |
| `02-baseline-original-test.log` | 1115 | `aa383a149c2524332487b9a82564d8d4476fd94b40def952a12844eb01087b8e` |
| `03-baseline-contract-probe.log` | 7155 | `f5e6b9b0d397ddab356ffb6b1807c6b6cc3ee2147ba61c3e8308548b364c140e` |
| `04-baseline-frame-sizes.log` | 488 | `bf9c0eb69bd3ac3015c4bb32cb895951ba51a918a919029b0f0b3b121ca4a37a` |
| `05-selector-invocation.txt` | 481 | `333d0e9181fa99b72d0200e6563e981a767b0c68ade2dcf7ed92423c0a3af8a5` |
| `05-selector-stderr.log` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `05-selector-stdout.json` | 10869 | `d8ac61fce7b0d1deec8310c12abb10de7ac4384ca4b8f4d8a78bf173a14c6d47` |
| `10-baseline-verdicts.json` | 29286 | `eb0c2f7740484ed71259fb6c6087471a503d721865f2a75aea4b308dde9b7cd6` |
| `11-candidate-verdicts.json` | 18024 | `83d7e6e1d5a861301a652474a7ea9b988a3fa68e7fcd99c7a044b59d06823f68` |
| `12-corpus-diff.log` | 2715 | `e7ce9c878ca1b442a8ac0068be05925f525c0857f9e0cb9b8228d4d04d248d3a` |
| `13-golden-baseline.json` | 251 | `1f7125d8300c3bb15ec84ebbccd3c4712d0d0d45e03c694aedaadc13f859752a` |
| `13-golden-candidate.json` | 251 | `1f7125d8300c3bb15ec84ebbccd3c4712d0d0d45e03c694aedaadc13f859752a` |
| `20-candidate-original-test.log` | 1113 | `923ef396a57b2e5454e92aaacaa0b3217a4f6fa4654bfdb3d62682c6c5799303` |
| `21-candidate-contract-probe.log` | 1342 | `537d1554d9772c83fbf0c2b0224f2d835fe6de5025355fdd13a78c958e09bbb8` |
| `22-candidate-builder-test.log` | 2158 | `83582d3037a7bd91c48dbd889cee2661def23f67836141d17eb12b0d49c64aa7` |
| `23-node24-all-tests.log` | 1818 | `816db98991ab9bed8c7e1d5029b142fd1b99c236779f034a29f498e625e07aee` |
| `24-node22-all-tests.log` | 4380 | `877f3b5ac40eec40868f458d231b875cde7f844eba5b82df44eda49b66ab373f` |
| `30-mutants.jsonl` | 6103 | `4f2b4c80af51b30e1487839c535da7b2f8ae182258d16119ba06a1205c3b7431` |
| `30-mutants.stderr.log` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `31-application-record.json` | 3415 | `da018c265816d42e552c933dd8d54a5e796fa00e1d779ef46fc0bc02480dd4ba` |
| `31-application-record.log` | 262 | `baad334cfefe65fa0b0dc015ae73afb4441476d8723abc81a45a728158ee4038` |
| `application-report.json` | 3277 | `fa52b805d45b9e9fac0fdf133247403def2122972738ffff54947c84fbe3baba` |
| `corpus-diff.mjs` | 2307 | `8bf27e7e897dfda091e061e1e01eb947ed6c48ab43ba9f9c565e7307e3a0afab` |
| `corpus.mjs` | 5458 | `f499fc202fff6582aecee7eb57e2c0cdc0520c04df5ffa5205c7dbc0ee42bff8` |
| `golden.mjs` | 372 | `c18037f0e4b3fafc932d043b13970a5b8f638c437ea38e1abe1f8e54255ac5ee` |
| `mutants.mjs` | 6142 | `1dc9fbb371728edf55a678048e734af39c763c712bbaa6afa7f5bf9836f7c9d4` |
| `probe-sizes.mjs` | 1344 | `939c2c44bd9742e3196773a2ef3dcaea3fea6b705565651f1b1b84e9682178e7` |
| `record.mjs` | 927 | `f3d44313fcba2355b2a3f5ad7da464634e4e91f776aef727c3d39e5eaa1a2a93` |
| `selector-context.json` | 4149 | `9b03f81394911b2559e04bd9df93122671becf2332d13b1d65a611caedd20f4f` |
| `mutants/results.json` | 7194 | `af78035a45c848eb5bc21502adc71143f75c17679736f0fc002a3ef162b44f4a` |
| `baseline-blob/claim-target.mjs` | 3615 | `def6398d0340c96d07fad7eaef6cb71a0fa87c149a607b59ebc2d9159c072015` |
