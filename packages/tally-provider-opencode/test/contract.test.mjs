/**
 * Contract-compliance tests for the opencode AgentProvider/1 adapter:
 * registry (opt-in only, no default, no fallback), attempt pin discipline,
 * observation-only vocabulary (liar rejection), reconcile semantics.
 *
 * Uses the real contract implementation from
 * packages/tally-agent-provider (workspace-relative import; the candidate
 * has zero npm dependencies).
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {
  AGENT_PROVIDER_CONTRACT_VERSION,
  createAgentProviderRegistry,
  createProviderAttemptPin,
  defineAgentProvider,
  normalizeProviderObservation,
} from '../../tally-agent-provider/src/index.mjs';
import {
  createOpenCodeAgentProvider,
  OPENCODE_PINNED_VERSION,
  OPENCODE_PROVIDER_ID,
} from '../src/index.mjs';

const FAKE_CLI = fileURLToPath(new URL('./fake-opencode.mjs', import.meta.url));
const makeProvider = (over = {}) =>
  createOpenCodeAgentProvider({cliPath: process.execPath, cliArgs: [FAKE_CLI], ...over});
const pin = (providerId = OPENCODE_PROVIDER_ID) => createProviderAttemptPin({
  taskId: '264', attemptId: 'attempt:264:r1', sourceSha: 'a'.repeat(40), providerId,
});

test('opencode adapter satisfies the mandatory AgentProvider/1 shape', () => {
  const provider = makeProvider();
  assert.equal(provider.id, OPENCODE_PROVIDER_ID);
  assert.equal(provider.contractVersion, AGENT_PROVIDER_CONTRACT_VERSION);
  assert.equal(provider.contractVersion, 1);
  // Honest empty baseline (#471 evidence): no streaming, no resume/fork, no
  // model.select, no session.isolation (canary evidence not held).
  assert.deepEqual(provider.capabilities, []);
  for (const operation of ['probe', 'execute', 'cancel', 'reconcile']) {
    assert.equal(typeof provider[operation], 'function');
  }
  assert.equal(OPENCODE_PINNED_VERSION, '1.19.6');
});

test('registry keeps opencode opt-in only: no default selection, no fallback', () => {
  const registry = createAgentProviderRegistry([makeProvider()]);
  assert.throws(() => registry.resolve(), /PROVIDER_NOT_INSTALLED/,
    'default (codex) must not silently resolve to opencode');
  assert.throws(() => registry.resolve({providerId: OPENCODE_PROVIDER_ID}),
    /PROVIDER_DISABLED/);
  const resolved = registry.resolve({
    providerId: OPENCODE_PROVIDER_ID, enabledProviderIds: [OPENCODE_PROVIDER_ID],
  });
  assert.equal(resolved.id, OPENCODE_PROVIDER_ID);
  // codex enabled but not installed while opencode is: selecting codex must
  // fail, never silently fall back to the installed opencode adapter.
  assert.throws(() => registry.resolve({
    providerId: 'codex', enabledProviderIds: ['codex', OPENCODE_PROVIDER_ID],
  }), /PROVIDER_NOT_INSTALLED/, 'no fallback to an installed provider');
});

test('one attempt pin cannot be silently switched or rebound', () => {
  const registry = createAgentProviderRegistry([makeProvider()]);
  assert.equal(
    registry.resolve({pin: pin(), enabledProviderIds: [OPENCODE_PROVIDER_ID]}).id,
    OPENCODE_PROVIDER_ID);
  assert.throws(() => registry.resolve({
    providerId: OPENCODE_PROVIDER_ID,
    enabledProviderIds: [OPENCODE_PROVIDER_ID],
    pin: pin('codex'),
  }), /PIN_MISMATCH/);
  assert.throws(() => pin('OPENCODE'), /PROVIDER_ID_INVALID/);
});

test('adapter observations stay observation-only; UNKNOWN always carries a reason',
  async () => {
    const provider = makeProvider();
    const reconcileDefault = await provider.reconcile(pin());
    assert.deepEqual(reconcileDefault,
      {state: 'UNKNOWN', reason: 'NOT_RECONCILED', evidence: null});
    const cancelIdle = await provider.cancel({...pin(), attemptId: 'attempt:264:missing'});
    assert.equal(cancelIdle.state, 'NOT_RUNNING');
    assert.throws(() => normalizeProviderObservation('reconcile', {state: 'UNKNOWN'}),
      /UNKNOWN_REASON_REQUIRED/);
    for (const field of ['taskPass', 'releasePass', 'claimAccepted', 'authorized',
      'candidateCommit']) {
      assert.throws(() =>
        normalizeProviderObservation('execute', {state: 'COMPLETED', [field]: true}),
      /UNEXPECTED_FIELD/);
    }
  });

test('injectable attempt records flow through reconcile under the same normalization',
  async () => {
    const records = new Map([
      ['attempt:264:r1', {state: 'FAILED', reason: 'EXIT_NONZERO', evidence: {exitCode: 7}}],
    ]);
    const provider = makeProvider({attemptRecords: records});
    const obs = await provider.reconcile(pin());
    assert.equal(obs.state, 'FAILED');
    assert.equal(obs.reason, 'EXIT_NONZERO');
    const bad = makeProvider({
      attemptRecords: new Map([['attempt:264:r1', {state: 'SUCCESS'}]]),
    });
    await assert.rejects(() => bad.reconcile(pin()), /STATE_INVALID/);
  });

test('a lying execute wrapper is rejected by the contract, not by adapter goodwill',
  async () => {
    const liar = defineAgentProvider({
      id: OPENCODE_PROVIDER_ID, contractVersion: 1, capabilities: [],
      probe: async () => ({state: 'AVAILABLE'}),
      execute: async () => ({state: 'COMPLETED', taskPass: true}),
      cancel: async () => ({state: 'CANCEL_REQUESTED'}),
      reconcile: async () => ({state: 'UNKNOWN', reason: 'NOT_RECONCILED'}),
    });
    // Hardened AgentProvider/1 boundary: a bare/unvalidated execute request is
    // rejected with a TypeError before any adapter code runs.
    await assert.rejects(() => liar.execute(),
      {name: 'TypeError', message: 'PROVIDER_REQUEST_INVALID'});
    // With a legal request the lie still cannot pass: output normalization
    // rejects the unexpected field, not adapter goodwill.
    const legalRequest = {
      taskId: '264', attemptId: 'attempt:264:r1', sourceSha: 'a'.repeat(40),
      prompt: 'x', worktreePath: '/tmp/ws-264',
    };
    await assert.rejects(() => liar.execute(legalRequest), /UNEXPECTED_FIELD/);
  });

test('factory rejects unsafe configuration before any process work', () => {
  const base = {cliPath: process.execPath, cliArgs: [FAKE_CLI]};
  assert.throws(() => createOpenCodeAgentProvider({}), /CLI_PATH_REQUIRED/);
  assert.throws(() => makeProvider({cliPath: ''}), /CLI_PATH_REQUIRED/);
  assert.throws(() => makeProvider({cliArgs: 'not-an-array'}), /CLI_ARGS_INVALID/);
  assert.throws(() => makeProvider({model: 'no-slash'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({model: 'a/b/c'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({model: '/leading'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({model: 'trailing/'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({model: 'a/b c'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({model: 'a/b\tc'}), /MODEL_FLAG/);
  assert.throws(() => makeProvider({agent: 'bad agent'}), /AGENT_FLAG/);
  assert.throws(() => makeProvider({agent: 'a/b'}), /AGENT_FLAG/);
  assert.throws(() => makeProvider({agent: "'quote'"}), /AGENT_FLAG/);
  assert.throws(() => makeProvider({pinnedVersion: ''}), /PINNED_VERSION_INVALID/);
  assert.throws(() => makeProvider({attemptRecords: {}}), /ATTEMPT_RECORDS_INVALID/);
});
