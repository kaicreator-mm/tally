import test from 'node:test';
import assert from 'node:assert/strict';
import {
  BRIDGE_REASONS, CODEX_PROVIDER_ID, EXECUTE_STATE_MAP, RECONCILE_STATE_MAP,
  createCodexBridge, mapAttemptOutcome, mapProbeOutcome, redactExecutionEvidence,
  sanitizeReason,
} from '../src/codex-bridge.mjs';

const sha40 = 'a'.repeat(40);
const exitedProcess = {status: 'EXITED', exitCode: 0, pid: 4242,
  startedAt: '2026-01-01T00:00:00.000Z', terminatedAt: '2026-01-01T00:00:01.500Z'};
const readyResult = {state: 'LOCAL_CANDIDATE_READY', reason: 'REAL_CODEX_CANDIDATE_RECORDED',
  agentLaunch: true, candidateTree: 'b'.repeat(40), candidateCommit: 'c'.repeat(40),
  candidateRef: 'refs/tally-candidates/attempt:264:r1', process: exitedProcess};

function fakeHostModule({runAttempt, probeProvider} = {}) {
  const calls = {created: [], runAttempt: [], probeProvider: []};
  const instance = {
    runAttempt: async config => {
      calls.runAttempt.push(config);
      return runAttempt ? runAttempt(config, calls) : readyResult;
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

const bridgeConfig = {hostStorePath: 'C:\\tally\\host.sqlite', repositoryPath: 'C:\\tally\\repo',
  cliPath: 'C:\\tools\\codex.exe', sandboxCliPath: 'C:\\tools\\codex.exe'};
const request = (overrides = {}) => ({taskId: '264', attemptId: 'attempt:264:r1',
  sourceSha: sha40, prompt: 'bridge must ignore this', worktreePath: 'C:\\tally\\wt\\r1',
  ...overrides});
const pin = (overrides = {}) => ({taskId: '264', attemptId: 'attempt:264:r1',
  sourceSha: sha40, providerId: CODEX_PROVIDER_ID, ...overrides});

async function until(condition, timeoutMs = 2000) {
  const start = Date.now();
  while (!condition()) {
    if (Date.now() - start > timeoutMs) throw new Error('condition not met in time');
    await new Promise(resolve => setTimeout(resolve, 2));
  }
}

test('state mapping tables match the documented host vocabulary exactly', () => {
  assert.deepEqual(EXECUTE_STATE_MAP, {
    LOCAL_CANDIDATE_READY: 'COMPLETED',
    LOCAL_ATTEMPT_FAILED: 'FAILED',
    LOCAL_ATTEMPT_CANCELLED: 'CANCELLED',
    LOCAL_ATTEMPT_UNKNOWN: 'UNKNOWN',
    AGENT_NOT_EXECUTED: 'FAILED',
    NOT_ADMITTED: 'FAILED',
  });
  assert.deepEqual(RECONCILE_STATE_MAP, {
    LOCAL_CANDIDATE_READY: 'COMPLETED',
    LOCAL_ATTEMPT_FAILED: 'FAILED',
    LOCAL_ATTEMPT_CANCELLED: 'CANCELLED',
    LOCAL_ATTEMPT_UNKNOWN: 'UNKNOWN',
    AGENT_NOT_EXECUTED: 'FAILED',
    NOT_ADMITTED: 'UNKNOWN',
  });
});

test('every host attempt terminal state maps positively for execute', () => {
  const cases = [
    [readyResult, 'COMPLETED', null],
    [{state: 'LOCAL_ATTEMPT_FAILED', reason: 'WRITE_SET_VIOLATION', agentLaunch: true,
      process: {...exitedProcess, exitCode: 7}}, 'FAILED', 'WRITE_SET_VIOLATION'],
    [{state: 'LOCAL_ATTEMPT_FAILED', reason: 'ATTEMPT_PROCESS_FAILED', agentLaunch: true,
      process: exitedProcess}, 'FAILED', 'ATTEMPT_PROCESS_FAILED'],
    [{state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED', agentLaunch: true,
      process: exitedProcess}, 'CANCELLED', 'ATTEMPT_CANCELLED'],
    [{state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'ATTEMPT_TIMEOUT_RECONCILED', agentLaunch: true,
      process: {status: 'UNKNOWN', exitCode: null, pid: null}},
      'UNKNOWN', 'ATTEMPT_TIMEOUT_RECONCILED'],
    [{state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'ATTEMPT_STATUS_UNKNOWN', agentLaunch: true,
      process: {status: 'RUNNING', pid: 99}},
      'UNKNOWN', 'ATTEMPT_STATUS_UNKNOWN'],
    [{state: 'AGENT_NOT_EXECUTED', reason: 'LOCAL_AGENT_UNAVAILABLE', agentLaunch: false,
      process: null}, 'FAILED', 'LOCAL_AGENT_UNAVAILABLE'],
    [{state: 'AGENT_NOT_EXECUTED', reason: 'BUILDER_CONFINEMENT_UNAVAILABLE',
      agentLaunch: false, process: null}, 'FAILED', 'BUILDER_CONFINEMENT_UNAVAILABLE'],
    [{state: 'NOT_ADMITTED', reason: 'CANDIDATE_WORKTREE_OVERLAP', agentLaunch: false,
      process: null}, 'FAILED', 'CANDIDATE_WORKTREE_OVERLAP'],
    [{state: 'NOT_ADMITTED', reason: 'CANDIDATE_WRITE_GRANT_OVERLAP', agentLaunch: false,
      process: null}, 'FAILED', 'CANDIDATE_WRITE_GRANT_OVERLAP'],
  ];
  for (const [result, expectedState, expectedReason] of cases) {
    const mapped = mapAttemptOutcome('execute', result);
    assert.equal(mapped.state, expectedState, result.state);
    assert.equal(mapped.reason, expectedReason, result.state);
  }
});

test('unknown host terminal states fail closed to UNKNOWN with MAPPING_MISS reason', () => {
  for (const hostState of ['SOMETHING_NEW', 'REVIEW_JOINED', 'STARTING', undefined]) {
    const mapped = mapAttemptOutcome('execute', {state: hostState, reason: 'FUTURE_STATE'});
    assert.equal(mapped.state, 'UNKNOWN', String(hostState));
    assert.equal(mapped.reason, BRIDGE_REASONS.HOST_STATE_MAPPING_MISS, String(hostState));
  }
});

test('reconcile mapping refines RUNNING, fails closed on NOT_ADMITTED, never fabricates terminals', () => {
  const cases = [
    [readyResult, 'COMPLETED', null],
    [{state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'ATTEMPT_STATUS_UNKNOWN', agentLaunch: true,
      process: {status: 'RUNNING', pid: 99, startedAt: '2026-01-01T00:00:00.000Z'}},
      'RUNNING', 'ATTEMPT_STATUS_UNKNOWN'],
    [{state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'ATTEMPT_STATUS_UNKNOWN', agentLaunch: true,
      process: {status: 'EXITED', exitCode: 1}}, 'UNKNOWN', 'ATTEMPT_STATUS_UNKNOWN'],
    [{state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'ATTEMPT_STATUS_UNKNOWN', agentLaunch: true,
      process: {status: 'UNKNOWN'}}, 'UNKNOWN', 'ATTEMPT_STATUS_UNKNOWN'],
    [{state: 'LOCAL_ATTEMPT_FAILED', reason: 'ATTEMPT_PROCESS_FAILED', agentLaunch: true,
      process: exitedProcess}, 'FAILED', 'ATTEMPT_PROCESS_FAILED'],
    [{state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED', agentLaunch: true,
      process: exitedProcess}, 'CANCELLED', 'ATTEMPT_CANCELLED'],
    [{state: 'AGENT_NOT_EXECUTED', reason: 'HIDDEN_VERIFIER_UNAVAILABLE', agentLaunch: false,
      process: null}, 'FAILED', 'HIDDEN_VERIFIER_UNAVAILABLE'],
    [{state: 'NOT_ADMITTED', reason: 'CANDIDATE_WORKTREE_OVERLAP', agentLaunch: false,
      process: null}, 'UNKNOWN', 'CANDIDATE_WORKTREE_OVERLAP'],
  ];
  for (const [result, expectedState, expectedReason] of cases) {
    const mapped = mapAttemptOutcome('reconcile', result);
    assert.equal(mapped.state, expectedState, `${result.state}/${result.process?.status}`);
    assert.equal(mapped.reason, expectedReason, `${result.state}/${result.process?.status}`);
  }
});

test('unusable host reasons degrade safely: UNKNOWN gets a reason, FAILED may drop it', () => {
  const unsafe = mapAttemptOutcome('execute',
    {state: 'LOCAL_ATTEMPT_UNKNOWN', reason: 'runtime oops', process: null});
  assert.equal(unsafe.state, 'UNKNOWN');
  assert.equal(unsafe.reason, BRIDGE_REASONS.HOST_REASON_UNSAFE);
  const failedUnsafe = mapAttemptOutcome('execute',
    {state: 'LOCAL_ATTEMPT_FAILED', reason: 'unexpected detail: x=1', process: exitedProcess});
  assert.equal(failedUnsafe.state, 'FAILED');
  assert.equal(failedUnsafe.reason, null);
  assert.equal(sanitizeReason('WRITE_SET_VIOLATION', 'X'), 'WRITE_SET_VIOLATION');
  assert.equal(sanitizeReason('nope', 'FALLBACK_REASON'), 'FALLBACK_REASON');
  assert.equal(sanitizeReason(7, 'FALLBACK_REASON'), 'FALLBACK_REASON');
});

test('execute evidence is redacted to launch/exit/session/duration only', () => {
  const mapped = mapAttemptOutcome('execute', readyResult);
  assert.deepEqual(mapped.evidence, {agentLaunch: true, exitCode: 0, pid: 4242,
    startedAt: '2026-01-01T00:00:00.000Z', terminatedAt: '2026-01-01T00:00:01.500Z',
    durationMs: 1500});
  const serialized = JSON.stringify(mapped);
  for (const forbidden of ['candidateCommit', 'candidateTree', 'candidateRef', 'taskPass',
    'reviewPass', 'releasePass', 'claimAccepted', 'builderSid', 'builderPrincipal'])
    assert.equal(serialized.includes(forbidden), false, forbidden);
  assert.equal(redactExecutionEvidence({state: 'NOT_ADMITTED', process: null}), null);
  assert.equal(redactExecutionEvidence(undefined), null);
  assert.deepEqual(redactExecutionEvidence({state: 'LOCAL_ATTEMPT_UNKNOWN',
    agentLaunch: true, process: {status: 'UNKNOWN', pid: null, exitCode: null}}),
    {agentLaunch: true});
});

test('probe mapping: available, unavailable with stage/exitCode, malformed', () => {
  const available = mapProbeOutcome({available: true, provider: 'codex-cli'});
  assert.deepEqual(available, {state: 'AVAILABLE', reason: null,
    evidence: {provider: 'codex-cli'}});
  const unavailable = mapProbeOutcome({available: false, reason: 'LOCAL_AGENT_UNAVAILABLE',
    stage: 'AUTH', exitCode: 1, outputMatched: false});
  assert.deepEqual(unavailable, {state: 'UNAVAILABLE', reason: 'LOCAL_AGENT_UNAVAILABLE',
    evidence: {stage: 'AUTH', exitCode: 1}});
  const unavailableUnsafe = mapProbeOutcome({available: false, reason: 'not-a-code'});
  assert.equal(unavailableUnsafe.state, 'UNAVAILABLE');
  assert.equal(unavailableUnsafe.reason, BRIDGE_REASONS.PROBE_UNAVAILABLE);
  const malformed = mapProbeOutcome({hello: true});
  assert.deepEqual(malformed, {state: 'UNKNOWN',
    reason: BRIDGE_REASONS.PROBE_OBSERVATION_INVALID, evidence: null});
  assert.equal(mapProbeOutcome(null).state, 'UNKNOWN');
});

test('probe delegates to host probeProvider and forwards the factory clock', async () => {
  const fake = fakeHostModule({probeProvider: async () =>
    ({available: false, reason: 'LOCAL_AGENT_UNAVAILABLE', stage: 'VERSION', exitCode: 1})});
  const now = () => Date.now();
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig, now});
  const probe = await bridge.probe();
  assert.equal(probe.state, 'UNAVAILABLE');
  assert.equal(probe.reason, 'LOCAL_AGENT_UNAVAILABLE');
  assert.deepEqual(probe.evidence, {stage: 'VERSION', exitCode: 1});
  assert.deepEqual(fake.calls.created, [{now}]);
  assert.equal(fake.calls.probeProvider.length, 1);
  assert.equal(fake.calls.probeProvider[0], bridgeConfig);
});

test('lazy loadHost failure degrades to HOST_MODULE_UNAVAILABLE per operation', async () => {
  const rejecting = () => Promise.reject(
    Object.assign(new Error("Cannot find package '@kaicreator/domain-harness-host-node'"),
      {code: 'ERR_MODULE_NOT_FOUND'}));
  const bridge = createCodexBridge({loadHost: rejecting, config: bridgeConfig});
  const probe = await bridge.probe();
  assert.deepEqual([probe.state, probe.reason], ['UNAVAILABLE', 'HOST_MODULE_UNAVAILABLE']);
  const execute = await bridge.execute(request(), {});
  assert.deepEqual([execute.state, execute.reason], ['UNKNOWN', 'HOST_MODULE_UNAVAILABLE']);
  const reconcile = await bridge.reconcile(pin(), {attemptConfig:
    {...bridgeConfig, attemptId: 'attempt:264:r1', sourceSha: sha40}});
  assert.deepEqual([reconcile.state, reconcile.reason], ['UNKNOWN', 'HOST_MODULE_UNAVAILABLE']);
  // without a replayable attempt config, reconcile fails closed on its own terms
  const untrackable = await bridge.reconcile(pin(), {});
  assert.deepEqual([untrackable.state, untrackable.reason],
    ['UNKNOWN', 'RECONCILE_STATE_UNAVAILABLE']);
  // a rejected load resets the memo, so a later healthy module can be loaded
  const fake = fakeHostModule();
  let loads = 0;
  const recovered = createCodexBridge({loadHost: async () => {
    if (loads++ === 0) return rejecting();
    return fake.module;
  }, config: bridgeConfig});
  assert.equal((await recovered.probe()).reason, 'HOST_MODULE_UNAVAILABLE');
  assert.equal((await recovered.probe()).state, 'AVAILABLE');
});

test('host module without createLocalCodexHost fails closed as HOST_MODULE_UNAVAILABLE', async () => {
  const bridge = createCodexBridge({loadHost: async () => ({}), config: bridgeConfig});
  const probe = await bridge.probe();
  assert.deepEqual([probe.state, probe.reason], ['UNAVAILABLE', 'HOST_MODULE_UNAVAILABLE']);
});

test('missing factory config fails closed before touching the host module', async () => {
  let loaded = false;
  const bridge = createCodexBridge({loadHost: async () => { loaded = true;
    return fakeHostModule().module; }, config: null});
  const probe = await bridge.probe();
  assert.deepEqual([probe.state, probe.reason], ['UNAVAILABLE', 'PROVIDER_CONFIG_UNAVAILABLE']);
  const execute = await bridge.execute(request(), {});
  assert.deepEqual([execute.state, execute.reason], ['UNKNOWN', 'PROVIDER_CONFIG_UNAVAILABLE']);
  assert.equal(loaded, false);
});

test('execute adapts the request onto the static config and never forwards the prompt', async () => {
  const fake = fakeHostModule();
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const completed = await bridge.execute(request({timeoutMs: 1234}), {env: {SECRET: 'x'}});
  assert.equal(completed.state, 'COMPLETED');
  const sent = fake.calls.runAttempt[0];
  assert.equal(sent.attemptId, 'attempt:264:r1');
  assert.equal(sent.sourceSha, sha40);
  assert.equal(sent.worktreePath, 'C:\\tally\\wt\\r1');
  assert.equal(sent.timeoutMs, 1234);
  assert.equal(sent.cliPath, bridgeConfig.cliPath);
  assert.equal(sent.hostStorePath, bridgeConfig.hostStorePath);
  assert.equal('prompt' in sent, false);
  assert.equal('env' in sent, false);
  assert.equal(typeof sent.signal.addEventListener, 'function');
  assert.equal(fake.calls.runAttempt.length, 1);
});

test('execute throw with a host error code keeps the code as the reason', async () => {
  const fake = fakeHostModule({runAttempt: async () => {
    throw Object.assign(new Error('denied'), {code: 'CANDIDATE_WORKTREE_ALREADY_EXISTS'});
  }});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const unknown = await bridge.execute(request(), {});
  assert.deepEqual([unknown.state, unknown.reason],
    ['UNKNOWN', 'CANDIDATE_WORKTREE_ALREADY_EXISTS']);
  const generic = fakeHostModule({runAttempt: async () => { throw new Error('boom'); }});
  const bridge2 = createCodexBridge({loadHost: async () => generic.module,
    config: bridgeConfig});
  const unknown2 = await bridge2.execute(request(), {});
  assert.deepEqual([unknown2.state, unknown2.reason],
    ['UNKNOWN', BRIDGE_REASONS.HOST_EXECUTION_ERROR]);
});

test('an already-aborted context signal cancels before launch without invoking the host', async () => {
  const fake = fakeHostModule({runAttempt: async config =>
    ({state: 'LOCAL_CANDIDATE_READY', reason: 'REAL_CODEX_CANDIDATE_RECORDED',
      agentLaunch: true, process: exitedProcess})});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const controller = new AbortController();
  controller.abort();
  const cancelled = await bridge.execute(request(), {signal: controller.signal});
  assert.deepEqual([cancelled.state, cancelled.reason],
    ['CANCELLED', BRIDGE_REASONS.CANCELLED_BEFORE_LAUNCH]);
  assert.equal(fake.calls.runAttempt.length, 0);
  const reconcile = await bridge.reconcile(pin(), {});
  assert.deepEqual([reconcile.state, reconcile.reason],
    ['UNKNOWN', BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE]);
});

test('cancel aborts the in-flight attempt: CANCEL_REQUESTED then execute CANCELLED', async () => {
  let seenAborted = null;
  const fake = fakeHostModule({runAttempt: config => new Promise(resolve => {
    config.signal.addEventListener('abort', () => {
      seenAborted = config.signal.aborted;
      resolve({state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED',
        agentLaunch: true, process: exitedProcess});
    }, {once: true});
  })});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const pending = bridge.execute(request(), {});
  await until(() => fake.calls.runAttempt.length === 1);
  const cancel = await bridge.cancel(pin());
  assert.deepEqual([cancel.state, cancel.reason, cancel.evidence],
    ['CANCEL_REQUESTED', null, {attemptId: 'attempt:264:r1'}]);
  const execute = await pending;
  assert.deepEqual([execute.state, execute.reason], ['CANCELLED', 'ATTEMPT_CANCELLED']);
  assert.equal(seenAborted, true);
});

test('cancel before the host launch records CANCELLED_BEFORE_LAUNCH without launching', async () => {
  const fake = fakeHostModule({runAttempt: config => new Promise(resolve => {
    config.signal.addEventListener('abort', () => resolve(
      {state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED', agentLaunch: true,
        process: exitedProcess}), {once: true});
  })});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const pending = bridge.execute(request(), {});
  const cancel = await bridge.cancel(pin());
  assert.deepEqual([cancel.state, cancel.reason, cancel.evidence],
    ['CANCEL_REQUESTED', null, {attemptId: 'attempt:264:r1'}]);
  const execute = await pending;
  assert.deepEqual([execute.state, execute.reason],
    ['CANCELLED', BRIDGE_REASONS.CANCELLED_BEFORE_LAUNCH]);
  assert.equal(fake.calls.runAttempt.length, 0);
  // no launch means no durable record: reconcile must not pretend one exists
  const reconcile = await bridge.reconcile(pin(), {});
  assert.deepEqual([reconcile.state, reconcile.reason],
    ['UNKNOWN', BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE]);
});

test('cancel with no in-flight attempt reports NOT_RUNNING without guessing', async () => {
  const fake = fakeHostModule();
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const cancel = await bridge.cancel(pin());
  assert.deepEqual([cancel.state, cancel.reason, cancel.evidence],
    ['NOT_RUNNING', BRIDGE_REASONS.NO_INFLIGHT_ATTEMPT, {attemptId: 'attempt:264:r1'}]);
  const after = await bridge.execute(request(), {});
  assert.equal(after.state, 'COMPLETED');
  const done = await bridge.cancel(pin());
  assert.equal(done.state, 'NOT_RUNNING');
});

test('cancel with a mismatched pin or malformed pin fails closed without aborting', async () => {
  const fake = fakeHostModule({runAttempt: config => new Promise(resolve => {
    config.signal.addEventListener('abort',
      () => resolve({state: 'LOCAL_ATTEMPT_CANCELLED', reason: 'ATTEMPT_CANCELLED',
        agentLaunch: true, process: exitedProcess}), {once: true});
  })});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const pending = bridge.execute(request(), {});
  await until(() => fake.calls.runAttempt.length === 1);
  const mismatched = await bridge.cancel(pin({sourceSha: 'b'.repeat(40)}));
  assert.deepEqual([mismatched.state, mismatched.reason],
    ['UNKNOWN', BRIDGE_REASONS.ATTEMPT_PIN_MISMATCH]);
  const wrongProvider = await bridge.cancel(pin({providerId: 'zcode'}));
  assert.deepEqual([wrongProvider.state, wrongProvider.reason],
    ['UNKNOWN', BRIDGE_REASONS.ATTEMPT_PIN_MISMATCH]);
  const malformed = await bridge.cancel({attemptId: 'attempt:264:r1'});
  assert.deepEqual([malformed.state, malformed.reason],
    ['UNKNOWN', BRIDGE_REASONS.ATTEMPT_PIN_INVALID]);
  const cancel = await bridge.cancel(pin());
  assert.equal(cancel.state, 'CANCEL_REQUESTED');
  assert.equal((await pending).state, 'CANCELLED');
});

test('reconcile replays through the host with the exact recorded attempt config', async () => {
  const results = [readyResult, {state: 'LOCAL_ATTEMPT_UNKNOWN',
    reason: 'ATTEMPT_STATUS_UNKNOWN', agentLaunch: true,
    process: {status: 'RUNNING', pid: 4242, startedAt: '2026-01-01T00:00:00.000Z'}}];
  let call = 0;
  const fake = fakeHostModule({runAttempt: async () => results[Math.min(call++, results.length - 1)]});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const reconciled = await bridge.reconcile(pin(), {});
  assert.deepEqual([reconciled.state, reconciled.reason],
    ['UNKNOWN', BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE]);
  assert.equal(fake.calls.runAttempt.length, 0);
  const completed = await bridge.execute(request(), {});
  assert.equal(completed.state, 'COMPLETED');
  const running = await bridge.reconcile(pin(), {});
  assert.deepEqual([running.state, running.reason], ['RUNNING', 'ATTEMPT_STATUS_UNKNOWN']);
  assert.equal(fake.calls.runAttempt.length, 2);
  assert.equal(fake.calls.runAttempt[1].attemptId, 'attempt:264:r1');
  assert.deepEqual(fake.calls.runAttempt[1], fake.calls.runAttempt[0]);
  const failed = await bridge.reconcile(pin({sourceSha: 'f'.repeat(40)}), {});
  assert.deepEqual([failed.state, failed.reason],
    ['UNKNOWN', BRIDGE_REASONS.ATTEMPT_PIN_MISMATCH]);
});

test('reconcile accepts an attemptConfig supplied by the trusted Host context', async () => {
  const fake = fakeHostModule({runAttempt: async () => readyResult});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const attemptConfig = {...bridgeConfig, attemptId: 'attempt:264:r1', sourceSha: sha40,
    worktreePath: 'C:\\tally\\wt\\r1'};
  const completed = await bridge.reconcile(pin(), {attemptConfig});
  assert.equal(completed.state, 'COMPLETED');
  assert.equal(fake.calls.runAttempt.length, 1);
  assert.equal(fake.calls.runAttempt[0], attemptConfig);
  const wrongSha = await bridge.reconcile(pin(), {attemptConfig:
    {...attemptConfig, sourceSha: 'e'.repeat(40)}});
  assert.deepEqual([wrongSha.state, wrongSha.reason],
    ['UNKNOWN', BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE]);
  assert.equal(fake.calls.runAttempt.length, 1);
});

test('reconcile replay of a terminal failure maps FAILED; NOT_ADMITTED replays as UNKNOWN', async () => {
  const results = [readyResult, {state: 'LOCAL_ATTEMPT_FAILED',
    reason: 'ATTEMPT_PROCESS_FAILED', agentLaunch: true, process: exitedProcess}];
  let call = 0;
  const fake = fakeHostModule({runAttempt: async () => results[Math.min(call++, results.length - 1)]});
  const bridge = createCodexBridge({loadHost: async () => fake.module, config: bridgeConfig});
  const executed = await bridge.execute(request(), {});
  assert.equal(executed.state, 'COMPLETED');
  const failed = await bridge.reconcile(pin(), {});
  assert.deepEqual([failed.state, failed.reason], ['FAILED', 'ATTEMPT_PROCESS_FAILED']);
  const denied = fakeHostModule({runAttempt: async () => ({state: 'NOT_ADMITTED',
    reason: 'CANDIDATE_WRITE_GRANT_OVERLAP', agentLaunch: false, process: null})});
  const bridge2 = createCodexBridge({loadHost: async () => denied.module,
    config: bridgeConfig});
  await bridge2.execute(request(), {});
  const unknown = await bridge2.reconcile(pin(), {});
  assert.deepEqual([unknown.state, unknown.reason],
    ['UNKNOWN', 'CANDIDATE_WRITE_GRANT_OVERLAP']);
});
