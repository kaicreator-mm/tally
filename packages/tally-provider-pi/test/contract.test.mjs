/**
 * tally-provider-pi contract-compliance tests: registry/pin/disabled/no-
 * fallback/liar refusal and observation discipline, exercised with the pi
 * adapter mounted in the shared tally-agent-provider registry.
 */

import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtempSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider,
  createAgentProviderRegistry, createProviderAttemptPin,
  normalizeProviderObservation,
} from '../../tally-agent-provider/src/index.mjs';
import {createPiAgentProvider, PI_PROVIDER_ID, PINNED_PI_VERSION} from '../src/index.mjs';

const fakeScript = fileURLToPath(new URL('./fake-pi.mjs', import.meta.url));
const SOURCE_SHA = 'a'.repeat(40);

const codexStandIn = () => ({
  id: 'codex', contractVersion: 1, capabilities: [],
  probe: async () => ({state: 'AVAILABLE'}),
  execute: async () => ({state: 'COMPLETED', evidence: {}}),
  cancel: async () => ({state: 'CANCEL_REQUESTED'}),
  reconcile: async () => ({state: 'UNKNOWN', reason: 'NOT_RECONCILED'}),
});

function piProvider() {
  return createPiAgentProvider({
    binPath: process.execPath,
    programArgs: [fakeScript],
    timeoutMs: 15000,
    inheritedEnv: {PATH: process.env.PATH ?? '/usr/bin'},
  });
}

const pin = (providerId = PI_PROVIDER_ID) => createProviderAttemptPin({
  taskId: '473', attemptId: 'attempt:473:r1', sourceSha: SOURCE_SHA, providerId,
});

test('pi adapter satisfies the AgentProvider/1 contract shape', () => {
  const provider = piProvider();
  assert.equal(provider.id, PI_PROVIDER_ID);
  assert.equal(provider.contractVersion, AGENT_PROVIDER_CONTRACT_VERSION);
  assert.deepEqual(provider.capabilities, []); // honest empty capability set
  for (const operation of ['probe', 'execute', 'cancel', 'reconcile']) {
    assert.equal(typeof provider[operation], 'function');
  }
  assert.equal(typeof PINNED_PI_VERSION, 'string');
});

test('default stays codex; pi requires explicit enabled selection (no fallback)', () => {
  const registry = createAgentProviderRegistry([piProvider(), codexStandIn()]);
  assert.equal(registry.resolve().id, 'codex');
  assert.throws(() => registry.resolve({providerId: PI_PROVIDER_ID}), /PROVIDER_DISABLED/);
  assert.equal(
    registry.resolve({providerId: PI_PROVIDER_ID, enabledProviderIds: ['codex', PI_PROVIDER_ID]}).id,
    PI_PROVIDER_ID,
  );
  // A pin for pi never silently reroutes to the default provider.
  assert.throws(() => registry.resolve({pin: pin(), enabledProviderIds: ['codex']}),
    /PROVIDER_DISABLED/);
  assert.equal(
    registry.resolve({pin: pin(), enabledProviderIds: [PI_PROVIDER_ID]}).id,
    PI_PROVIDER_ID,
  );
});

test('a pin cannot be silently switched or rebound to another provider', () => {
  const registry = createAgentProviderRegistry([piProvider(), codexStandIn()]);
  assert.throws(() => registry.resolve({providerId: 'codex', pin: pin(PI_PROVIDER_ID)}),
    /PIN_MISMATCH/);
  // A codex-pinned attempt with only pi enabled is a hard error — never a
  // silent fallback to the installed provider.
  assert.throws(() => registry.resolve({pin: pin('codex'), enabledProviderIds: [PI_PROVIDER_ID]}),
    /PROVIDER_DISABLED/);
  assert.throws(() => pin('PI'), /PROVIDER_ID_INVALID/);
  assert.throws(() => createProviderAttemptPin({
    taskId: '473', attemptId: 'attempt:473:r1', sourceSha: 'nope', providerId: PI_PROVIDER_ID,
  }), /SOURCE_SHA/);
});

test('duplicate registration and uninstall edge refused', () => {
  assert.throws(() => createAgentProviderRegistry([piProvider(), piProvider()]), /DUPLICATE/);
  assert.throws(() => createAgentProviderRegistry([codexStandIn()])
    .resolve({providerId: PI_PROVIDER_ID, enabledProviderIds: [PI_PROVIDER_ID]}),
  /NOT_INSTALLED/);
});

test('provider observations cannot assert authority or a Task PASS', async () => {
  for (const field of ['taskPass', 'releasePass', 'claimAccepted', 'authorized', 'candidateCommit']) {
    assert.throws(() => normalizeProviderObservation('execute',
      {state: 'COMPLETED', [field]: true}), /UNEXPECTED_FIELD/);
  }
  const liar = defineAgentProvider({...codexStandIn(),
    execute: async () => ({state: 'COMPLETED', taskPass: true})});
  // Contract-legal request shape: the liar's OBSERVATION (not the input
  // boundary) is what must be refused here.
  await assert.rejects(
    () => liar.execute({
      taskId: '473', attemptId: 'attempt:473:r1', sourceSha: SOURCE_SHA,
      prompt: 'x', worktreePath: 'w',
    }),
    /UNEXPECTED_FIELD/);
  assert.throws(() => normalizeProviderObservation('execute', {state: 'UNKNOWN'}),
    /UNKNOWN_REASON_REQUIRED/);
  assert.throws(() => normalizeProviderObservation('execute', {state: 'SUCCESS'}),
    /STATE_INVALID/);
  assert.throws(() => normalizeProviderObservation('probe', {state: 'FAILED'}),
    /STATE_INVALID/);
});

test('the pi adapter observations normalize cleanly under the contract', async (t) => {
  const dir = mkdtempSync(path.join(tmpdir(), 'pi-prov-contract-'));
  t.after(() => { rmSync(dir, {recursive: true, force: true}); });
  const provider = piProvider();
  const worktree = path.join(dir, 'ws');
  const {mkdirSync} = await import('node:fs');
  mkdirSync(worktree);
  const probe = await provider.probe({env: {DEEPSEEK_API_KEY: 'contract-key'}});
  assert.equal(probe.state, 'AVAILABLE');
  const done = await provider.execute(
    {taskId: '473', attemptId: 'r1', sourceSha: SOURCE_SHA, prompt: 'x', worktreePath: worktree},
    {env: {DEEPSEEK_API_KEY: 'contract-key'}},
  );
  assert.equal(done.state, 'COMPLETED');
  assert.deepEqual(Object.keys(done).sort(), ['evidence', 'reason', 'state']);
  assert.equal(done.reason, null);
  const cancelled = await provider.execute(
    {taskId: '473', attemptId: 'r2', sourceSha: SOURCE_SHA, prompt: 'x', worktreePath: worktree},
    {env: {DEEPSEEK_API_KEY: 'contract-key'}, signal: AbortSignal.abort()},
  );
  assert.equal(cancelled.state, 'CANCELLED');
  assert.match(cancelled.reason, /^PI_/);
  const reconciled = await provider.reconcile(pin());
  assert.equal(reconciled.state, 'UNKNOWN');
  assert.equal(reconciled.reason, 'NOT_RECONCILED');
});
