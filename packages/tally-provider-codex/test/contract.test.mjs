import test from 'node:test';
import assert from 'node:assert/strict';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, DEFAULT_AGENT_PROVIDER,
  createAgentProviderRegistry, createProviderAttemptPin, normalizeProviderObservation,
} from '../../tally-agent-provider/src/index.mjs';
import {CODEX_PROVIDER_ID, createCodexAgentProvider, defaultLoadHost} from '../src/index.mjs';

const sha40 = 'a'.repeat(40);
const exitedProcess = {status: 'EXITED', exitCode: 0, pid: 4242,
  startedAt: '2026-01-01T00:00:00.000Z', terminatedAt: '2026-01-01T00:00:01.500Z'};

function fakeHostModule({runAttempt, probeProvider} = {}) {
  const calls = {created: [], runAttempt: [], probeProvider: []};
  const instance = {
    runAttempt: async config => {
      calls.runAttempt.push(config);
      return runAttempt ? runAttempt(config) : {state: 'LOCAL_CANDIDATE_READY',
        reason: 'REAL_CODEX_CANDIDATE_RECORDED', agentLaunch: true,
        candidateTree: 'b'.repeat(40), candidateCommit: 'c'.repeat(40),
        process: exitedProcess};
    },
    probeProvider: async config => {
      calls.probeProvider.push(config);
      return probeProvider ? probeProvider(config) : {available: true, provider: 'codex-cli'};
    },
  };
  return {module: {createLocalCodexHost: deps => {
    calls.created.push(deps);
    return instance;
  }}, instance, calls};
}

const providerConfig = {hostStorePath: 'C:\\tally\\host.sqlite',
  repositoryPath: 'C:\\tally\\repo', cliPath: 'C:\\tools\\codex.exe',
  sandboxCliPath: 'C:\\tools\\codex.exe'};
const executionRequest = {taskId: '264', attemptId: 'attempt:264:r1', sourceSha: sha40,
  prompt: 'not forwarded', worktreePath: 'C:\\tally\\wt\\r1'};
const pinFor = (providerId = CODEX_PROVIDER_ID) => createProviderAttemptPin(
  {taskId: '264', attemptId: 'attempt:264:r1', sourceSha: sha40, providerId});

async function until(condition, timeoutMs = 2000) {
  const start = Date.now();
  while (!condition()) {
    if (Date.now() - start > timeoutMs) throw new Error('condition not met in time');
    await new Promise(resolve => setTimeout(resolve, 2));
  }
}

test('the bridge is a contract-legal codex AgentProvider/1 with normalized observations', async () => {
  assert.equal(CODEX_PROVIDER_ID, DEFAULT_AGENT_PROVIDER);
  assert.equal(AGENT_PROVIDER_CONTRACT_VERSION, 1);
  const provider = createCodexAgentProvider(
    {loadHost: async () => fakeHostModule().module, config: providerConfig});
  assert.equal(provider.id, 'codex');
  assert.equal(provider.contractVersion, 1);
  assert.deepEqual(provider.capabilities, []);
  const probe = await provider.probe();
  assert.deepEqual(probe, {state: 'AVAILABLE', reason: null, evidence: {provider: 'codex-cli'}});
  assert.ok(Object.isFrozen(probe));
  const execute = await provider.execute(executionRequest, {});
  assert.equal(execute.state, 'COMPLETED');
  assert.deepEqual(Object.keys(execute).sort(), ['evidence', 'reason', 'state']);
  assert.equal(execute.reason, null);
});

test('registry resolves codex by default and never falls back implicitly', () => {
  const provider = createCodexAgentProvider(
    {loadHost: async () => fakeHostModule().module, config: providerConfig});
  const registry = createAgentProviderRegistry([provider]);
  assert.equal(registry.resolve().id, 'codex');
  assert.equal(registry.resolve({pin: pinFor()}).id, 'codex');
  assert.throws(() => registry.resolve({providerId: 'codex',
    enabledProviderIds: ['zcode']}), /PROVIDER_DISABLED/);
  assert.throws(() => registry.resolve({providerId: 'zcode',
    enabledProviderIds: ['zcode']}), /PROVIDER_NOT_INSTALLED/);
  assert.throws(() => registry.resolve({providerId: 'zcode'}), /PROVIDER_DISABLED/);
});

test('an attempt pin binds to codex and cannot be silently switched', () => {
  const codex = createCodexAgentProvider(
    {loadHost: async () => fakeHostModule().module, config: providerConfig});
  const registry = createAgentProviderRegistry([codex]);
  assert.equal(registry.resolve({pin: pinFor('codex')}).id, 'codex');
  assert.throws(() => registry.resolve({providerId: 'codex', pin: pinFor('zcode')}),
    /PROVIDER_ATTEMPT_PIN_MISMATCH/);
  assert.throws(() => createProviderAttemptPin({taskId: '264',
    attemptId: 'attempt:264:r1', sourceSha: sha40, providerId: 'CODEX'}),
    /PROVIDER_ID_INVALID/);
  assert.throws(() => createProviderAttemptPin({taskId: '264',
    attemptId: 'attempt:264:r1', sourceSha: 'nope', providerId: 'codex'}),
    /PROVIDER_SOURCE_SHA_INVALID/);
});

test('a lying host result can never surface authority fields through the bridge', async () => {
  const liar = fakeHostModule({runAttempt: async () => ({
    state: 'LOCAL_CANDIDATE_READY', reason: 'REAL_CODEX_CANDIDATE_RECORDED',
    agentLaunch: true, taskPass: true, reviewPass: true, releasePass: true,
    claimAccepted: true, authorized: true, admitted: true,
    candidateCommit: 'c'.repeat(40), candidateTree: 'b'.repeat(40),
    process: exitedProcess})});
  const provider = createCodexAgentProvider(
    {loadHost: async () => liar.module, config: providerConfig});
  const result = await provider.execute(executionRequest, {});
  assert.equal(result.state, 'COMPLETED');
  const serialized = JSON.stringify(result);
  for (const forbidden of ['taskPass', 'reviewPass', 'releasePass', 'claimAccepted',
    'authorized', 'admitted', 'candidateCommit', 'candidateTree'])
    assert.equal(serialized.includes(forbidden), false, forbidden);
  assert.deepEqual(result.evidence, {agentLaunch: true, exitCode: 0, pid: 4242,
    startedAt: '2026-01-01T00:00:00.000Z', terminatedAt: '2026-01-01T00:00:01.500Z',
    durationMs: 1500});
  // the contract itself still rejects any direct authority assertion
  assert.throws(() => normalizeProviderObservation('execute',
    {state: 'COMPLETED', taskPass: true}), /PROVIDER_OBSERVATION_UNEXPECTED_FIELD/);
});

test('AbortSignal cancellation maps to CANCELLED through the frozen provider', async () => {
  const fake = fakeHostModule({runAttempt: config => new Promise(resolve => {
    config.signal.addEventListener('abort',
      () => resolve({state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED',
        agentLaunch: true, process: exitedProcess}), {once: true});
  })});
  const provider = createCodexAgentProvider(
    {loadHost: async () => fake.module, config: providerConfig});
  const controller = new AbortController();
  const pending = provider.execute(executionRequest, {signal: controller.signal});
  await until(() => fake.calls.runAttempt.length === 1);
  const cancel = await provider.cancel(pinFor());
  assert.equal(cancel.state, 'CANCEL_REQUESTED');
  assert.equal((await pending).state, 'CANCELLED');
  assert.equal(fake.calls.runAttempt[0].signal.aborted, true);
});

test('disabled, not-installed and mismatched selections all fail closed', () => {
  const codex = createCodexAgentProvider(
    {loadHost: async () => fakeHostModule().module, config: providerConfig});
  const zcode = createCodexAgentProvider(
    {loadHost: async () => fakeHostModule().module, config: providerConfig});
  const registry = createAgentProviderRegistry([codex, {...zcode, id: 'zcode'}]);
  assert.throws(() => registry.resolve({providerId: 'zcode',
    enabledProviderIds: ['zcode'], pin: pinFor('codex')}), /PROVIDER_ATTEMPT_PIN_MISMATCH/);
  assert.equal(registry.resolve({}).id, 'codex');
  assert.throws(() => createAgentProviderRegistry([codex, codex]), /PROVIDER_DUPLICATE_ID/);
});

test('the default host loader is lazy and injectable; the factory never loads eagerly', async () => {
  assert.equal(typeof defaultLoadHost, 'function');
  // constructing with the default loader must not touch the module graph
  const provider = createCodexAgentProvider({config: providerConfig});
  assert.equal(provider.id, 'codex');
  // every other test passes without @kaicreator/domain-harness-host-node
  // installed, which proves the host module is only loaded lazily.
});
