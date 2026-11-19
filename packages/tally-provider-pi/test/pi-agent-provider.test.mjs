/**
 * tally-provider-pi adapter tests: every process boundary is exercised
 * against a real fake CLI child process (argv shape, env strip/force, NDJSON
 * parsing, exit-code classification, timeout/abort tree termination,
 * isolation cleanup). No real pi turn is ever run — live acceptance belongs
 * to the controller.
 */

import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtempSync, mkdirSync, writeFileSync, existsSync, readFileSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {
  createPiAgentProvider, PINNED_PI_VERSION, parsePiStream, buildPiCliArgs,
} from '../src/index.mjs';
import {
  PI_CREDENTIAL_ENV_VARS, PI_CONFIG_ENV_VARS, isForcedEnvName,
  buildPiSpawnEnv, createIsolationDirs, removeIsolationDirs,
} from '../src/pi-env.mjs';
import {
  createProviderAttemptPin,
} from '../../tally-agent-provider/src/index.mjs';

const fakeScript = fileURLToPath(new URL('./fake-pi.mjs', import.meta.url));
const FAKE_SESSION_ID = 'f0000000-0000-4000-8000-000000000001';
const FAKE_MODEL = 'fake-model';
const FAKE_PROVIDER = 'fake-provider';
const SOURCE_SHA = 'a'.repeat(40);

// ---- test infrastructure ---------------------------------------------------

function newTestDir(t) {
  const dir = mkdtempSync(path.join(tmpdir(), 'pi-prov-test-'));
  t.after(() => { rmSync(dir, {recursive: true, force: true}); });
  return dir;
}

function testInheritedEnv(recordPath, extra = {}) {
  const env = {
    PATH: process.env.PATH ?? '/usr/bin',
    ...(process.env.SystemRoot ? {SystemRoot: process.env.SystemRoot} : {}),
    ...(recordPath ? {TALLY_FAKE_PI_RECORD: recordPath} : {}),
    ...extra,
  };
  return env;
}

function makeProvider(t, {
  recordPath, inheritedExtra = {}, factory = {},
} = {}) {
  return createPiAgentProvider({
    binPath: process.execPath,
    programArgs: [fakeScript],
    timeoutMs: 15000,
    inheritedEnv: testInheritedEnv(recordPath, inheritedExtra),
    ...factory,
  });
}

function makeRequest({prompt = 'x', worktree, timeoutMs, taskId = 't1', attemptId = 'a1'} = {}) {
  return {
    taskId, attemptId, sourceSha: SOURCE_SHA,
    prompt, worktreePath: worktree, ...(timeoutMs !== undefined ? {timeoutMs} : {}),
  };
}

function hostContext(env = {}) {
  return {env};
}

async function runExecute(provider, options = {}, env = {}) {
  return provider.execute(makeRequest(options), hostContext(env));
}

function readRecords(recordPath) {
  if (!existsSync(recordPath)) return [];
  return readFileSync(recordPath, 'utf8')
    .split('\n')
    .filter((line) => line.trim() !== '')
    .map((line) => JSON.parse(line));
}

function lastRecord(t, recordPath) {
  const rows = readRecords(recordPath);
  assert.ok(rows.length > 0, `expected at least one fake record in ${recordPath}`);
  return rows[rows.length - 1];
}

async function waitFor(condition, {deadlineMs = 10000, stepMs = 25} = {}) {
  const deadline = Date.now() + deadlineMs;
  for (;;) {
    const value = condition();
    if (value) return value;
    if (Date.now() > deadline) throw new Error('waitFor: condition not met in time');
    await new Promise((resolve) => setTimeout(resolve, stepMs));
  }
}

function pidAlive(pid) {
  if (!pid || pid <= 0) return false;
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

function assertNoPrintSpawn(recordPath) {
  const rows = readRecords(recordPath);
  assert.equal(rows.filter((row) => row.kind === 'print').length, 0,
    'no pi print process must be spawned');
}

// ---- probe: pinned version gate + credential gate ---------------------------

test('probe AVAILABLE when pinned version matches and credential is injected', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const provider = makeProvider(t, {recordPath: record});
  const result = await provider.probe(hostContext({DEEPSEEK_API_KEY: 'test-key'}));
  assert.equal(result.state, 'AVAILABLE');
  assert.equal(result.reason, null);
  assert.equal(result.evidence.cliVersion, PINNED_PI_VERSION);
  assert.equal(result.evidence.pinnedVersion, PINNED_PI_VERSION);
  assert.equal(result.evidence.credentialEnvVar, 'DEEPSEEK_API_KEY');
  // Canary (real pi turn) is NOT run by this adapter — honest NOT_RUN.
  assert.equal(result.evidence.canaryTurn, 'NOT_RUN');
});

test('probe UNAVAILABLE on missing binary and never spawns', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const provider = createPiAgentProvider({
    binPath: path.join(dir, 'no-such-pi'),
    inheritedEnv: testInheritedEnv(record),
  });
  const result = await provider.probe(hostContext({DEEPSEEK_API_KEY: 'test-key'}));
  assert.equal(result.state, 'UNAVAILABLE');
  assert.equal(result.reason, 'PI_BINARY_MISSING');
  assertNoPrintSpawn(record);
});

test('probe UNAVAILABLE on version pin mismatch and never spawns a turn', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const provider = makeProvider(t, {recordPath: record});
  // The identity probe's spawn env is the Host-injected env only (mirrors
  // the #469 Go candidate), so the fake version override rides context.env.
  const result = await provider.probe(hostContext({
    DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_VERSION: '0.86.0',
  }));
  assert.equal(result.state, 'UNAVAILABLE');
  assert.equal(result.reason, 'PI_VERSION_PIN_MISMATCH');
  assert.equal(result.evidence.observedVersion, '0.86.0');
  assert.equal(result.evidence.pinnedVersion, PINNED_PI_VERSION);
  assertNoPrintSpawn(record);
});

test('probe UNAVAILABLE when version is unobservable (fail-closed)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const provider = makeProvider(t, {recordPath: record});
  const result = await provider.probe(hostContext({
    DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'versionfail',
  }));
  assert.equal(result.state, 'UNAVAILABLE');
  assert.equal(result.reason, 'PI_VERSION_UNOBSERVABLE');
  assertNoPrintSpawn(record);
});

test('probe UNAVAILABLE when credential env var is not explicitly injected', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const provider = makeProvider(t, {recordPath: record});
  for (const env of [{}, {DEEPSEEK_API_KEY: '   '}]) {
    const result = await provider.probe(hostContext(env));
    assert.equal(result.state, 'UNAVAILABLE');
    assert.equal(result.reason, 'PI_CREDENTIAL_MISSING');
    assert.equal(result.evidence.credentialEnvVar, 'DEEPSEEK_API_KEY');
  }
  assertNoPrintSpawn(record);
});

test('version observation is cached per provider (single --version spawn)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const env = {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_RECORD: record};
  await provider.probe(hostContext(env));
  await provider.probe(hostContext(env));
  const done = await runExecute(provider, {worktree}, env);
  assert.equal(done.state, 'COMPLETED');
  const versionRows = readRecords(record).filter((row) => row.kind === 'version');
  assert.equal(versionRows.length, 1, 'exactly one --version observation expected');
});

// ---- execute: happy path, observation vs expectation, argv shape ------------

test('execute never leaves the child stdin open (real 0.85.1 `-p` reads stdin to EOF)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  // stdinwait answers ONLY after stdin ends. A default `pipe` stdin that the
  // adapter never closes makes this scenario hang until the bounded timeout
  // (UNKNOWN); with stdin ignored (NUL) EOF is immediate and the turn
  // completes. Regression for the real-CLI hang found by the controller smoke.
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree, timeoutMs: 8000}, {
    DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'stdinwait',
  });
  assert.equal(done.state, 'COMPLETED');
  assert.equal(done.reason, null);
  assert.equal(done.evidence.summary, 'stdin closed, turn complete');
});

test('execute COMPLETED: claim evidence comes from the stream, not from expected flags', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record, factory: {
    provider: 'deepseek', model: 'deepseek-v4-flash', thinkingLevel: 'low',
  }});
  const done = await runExecute(provider, {worktree}, {DEEPSEEK_API_KEY: 'test-key'});
  assert.equal(done.state, 'COMPLETED');
  assert.equal(done.reason, null);
  const ev = done.evidence;
  assert.equal(ev.summary, 'ok');
  assert.equal(ev.sessionId, FAKE_SESSION_ID);
  // Observed identity is the stream's self-report, never the expected flags.
  assert.equal(ev.observedModel, FAKE_MODEL);
  assert.equal(ev.observedProvider, FAKE_PROVIDER);
  assert.notEqual(ev.observedModel, 'deepseek-v4-flash');
  assert.equal(ev.stopReason, 'stop');
  assert.deepEqual(ev.usage, {input: 11, output: 3, cacheRead: 0});
  assert.equal(ev.cliVersion, PINNED_PI_VERSION);
  assert.equal(ev.outputMode, 'json');
});

test('execute passes hardened argv exactly; expected flags precede `--`; instruction is last', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record, factory: {
    provider: 'deepseek', model: 'deepseek-v4-flash',
  }});
  const done = await runExecute(provider, {worktree, prompt: 'add a test'},
    {DEEPSEEK_API_KEY: 'test-key'});
  assert.equal(done.state, 'COMPLETED');
  const rec = lastRecord(t, record);
  assert.equal(rec.cwd, worktree);
  const cliArgs = rec.argv.slice(2); // strip [node, fake-pi.mjs]
  const sessionDir = cliArgs[cliArgs.indexOf('--session-dir') + 1];
  assert.deepEqual(cliArgs, [
    '-p', '--mode', 'json',
    '--no-session', '--session-dir', sessionDir,
    '--no-extensions', '--no-skills', '--no-prompt-templates',
    '--no-themes', '--no-context-files', '--no-approve',
    '--provider', 'deepseek', '--model', 'deepseek-v4-flash',
    '--', 'add a test',
  ]);
  // Isolation flags all on; --no-session present; trailing `--` + instruction.
  assert.equal(rec.no_session, true);
  assert.equal(rec.no_extensions, true);
  assert.equal(rec.no_skills, true);
  assert.equal(rec.no_prompt_templates, true);
  assert.equal(rec.no_themes, true);
  assert.equal(rec.no_context_files, true);
  assert.equal(rec.no_approve, true);
  assert.equal(cliArgs[cliArgs.length - 1], 'add a test');
  // The session dir must live under the per-attempt isolation root.
  assert.match(sessionDir, /tally-pi-attempt-[A-Za-z0-9]+[\\/]sessions$/);
});

test('execute pins workspace via process cwd (effect-level proof)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  writeFileSync(path.join(worktree, 'marker.txt'), 'cwd-bound-answer\n');
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree, prompt: 'read marker.txt'},
    {DEEPSEEK_API_KEY: 'test-key'});
  assert.equal(done.state, 'COMPLETED');
  assert.equal(done.evidence.summary, 'cwd-bound-answer');
  const rec = lastRecord(t, record);
  assert.equal(rec.cwd, worktree);
});

test('text mode: summary is the raw stdout answer, usage stays unobserved', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record, factory: {outputMode: 'text'}});
  const done = await runExecute(provider, {worktree}, {DEEPSEEK_API_KEY: 'test-key'});
  assert.equal(done.state, 'COMPLETED');
  assert.equal(done.evidence.summary, 'ok');
  assert.equal(done.evidence.outputMode, 'text');
  assert.equal('usage' in done.evidence, false);
  assert.equal('observedModel' in done.evidence, false);
});

// ---- execute: env strip / force / injection channel --------------------------

test('spawn env strips all 42 credential vars, forces isolation keys, keeps only explicit injection', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const owner = path.join(dir, 'owner');
  const ownerAgentDir = path.join(owner, 'owner-pi-agent');
  const ownerHome = path.join(dir, 'owner-home');
  mkdirSync(ownerAgentDir, {recursive: true});
  mkdirSync(ownerHome, {recursive: true});

  const inherited = {
    PATH: '/kept/bin',
    PI_PACKAGE_DIR: path.join(owner, 'packages'),
    PI_CODING_AGENT_DIR: ownerAgentDir,
    PI_CODING_AGENT_SESSION_DIR: path.join(owner, 'owner-sessions'),
    PI_SHARE_VIEWER_URL: 'https://owner.example/share',
    HOME: ownerHome,
    USERPROFILE: ownerHome,
    PI_OFFLINE: '0',
    PI_TELEMETRY: '1',
  };
  for (const name of PI_CREDENTIAL_ENV_VARS) inherited[name] = `operator-${name}`;

  const injected = {
    DEEPSEEK_API_KEY: 'explicit-key', // the single explicit credential channel
    PI_OFFLINE: '0',                  // forced keys override injections
    PI_CODING_AGENT_DIR: ownerAgentDir,
    home: 'must-drop',                // case-variant forced key: overridden
    CUSTOM_FLAG: 'kept',
  };

  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree}, injected);
  assert.equal(done.state, 'COMPLETED');

  const rec = lastRecord(t, record);
  // Every stripped credential var is gone except the explicitly injected one.
  assert.deepEqual(rec.credentials_present, ['DEEPSEEK_API_KEY']);
  // Forced isolation keys: per-attempt dirs, offline + telemetry hardening.
  assert.equal(rec.pi_offline, '1');
  assert.equal(rec.pi_telemetry, '0');
  assert.match(rec.pi_agent_dir, /tally-pi-attempt-/);
  assert.notEqual(rec.pi_agent_dir, ownerAgentDir);
  assert.match(rec.pi_session_dir_env, /tally-pi-attempt-/);
  assert.match(rec.session_dir, /tally-pi-attempt-/);
  assert.match(rec.home, /tally-pi-attempt-/);
  assert.equal(rec.home, rec.userprofile);
  assert.notEqual(rec.home, ownerHome);
  assert.equal(path.dirname(rec.home), path.dirname(rec.pi_agent_dir),
    'home and agent-config share one per-attempt root');
  // pi config switches stripped from the inherited environment.
  assert.equal(rec.pi_package_dir, '');
  assert.equal(rec.pi_share_viewer_url, '');
  // Case-variant forced key from injection must not win.
  assert.notEqual(rec.home, 'must-drop');

  // Pure-function cross-check: PATH kept, CUSTOM_FLAG kept, no case-variant
  // duplicates.
  const dirs = createIsolationDirs({parent: dir});
  try {
    const env = buildPiSpawnEnv({inheritedEnv: inherited, injectedEnv: injected, dirs});
    assert.equal(env.PATH, '/kept/bin');
    assert.equal(env.CUSTOM_FLAG, 'kept');
    assert.equal(env.DEEPSEEK_API_KEY, 'explicit-key');
    assert.equal(env.PI_OFFLINE, '1');
    assert.equal(env.PI_TELEMETRY, '0');
    assert.equal(env.HOME, dirs.home);
    assert.equal(env.USERPROFILE, dirs.home);
    assert.equal(env.PI_CODING_AGENT_DIR, dirs.agentConfig);
    assert.equal(env.PI_CODING_AGENT_SESSION_DIR, dirs.sessions);
    assert.equal('PI_PACKAGE_DIR' in env, false);
    assert.equal('PI_SHARE_VIEWER_URL' in env, false);
    assert.equal('home' in env, false);
    assert.equal('pi_offline' in env, false);
    const injectedNames = new Set(Object.keys(injected).map((k) => k.toLowerCase()));
    for (const name of PI_CREDENTIAL_ENV_VARS) {
      if (injectedNames.has(name.toLowerCase())) continue; // explicit injection channel
      const folded = name.toLowerCase();
      for (const key of Object.keys(env)) {
        assert.notEqual(key.toLowerCase(), folded,
          `stripped credential ${name} must not appear under any casing`);
      }
    }
    for (const name of PI_CONFIG_ENV_VARS) {
      if (isForcedEnvName(name)) continue; // forced keys exist by design
      const folded = name.toLowerCase();
      for (const key of Object.keys(env)) {
        assert.notEqual(key.toLowerCase(), folded,
          `stripped config switch ${name} must not appear under any casing`);
      }
    }
  } finally {
    removeIsolationDirs(dirs);
  }
});

test('secret-shaped operator env names outside the 42-var list are stripped too (review P1)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  // The inherited environment is the FULL operator env: names that merely are
  // not on pi's own help list (CI/SCM tokens, service secrets) must not reach
  // the agent process, while OS/functional keys survive and the explicit
  // Host injection (DEEPSEEK_API_KEY) is applied AFTER stripping.
  const provider = makeProvider(t, {recordPath: record, inheritedExtra: {
    GITHUB_TOKEN: 'gh-operator', GH_HOST: 'github.com',
    WOODPECKER_TOKEN: 'wp-operator', MY_SERVICE_SECRET: 'service',
    CI_JOB_TOKEN: 'ci', NPM_AUTH: 'should-this-stay',
    DEPLOY_PASSWORD: 'pw', SSH_KEY_PATH: 'kept-but-harmless-path',
  }});
  const done = await runExecute(provider, {worktree}, {DEEPSEEK_API_KEY: 'explicit-key'});
  assert.equal(done.state, 'COMPLETED');
  const rec = lastRecord(t, record);
  const names = new Set(rec.env_names.map((n) => n.toLowerCase()));
  assert.equal(names.has('github_token'), false);
  assert.equal(names.has('woodpecker_token'), false);
  assert.equal(names.has('my_service_secret'), false);
  assert.equal(names.has('ci_job_token'), false);
  assert.equal(names.has('deploy_password'), false);
  assert.equal(names.has('deepseek_api_key'), true, 'explicit injection channel survives');
  assert.ok(names.has('path'), 'OS functional keys survive');
  assert.ok(!names.has('github_token'), 'no case tricks');
});

test('credential values never enter argv or evidence', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree},
    {DEEPSEEK_API_KEY: 'explicit-secret-value-123'});
  assert.equal(done.state, 'COMPLETED');
  const rec = lastRecord(t, record);
  const argvText = JSON.stringify(rec.argv);
  assert.equal(argvText.includes('explicit-secret-value-123'), false);
  assert.equal(argvText.includes('--api-key'), false);
  assert.equal(JSON.stringify(done).includes('explicit-secret-value-123'), false);
});

// ---- exit-code classification -------------------------------------------------

test('keyless run (exit 1 + observed stderr) is a FAILED observation, not a throw', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree},
    {TALLY_FAKE_PI_REQUIRE_KEY: '1'}); // NO credential injected
  assert.equal(done.state, 'FAILED');
  assert.equal(done.reason, 'PI_EXIT_NONZERO');
  assert.equal(done.evidence.exitCode, 1);
  assert.match(done.evidence.stderrTail, /No API key found for deepseek\./);
});

test('non-zero exit after partial stream: FAILED, partial output never becomes the answer', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'exitfail'});
  assert.equal(done.state, 'FAILED');
  assert.equal(done.reason, 'PI_EXIT_NONZERO');
  assert.equal(done.evidence.exitCode, 7);
  assert.equal(done.evidence.stderrTail.includes('partial'), false);
});

test('exit 0 without a parseable answer fails safe (no guessed success)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'emptyok'});
  assert.equal(done.state, 'FAILED');
  assert.equal(done.reason, 'PI_CONTRACT_NO_ANSWER');
});

test('failure stderr with secret-shaped text is redacted in evidence', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await runExecute(provider, {worktree},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'leakstderr'});
  assert.equal(done.state, 'FAILED');
  assert.equal(done.reason, 'PI_EXIT_NONZERO');
  const evidenceText = JSON.stringify(done.evidence);
  assert.equal(evidenceText.includes('sk-fake-secret-1234567890abcdef'), false,
    'secret must not leak into evidence');
  assert.match(evidenceText, /\[REDACTED\]/);
});

test('hung process + timeout: tree killed, UNKNOWN with reason, no orphan left', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const donePromise = runExecute(provider, {worktree, timeoutMs: 400},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'hang'});
  const rec = await waitFor(() => {
    const rows = readRecords(record).filter((row) => row.child_pid > 0);
    return rows.length > 0 ? rows[rows.length - 1] : null;
  });
  assert.ok(pidAlive(rec.child_pid), 'child should be alive before the timeout fires');
  const done = await donePromise;
  assert.equal(done.state, 'UNKNOWN');
  assert.equal(done.reason, 'PI_EXEC_TIMEOUT');
  await waitFor(() => !pidAlive(rec.child_pid), {deadlineMs: 10000});
  assert.equal(pidAlive(rec.child_pid), false, 'no orphan process may survive');
  // Isolation dirs are cleaned even on the UNKNOWN path.
  await waitFor(() => !existsSync(rec.pi_agent_dir), {deadlineMs: 10000});
});

// ---- cancellation ---------------------------------------------------------------

test('AbortSignal mid-run: CANCELLED, tree terminated, no orphan', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const controller = new AbortController();
  const donePromise = provider.execute(
    makeRequest({worktree, prompt: 'x'}),
    {env: {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'hang'}, signal: controller.signal},
  );
  const rec = await waitFor(() => {
    const rows = readRecords(record).filter((row) => row.child_pid > 0);
    return rows.length > 0 ? rows[rows.length - 1] : null;
  });
  controller.abort();
  const done = await donePromise;
  assert.equal(done.state, 'CANCELLED');
  assert.equal(done.reason, 'PI_EXEC_CANCELLED');
  await waitFor(() => !pidAlive(rec.child_pid), {deadlineMs: 10000});
});

test('pre-aborted signal: CANCELLED before start, nothing spawns', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const done = await provider.execute(
    makeRequest({worktree}),
    {env: {DEEPSEEK_API_KEY: 'test-key'}, signal: AbortSignal.abort()},
  );
  assert.equal(done.state, 'CANCELLED');
  assert.equal(done.reason, 'PI_CANCELLED_BEFORE_START');
  assertNoPrintSpawn(record);
});

test('cancel(attempt pin): CANCEL_REQUESTED then CANCELLED; afterwards NOT_RUNNING', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const {createProviderAttemptPin} = await import('../../tally-agent-provider/src/index.mjs');
  const provider = makeProvider(t, {recordPath: record});
  const pin = createProviderAttemptPin({
    taskId: 't1', attemptId: 'a1', sourceSha: SOURCE_SHA, providerId: 'pi',
  });
  const donePromise = runExecute(provider, {worktree},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_SCENARIO: 'hang'});
  await waitFor(() => readRecords(record).some((row) => row.child_pid > 0));
  const cancelResult = await provider.cancel(pin);
  assert.equal(cancelResult.state, 'CANCEL_REQUESTED');
  const done = await donePromise;
  assert.equal(done.state, 'CANCELLED');
  assert.equal(done.reason, 'PI_EXEC_CANCELLED');
  assert.equal((await provider.cancel(pin)).state, 'NOT_RUNNING');
  const unknownPin = createProviderAttemptPin({
    taskId: 't1', attemptId: 'no-such-attempt', sourceSha: SOURCE_SHA, providerId: 'pi',
  });
  assert.equal((await provider.cancel(unknownPin)).state, 'NOT_RUNNING');
});

// ---- execute-level gate failures (no spawn) -------------------------------------

test('execute refuses missing binary / version mismatch without spawning a turn', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const missing = createPiAgentProvider({
    binPath: path.join(dir, 'no-such-pi'),
    inheritedEnv: testInheritedEnv(record),
  });
  assert.equal((await runExecute(missing, {worktree})).reason, 'PI_BINARY_MISSING');

  const mismatch = makeProvider(t, {recordPath: record});
  const done = await runExecute(mismatch, {worktree},
    {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_VERSION: '0.99.0'});
  assert.equal(done.state, 'FAILED');
  assert.equal(done.reason, 'PI_VERSION_PIN_MISMATCH');
  assertNoPrintSpawn(record);
});

test('execute validates prompt and workspace before any process', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  // Contract-legal shape, adapter-refined illegality: a whitespace-only
  // prompt is still a string, so it reaches the adapter gate.
  assert.equal((await runExecute(provider, {prompt: '   ', worktree})).reason,
    'PI_PROMPT_INVALID');
  // Contract boundary: a malformed binding (empty worktreePath) throws
  // TypeError before the adapter is ever reached.
  await assert.rejects(() => runExecute(provider, {prompt: 'x', worktree: ''}),
    {name: 'TypeError', message: 'PROVIDER_WORKTREE_PATH_INVALID'});
  assertNoPrintSpawn(record);
});

// ---- per-attempt isolation lifecycle ----------------------------------------------

test('each attempt gets a fresh isolation group and it is removed after the attempt', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const first = await runExecute(provider, {worktree}, {DEEPSEEK_API_KEY: 'test-key'});
  const second = await runExecute(provider, {worktree, attemptId: 'a2'},
    {DEEPSEEK_API_KEY: 'test-key'});
  assert.equal(first.state, 'COMPLETED');
  assert.equal(second.state, 'COMPLETED');
  const rows = readRecords(record).filter((row) => row.kind === 'print');
  assert.equal(rows.length, 2);
  const roots = rows.map((row) => path.dirname(row.pi_agent_dir));
  assert.notEqual(roots[0], roots[1], 'attempts must not share isolation groups');
  for (const row of rows) {
    assert.equal(existsSync(row.pi_agent_dir), false, 'agent-config removed after attempt');
    assert.equal(existsSync(path.dirname(row.pi_agent_dir)), false, 'isolation root removed after attempt');
  }
});

test('execute gates itself on the cached version probe (no extra --version after probe)', async (t) => {
  const dir = newTestDir(t);
  const record = path.join(dir, 'calls.jsonl');
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const provider = makeProvider(t, {recordPath: record});
  const env = {DEEPSEEK_API_KEY: 'test-key', TALLY_FAKE_PI_RECORD: record};
  const probe = await provider.probe(hostContext(env));
  assert.equal(probe.state, 'AVAILABLE');
  const done = await runExecute(provider, {worktree}, env);
  assert.equal(done.state, 'COMPLETED');
  const versionRows = readRecords(record).filter((row) => row.kind === 'version');
  assert.equal(versionRows.length, 1);
});

// ---- reconcile ---------------------------------------------------------------------

test('reconcile is always UNKNOWN/NOT_RECONCILED (no second persistence layer)', async (t) => {
  const dir = newTestDir(t);
  const provider = makeProvider(t, {});
  const pin = createProviderAttemptPin({
    taskId: 't1', attemptId: 'a1', sourceSha: SOURCE_SHA, providerId: 'pi',
  });
  const result = await provider.reconcile(pin);
  assert.equal(result.state, 'UNKNOWN');
  assert.equal(result.reason, 'NOT_RECONCILED');
});

// ---- factory validation -------------------------------------------------------------

test('factory refuses invalid configuration before any operation', (t) => {
  const dir = newTestDir(t);
  const base = {binPath: process.execPath, programArgs: [fakeScript]};
  assert.throws(() => createPiAgentProvider({...base, binPath: '   '}), /PI_BIN_PATH_REQUIRED/);
  assert.throws(() => createPiAgentProvider({...base, outputMode: 'rpc'}), /PI_OUTPUT_MODE_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, outputMode: 'JSON'}), /PI_OUTPUT_MODE_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, thinkingLevel: 'ultra'}), /PI_THINKING_LEVEL_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, provider: 'deepseek official'}), /PI_PROVIDER_TOKEN_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, model: "bad'model"}), /PI_MODEL_TOKEN_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, timeoutMs: 0}), /PI_TIMEOUT_INVALID/);
  assert.throws(() => createPiAgentProvider({...base, binPath: undefined}), /PI_BIN_PATH_REQUIRED/);
});

test('buildPiCliArgs assembles the hardened argv shape with the trailing `--` separator', () => {
  const args = buildPiCliArgs({sessionDir: 'S', instruction: 'do it'});
  assert.deepEqual(args, [
    '-p', '--mode', 'json',
    '--no-session', '--session-dir', 'S',
    '--no-extensions', '--no-skills', '--no-prompt-templates',
    '--no-themes', '--no-context-files', '--no-approve',
    '--', 'do it',
  ]);
  const withFlags = buildPiCliArgs({
    sessionDir: 'S', provider: 'deepseek', model: 'deepseek-v4-flash',
    thinkingLevel: 'high', instruction: 'x',
  });
  assert.deepEqual(withFlags.slice(12, 18),
    ['--provider', 'deepseek', '--model', 'deepseek-v4-flash', '--thinking', 'high']);
  assert.equal(withFlags[withFlags.length - 1], 'x');
  assert.throws(() => buildPiCliArgs({sessionDir: 'S', outputMode: 'rpc', instruction: 'x'}),
    /PI_OUTPUT_MODE_INVALID/);
  assert.throws(() => buildPiCliArgs({sessionDir: 'S', instruction: ''}), /PI_INSTRUCTION_INVALID/);
});

// ---- NDJSON parsing (pure function surface) -------------------------------------------

test('parsePiStream: session observed, truncated tail skipped, no assistant means no success', () => {
  const partial = `{"type":"session","version":3,"id":"${FAKE_SESSION_ID}","cwd":"C:\\\\ws"}\n`
    + '{"type":"agent_start"}\n'
    + '{"type":"message_update","assistantMessageEvent":{"delta":"trunc';
  const obs = parsePiStream(partial);
  assert.equal(obs.sessionHandle, FAKE_SESSION_ID);
  assert.equal(obs.hasAssistant, false);
  assert.equal(obs.usage, null);
});

test('parsePiStream: last text-bearing assistant message_end wins; noise and user rows ignored', () => {
  const full = [
    `{"type":"session","version":3,"id":"${FAKE_SESSION_ID}","cwd":"C:\\\\ws"}`,
    '{"type":"message_end","message":{"role":"user","content":[{"type":"text","text":"q"}]}}',
    '{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"part1"}],"provider":"p1","model":"m1","usage":{"input":1,"output":2},"stopReason":"stop"}}',
    'not json at all',
    '{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"final"},{"type":"text","text":"part2"}],"provider":"p2","model":"m2","usage":{"input":5,"output":7,"cacheRead":9},"stopReason":"stop"}}',
    '{"type":"agent_settled"}',
  ].join('\n');
  const obs = parsePiStream(full);
  assert.equal(obs.hasAssistant, true);
  assert.equal(obs.text, 'final\npart2');
  assert.equal(obs.model, 'm2');
  assert.equal(obs.provider, 'p2');
  assert.deepEqual(obs.usage, {input: 5, output: 7, cacheRead: 9});
});

test('parsePiStream: empty and truncated-only inputs yield no success', () => {
  for (const input of ['', '\n', '{"type":"session"', '{"type":"session"}']) {
    const obs = parsePiStream(input);
    assert.equal(obs.hasAssistant, false, `input ${JSON.stringify(input)} must not count as success`);
  }
});


test('completed JSON and text summaries redact secret-shaped agent output', async (t) => {
  const dir = newTestDir(t);
  const worktree = path.join(dir, 'ws');
  mkdirSync(worktree);
  const sample = 'sk-0123456789abcdefghijklmnopqrstuv';
  writeFileSync(path.join(worktree, 'secret-note.txt'), sample + '\n');
  for (const outputMode of ['json', 'text']) {
    const provider = makeProvider(t, {factory: {outputMode}});
    const result = await runExecute(provider, {
      prompt: 'read secret-note.txt', worktree,
      attemptId: 'secret-' + outputMode,
    }, {DEEPSEEK_API_KEY: 'test-key'});
    assert.equal(result.state, 'COMPLETED');
    assert.doesNotMatch(JSON.stringify(result.evidence), /sk-0123456789abcdefghijklmnopqrstuv/,
      'agent stdout must never echo secret-shaped material into Host observation');
    assert.match(result.evidence.summary, /\\[REDACTED\\]/);
  }
});
