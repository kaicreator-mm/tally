/**
 * Tally v2.1 candidate: AgentProvider/1 adapter for the OpenCode CLI
 * (opencode-ai@1.19.6), implemented in Node from the field-verified CLI
 * contract captured in PR #471. The Go runtime (runtimes/opencode) is INPUT
 * EVIDENCE ONLY — this package never ports or bridges the Go worker; v2
 * Tasks are executed exclusively through the CLI process surface below.
 *
 * CLI contract (all flags/env observed on opencode-ai@1.19.6, nothing invented):
 *   - one-shot turn: `opencode run --format json [--model provider/model]
 *     [--agent <name>] --pure "<instruction>"` (instruction last);
 *   - never passed: `--attach` (shared-server face), `--standalone` (absent
 *     in 1.19.6), `--auto`, and the OPENCODE_CONFIG env var (single config
 *     pipeline = flags only);
 *   - without --attach each run starts its own in-process temporary local
 *     server on a random port: single process tree, no residue after exit;
 *   - PWD trap: opencode resolves the project dir from the PWD env var, NOT
 *     from the process cwd (live-verified on Windows, 1.19.6) → PWD/OLDPWD
 *     are force-pinned to the workspace, last-wins over context.env;
 *   - isolation: XDG_CONFIG_HOME / XDG_DATA_HOME / XDG_CACHE_HOME /
 *     XDG_STATE_HOME (+ platform TMP/TEMP) redirected to a fresh per-attempt
 *     temp tree, removed when the run ends; owner config/auth.json/sessions
 *     are never read or written;
 *   - stdout: NDJSON envelope {type,timestamp,sessionID:"ses_*",part};
 *     summary = last `text` part; usage = Σ step_finish tokens; unknown
 *     types ignored; plain non-JSON stdout degrades to the raw text;
 *   - exit codes: 0 = success shape (completed only when an answer was
 *     observed — exit 0 with no output is a FAILED fail-safe), non-zero =
 *     FAILED, timeout = tree-killed UNKNOWN, AbortSignal/cancel = CANCELLED;
 *   - pinned-version identity gate: `--version` must print exactly
 *     1.19.6 (fail-closed); the argv/output contract is valid only for the
 *     pinned release;
 *   - credentials flow only through the Host-supplied context.env (e.g.
 *     DEEPSEEK_API_KEY). No MISSING_CREDENTIAL errors are invented: a
 *     keyless run falls back to the CLI's keyless default model and is
 *     classified by the real exit code/output.
 *
 * Contract discipline (packages/tally-agent-provider): observations only,
 * no authority fields, no fallback, UNKNOWN always carries a machine reason,
 * evidence is adapter-redacted (injected credential values and sk- tokens
 * are replaced before anything leaves the process boundary).
 *
 * Capabilities stay an honest empty set: the NDJSON stream is the stdout of
 * a one-shot process, not a subscribable session stream; every run is a new
 * session (no resume/fork); requested model/agent are expectations — the
 * effective model is not per-run observable. session.isolation stays
 * undeclared until the shared conformance canary passes (evidence gate).
 */

import {spawn} from 'node:child_process';
import {mkdir, mkdtemp, rm, stat} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider,
} from '../../tally-agent-provider/src/index.mjs';
import {createNdjsonEventSink} from './ndjson.mjs';

/** Provider id (selection vocabulary only; no control flow branches on it). */
export const OPENCODE_PROVIDER_ID = 'opencode';

/** Pinned upstream release the argv/output contract is anchored to. */
export const OPENCODE_PINNED_VERSION = '1.19.6';

const FLAG_TOKEN = /^[A-Za-z0-9._-]+$/;
const REDACT_MARKER = '[REDACTED]';
const SK_PATTERN = /sk-[A-Za-z0-9]{16,}/g;
const STDOUT_CAP = 64 * 1024;
const STDERR_CAP = 8 * 1024;
const DEFAULT_WAIT_DELAY_MS = 5000;
const VERSION_PROBE_TIMEOUT_MS = 15000;
const ISOLATION_SUBDIRS = ['config', 'data', 'cache', 'state', 'tmp'];

function fail(reason) { throw new TypeError(`OPENCODE_${reason}`); }
function isPlainObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

// Observed flag value shapes: model = "provider/model" (exactly one '/', both
// sides non-empty), agent = single token. Restricted to argv-safe characters;
// suspicious values are rejected, not escaped.
function validateModelFlag(model) {
  if (model === null || model === undefined) return;
  if (typeof model !== 'string' || model === '') fail('MODEL_FLAG_INVALID');
  const slashAt = model.indexOf('/');
  const provider = model.slice(0, slashAt);
  const spec = slashAt === -1 ? '' : model.slice(slashAt + 1);
  if (slashAt === -1 || provider === '' || spec === '' || spec.includes('/')) {
    fail('MODEL_FLAG_INVALID');
  }
  if (!FLAG_TOKEN.test(provider) || !FLAG_TOKEN.test(spec)) fail('MODEL_FLAG_UNSAFE');
}

function validateAgentFlag(agent) {
  if (agent === null || agent === undefined) return;
  if (typeof agent !== 'string' || agent === '' || !FLAG_TOKEN.test(agent)) {
    fail('AGENT_FLAG_INVALID');
  }
}

// Values the Host injected as credentials never appear in observations.
function collectSecrets(env) {
  const secrets = new Set();
  if (!isPlainObject(env)) return [];
  for (const [key, value] of Object.entries(env)) {
    if (typeof value !== 'string' || value.length < 8) continue;
    if (key === 'DEEPSEEK_API_KEY' ||
        /(API_?KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)/i.test(key)) {
      secrets.add(value);
    }
  }
  return [...secrets];
}

function redact(text, secrets) {
  let out = String(text ?? '');
  for (const secret of secrets) out = out.split(secret).join(REDACT_MARKER);
  return out.replace(SK_PATTERN, REDACT_MARKER);
}

const unknown = (reason, evidence) => ({state: 'UNKNOWN', reason, evidence});
const failed = (reason, evidence) => ({state: 'FAILED', reason, evidence});

// Whole-tree termination: Windows taskkill /T /F (observed zero-residue),
// POSIX detached process group SIGKILL.
function killTree(child) {
  return new Promise(resolve => {
    if (!child || child.pid === undefined ||
        (child.exitCode !== null || child.signalCode !== null)) {
      resolve();
      return;
    }
    if (process.platform === 'win32') {
      const killer = spawn('taskkill', ['/pid', String(child.pid), '/T', '/F'],
        {windowsHide: true, stdio: 'ignore'});
      killer.once('close', () => resolve());
      killer.once('error', () => {
        try { child.kill(); } catch { /* already gone */ }
        resolve();
      });
    } else {
      try { process.kill(-child.pid, 'SIGKILL'); }
      catch {
        try { child.kill('SIGKILL'); } catch { /* already gone */ }
      }
      resolve();
    }
  });
}

/**
 * Create the opencode AgentProvider/1 adapter.
 *
 * @param {object} options
 *   - cliPath {string} required: explicit, directly spawnable opencode
 *     executable path (no implicit discovery; test doubles welcome).
 *   - cliArgs {string[]} optional: arguments inserted before the subcommand
 *     (test seam; production uses cliPath directly with []).
 *   - pinnedVersion {string} optional, default '1.19.6'. Deliberately has no
 *     "off" value: the contract does not exist without a version anchor.
 *   - model {string} optional: `--model provider/model` expectation.
 *   - agent {string} optional: `--agent <name>` expectation.
 *   - attemptRecords {{get(attemptId): object|undefined}|Map} optional:
 *     injectable attempt-record lookup for reconcile (the adapter keeps no
 *     persistent execution journal; default reconcile is UNKNOWN/
 *     NOT_RECONCILED).
 *   - waitDelayMs {number} optional, default 5000: cap on how long a
 *     pipe-holding orphan may delay result collection after exit.
 */
export function createOpenCodeAgentProvider(options = {}) {
  const {
    cliPath,
    cliArgs = [],
    pinnedVersion = OPENCODE_PINNED_VERSION,
    model = null,
    agent = null,
    attemptRecords = null,
    waitDelayMs = DEFAULT_WAIT_DELAY_MS,
  } = options ?? {};

  if (typeof cliPath !== 'string' || cliPath === '') fail('CLI_PATH_REQUIRED');
  if (!Array.isArray(cliArgs) || cliArgs.some(a => typeof a !== 'string')) {
    fail('CLI_ARGS_INVALID');
  }
  if (typeof pinnedVersion !== 'string' || pinnedVersion === '') {
    fail('PINNED_VERSION_INVALID');
  }
  validateModelFlag(model);
  validateAgentFlag(agent);
  if (attemptRecords !== null &&
      (typeof attemptRecords !== 'object' || typeof attemptRecords.get !== 'function')) {
    fail('ATTEMPT_RECORDS_INVALID');
  }
  const waitDelay = Number.isFinite(waitDelayMs) && waitDelayMs > 0
    ? waitDelayMs : DEFAULT_WAIT_DELAY_MS;

  // Successful `--version` observations only (failures are never cached).
  let observedCliVersion = null;
  // In-flight execution handles — execution control for cancel(), not a
  // journal. Deleted the moment an execute settles.
  const inFlight = new Map();

  async function checkBinary() {
    try {
      const info = await stat(cliPath);
      return info.isFile() ? {ok: true} : {ok: false};
    } catch {
      return {ok: false};
    }
  }

  // 'close' can fire in the SAME tick as 'exit' (observed on Windows): the
  // listener must be attached at spawn time, never after awaiting exit.
  // Result collection is still capped by waitDelay so a pipe-holding orphan
  // can never delay it unboundedly (#471 WaitDelay discipline).
  function closeWithin(child, closed) {
    return new Promise(resolve => {
      let settled = false;
      const finish = () => {
        if (settled) return;
        settled = true;
        clearTimeout(guard);
        resolve();
      };
      const guard = setTimeout(() => {
        try { child.stdout?.destroy(); } catch { /* ignore */ }
        try { child.stderr?.destroy(); } catch { /* ignore */ }
        finish();
      }, waitDelay);
      closed.then(finish, finish);
    });
  }

  // Live `--version` observation (bare version line, exit 0 = observed shape).
  async function runVersionProbe(env) {
    let child;
    try {
      child = spawn(cliPath, [...cliArgs, '--version'],
        {env: {...env}, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe']});
    } catch {
      return {ok: false, reason: 'CLI_NOT_FOUND', evidence: {}};
    }
    const closed = new Promise(resolve => child.once('close', resolve));
    let out = '';
    child.stdout.setEncoding('utf8');
    child.stdout.on('data', chunk => {
      if (out.length < 4096) out += chunk;
    });
    const outcome = await new Promise(resolve => {
      const guard = setTimeout(() => {
        try { child.kill(); } catch { /* ignore */ }
      }, VERSION_PROBE_TIMEOUT_MS);
      child.once('exit', code => { clearTimeout(guard); resolve({code}); });
      child.once('error', () => { clearTimeout(guard); resolve({code: null}); });
    });
    await closeWithin(child, closed);
    const line = out.split(/\r?\n/).map(s => s.trim()).find(Boolean) ?? null;
    if (outcome.code === null || outcome.code !== 0 || line === null) {
      return {ok: false, reason: 'VERSION_UNOBSERVABLE', evidence: {exitCode: outcome.code}};
    }
    if (line !== pinnedVersion) {
      return {ok: false, reason: 'VERSION_MISMATCH',
        evidence: {observed: line, pinnedVersion}};
    }
    return {ok: true, version: line};
  }

  async function ensureCliVersion(env) {
    if (observedCliVersion !== null) return {ok: true, version: observedCliVersion};
    const observed = await runVersionProbe(env);
    if (!observed.ok) return observed;
    observedCliVersion = observed.version;
    return {ok: true, version: observed.version};
  }

  // VERSION gate first, then CREDENTIALS (explicit env only), per the #471
  // probe ladder. No real turn is made: the canary stays NOT_RUN.
  async function probe(context) {
    const env = isPlainObject(context?.env) ? context.env : {};
    const binary = await checkBinary();
    if (!binary.ok) {
      return {state: 'UNAVAILABLE', reason: 'CLI_NOT_FOUND',
        evidence: {cliPath, pinnedVersion}};
    }
    const identity = await ensureCliVersion(env);
    if (!identity.ok) {
      return {state: 'UNAVAILABLE', reason: identity.reason,
        evidence: {...(identity.evidence ?? {}), pinnedVersion}};
    }
    if (typeof env.DEEPSEEK_API_KEY !== 'string' || env.DEEPSEEK_API_KEY === '') {
      return {state: 'UNAVAILABLE', reason: 'CREDENTIALS_MISSING',
        evidence: {credentials: 'DEEPSEEK_API_KEY', present: false, canary: 'NOT_RUN'}};
    }
    return {state: 'AVAILABLE', reason: null,
      evidence: {cliVersion: identity.version, pinnedVersion,
        credentials: 'DEEPSEEK_API_KEY', present: true, canary: 'NOT_RUN'}};
  }

  function buildRunEnv(envIn, worktreePath, root) {
    // Host-supplied snapshot first; forced isolation keys last (last-wins).
    return {
      ...envIn,
      PWD: worktreePath,
      OLDPWD: worktreePath,
      XDG_CONFIG_HOME: join(root, 'config'),
      XDG_DATA_HOME: join(root, 'data'),
      XDG_CACHE_HOME: join(root, 'cache'),
      XDG_STATE_HOME: join(root, 'state'),
      TMP: join(root, 'tmp'),
      TEMP: join(root, 'tmp'),
    };
  }

  async function createAttemptDirs() {
    const root = await mkdtemp(join(tmpdir(), 'tally-opencode-attempt-'));
    await Promise.all(ISOLATION_SUBDIRS.map(
      sub => mkdir(join(root, sub), {recursive: true})));
    return root;
  }

  async function removeAttemptDirs(root) {
    if (!root) return;
    try {
      await rm(root, {recursive: true, force: true, maxRetries: 3, retryDelay: 100});
    } catch { /* best effort; OS temp cleanup is the backstop */ }
  }

  async function execute(request, context) {
    if (!isPlainObject(request) ||
        typeof request.prompt !== 'string' || request.prompt === '' ||
        typeof request.worktreePath !== 'string' || request.worktreePath === '' ||
        typeof request.attemptId !== 'string' || request.attemptId === '') {
      fail('EXECUTE_REQUEST_INVALID');
    }
    const envIn = isPlainObject(context?.env) ? context.env : {};
    const signal = context?.signal ?? null;
    const secrets = collectSecrets(envIn);
    const attemptKey = request.attemptId;

    // Version identity gate (fail-closed). Environment problems are UNKNOWN
    // observations, never a FAILED claim for the candidate.
    const binary = await checkBinary();
    if (!binary.ok) {
      return unknown('CLI_NOT_FOUND', {cliPath, pinnedVersion});
    }
    const identity = await ensureCliVersion(envIn);
    if (!identity.ok) {
      return unknown(identity.reason, {...(identity.evidence ?? {}), pinnedVersion});
    }

    if (signal?.aborted) {
      return {state: 'CANCELLED', reason: null, evidence: {durationMs: 0}};
    }
    if (inFlight.has(attemptKey)) {
      return unknown('ATTEMPT_IN_FLIGHT', {});
    }

    let root = null;
    try {
      root = await createAttemptDirs();
    } catch (error) {
      return unknown('ISOLATION_DIRS_FAILED',
        {detail: redact(String(error?.message ?? error), secrets)});
    }

    const env = buildRunEnv(envIn, request.worktreePath, root);
    const args = ['run', '--format', 'json'];
    if (model !== null) args.push('--model', model);
    if (agent !== null) args.push('--agent', agent);
    args.push('--pure', request.prompt);

    const startedAt = Date.now();
    const finish = async result => {
      await removeAttemptDirs(root);
      return result;
    };

    let child;
    try {
      child = spawn(cliPath, [...cliArgs, ...args], {
        cwd: request.worktreePath,
        env,
        windowsHide: true,
        detached: process.platform !== 'win32',
        stdio: ['ignore', 'pipe', 'pipe'],
      });
    } catch (error) {
      return finish(unknown('SPAWN_FAILED',
        {detail: redact(String(error?.message ?? error), secrets)}));
    }
    const entry = {child, cancelRequested: false, timedOut: false};
    inFlight.set(attemptKey, entry);
    const closed = new Promise(resolve => child.once('close', resolve));

    const sink = createNdjsonEventSink();
    let stdoutRaw = '';
    let stderrRaw = '';
    child.stdout.setEncoding('utf8');
    child.stdout.on('data', chunk => {
      sink.push(chunk);
      if (stdoutRaw.length < STDOUT_CAP) stdoutRaw += chunk;
    });
    child.stderr.setEncoding('utf8');
    child.stderr.on('data', chunk => {
      if (stderrRaw.length < STDERR_CAP * 2) stderrRaw += chunk;
    });

    const onAbort = () => {
      entry.cancelRequested = true;
      void killTree(child);
    };
    if (signal) signal.addEventListener('abort', onAbort, {once: true});

    let timer = null;
    if (typeof request.timeoutMs === 'number' &&
        Number.isFinite(request.timeoutMs) && request.timeoutMs > 0) {
      timer = setTimeout(() => {
        entry.timedOut = true;
        void killTree(child);
      }, request.timeoutMs);
    }

    let exitCode = null;
    let exitSignal = null;
    try {
      [exitCode, exitSignal] = await new Promise(resolve => {
        child.once('exit', (code, name) => resolve([code, name]));
        child.once('error', () => resolve([null, null]));
      });
    } finally {
      if (timer !== null) clearTimeout(timer);
    }
    await closeWithin(child, closed);
    if (signal) signal.removeEventListener('abort', onAbort);
    inFlight.delete(attemptKey);

    const durationMs = Date.now() - startedAt;
    const snapshot = sink.snapshot();

    if (entry.timedOut) {
      return finish(unknown('TIMEOUT_KILLED',
        {timeoutMs: request.timeoutMs, durationMs}));
    }
    if (entry.cancelRequested) {
      return finish({state: 'CANCELLED', reason: null, evidence: {durationMs}});
    }
    if (exitCode === null) {
      return finish(unknown('SPAWN_FAILED', {durationMs}));
    }
    if (exitCode !== 0) {
      // Failure claims never carry stdout partial text as an answer; only a
      // redacted stderr excerpt travels with the observation.
      return finish(failed('EXIT_NONZERO', {
        exitCode,
        terminatedBy: exitSignal ?? null,
        durationMs,
        stderr: stderrRaw === '' ? null : redact(stderrRaw.slice(0, STDERR_CAP), secrets),
      }));
    }
    const answer = snapshot.sawJSON
      ? snapshot.summary
      : (stdoutRaw.trim() === '' ? null : stdoutRaw.trim());
    if (answer === null || answer === '') {
      // exit 0 without any observed answer: the stdout contract was not
      // honored — fail-safe, never guess success.
      return finish(failed('NO_OUTPUT', {exitCode: 0, durationMs}));
    }
    return finish({state: 'COMPLETED', reason: null, evidence: {
      exitCode: 0,
      durationMs,
      summary: redact(answer, secrets),
      usage: snapshot.usage,
      sessionId: snapshot.sessionId,
      degraded: !snapshot.sawJSON,
    }});
  }

  async function cancel(attempt) {
    const id = typeof attempt?.attemptId === 'string' ? attempt.attemptId : null;
    const entry = id === null ? undefined : inFlight.get(id);
    if (!entry) return {state: 'NOT_RUNNING', reason: null, evidence: null};
    entry.cancelRequested = true;
    await killTree(entry.child);
    return {state: 'CANCEL_REQUESTED', reason: null, evidence: {attemptId: id}};
  }

  async function reconcile(attempt) {
    const id = typeof attempt?.attemptId === 'string' ? attempt.attemptId : null;
    if (id !== null && inFlight.has(id)) {
      return {state: 'RUNNING', reason: null, evidence: {observedBy: 'in-process'}};
    }
    if (attemptRecords !== null && id !== null) {
      const record = attemptRecords.get(id);
      if (record !== undefined && record !== null) return record;
    }
    // No persistent execution journal exists (the contract registry keeps
    // none either): an unknown attempt is honestly NOT reconciled.
    return {state: 'UNKNOWN', reason: 'NOT_RECONCILED', evidence: null};
  }

  return defineAgentProvider({
    id: OPENCODE_PROVIDER_ID,
    contractVersion: AGENT_PROVIDER_CONTRACT_VERSION,
    capabilities: [],
    probe,
    execute,
    cancel,
    reconcile,
  });
}
