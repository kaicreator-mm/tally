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
