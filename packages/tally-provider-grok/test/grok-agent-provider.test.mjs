/**
 * Behavioral tests for the grok AgentProvider/1 Node adapter, driven entirely
 * through the stand-in CLI (test/fake-grok.mjs) at a REAL process boundary -
 * the same discipline as #468 runtimes/grok (no in-memory mocks): argv
 * assembly, spawn env allowlist, GROK_HOME isolation, exit-code mapping,
 * whole-tree teardown are all really exercised.
 *
 * This machine has no real grok binary and no XAI_API_KEY; every process
 * here is the stand-in. The keyless refusal is asserted as real behaviour -
 * no test fabricates a real-execution PASS.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {existsSync, mkdirSync, readFileSync, rmSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {
  createGrokAgentProvider,
  GrokProviderRefusalError,
  GROK_PROVIDER_ID,
  PINNED_GROK_VERSION,
} from '../src/index.mjs';

const FAKE_GROK_SCRIPT = fileURLToPath(new URL('./fake-grok.mjs', import.meta.url));

// ---- helpers -----------------------------------------------------------------

// NEVER block the event loop while a run is in flight: the adapter needs the
// loop to spawn the stand-in. All in-flight polling helpers are async.
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

/** Provider wired to the stand-in CLI (binPath = node, binArgs = [script]). */
function makeProvider(config = {}) {
  return createGrokAgentProvider({
    binPath: process.execPath,
    binArgs: [FAKE_GROK_SCRIPT],
    ...config,
  });
}

/** Trusted-Host context env for a test call (never real credentials). */
function hostEnv({key = 'test-key', record, scenario, fakeVersion} = {}) {
  const env = {};
  if (key !== null) env.XAI_API_KEY = key;
  if (record) env.TALLY_FAKE_GROK_RECORD = record;
  if (scenario) env.TALLY_FAKE_GROK_SCENARIO = scenario;
  if (fakeVersion) env.TALLY_FAKE_GROK_VERSION = fakeVersion;
  return env;
}

function newWorktree(t) {
  const dir = path.join(tmpdir(), `tally-grok-ws-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  mkdirSync(dir, {recursive: true});
  t.after(() => {
    // Windows can transiently hold a just-exited child's cwd: retry a bit.
    try { rmSync(dir, {recursive: true, force: true, maxRetries: 5, retryDelay: 100}); } catch {}
  });
  return dir;
}

function newRecordPath(t) {
  const file = path.join(tmpdir(), `tally-grok-rec-${Date.now()}-${Math.random().toString(36).slice(2)}.jsonl`);
  t.after(() => {
    try { if (existsSync(file)) rmSync(file, {force: true, maxRetries: 5, retryDelay: 100}); } catch {}
  });
  return file;
}

/** Poll the stand-in's JSONL record until at least one complete line exists. */
async function readRecords(recordPath, {minLines = 1, wantChildPid = false} = {}) {
  const deadline = Date.now() + 15_000;
  for (;;) {
    let lines = [];
    try {
      lines = readFileSync(recordPath, 'utf8').split('\n').filter(l => l.trim() !== '');
    } catch { /* not written yet */ }
    const parsed = [];
    let allValid = lines.length >= minLines;
    if (allValid) {
      for (const line of lines) {
        try { parsed.push(JSON.parse(line)); } catch { allValid = false; break; }
      }
    }
    if (allValid && (!wantChildPid || (parsed.at(-1)?.childPid ?? 0) > 0)) return parsed;
    if (Date.now() > deadline) {
      throw new Error(`record ${recordPath} did not produce ${minLines} complete line(s) in time`);
    }
    await sleep(20);
  }
}

async function lastRecord(recordPath) {
  const records = await readRecords(recordPath);
  return records.at(-1);
}

function processAlive(pid) {
  if (!pid || pid <= 0) return false;
  if (process.platform !== 'win32') {
    try { process.kill(pid, 0); return true; }
    catch (error) { return error.code === 'EPERM'; } // exists but not ours
  }
  const res = spawnSync('tasklist', ['/FI', `PID eq ${pid}`, '/FO', 'CSV', '/NH'],
    {encoding: 'utf8', timeout: 10_000});
  if (res.status !== 0) return false;
  const line = (res.stdout ?? '').trim();
  if (line === '') return false;
  const image = line.split(',')[0]?.replace(/"/g, '').trim().toLowerCase();
  // The stand-in and its child are both node.exe; a match on the image name
  // is the same (documented) heuristic as #468's fakegrok.exe check.
  return image === 'node.exe';
}

/** Poll until the pid is gone (whole-tree teardown convergence). */
async function waitForGone(pid, label) {
  const deadline = Date.now() + 15_000;
  while (processAlive(pid)) {
    if (Date.now() > deadline) {
      throw new Error(`process ${pid} (${label}) still alive after whole-tree teardown`);
    }
    await sleep(50);
  }
}

const baseRequest = (over = {}) => ({
  taskId: '473',
  attemptId: 'attempt:473:r1',
  sourceSha: 'a'.repeat(40),
  prompt: 'add a test',
  worktreePath: newWorktree2(),
  ...over,
});

// Version without the test hook: a plain throwaway dir factory that does not
// need `t` (used only where cleanup is already covered by the OS temp dir).
function newWorktree2() {
  const dir = path.join(tmpdir(), `tally-grok-ws-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  mkdirSync(dir, {recursive: true});
  return dir;
}

// ---- probe -------------------------------------------------------------------

test('probe AVAILABLE: pinned version observed + credential present, honest empty capabilities', async t => {
  const provider = makeProvider();
  assert.equal(provider.id, GROK_PROVIDER_ID);
  assert.equal(provider.contractVersion, 1);
  assert.deepEqual(provider.capabilities, []);

  const observation = await provider.probe({env: hostEnv()});
  assert.equal(observation.state, 'AVAILABLE');
  assert.equal(observation.reason, null);
  assert.equal(observation.evidence.observed, PINNED_GROK_VERSION);
  assert.equal(observation.evidence.credential, 'present');
});

test('probe UNAVAILABLE + GROK_NOT_AUTHENTICATED without XAI_API_KEY (default state on this machine)', async t => {
  const provider = makeProvider();
  const observation = await provider.probe({env: hostEnv({key: null})});
  assert.equal(observation.state, 'UNAVAILABLE');
  assert.equal(observation.reason, 'GROK_NOT_AUTHENTICATED');
  assert.equal(observation.evidence.credential, 'missing');
  assert.equal(observation.evidence.observed, PINNED_GROK_VERSION);
});

test('probe UNAVAILABLE + GROK_VERSION_MISMATCH: observed version differs from the pin (fail closed)', async t => {
  const provider = makeProvider();
  const observation = await provider.probe({
    env: hostEnv({fakeVersion: 'grok 1.1.0 (ffffffffffff)'}),
  });
  assert.equal(observation.state, 'UNAVAILABLE');
  assert.equal(observation.reason, 'GROK_VERSION_MISMATCH');
  assert.equal(observation.evidence.observed, '1.1.0');
  assert.equal(observation.evidence.pinned, PINNED_GROK_VERSION);
});

test('probe UNAVAILABLE + GROK_PROBE_FAILED when --version is unobservable', async t => {
  const provider = makeProvider();
  const observation = await provider.probe({env: hostEnv({scenario: 'versionfail'})});
  assert.equal(observation.state, 'UNAVAILABLE');
  assert.equal(observation.reason, 'GROK_PROBE_FAILED');

  const missing = makeProvider({binPath: path.join(tmpdir(), 'no-such-grok-binary')});
  const missingObservation = await missing.probe({env: hostEnv()});
  assert.equal(missingObservation.state, 'UNAVAILABLE');
  assert.equal(missingObservation.reason, 'GROK_PROBE_FAILED');
});

// ---- keyless refusal (the honest negative) -----------------------------------

test('execute without XAI_API_KEY is a typed refusal BEFORE spawn - never a COMPLETED', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const worktree = newWorktree(t);
  await assert.rejects(
    () => provider.execute(baseRequest({worktreePath: worktree}),
      {env: hostEnv({key: null, record})}),
    error => error instanceof GrokProviderRefusalError &&
      error.code === 'GROK_NOT_AUTHENTICATED');
  // No process was ever spawned: the record file does not exist.
  await sleep(300);
  assert.equal(existsSync(record), false, 'stand-in CLI must not have been spawned');
});

// ---- happy path, argv shape, cwd pinning --------------------------------------

test('execute exit 0: COMPLETED with stdout as the final answer evidence', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(baseRequest(), {env: hostEnv({record})});
  assert.equal(observation.state, 'COMPLETED');
  assert.equal(observation.reason, null);
  assert.equal(observation.evidence.agentSummary, 'ok');
  assert.equal(observation.evidence.exitCode, 0);
  assert.equal(typeof observation.evidence.durationMs, 'number');
});

test('argv shape: --no-auto-update + attached --single= form; zero overlay; no bare -p', async t => {
  const record = newRecordPath(t);
  const worktree = newWorktree(t);
  const marker = path.join(worktree, 'marker.txt');
  writeFileSync(marker, 'cwd-bound-answer');
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest({worktreePath: worktree, prompt: 'read marker.txt'}),
    {env: hostEnv({record})});
  assert.equal(observation.state, 'COMPLETED');
  assert.equal(observation.evidence.agentSummary, 'cwd-bound-answer',
    'final answer must come from the file the stand-in really read in its cwd');

  const rec = await lastRecord(record);
  assert.equal(rec.cwd, worktree, 'workspace is pinned via spawn cwd (workspaceRoot=process.cwd())');
  assert.equal(rec.prompt, 'read marker.txt', 'instruction rides --single= verbatim');
  assert.deepEqual(rec.argv, ['--no-auto-update', '--single=read marker.txt']);
  assert.ok(!rec.argv.some(a => a === '-p' || a.startsWith('-p=')),
    'the misparse-prone bare -p form is never emitted');
});

test('leading-dash instruction survives via the attached --single= form', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest({prompt: '-leading dash task'}), {env: hostEnv({record})});
  assert.equal(observation.state, 'COMPLETED');
  const rec = await lastRecord(record);
  assert.deepEqual(rec.argv, ['--no-auto-update', '--single=-leading dash task']);
});

test('model and permission-mode pass through observed flags only when configured', async t => {
  const record = newRecordPath(t);
  const configured = makeProvider({model: 'grok-4.7', permissionMode: 'acceptEdits'});
  const observation = await configured.execute(baseRequest(), {env: hostEnv({record})});
  assert.equal(observation.state, 'COMPLETED');
  const rec = await lastRecord(record);
  assert.equal(rec.model, 'grok-4.7');
  assert.equal(rec.permissionMode, 'acceptEdits');
  assert.deepEqual(rec.argv,
    ['--no-auto-update', '-m', 'grok-4.7', '--permission-mode', 'acceptEdits',
      '--single=add a test']);

  const plainRecord = newRecordPath(t);
  const plain = makeProvider();
  await plain.execute(baseRequest(), {env: hostEnv({record: plainRecord})});
  const plainRec = await lastRecord(plainRecord);
  assert.deepEqual(plainRec.argv, ['--no-auto-update', '--single=add a test'],
    'zero overlay: no -m / --permission-mode flags when not configured');
});

// ---- isolation: env allowlist + per-attempt GROK_HOME -------------------------

test('spawn env is an allowlist: injected key reaches the CLI, owner credentials structurally cannot', async t => {
  const ownerHome = path.join(tmpdir(), `tally-grok-owner-home-${Date.now()}`);
  mkdirSync(ownerHome, {recursive: true});
  const savedDeepSeek = process.env.DEEPSEEK_API_KEY;
  const savedGrokHome = process.env.GROK_HOME;
  process.env.DEEPSEEK_API_KEY = 'owner-secret-not-for-grok';
  process.env.GROK_HOME = ownerHome;
  try {
    const record = newRecordPath(t);
    const provider = makeProvider();
    const observation = await provider.execute(
      baseRequest(), {env: hostEnv({record, })});
    assert.equal(observation.state, 'COMPLETED');
    const rec = await lastRecord(record);
    assert.equal(rec.xaiApiKeySet, true,
      'Host-context XAI_API_KEY reaches the spawned CLI (observed auth env name)');
    assert.equal(rec.deepseekKeySet, false,
      'owner env DEEPSEEK_API_KEY must structurally not reach the spawned CLI');
    assert.ok(rec.grokHome !== '' && rec.grokHome !== ownerHome,
      `GROK_HOME must be a fresh per-attempt dir, got ${JSON.stringify(rec.grokHome)}`);
    assert.ok(rec.grokHome.startsWith(tmpdir()),
      'per-attempt home lives under the OS temp dir');

    // Same-named key in the trusted Host context env is ALSO overridden.
    const record2 = newRecordPath(t);
    await provider.execute(
      baseRequest(), {env: {...hostEnv({record: record2}), GROK_HOME: ownerHome}});
    const rec2 = await lastRecord(record2);
    assert.ok(rec2.grokHome !== ownerHome && rec2.grokHome !== rec.grokHome,
      'GROK_HOME is forced per attempt; context env cannot defeat it');
  } finally {
    if (savedDeepSeek === undefined) delete process.env.DEEPSEEK_API_KEY;
    else process.env.DEEPSEEK_API_KEY = savedDeepSeek;
    if (savedGrokHome === undefined) delete process.env.GROK_HOME;
    else process.env.GROK_HOME = savedGrokHome;
  }
});

test('per-attempt GROK_HOME is removed after the run settles', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  await provider.execute(baseRequest(), {env: hostEnv({record})});
  const rec = await lastRecord(record);
  const deadline = Date.now() + 10_000;
  while (existsSync(rec.grokHome)) {
    if (Date.now() > deadline) {
      throw new Error(`per-attempt GROK_HOME ${rec.grokHome} was not removed after the run`);
    }
    await sleep(50);
  }
});

// ---- exit-code mapping --------------------------------------------------------

test('exit 1 with the observed keyless stderr maps to FAILED with the stderr summary', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest(), {env: hostEnv({record, scenario: 'keyless'})});
  assert.equal(observation.state, 'FAILED');
  assert.equal(observation.reason, 'GROK_RUN_FAILED');
  assert.equal(observation.evidence.exitCode, 1);
  assert.ok(observation.evidence.stderr.includes('Not signed in'),
    'failure evidence carries the observed keyless stderr text');
  assert.equal(observation.evidence.agentSummary, undefined,
    'a failed run has no final-answer evidence');
});

test('exit 2 usage error maps to FAILED + GROK_USAGE_ERROR', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest(), {env: hostEnv({record, scenario: 'usage'})});
  assert.equal(observation.state, 'FAILED');
  assert.equal(observation.reason, 'GROK_USAGE_ERROR');
  assert.equal(observation.evidence.exitCode, 2);
});

test('non-zero crash (exit 7): FAILED claim; partial stdout never surfaces as the answer', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest(), {env: hostEnv({record, scenario: 'exitfail'})});
  assert.equal(observation.state, 'FAILED');
  assert.equal(observation.reason, 'GROK_RUN_FAILED');
  assert.equal(observation.evidence.exitCode, 7);
  assert.equal(observation.evidence.agentSummary, undefined);
  assert.ok(!JSON.stringify(observation).includes('partial answer'),
    'the pre-crash partial output must not appear anywhere in the claim');
});

test('exit 0 without a final answer fails safe (GROK_EMPTY_OUTPUT), no guessing', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest(), {env: hostEnv({record, scenario: 'emptyok'})});
  assert.equal(observation.state, 'FAILED');
  assert.equal(observation.reason, 'GROK_EMPTY_OUTPUT');
});

test('credentials echoed by the CLI into diagnostics are redacted before storage', async t => {
  const record = newRecordPath(t);
  const secret = 'xai-secret-probe-value-7f3a9';
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest(), {env: hostEnv({record, scenario: 'leak', key: secret})});
  assert.equal(observation.state, 'FAILED');
  assert.ok(observation.evidence.stderr.includes('[REDACTED]'),
    'stderr evidence carries the redaction marker');
  assert.ok(!JSON.stringify(observation).includes(secret),
    'the injected credential value must not leak into evidence');
});

// ---- cancellation / timeout / tree teardown -----------------------------------

test('timeout kills the whole tree and settles UNKNOWN + GROK_RUN_TIMEOUT', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const observation = await provider.execute(
    baseRequest({prompt: 'tally-conformance-long run', timeoutMs: 2_000}),
    {env: hostEnv({record})});
  assert.equal(observation.state, 'UNKNOWN');
  assert.equal(observation.reason, 'GROK_RUN_TIMEOUT');
  assert.equal(observation.evidence.timedOut, true);
  const rec = await lastRecord(record);
  assert.ok((rec.childPid ?? 0) > 0, 'stand-in recorded its child pid');
  await waitForGone(rec.childPid, 'long-run child after timeout');
});

test('AbortSignal cancels the run: CANCELLED and no orphan subprocess remains', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const controller = new AbortController();
  const pending = provider.execute(
    baseRequest({prompt: 'tally-conformance-long run'}), {env: hostEnv({record}), signal: controller.signal});
  const rec = (await readRecords(record, {wantChildPid: true})).at(-1);
  assert.ok((rec.childPid ?? 0) > 0);
  controller.abort();
  const observation = await pending;
  assert.equal(observation.state, 'CANCELLED');
  assert.equal(observation.evidence.cause, 'abort-signal');
  await waitForGone(rec.childPid, 'long-run child after abort');
});

test('cancel(): CANCEL_REQUESTED while in flight, NOT_RUNNING otherwise; tree torn down', async t => {
  const record = newRecordPath(t);
  const provider = makeProvider();
  const request = baseRequest({taskId: '9001', attemptId: 'attempt:9001:r1',
    prompt: 'tally-conformance-long run'});
  const pending = provider.execute(request, {env: hostEnv({record})});
  const rec = (await readRecords(record, {wantChildPid: true})).at(-1);

  const unknownPin = {taskId: '9001', attemptId: 'attempt:9001:other', sourceSha: request.sourceSha,
    providerId: 'grok'};
  const notRunning = await provider.cancel(unknownPin);
  assert.equal(notRunning.state, 'NOT_RUNNING');

  const cancelRequest = {taskId: request.taskId, attemptId: request.attemptId,
    sourceSha: request.sourceSha, providerId: 'grok'};
  const requested = await provider.cancel(cancelRequest);
  assert.equal(requested.state, 'CANCEL_REQUESTED');

  const observation = await pending;
  assert.equal(observation.state, 'CANCELLED');
  assert.equal(observation.evidence.cause, 'cancel-request');
  await waitForGone(rec.childPid, 'long-run child after cancel()');
});

// ---- reconcile + construction gates -------------------------------------------

test('reconcile stays UNKNOWN + NOT_RECONCILED (no execution journal to reconcile from)', async t => {
  const provider = makeProvider();
  const observation = await provider.reconcile({taskId: '473', attemptId: 'attempt:473:r1',
    sourceSha: 'a'.repeat(40), providerId: 'grok'});
  assert.equal(observation.state, 'UNKNOWN');
  assert.equal(observation.reason, 'NOT_RECONCILED');
});

test('construction fails closed on invalid config (explicit binPath, enum permission, non-blank model)', () => {
  assert.throws(() => createGrokAgentProvider({}), /binPath/);
  assert.throws(() => createGrokAgentProvider({binPath: '   '}), /binPath/);
  assert.throws(() => makeProvider({permissionMode: 'yolo'}), /permissionMode/);
  assert.throws(() => makeProvider({model: '   '}), /model/);
});
