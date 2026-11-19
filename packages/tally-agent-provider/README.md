# Tally AgentProvider/1 (v2 Node contract candidate)

Single TypeScript/Node contract for all coding-agent CLI adapters. Source-only
candidate for [#473](https://github.com/kaicreator-mm/tally/issues/473).
It does **not** mount providers into the selected v2 Host and is not a Core MVP
or Release Qualification gate. Default provider is `codex`; others require an
explicit enabled list and actual Host qualification.

## Ownership

Provider owns `probe`, process-level `execute`, `cancel`, `reconcile` and
adapter-specific CLI args, redacted environment, output parsing. Host owns
accepted Claim/FSW, permission, K admission, worktree, write-set grants,
independent Git candidate creation/readback, Review, CI and Task Outcome.
A Provider `COMPLETED` result is **not** Task PASS or PR/Release authority.
The registry is an in-memory lookup, not a scheduler or execution journal.

## Public interface

`AgentProvider` has `id`, `contractVersion:1`, declared `capabilities`, and
four asynchronous methods. Each returns `{state, reason?, evidence?}`.
`evidence` is adapter-redacted, **untrusted**, and must be independently
verified by Host before any positive outcome. Top-level authority fields and
unknown fields are refused. `UNKNOWN` must include an error reason code.
`createProviderAttemptPin` records the Host-selected task, attempt, source SHA
and provider choice; the pin itself grants no permissions.

```js
import {defineAgentProvider, createAgentProviderRegistry,
  createProviderAttemptPin} from '@kaicreator/tally-agent-provider';

const codex = defineAgentProvider({
  id: 'codex', contractVersion: 1, capabilities: [],
  probe: async () => ({state: 'AVAILABLE'}),
  execute: async (_request, _hostContext) => ({state: 'UNKNOWN',reason:'NOT_WIRED'}),
  cancel: async () => ({state: 'UNKNOWN',reason:'NOT_WIRED'}),
  reconcile: async () => ({state: 'UNKNOWN',reason:'NOT_WIRED'}),
});
const registry = createAgentProviderRegistry([codex]);
const pin = createProviderAttemptPin({
  taskId:'264', attemptId:'attempt:264:e1',sourceSha:'a'.repeat(40),providerId:'codex'});
const provider = registry.resolve({pin, enabledProviderIds:['codex']});
```

The example's `UNKNOWN` handler is intentionally **not a production Codex
adapter**. Do not claim any provider has been migrated until its real Node
process/Host wiring and source-qualified evidence are available.

## Implementation order

1. Contract (this package) and independent delta review.
2. Node process adapters for existing Codex, ZCode #470 and DSH #467;
   translate OpenCode #471, Pi #469 and Grok #468 observed CLI contracts
   from their Go candidates into **Node** adapters without Go bridging.
3. A single E Writer, after R102 frees shared files, connects Registry, Task
   provider selection and existing B→K→Host/C. Task UI displays selected
   provider and authentic provenance. No fallback on failure.
4. Minimal source-qualified Windows live CLI smoke per enabled adapter and
   two disjoint admitted Task attempts with two different providers, using
   existing Host grants and worktree separation.

No changes to old Go Worker, no new scheduler, new CAS, security authority,
release route or premature main/tag/deploy. See #392 #64 #283 #264.
