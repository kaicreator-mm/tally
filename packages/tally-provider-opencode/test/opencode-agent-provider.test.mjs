/**
 * OpenCode AgentProvider/1 adapter tests — node:test + fake CLI only.
 * NO real opencode process is ever spawned here (real-turn acceptance is the
 * controller's job; see README NOT_RUN list).
 *
 * Process boundary, argv shape, PWD/XDG isolation, exit-code classification,
 * tree teardown and redaction are exercised against the fake CLI
 * (test/fake-opencode.mjs) exactly like the #471 Go tests drive fakeopencode.
 */
import test from 'node:test';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {existsSync, mkdtempSync, readFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {basename, dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createProviderAttemptPin} from '../../tally-agent-provider/src/index.mjs';
import {createNdjsonEventSink, parseNdjsonRunOutput} from '../src/ndjson.mjs';
import {createOpenCodeAgentProvider, OPENCODE_PINNED_VERSION} from '../src/index.mjs';

const FAKE_CLI = fileURLToPath(new URL('./fake-opencode.mjs', import.meta.url));
const KEY = 'test-key-0123456789abcdef';
const LONG_MARKER = 'tally-conformance-long';

const makeProvider = (over = {}) =>
  createOpenCodeAgentProvider({cliPath: process.execPath, cliArgs: [FAKE_CLI], ...over});
const ctx = env => ({env: {DEEPSEEK_API_KEY: KEY, ...env}});
const req = (prompt, worktreePath, over = {}) => ({
  taskId: '264',
  attemptId: 'attempt:264:r1',
  sourceSha: 'a'.repeat(40),
  prompt,
  worktreePath,
  ...over,
});
const pin = (over = {}) => createProviderAttemptPin({
  taskId: '264', attemptId: 'attempt:264:r1', sourceSha: 'a'.repeat(40),
  providerId: 'opencode', ...over,
});
const newWs = () => mkdtempSync(join(tmpdir(), 'oc-ws-'));
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));

async function waitForRecord(path, predicate = () => true, timeoutMs = 15000) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    let last = null;
    try {
      const lines = readFileSync(path, 'utf8').split('\n').filter(l => l.trim() !== '');
      if (lines.length > 0) last = JSON.parse(lines[lines.length - 1]);
    } catch { /* record not written (completely) yet */ }
    if (last !== null && predicate(last)) return last;
    if (Date.now() > deadline) throw new Error(`fake record not ready: ${path}`);
    await delay(20);
  }
}

function pidAlive(pid) {
  if (!Number.isInteger(pid) || pid <= 0) return false;
  if (process.platform === 'win32') {
    let listing;
    try {
      listing = execFileSync('tasklist',
        ['/FI', `PID eq ${pid}`, '/FO', 'CSV', '/NH'],
        {encoding: 'utf8', timeout: 10000, windowsHide: true});
    } catch {
      return false;
    }
    const line = listing.trim();
    if (line === '' || /^INFO/i.test(line)) return false;
    const image = (line.split(',')[0] ?? '').replace(/"/g, '').trim().toLowerCase();
    return image === basename(process.execPath).toLowerCase();
  }
  try {
    process.kill(pid, 0);
  } catch {
    return false;
  }
  // signal 0 succeeds for zombies too: a killed-but-unreaped orphan
  // (reparented to a container PID 1 that does not adopt-reap) is DEAD
  // for the no-orphan assertions — treat kernel state Z as not alive.
  try {
    const stat=readFileSync('/proc/'+pid+'/stat','utf8');
    if (/^\s*Z/.test(stat.slice(stat.lastIndexOf(')')+1))) return false;
  } catch {/* keep the signal-0 answer */}
  return true;
}

async function waitForGone(pid, timeoutMs = 15000) {
  const deadline = Date.now() + timeoutMs;
  while (pidAlive(pid)) {
    if (Date.now() > deadline) throw new Error(`pid ${pid} still alive after tree kill`);
    await delay(50);
  }
}

const ZERO_USAGE = {inputTokens: 0, outputTokens: 0, reasoningTokens: 0,
  cacheReadTokens: 0, cacheWriteTokens: 0};

// ---- probe ladder -------------------------------------------------------------

test('probe AVAILABLE: version observed, credentials explicit, canary NOT_RUN',
  {timeout: 30000}, async () => {
    const provider = makeProvider();
    const obs = await provider.probe(ctx({}));
    assert.equal(obs.state, 'AVAILABLE');
    assert.equal(obs.evidence.cliVersion, OPENCODE_PINNED_VERSION);
    assert.equal(obs.evidence.pinnedVersion, OPENCODE_PINNED_VERSION);
    assert.equal(obs.evidence.canary, 'NOT_RUN');
    assert.equal(obs.evidence.present, true);
  });

test('probe UNAVAILABLE when the CLI binary is missing', {timeout: 30000}, async () => {
  const missing = join(tmpdir(), `no-such-opencode-${Date.now()}`);
  const provider = createOpenCodeAgentProvider({cliPath: missing});
  const obs = await provider.probe(ctx({}));
  assert.equal(obs.state, 'UNAVAILABLE');
  assert.equal(obs.reason, 'CLI_NOT_FOUND');
});

test('probe fail-closed on version mismatch and unobservable version',
  {timeout: 30000}, async () => {
    const provider = makeProvider();
    const mismatch = await provider.probe(ctx({TALLY_FAKE_OPENCODE_VERSION: '0.2.0'}));
    assert.equal(mismatch.state, 'UNAVAILABLE');
    assert.equal(mismatch.reason, 'VERSION_MISMATCH');
    assert.equal(mismatch.evidence.observed, '0.2.0');
    assert.equal(mismatch.evidence.pinnedVersion, OPENCODE_PINNED_VERSION);
    const unobservable = await provider.probe(ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'versionfail'}));
    assert.equal(unobservable.state, 'UNAVAILABLE');
    assert.equal(unobservable.reason, 'VERSION_UNOBSERVABLE');
  });

test('probe gates credentials by explicit env only', {timeout: 30000}, async () => {
  const provider = makeProvider();
  const obs = await provider.probe({env: {}});
  assert.equal(obs.state, 'UNAVAILABLE');
  assert.equal(obs.reason, 'CREDENTIALS_MISSING');
  assert.equal(obs.evidence.canary, 'NOT_RUN');
});

// ---- execute: happy path + argv shape ------------------------------------------

test('execute happy path: argv shape, NDJSON summary/usage/session evidence',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider({model: 'deepseek/deepseek-v4-pro', agent: 'build'});
    const obs = await provider.execute(req('add a test', ws), ctx({TALLY_FAKE_OPENCODE_RECORD: record}));
    assert.equal(obs.state, 'COMPLETED');
    assert.equal(obs.reason, null);
    assert.equal(obs.evidence.exitCode, 0);
    assert.equal(obs.evidence.summary, 'ok');
    assert.equal(obs.evidence.sessionId, 'ses_fake');
    assert.equal(obs.evidence.degraded, false);
    assert.deepEqual(obs.evidence.usage,
      {inputTokens: 38, outputTokens: 2, reasoningTokens: 0,
        cacheReadTokens: 0, cacheWriteTokens: 0});
    const rec = await waitForRecord(record);
    assert.deepEqual(rec.argv.slice(2),
      ['run', '--format', 'json', '--model', 'deepseek/deepseek-v4-pro',
        '--agent', 'build', '--pure', 'add a test']);
    assert.ok(!rec.argv.includes('--attach'), '--attach must never be passed');
    assert.ok(!rec.argv.includes('--standalone'), '--standalone must never be passed');
    assert.ok(rec.argv.lastIndexOf('--pure') < rec.argv.indexOf('add a test'),
      'instruction must be the last argument');
    assert.equal(rec.message, 'add a test');
    assert.equal(rec.model, 'deepseek/deepseek-v4-pro');
    assert.equal(rec.agent, 'build');
  });

test('execute without model/agent expectations passes only observed flags',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    const obs = await provider.execute(req('add a test', ws), ctx({TALLY_FAKE_OPENCODE_RECORD: record}));
    assert.equal(obs.state, 'COMPLETED');
    const rec = await waitForRecord(record);
    assert.deepEqual(rec.argv.slice(2),
      ['run', '--format', 'json', '--pure', 'add a test']);
    assert.equal(rec.model, null);
    assert.equal(rec.agent, null);
  });

// ---- execute: isolation (PWD trap + XDG redirection + cleanup) ------------------

test('execute pins PWD/OLDPWD to the workspace and redirects XDG roots per attempt',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const owner = mkdtempSync(join(tmpdir(), 'oc-owner-'));
    const provider = makeProvider();
    await provider.execute(req('x', ws), ctx({
      XDG_CONFIG_HOME: join(owner, 'cfg'),
      XDG_DATA_HOME: join(owner, 'data'),
      XDG_CACHE_HOME: join(owner, 'cache'),
      XDG_STATE_HOME: join(owner, 'state'),
      TMP: join(owner, 't'),
      TEMP: join(owner, 't'),
      PWD: join(owner, 'stale-pwd'),
      OLDPWD: join(owner, 'stale-oldpwd'),
      TALLY_FAKE_OPENCODE_RECORD: record,
    }));
    const rec = await waitForRecord(record);
    assert.equal(rec.cwd, ws);
    assert.equal(rec.pwd_env, ws, 'PWD must be pinned to the workspace');
    assert.equal(rec.oldpwd_env, ws, 'OLDPWD must be pinned to the workspace');
    const root = dirname(rec.xdg_data);
    assert.ok(basename(root).startsWith('tally-opencode-attempt-'),
      `attempt root naming: ${root}`);
    for (const [key, sub] of [
      ['xdg_config', 'config'], ['xdg_data', 'data'],
      ['xdg_cache', 'cache'], ['xdg_state', 'state'],
      ['tmp', 'tmp'], ['temp', 'tmp'],
    ]) {
      assert.equal(rec[key], join(root, sub), `${key} must sit under the attempt root`);
      assert.ok(!String(rec[key]).startsWith(owner), `${key} must not point at owner dirs`);
    }
    assert.equal(existsSync(root), false, 'attempt tree must be removed after the run');
  });

// ---- NDJSON parser (unit) -------------------------------------------------------

test('ndjson: multi-step stream sums usage and takes the last text as summary', () => {
  const stream = [
    '{"type":"step_start","timestamp":1,"sessionID":"ses_a","part":{"type":"step-start"}}',
    '{"type":"tool_use","timestamp":2,"sessionID":"ses_a","part":{"type":"tool","tool":"write"}}',
    '{"type":"step_finish","timestamp":3,"sessionID":"ses_a","part":{"type":"step-finish","reason":"tool-calls","tokens":{"total":130,"input":47,"output":83}}}',
    '{"type":"step_start","timestamp":4,"sessionID":"ses_a","part":{"type":"step-start"}}',
    '{"type":"text","timestamp":5,"sessionID":"ses_a","part":{"type":"text","text":"earlier"}}',
    '{"type":"text","timestamp":6,"sessionID":"ses_a","part":{"type":"text","text":"done"}}',
    '{"type":"step_finish","timestamp":7,"sessionID":"ses_a","part":{"type":"step-finish","reason":"stop","tokens":{"total":48,"input":46,"output":2,"reasoning":3,"cache":{"read":5,"write":7}}}}',
    '',
    'not json at all',
  ].join('\n');
  const snap = parseNdjsonRunOutput(stream);
  assert.equal(snap.summary, 'done');
  assert.equal(snap.sessionId, 'ses_a');
  assert.equal(snap.sawJSON, true);
  assert.equal(snap.eventCount, 7);
  assert.deepEqual(snap.usage,
    {inputTokens: 93, outputTokens: 85, reasoningTokens: 3,
      cacheReadTokens: 5, cacheWriteTokens: 7});
});

test('ndjson: noise prefix, unknown types, chunked delivery, truncated tail', () => {
  const sink = createNdjsonEventSink();
  sink.push('noise before json {"type":"text","timestamp":1,"sessionID":"ses_b","part":{"type":"text","text":"hi"}}\n');
  sink.push('{"type":"brand_new_type","timestamp":2,"sessionID":"ses_b","part":{"type":"mystery"}}\n');
  sink.push('{"type":"step_fin');
  sink.push('ish","timestamp":3,"sessionID":"ses_b","part":{"type":"step-finish","tokens":{"input":4,"output":1}}}\n');
  sink.push('{"type":"text","timestamp":4,"part":{"te');
  sink.end();
  const snap = sink.snapshot();
  assert.equal(snap.summary, 'hi');
  assert.equal(snap.sessionId, 'ses_b');
  assert.equal(snap.sawJSON, true);
  assert.deepEqual(snap.usage, {...ZERO_USAGE,
    inputTokens: 4, outputTokens: 1});
});

test('ndjson: non-JSON output degrades with zero usage', () => {
  const snap = parseNdjsonRunOutput('plain text answer\n');
  assert.equal(snap.sawJSON, false);
  assert.equal(snap.summary, null);
  assert.deepEqual(snap.usage, ZERO_USAGE);
  const empty = parseNdjsonRunOutput('');
  assert.equal(empty.sawJSON, false);
  assert.deepEqual(empty.usage, ZERO_USAGE);
});

// ---- exit-code classification ---------------------------------------------------

test('non-zero exit yields FAILED with exit code, never a partial answer',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    const obs = await provider.execute(req('x', ws),
      ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'exitfail', TALLY_FAKE_OPENCODE_RECORD: record}));
    assert.equal(obs.state, 'FAILED');
    assert.equal(obs.reason, 'EXIT_NONZERO');
    assert.equal(obs.evidence.exitCode, 7);
    assert.ok(String(obs.evidence.stderr).includes('UnknownError'));
    assert.ok(!JSON.stringify(obs).includes('partial answer before crash'),
      'partial stdout must not masquerade as an answer');
    const root = dirname((await waitForRecord(record)).xdg_data);
    assert.equal(existsSync(root), false, 'attempt tree cleaned after FAILED');
  });

test('exit 0 without any output fails safe (NO_OUTPUT)', {timeout: 30000}, async () => {
  const provider = makeProvider();
  const obs = await provider.execute(req('x', newWs()),
    ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'emptyok'}));
  assert.equal(obs.state, 'FAILED');
  assert.equal(obs.reason, 'NO_OUTPUT');
});

test('exit 0 with bare non-JSON stdout degrades to COMPLETED with zero usage',
  {timeout: 30000}, async () => {
    const provider = makeProvider();
    const obs = await provider.execute(req('x', newWs()),
      ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'plainok'}));
    assert.equal(obs.state, 'COMPLETED');
    assert.equal(obs.evidence.summary, 'plain-ok');
    assert.equal(obs.evidence.degraded, true);
    assert.deepEqual(obs.evidence.usage, ZERO_USAGE);
  });

test('timeout kills the hung run and reports UNKNOWN with a reason',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    const pending = provider.execute(req('x', ws, {timeoutMs: 800}),
      ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'hang', TALLY_FAKE_OPENCODE_RECORD: record}));
    await waitForRecord(record);
    const obs = await pending;
    assert.equal(obs.state, 'UNKNOWN');
    assert.equal(obs.reason, 'TIMEOUT_KILLED');
    assert.equal(obs.evidence.timeoutMs, 800);
    const root = dirname((await waitForRecord(record)).xdg_data);
    assert.equal(existsSync(root), false, 'attempt tree cleaned after timeout kill');
  });

test('AbortSignal before spawn yields CANCELLED without spawning', {timeout: 30000}, async () => {
  const provider = makeProvider();
  const controller = new AbortController();
  controller.abort();
  const obs = await provider.execute(req('x', newWs()),
    {...ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'hang'}), signal: controller.signal});
  assert.equal(obs.state, 'CANCELLED');
});

test('AbortSignal during a hung run kills the tree and yields CANCELLED',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    const controller = new AbortController();
    const pending = provider.execute(req('x', ws),
      {...ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'hang', TALLY_FAKE_OPENCODE_RECORD: record}),
        signal: controller.signal});
    await waitForRecord(record);
    controller.abort();
    const obs = await pending;
    assert.equal(obs.state, 'CANCELLED');
    assert.ok(obs.evidence.durationMs >= 0);
  });

test('cancel() kills the whole process tree (no orphan), execute resolves CANCELLED',
  {timeout: 40000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    const pending = provider.execute(req(`${LONG_MARKER} run`, ws),
      ctx({TALLY_FAKE_OPENCODE_RECORD: record}));
    const rec = await waitForRecord(record, r => (r.child_pid ?? 0) > 0);
    assert.ok(pidAlive(rec.child_pid), 'fake child must be alive before cancel');
    assert.equal((await provider.cancel(pin({attemptId: 'attempt:264:missing'}))).state,
      'NOT_RUNNING');
    assert.equal((await provider.cancel(pin())).state, 'CANCEL_REQUESTED');
    const obs = await pending;
    assert.equal(obs.state, 'CANCELLED');
    await waitForGone(rec.child_pid);
    assert.equal((await provider.cancel(pin())).state, 'NOT_RUNNING');
  });

// ---- version gate on execute + missing CLI + credential semantics ---------------

test('execute fail-closed on version mismatch is UNKNOWN, never FAILED',
  {timeout: 30000}, async () => {
    const provider = makeProvider();
    const obs = await provider.execute(req('x', newWs()),
      ctx({TALLY_FAKE_OPENCODE_VERSION: '0.2.0'}));
    assert.equal(obs.state, 'UNKNOWN');
    assert.equal(obs.reason, 'VERSION_MISMATCH');
    assert.equal(obs.evidence.observed, '0.2.0');
  });

test('missing CLI: probe UNAVAILABLE, execute UNKNOWN (environment, not candidate)',
  {timeout: 30000}, async () => {
    const missing = join(tmpdir(), `no-such-opencode-${Date.now()}`);
    const provider = createOpenCodeAgentProvider({cliPath: missing});
    const probeObs = await provider.probe(ctx({}));
    assert.equal(probeObs.state, 'UNAVAILABLE');
    assert.equal(probeObs.reason, 'CLI_NOT_FOUND');
    const execObs = await provider.execute(req('x', newWs()), ctx({}));
    assert.equal(execObs.state, 'UNKNOWN');
    assert.equal(execObs.reason, 'CLI_NOT_FOUND');
  });

test('keyless execute is not blocked and no credential is invented',
  {timeout: 30000}, async () => {
    const ws = newWs();
    const record = join(ws, 'calls.jsonl');
    const provider = makeProvider();
    // Host env carries no DEEPSEEK_API_KEY (and nothing leaks in from the
    // parent process env: the adapter env is exactly the Host snapshot plus
    // the forced isolation keys).
    const obs = await provider.execute(req('x', ws),
      {env: {TALLY_FAKE_OPENCODE_RECORD: record}});
    assert.equal(obs.state, 'COMPLETED');
    const rec = await waitForRecord(record);
    assert.equal(rec.api_key_set, false);
  });

test('injected credentials are redacted out of failure evidence', {timeout: 30000}, async () => {
  const provider = makeProvider();
  const obs = await provider.execute(req('x', newWs()),
    ctx({TALLY_FAKE_OPENCODE_SCENARIO: 'leakstderr'}));
  assert.equal(obs.state, 'FAILED');
  assert.equal(obs.reason, 'EXIT_NONZERO');
  assert.ok(!JSON.stringify(obs).includes(KEY), 'credential value leaked into evidence');
  assert.ok(String(obs.evidence.stderr).includes('[REDACTED]'));
});
