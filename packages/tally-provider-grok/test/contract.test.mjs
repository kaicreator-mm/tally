/**
 * Contract-compliance tests for the grok adapter against the
 * tally-agent-provider AgentProvider/1 contract (#473):
 *   - registry defaults never include grok (default DISABLED);
 *   - grok runs only behind an explicit enabledProviderIds + attempt pin;
 *   - observations are normalized: no authority fields, no state invention,
 *     UNKNOWN always carries a reason;
 *   - one Attempt pin cannot be silently rebound to another provider.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {
  AGENT_PROVIDER_CONTRACT_VERSION,
  DEFAULT_AGENT_PROVIDER,
  defineAgentProvider,
  createAgentProviderRegistry,
  createProviderAttemptPin,
  normalizeProviderObservation,
} from '../../tally-agent-provider/src/index.mjs';
import {
  createGrokAgentProvider,
  GROK_PROVIDER_ID,
  PINNED_GROK_VERSION,
} from '../src/index.mjs';

const FAKE_GROK_SCRIPT = fileURLToPath(new URL('./fake-grok.mjs', import.meta.url));

const good = (id = 'codex') => ({
  id, contractVersion: 1, capabilities: [],
  probe: async () => ({state: 'AVAILABLE'}),
  execute: async () => ({state: 'COMPLETED', evidence: {agentSummary: 'done'}}),
  cancel: async () => ({state: 'CANCEL_REQUESTED'}),
  reconcile: async () => ({state: 'UNKNOWN', reason: 'NOT_RECONCILED'}),
});

const grokProvider = () => createGrokAgentProvider({
  binPath: process.execPath,
  binArgs: [FAKE_GROK_SCRIPT],
});

const pin = (providerId = 'grok') => createProviderAttemptPin({
  taskId: '473', attemptId: 'attempt:473:r1', sourceSha: 'a'.repeat(40), providerId,
});

test('grok adapter satisfies the AgentProvider/1 shape: all four operations, honest empty capabilities', async () => {
  const provider = grokProvider();
  assert.equal(provider.id, GROK_PROVIDER_ID);
  assert.equal(provider.contractVersion, AGENT_PROVIDER_CONTRACT_VERSION);
  assert.deepEqual(provider.capabilities, []);
  for (const operation of ['probe', 'execute', 'cancel', 'reconcile']) {
    assert.equal(typeof provider[operation], 'function', `${operation} is mandatory`);
  }
  // reconcile is normalized (frozen, reason preserved, defaults filled).
  const reconciled = await provider.reconcile(pin());
  assert.deepEqual(reconciled, {state: 'UNKNOWN', reason: 'NOT_RECONCILED', evidence: null});
  assert.equal(Object.isFrozen(reconciled), true);
});

test('default DISABLED: the registry default is codex and grok resolves only when explicitly enabled', () => {
  const registry = createAgentProviderRegistry([good(), grokProvider()]);
  assert.equal(DEFAULT_AGENT_PROVIDER, 'codex');
  assert.equal(registry.resolve().id, 'codex');
  assert.ok(registry.list().some(p => p.id === GROK_PROVIDER_ID),
    'grok can be installed (opt-in) but is never the default');

  // Without explicit enablement: PROVIDER_DISABLED, regardless of install.
  assert.throws(() => registry.resolve({providerId: GROK_PROVIDER_ID}), /PROVIDER_DISABLED/);
  assert.throws(() => registry.resolve({pin: pin(GROK_PROVIDER_ID)}), /PROVIDER_DISABLED/);

  // Explicit enablement + matching pin: resolvable.
  const resolved = registry.resolve({
    providerId: GROK_PROVIDER_ID,
    enabledProviderIds: ['codex', GROK_PROVIDER_ID],
  });
  assert.equal(resolved.id, GROK_PROVIDER_ID);
  const viaPin = registry.resolve({
    pin: pin(GROK_PROVIDER_ID),
    enabledProviderIds: [GROK_PROVIDER_ID],
  });
  assert.equal(viaPin.id, GROK_PROVIDER_ID);
});

test('an Attempt pin cannot be silently rebound between providers', () => {
  const registry = createAgentProviderRegistry([good(), grokProvider()]);
  assert.throws(
    () => registry.resolve({
      providerId: 'codex',
      pin: pin(GROK_PROVIDER_ID),
      enabledProviderIds: ['codex', GROK_PROVIDER_ID],
    }),
    /PROVIDER_ATTEMPT_PIN_MISMATCH/);
  // Pin alone does not enable: no explicit list -> still disabled.
  assert.throws(
    () => registry.resolve({pin: pin('codex'), enabledProviderIds: [GROK_PROVIDER_ID]}),
    /PROVIDER_DISABLED|PIN_MISMATCH|NOT_INSTALLED/);
});

test('pin probing works through the registry path: UNAVAILABLE without credentials, never a fake AVAILABLE', async () => {
  const registry = createAgentProviderRegistry([grokProvider()]);
  const provider = registry.resolve({
    providerId: GROK_PROVIDER_ID,
    enabledProviderIds: [GROK_PROVIDER_ID],
    pin: pin(GROK_PROVIDER_ID),
  });
  const observation = await provider.probe({env: {}});
  assert.equal(observation.state, 'UNAVAILABLE');
  assert.equal(observation.reason, 'GROK_NOT_AUTHENTICATED');
});

test('provider observations cannot assert authority: the contract refuses liar fields on every adapter path', async () => {
  for (const field of ['taskPass', 'releasePass', 'claimAccepted', 'authorized', 'candidateCommit']) {
    assert.throws(
      () => normalizeProviderObservation('execute', {state: 'COMPLETED', [field]: true}),
      /UNEXPECTED_FIELD/);
  }
  // The adapter itself is built through defineAgentProvider, so even a
  // hostile/broken internal path cannot emit an un-normalized observation.
  const liar = defineAgentProvider({
    ...good(), id: GROK_PROVIDER_ID,
    execute: async () => ({state: 'COMPLETED', taskPass: true}),
  });
  // Contract v1 boundary: execute now requires a validated request shape
  // {taskId, attemptId, sourceSha, prompt, worktreePath}; a legal request
  // passes the boundary and the hostile internal path is still refused by
  // the observation normalizer (UNEXPECTED_FIELD), which is the point here.
  await assert.rejects(
    () => liar.execute({taskId: '473', attemptId: 'attempt:473:r1',
      sourceSha: 'a'.repeat(40), prompt: 'add a test', worktreePath: '/tmp/ws'}),
    /UNEXPECTED_FIELD/);
  assert.throws(
    () => normalizeProviderObservation('probe', {state: 'UNKNOWN'}),
    /UNKNOWN_REASON_REQUIRED/);
});

test('UNKNOWN always carries a reason across adapter surfaces', async () => {
  const provider = grokProvider();
  const reconciled = await provider.reconcile(pin());
  assert.equal(reconciled.state, 'UNKNOWN');
  assert.match(reconciled.reason ?? '', /^[A-Z][A-Z0-9_]{0,95}$/);
  // Sanity: pinned version constant stays the #468 anchor.
  assert.equal(PINNED_GROK_VERSION, '1.0.87');
});
