/**
 * Tally AgentProvider/1 Node adapter for the xAI official Grok Build CLI
 * (`@xai-official/grok`, native Rust binary; bin: `grok`).
 *
 * Every CLI surface used here was OBSERVED on the pinned binary
 * grok 1.0.87 (commit 929e6dd187b1, win32-x64, observed 2026-11-19) and is
 * carried by PR #468 (`runtimes/grok/runtime.go` + `isolation.go` +
 * `testdata/fakegrok/main.go`). No flag or protocol is invented here;
 * unobserved surfaces stay unused (runtime lock).
 *
 * Observed headless contract (#468):
 *   - One execute = one one-shot process:
 *       grok --no-auto-update [-m <model>] [--permission-mode <mode>] --single=<instruction>
 *     The attached `--single=<text>` form is mandatory: a bare `-p "-dash"`
 *     value is misparsed by clap as an unknown flag -> exit 2, while
 *     `--single="-leading dash task"` is accepted as a value for any
 *     instruction text. Instruction is always a single argv element; no shell.
 *   - stdout = final answer (plain output). exit 0 = completed; exit 1 =
 *     runtime error (auth failures included); exit 2 = usage error.
 *   - Auth = XAI_API_KEY environment variable. Keyless observed shape:
 *     empty stdout, stderr "Not signed in. ...", exit 1.
 *   - `--no-auto-update` is mandatory for scripted environments (official
 *     headless guidance; hidden flag observed accepted). Auto-update would
 *     write a new binary into GROK_HOME/bin - never allowed on candidate
 *     execution.
 *   - Isolation: GROK_HOME redirects the per-attempt home (sessions,
 *     config.toml, leader socket all live under it; an empty temp home was
 *     observed usable with no ~/.grok writes). Spawn env is a strict
 *     ALLOWLIST of OS base keys: no other owner-environment credential
 *     (e.g. DEEPSEEK_API_KEY) can structurally reach the spawned agent.
 *   - Version trap: npm dist-tag latest is stale (0.1.4) while the real line
 *     is 1.0.x -> the adapter pins 1.0.87 and fails closed on mismatch.
 *
 * Honesty stance (this repo, this machine): there are NO real Grok
 * credentials here, so the provider ships DISABLED/UNAVAILABLE. The keyless
 * refusal path is a real, asserted behaviour (typed refusal before any
 * spawn) - never a fabricated COMPLETED. REAL_EXECUTION stays NOT_RUN until
 * a credentialed deployment runs a real turn (see README.md).
 *
 * Contract discipline: observations only, no authority fields, no fallback,
 * every UNKNOWN carries a reason; `COMPLETED` is a provider claim
 * (INV-AUTH-01), never a Task/Review/Release fact.
 */
import {spawn} from 'node:child_process';
import {mkdtemp, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {defineAgentProvider} from '../../tally-agent-provider/src/index.mjs';

/** Provider id inside the AgentProvider/1 registry. */
export const GROK_PROVIDER_ID = 'grok';

/**
 * Upstream exact version this adapter's argv/exit-code contract is anchored
 * to (npm @xai-official/grok@1.0.87 native binary, commit 929e6dd187b1).
 * There is deliberately no "turn the pin off" value: a fast-moving surface
 * (242 npm releases in ~13 months; dist-tag latest observed stale) has no
 * contract without a version anchor.
 */
export const PINNED_GROK_VERSION = '1.0.87';

/**
 * --permission-mode values, verbatim from `grok 1.0.87 --help`
 * ("[possible values: default, acceptEdits, auto, dontAsk,
 * bypassPermissions, plan]"). Runtime semantics of each value were NOT
 * observed (no credentials on this machine): the adapter only validates and
 * passes the enum through, it never interprets it.
 */
export const GROK_PERMISSION_MODES = Object.freeze([
  'default', 'acceptEdits', 'auto', 'dontAsk', 'bypassPermissions', 'plan',
]);

/**
 * Spawn-environment allowlist, carried 1:1 from #468 isolation.go: OS base
 * keys only, copied from the owner environment when present. Everything else
 * in the owner environment - including every other provider's credentials -
 * structurally cannot reach the spawned agent or its tool subprocesses.
 * PATH lets the agent run git/node etc.; HOME/USERPROFILE/HOMEDRIVE/HOMEPATH
 * let its tools resolve home; the rest is Windows/POSIX system surface.
 * Both platforms' keys live in one list, copied only when present.
 */
const ENV_ALLOWLIST = Object.freeze([
  // Process search and system surface (both platforms).
  'PATH', 'OS', 'LANG', 'LC_ALL',
  'NUMBER_OF_PROCESSORS', 'PROCESSOR_ARCHITECTURE',
  'PROCESSOR_IDENTIFIER', 'PROCESSOR_LEVEL', 'PROCESSOR_REVISION',
  // Windows system surface.
  'SYSTEMROOT', 'SYSTEMDRIVE', 'WINDIR', 'COMSPEC', 'PATHEXT', 'TEMP', 'TMP',
  // Home resolution (Windows: USERPROFILE/HOMEDRIVE/HOMEPATH; POSIX: HOME).
  'HOME', 'USERPROFILE', 'HOMEDRIVE', 'HOMEPATH',
  // POSIX temp dir.
  'TMPDIR',
]);

const GROK_HOME_ENV_KEY = 'GROK_HOME';
const IS_WINDOWS = process.platform === 'win32';
const SECRET_KEY_PATTERN = /(?:API_?KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL)/i;
const SUMMARY_MAX_CHARS = 400;
// Bound on "process exited but output pipes are still held by an orphaned
// subprocess" (the WaitDelay equivalent from #468): after the run process
// exits we wait this long for stdio to close, then settle with what we have.
const PIPE_GRACE_MS = 5_000;
const PROBE_TIMEOUT_MS = 30_000;

/**
 * Typed refusal: thrown BEFORE any spawn when a precondition is not met
 * (currently: no XAI_API_KEY in the trusted Host context). Callers can
 * branch on `err instanceof GrokProviderRefusalError` / `err.code` instead
 * of string matching. A refusal is not an execution observation - the
 * adapter never produces a COMPLETED/FAILED claim for a run it refused to
 * start.
 */
export class GrokProviderRefusalError extends Error {
  /**
   * @param {string} code stable machine reason (e.g. GROK_NOT_AUTHENTICATED)
   * @param {string} message human-readable explanation
   */
  constructor(code, message) {
    super(message);
    this.name = 'GrokProviderRefusalError';
    this.code = code;
  }
}

function fail(message) { throw new TypeError(`tally-provider-grok: ${message}`); }

/**
 * Observed `grok --version` first line: "grok 1.0.87 (929e6dd187b1)".
 * Parsing fills only the recognizable fields (never guesses): the commit is
 * informational (not part of the pin check) and may be any token.
 */
export function parseGrokVersionLine(stdout) {
  const firstLine = String(stdout ?? '').split('\n', 1)[0]?.trim() ?? '';
  const match = /^grok (\S+?)(?:\s+\(([^)]+)\))?$/.exec(firstLine);
  if (!match) return {name: '', version: '', commit: '', observed: false};
  return {name: 'grok', version: match[1], commit: match[2] ?? '', observed: true};
}

function assertConfig(config) {
  if (!config || typeof config !== 'object') fail('config object required');
  if (typeof config.binPath !== 'string' || config.binPath.trim() === '')
    fail('config.binPath is explicitly required (npm dist-tag latest observed stale; a "grok" on PATH has uncontrolled provenance - no implicit discovery)');
  if (config.pinnedVersion !== undefined && config.pinnedVersion !== null &&
      (typeof config.pinnedVersion !== 'string' || config.pinnedVersion.trim() === ''))
    fail('config.pinnedVersion, when given, must be a non-empty string');
  if (config.model !== undefined && config.model !== null &&
      (typeof config.model !== 'string' || config.model.trim() === ''))
    fail('config.model must be a non-blank string or undefined (blank = never pass -m)');
  if (config.permissionMode !== undefined && config.permissionMode !== null &&
      !GROK_PERMISSION_MODES.includes(config.permissionMode))
    fail(`config.permissionMode must be one of the grok 1.0.87 --help observed values: ${GROK_PERMISSION_MODES.join(' | ')}; got ${JSON.stringify(config.permissionMode)}`);
  if (config.binArgs !== undefined && config.binArgs !== null &&
      (!Array.isArray(config.binArgs) || config.binArgs.some(a => typeof a !== 'string')))
    fail('config.binArgs, when given, must be an array of strings');
}

/**
 * argv for one headless run. Zero overlay: expectation flags are appended
 * only when configured; the instruction always rides the ATTACHED
 * `--single=<text>` form (single argv element, safe for any text including
 * leading dashes). A bare `-p <text>` form is never emitted (observed clap
 * misparse -> exit 2).
 */
function buildRunArgv(config, prompt) {
  const argv = [config.binPath, ...(config.binArgs ?? []), '--no-auto-update'];
  if (config.model) argv.push('-m', config.model);
  if (config.permissionMode) argv.push('--permission-mode', config.permissionMode);
  argv.push(`--single=${prompt}`);
  return argv;
}

/**
 * Spawn environment: allowlisted owner keys -> trusted Host context env ->
 * GROK_HOME forced to the per-attempt temp home. Construction order makes
 * the last write win: a GROK_HOME coming from the Host context env (or from
 * the owner environment, which is not allowlisted at all) can never defeat
 * the per-attempt home.
 */
function buildSpawnEnv(contextEnv, grokHome) {
  const env = {};
  for (const key of ENV_ALLOWLIST) {
    const value = process.env[key];
    if (value !== undefined && value !== '') env[key] = value;
  }
  for (const [key, value] of Object.entries(contextEnv ?? {})) {
    if (typeof value === 'string') env[key] = value;
  }
  env[GROK_HOME_ENV_KEY] = grokHome;
  return env;
}

/**
 * Replace every occurrence of secret-valued context env entries with
 * "[REDACTED]" and clamp length. Evidence crosses a durable boundary; an
 * external CLI may echo injected credentials into its diagnostics, so
 * redaction happens before anything is stored.
 */
function redactAndClamp(text, contextEnv) {
  let out = String(text ?? '');
  for (const [key, value] of Object.entries(contextEnv ?? {})) {
    if (SECRET_KEY_PATTERN.test(key) && typeof value === 'string' && value.length >= 4) {
      out = out.split(value).join('[REDACTED]');
    }
  }
  if (out.length > SUMMARY_MAX_CHARS) out = `${out.slice(0, SUMMARY_MAX_CHARS)}...`;
  return out;
}

/**
 * Kill the whole process tree. The real CLI spawns tool subprocesses; a
 * single-process kill would leave pipe-holding orphans behind (the #468
 * fakegrok long-run variant exists precisely to prove tree teardown).
 * Windows: `taskkill /pid <pid> /t /f`. POSIX: the run was spawned detached
 * (own process group), so kill the group, falling back to the single pid.
 */
function killProcessTree(pid) {
  if (!Number.isInteger(pid) || pid <= 0) return;
  if (IS_WINDOWS) {
    try {
      const killer = spawn('taskkill', ['/pid', String(pid), '/t', '/f'],
        {stdio: 'ignore', windowsHide: true});
      killer.on('error', () => {}); // best effort; bookkeeping proceeds
    } catch { /* best effort */ }
    return;
  }
  try { process.kill(-pid, 'SIGKILL'); }
  catch { try { process.kill(pid, 'SIGKILL'); } catch { /* already gone */ } }
}

function newAttemptKey(request) {
  return `${request?.taskId}:${request?.attemptId}`;
}

function credentialFrom(context) {
  const value = context?.env?.XAI_API_KEY;
  return (typeof value === 'string' && value.trim() !== '') ? value : null;
}

/** Spawn one child and collect its full outcome. Never rejects. */
function runChild({argv, cwd, env, timeoutMs, signal, onSpawn}) {
  return new Promise(resolve => {
    let timeoutTimer = null;
    let graceTimer = null;
    let child = null;
    let settled = false;
    let stdout = '';
    let stderr = '';
    let exitCode = null;
    let exitSignal = null;
    let spawnError = null;
    let timedOut = false;
    const settle = () => {
      if (settled) return;
      settled = true;
      if (timeoutTimer) clearTimeout(timeoutTimer);
      if (graceTimer) clearTimeout(graceTimer);
      if (signal) signal.removeEventListener('abort', onAbort);
      resolve({child, stdout, stderr, exitCode, exitSignal, spawnError, timedOut});
    };
    const terminate = () => { if (child?.pid) killProcessTree(child.pid); };
    const onAbort = () => terminate();
    if (timeoutMs > 0) {
      timeoutTimer = setTimeout(() => { timedOut = true; terminate(); }, timeoutMs);
    }
    if (signal) {
      if (signal.aborted) onAbort();
      else signal.addEventListener('abort', onAbort, {once: true});
    }
    try {
      child = spawn(argv[0], argv.slice(1), {
        cwd,
        env,
        stdio: ['ignore', 'pipe', 'pipe'],
        detached: !IS_WINDOWS,
        windowsHide: true,
      });
    } catch (error) {
      spawnError = error;
      resolve({child: null, stdout: '', stderr: '', exitCode: null, exitSignal: null,
        spawnError, timedOut});
      return;
    }
    if (onSpawn) onSpawn(child);
    child.stdout.on('data', chunk => { stdout += chunk; });
    child.stderr.on('data', chunk => { stderr += chunk; });
    child.on('error', error => { spawnError = error; settle(); });
    child.on('exit', (code, signalName) => {
      exitCode = code;
      exitSignal = signalName;
      // WaitDelay equivalent: give held pipes a bounded grace period, then
      // settle with the collected output anyway.
      graceTimer = setTimeout(settle, PIPE_GRACE_MS);
    });
    child.on('close', () => settle());
  });
}

/**
 * Build the grok AgentProvider/1 adapter.
 *
 * @param {object} config
 * @param {string} config.binPath explicit path to the grok executable.
 *   Required - no implicit discovery (npm dist-tag latest observed stale at
 *   0.1.4 while the real line is 1.0.x; a PATH "grok" has uncontrolled
 *   provenance).
 * @param {string[]} [config.binArgs] leading argv elements inserted before
 *   the grok flags. Test-injection seam ONLY (lets the stand-in CLI be an
 *   interpreted script: binPath = node executable, binArgs = [script]).
 *   Production must leave it empty so argv[0] is the native binary itself.
 * @param {string} [config.pinnedVersion] defaults to PINNED_GROK_VERSION;
 *   probe fails closed when the observed `--version` differs.
 * @param {string} [config.model] desired model passed verbatim via the
 *   observed `-m <model>` flag; undefined = flag not sent (zero overlay).
 *   Expectation only - the plain headless surface never reports the
 *   effective model, and expected config is never backfilled into any
 *   observed field.
 * @param {string} [config.permissionMode] one of GROK_PERMISSION_MODES,
 *   passed via the observed `--permission-mode <mode>` flag; undefined =
 *   flag not sent (CLI default).
 * @returns {Readonly<object>} frozen AgentProvider (id 'grok',
 *   contractVersion 1, capabilities: [] - the honest empty set: the one-shot
 *   `--single` surface supports no streaming, resume, fork, structured
 *   output, tool approval or native sandbox that has been actually
 *   observed).
 */
export function createGrokAgentProvider(config) {
  assertConfig(config);
  const pinnedVersion = config.pinnedVersion ?? PINNED_GROK_VERSION;
  // Key `${taskId}:${attemptId}` -> in-flight child. Process-lifetime
  // coordination for cancel() only; explicitly NOT an execution journal.
  const inFlight = new Map();

  /**
   * VERSION probe step: observe `grok --version`, compare against the pin,
   * fail closed on any unobservability (this is a deliberate, documented
   * divergence from best-effort probes: the argv/exit-code contract only
   * holds for the pinned version of a fast-moving surface).
   */
  async function probeVersion(contextEnv) {
    // Dedicated throwaway home for the probe: the observed safe shape is an
    // empty redirected GROK_HOME (--version exit 0, nothing written); the
    // real ~/.grok on the owner machine is never touched. The trusted Host
    // context env rides along (the #468 probe env = adapter env, same trust
    // anchor: the pinned binary itself).
    const probeHome = await mkdtemp(path.join(tmpdir(), 'tally-grok-probe-home-'));
    let result;
    try {
      result = await runChild({
        argv: [config.binPath, ...(config.binArgs ?? []), '--version'],
        env: buildSpawnEnv(contextEnv, probeHome),
        timeoutMs: PROBE_TIMEOUT_MS,
      });
    } finally {
      await rm(probeHome, {recursive: true, force: true}).catch(() => {});
    }
    if (result.timedOut) {
      return {ok: false, reason: 'GROK_PROBE_TIMEOUT',
        evidence: {step: 'VERSION', timedOut: true}};
    }
    if (result.spawnError) {
      return {ok: false, reason: 'GROK_PROBE_FAILED',
        evidence: {step: 'VERSION', error: result.spawnError.code ?? 'SPAWN_ERROR'}};
    }
    if (result.exitCode !== 0) {
      return {ok: false, reason: 'GROK_PROBE_FAILED',
        evidence: {step: 'VERSION', exitCode: result.exitCode,
          stderr: redactAndClamp(result.stderr, undefined)}};
    }
    const identity = parseGrokVersionLine(result.stdout);
    if (!identity.observed) {
      return {ok: false, reason: 'GROK_PROBE_FAILED',
        evidence: {step: 'VERSION', problem: 'VERSION_UNPARSEABLE',
          stdout: redactAndClamp(result.stdout, undefined)}};
    }
    if (identity.version !== pinnedVersion) {
      return {ok: false, reason: 'GROK_VERSION_MISMATCH',
        evidence: {step: 'VERSION', observed: identity.version, pinned: pinnedVersion}};
    }
    return {ok: true, reason: null, evidence: {step: 'VERSION', observed: identity.version}};
  }

  return defineAgentProvider({
    id: GROK_PROVIDER_ID,
    contractVersion: 1,
    // Honest capability set: empty. One-shot `--single` with plain stdout
    // gives no observed event vocabulary, resume/fork handle, structured
    // output, approval channel, sandbox profile or effective-model report.
    // (#468 declares none either; declarable capabilities appear only after
    // shared conformance canary evidence exists.)
    capabilities: [],

    /**
     * VERSION (pinned 1.0.87, fail-closed) -> CREDENTIALS (trusted Host
     * context must carry XAI_API_KEY). No canary step: this machine holds no
     * real Grok credential, and a canary without credentials would be
     * theatre. Default result on this deployment is therefore UNAVAILABLE.
     */
    async probe(context) {
      const version = await probeVersion(context?.env);
      if (!version.ok) {
        return {state: 'UNAVAILABLE', reason: version.reason, evidence: version.evidence};
      }
      if (!credentialFrom(context)) {
        return {state: 'UNAVAILABLE', reason: 'GROK_NOT_AUTHENTICATED',
          evidence: {...version.evidence, step: 'CREDENTIALS', credential: 'missing'}};
      }
      return {state: 'AVAILABLE', reason: null,
        evidence: {...version.evidence, step: 'CREDENTIALS', credential: 'present'}};
    },

    async execute(request, context) {
      if (!request || typeof request !== 'object') fail('execute request object required');
      if (typeof request.prompt !== 'string' || request.prompt.trim() === '')
        fail('execute request.prompt must be a non-blank string');
      if (typeof request.worktreePath !== 'string' || request.worktreePath.trim() === '')
        fail('execute request.worktreePath must be a non-blank string (workspace is pinned by spawn cwd)');
      // Typed refusal BEFORE any spawn: without a Host-supplied XAI_API_KEY
      // the observed CLI contract is a guaranteed auth failure (exit 1,
      // "Not signed in"). This deployment has no credential, so refusing is
      // the honest behaviour - never a fabricated COMPLETED.
      if (!credentialFrom(context)) {
        throw new GrokProviderRefusalError('GROK_NOT_AUTHENTICATED',
          'grok provider refused to spawn: no XAI_API_KEY in the trusted Host context (observed keyless contract: exit 1, "Not signed in"); provision credentials and explicitly enable this provider first');
      }
      if (context?.signal?.aborted) {
        return {state: 'CANCELLED', reason: null,
          evidence: {cancelled: true, cause: 'abort-signal', spawned: false}};
      }

      const grokHome = await mkdtemp(path.join(tmpdir(), 'tally-grok-home-'));
      const argv = buildRunArgv(config, request.prompt);
      const spawnEnv = buildSpawnEnv(context.env, grokHome);
      const startedAtMs = Date.now();
      const attemptKey = newAttemptKey(request);
      let cancelRequested = false;
      let child = null;

      const outcome = await runChild({
        argv,
        cwd: request.worktreePath,
        env: spawnEnv,
        timeoutMs: request.timeoutMs > 0 ? request.timeoutMs : 0,
        signal: context?.signal,
        onSpawn: spawned => {
          child = spawned;
          // Registered at spawn time so a cancel() racing the run finds the
          // live child (no execution journal: the entry is dropped as soon
          // as the run settles).
          inFlight.set(attemptKey, {
            requestCancel: () => {
              cancelRequested = true;
              killProcessTree(child?.pid);
            },
            child,
          });
        },
      });
      inFlight.delete(attemptKey);
      try {
        await rm(grokHome, {recursive: true, force: true}); // best effort
      } catch { /* teardown is best effort by policy */ }

      const durationMs = Date.now() - startedAtMs;
      const baseEvidence = {exitCode: outcome.exitCode ?? null, durationMs};

      if (outcome.spawnError && outcome.exitCode === null && outcome.stdout === '' &&
          outcome.stderr === '') {
        return {state: 'FAILED', reason: 'GROK_SPAWN_FAILED',
          evidence: {error: outcome.spawnError.code ?? 'SPAWN_ERROR'}};
      }
      if (outcome.timedOut) {
        // Killed for exceeding the time budget: the run neither completed nor
        // observably failed - UNKNOWN with the mandatory reason, tree killed.
        return {state: 'UNKNOWN', reason: 'GROK_RUN_TIMEOUT',
          evidence: {...baseEvidence, timedOut: true, timeoutMs: request.timeoutMs}};
      }
      if (context?.signal?.aborted) {
        return {state: 'CANCELLED', reason: null,
          evidence: {...baseEvidence, cancelled: true, cause: 'abort-signal'}};
      }
      if (cancelRequested) {
        return {state: 'CANCELLED', reason: null,
          evidence: {...baseEvidence, cancelled: true, cause: 'cancel-request'}};
      }
      if (outcome.exitCode === 0 && outcome.stdout.trim() !== '') {
        // stdout = final answer (plain); the trailing newline is formatting,
        // not content. Evidence is untrusted and adapter-redacted; the Host
        // qualifies any outcome.
        return {state: 'COMPLETED', reason: null,
          evidence: {...baseEvidence,
            agentSummary: redactAndClamp(outcome.stdout.trim(), context.env)}};
      }
      if (outcome.exitCode === 0) {
        // exit 0 without a final answer: the observed contract
        // (stdout = final answer) was not honored -> fail-safe, no guessing.
        return {state: 'FAILED', reason: 'GROK_EMPTY_OUTPUT', evidence: baseEvidence};
      }
      if (outcome.exitCode === 2) {
        return {state: 'FAILED', reason: 'GROK_USAGE_ERROR',
          evidence: {...baseEvidence,
            stderr: redactAndClamp(outcome.stderr, context.env)}};
      }
      // exit 1 (auth/keyless "Not signed in", runtime errors) and any other
      // non-zero exit: honest FAILED claim with the redacted stderr summary.
      // stdout is deliberately NOT attached: on a failed run it is at most a
      // partial answer and must never surface as the run's output.
      return {state: 'FAILED', reason: 'GROK_RUN_FAILED',
        evidence: {...baseEvidence,
          stderr: redactAndClamp(outcome.stderr, context.env)}};
    },

    async cancel(attempt) {
      const entry = inFlight.get(newAttemptKey(attempt));
      if (!entry) {
        return {state: 'NOT_RUNNING', reason: null, evidence: null};
      }
      entry.requestCancel();
      // The request was issued; execute() settles CANCELLED when the tree
      // teardown lands.
      return {state: 'CANCEL_REQUESTED', reason: null, evidence: null};
    },

    async reconcile() {
      // One-shot provider keeps no durable run state to reconcile from (the
      // contract forbids an in-memory execution journal); the honest default
      // is UNKNOWN with the mandatory reason.
      return {state: 'UNKNOWN', reason: 'NOT_RECONCILED', evidence: null};
    },
  });
}
