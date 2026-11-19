/**
 * Tally v2.1 candidate AgentProvider/1 Node adapter for the pi coding agent
 * CLI (pi 0.85.1). Ported from the PR #469 Go candidate's observed CLI
 * contract — the Go source is evidence input only; nothing is bridged,
 * imported or executed from it.
 *
 * Observed CLI contract (pi 0.85.1, host binary, 2026-11-19 — see #469
 * runtimes/pi/{runtime.go,process.go,isolation.go,identity.go}; every flag
 * and output shape below is transcribed from those observations):
 *
 *   - One-shot process: `pi -p --mode json --no-session --session-dir <dir>
 *     --no-extensions --no-skills --no-prompt-templates --no-themes
 *     --no-context-files --no-approve [--provider p] [--model m]
 *     [--thinking t] -- "<instruction>"`. The trailing `--` ends option
 *     parsing; the instruction is always the single positional argument.
 *   - `--mode json` stdout is an NDJSON event stream: a first `session` line
 *     ({type:"session",id:<uuid>,cwd,...}), then agent/turn/message events;
 *     the assistant `message_end` carries content[].text, provider, model,
 *     usage{input,output,cacheRead,...} and stopReason. Observed values only:
 *     expected `--provider/--model` flags are wants, never backfilled into
 *     observations.
 *   - Exit codes observed: 0 = success (json stream contains an assistant
 *     message_end with text; exit 0 without one is a contract violation and
 *     fails safe); 1 = incomplete run (keyless: stderr "No API key found for
 *     <provider>."; bad model: provider 400 text). Cancellation = the stream
 *     is truncated mid-line; the parser skips incomplete tail lines and
 *     success is never guessed from a truncated stream.
 *   - workspace is pinned via the process cwd (pi has no cwd flag; its tools
 *     really write inside the process cwd — observed).
 *   - Credentials flow exclusively through environment variables (observed:
 *     deepseek readiness flips with DEEPSEEK_API_KEY presence). The
 *     `--api-key` flag exists but is never used: credentials never enter
 *     argv (a process-list observable surface) and never enter evidence.
 *   - Version gate: `pi --version` prints a bare version line (observed
 *     "0.85.1", exit 0). Any unobservable or mismatching version fails
 *     closed — the argv/output contract only holds for the pinned version.
 *   - Tree termination: Windows `taskkill /PID <pid> /T /F`; POSIX the child
 *     runs detached (own process group) and cancellation SIGKILLs the whole
 *     group. Orphan-holding stdout pipes are bounded by a WaitDelay-style
 *     force-finish so a hung classification cannot occur.
 *
 * Contract discipline (packages/tally-agent-provider):
 *   - Observations only. A COMPLETED result is an agent claim, never a Task
 *     PASS / Release / Candidate fact. No authority fields are emitted (the
 *     contract layer rejects them).
 *   - No fallback: selection failures (disabled, pin mismatch, not installed)
 *     are hard errors raised by the registry, never silently rerouted.
 *   - UNKNOWN always carries a reason code.
 *   - reconcile has no second persistence layer: always UNKNOWN/NOT_RECONCILED.
 *   - Canary (real pi turn) is NOT run by this adapter; probe reports
 *     canaryTurn: 'NOT_RUN'. Real acceptance is executed by the controller.
 */

import {spawn} from 'node:child_process';
import {statSync} from 'node:fs';
import {defineAgentProvider} from '../../tally-agent-provider/src/index.mjs';
import {
  buildPiSpawnEnv, createIsolationDirs, removeIsolationDirs,
} from './pi-env.mjs';

/** Registry id of this provider. */
export const PI_PROVIDER_ID = 'pi';

/** Adapter (descriptor) version — NOT the CLI version, which is observed. */
export const PI_ADAPTER_VERSION = '1';

/** Pinned upstream CLI version; all argv/output contract evidence anchor. */
export const PINNED_PI_VERSION = '0.85.1';

/**
 * Default credential env var name the probe requires to be explicitly
 * injected by the Host in context.env (observed deepseek readiness gate).
 */
export const PI_DEFAULT_CREDENTIAL_ENV_VAR = 'DEEPSEEK_API_KEY';

export const PI_DEFAULT_OUTPUT_MODE = 'json';

const OUTPUT_MODES = new Set(['json', 'text']);
const THINKING_LEVELS = new Set([
  'off', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max',
]);

const DEFAULT_TIMEOUT_MS = 10 * 60 * 1000;
const DEFAULT_WAIT_DELAY_MS = 5000;
const VERSION_PROBE_TIMEOUT_MS = 30 * 1000;
const TREE_KILL_TIMEOUT_MS = 10 * 1000;
const STDERR_LIMIT = 64 * 1024;
const STDOUT_LIMIT = 8 * 1024 * 1024;
const EVIDENCE_SUMMARY_CAP = 4000;
const EVIDENCE_STDERR_TAIL_CAP = 2000;
const EVIDENCE_MESSAGE_CAP = 200;

// ---- secret redaction (port of internal/logpipe RedactFine patterns) ----

const REDACTED = '[REDACTED]';
const RE_GITHUB = /\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})/g;
const RE_SK_KEY = /\bsk-[A-Za-z0-9_-]{20,}/g;
const RE_AWS = /\b(?:AKIA|ASIA)[0-9A-Z]{16}\b/g;
const RE_GITLAB = /\bglpat-[A-Za-z0-9_-]{20,}/g;
const RE_SLACK = /\bxox[baprs]-[A-Za-z0-9-]{10,}/g;
const RE_JWT = /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}/g;
const RE_AUTH_HEADER = /\b(Authorization|Proxy-Authorization)\s*[:=]\s*(?:Bearer\s+)?\S+/gi;
const RE_USER_PASS = /\b([a-z][a-z0-9+.-]*):\/\/[^:/\s@]+:[^@\s/]+@/g;
const RE_SECRET_KV = /\b(password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|client[_-]?secret)\s*[:=]\s*(?:"[^"]*"|'[^']*'|\S+)/gi;
const RE_LONG_B64 = /\b[A-Za-z0-9+/]{40,}={0,2}\b/g;

/**
 * Redact secret-shaped text in failure diagnostics before they can enter an
 * observation (external CLI stderr may carry credential-shaped text; prefer
 * over-redaction to any leak).
 */
export function redactFine(s) {
  let out = String(s);
  out = out.replace(RE_GITHUB, REDACTED);
  out = out.replace(RE_SK_KEY, REDACTED);
  out = out.replace(RE_AWS, REDACTED);
  out = out.replace(RE_GITLAB, REDACTED);
  out = out.replace(RE_SLACK, REDACTED);
  out = out.replace(RE_JWT, REDACTED);
  out = out.replace(RE_AUTH_HEADER, `$1: ${REDACTED}`);
  out = out.replace(RE_USER_PASS, `$1://${REDACTED}@`);
  out = out.replace(RE_SECRET_KV, `$1=${REDACTED}`);
  out = out.replace(RE_LONG_B64, REDACTED);
  return out;
}

const capText = (s, max, suffix = '...') => {
  const text = String(s);
  return text.length <= max ? text : text.slice(0, max) + suffix;
};

// ---- NDJSON stream parsing (pure) -----------------------------------------

function assistantText(content) {
  if (!Array.isArray(content)) return '';
  const texts = [];
  for (const block of content) {
    if (block && block.type === 'text' && typeof block.text === 'string' && block.text !== '') {
      texts.push(block.text);
    }
  }
  return texts.join('\n');
}

function normalizeUsage(usage) {
  if (!usage || typeof usage !== 'object') return null;
  const out = {};
  for (const field of ['input', 'output', 'cacheRead']) {
    const value = usage[field];
    if (typeof value === 'number' && Number.isFinite(value)) out[field] = value;
  }
  return Object.keys(out).length > 0 ? out : null;
}

/**
 * Parse a pi `--mode json` NDJSON stdout (0.85.1 observed shapes only).
 * Line-based: JSON-decode each non-empty line; malformed lines (truncated
 * tail after cancellation/crash, noise) are skipped — the stream is judged
 * by whether an assistant message_end with text appeared, never by guessing
 * success from the session line. The LAST assistant message_end with text
 * wins (multi-turn streams); its usage/provider/model/stopReason are carried
 * uninterpreted beyond the observed fields.
 */
export function parsePiStream(stdoutText) {
  const obs = {
    sessionHandle: '', cwd: '', hasAssistant: false, text: '',
    provider: '', model: '', stopReason: '', usage: null,
  };
  for (const rawLine of String(stdoutText).split('\n')) {
    const line = rawLine.trim();
    if (line === '') continue;
    let ev;
    try {
      ev = JSON.parse(line);
    } catch {
      continue; // truncated tail or noise line: fail-safe skip
    }
    if (!ev || typeof ev !== 'object') continue;
    if (ev.type === 'session') {
      if (typeof ev.id === 'string') obs.sessionHandle = ev.id;
      if (typeof ev.cwd === 'string') obs.cwd = ev.cwd;
    } else if (ev.type === 'message_end') {
      const message = ev.message;
      if (!message || message.role !== 'assistant') continue;
      const text = assistantText(message.content);
      if (text === '') continue;
      obs.hasAssistant = true;
      obs.text = text;
      obs.provider = typeof message.provider === 'string' ? message.provider : '';
      obs.model = typeof message.model === 'string' ? message.model : '';
      obs.stopReason = typeof message.stopReason === 'string' ? message.stopReason : '';
      obs.usage = normalizeUsage(message.usage);
    }
  }
  return obs;
}

// ---- argv assembly ---------------------------------------------------------

/**
 * True when `s` is a token safe to place bare into a pi flag value position
 * (observed token shapes: "deepseek", "deepseek-v4-flash",
 * "anthropic/claude-sonnet:high"; letters, digits and . _ - / : + only;
 * spaces/quotes/control characters are rejected outright — no escaping
 * layer is invented).
 */
export function isSafePiToken(s) {
  return typeof s === 'string' && s !== '' && /^[A-Za-z0-9._/:+-]+$/.test(s);
}

/**
 * Assemble the pi print-mode CLI args (after binPath / programArgs):
 *
 *   -p --mode <mode> --no-session --session-dir <dir> --no-extensions
 *   --no-skills --no-prompt-templates --no-themes --no-context-files
 *   --no-approve [--provider <p>] [--model <m>] [--thinking <t>]
 *   -- <instruction>
 *
 * Expected provider/model/thinking flags are passed only when configured
 * (they are wants, never observations). The instruction is always the last
 * positional argument after `--`.
 */
export function buildPiCliArgs({
  outputMode = PI_DEFAULT_OUTPUT_MODE, sessionDir,
  provider = '', model = '', thinkingLevel = '', instruction,
} = {}) {
  if (!OUTPUT_MODES.has(outputMode)) throw new TypeError('PI_OUTPUT_MODE_INVALID');
  if (thinkingLevel !== '' && !THINKING_LEVELS.has(thinkingLevel)) {
    throw new TypeError('PI_THINKING_LEVEL_INVALID');
  }
  if (provider !== '' && !isSafePiToken(provider)) throw new TypeError('PI_PROVIDER_TOKEN_INVALID');
  if (model !== '' && !isSafePiToken(model)) throw new TypeError('PI_MODEL_TOKEN_INVALID');
  if (typeof instruction !== 'string' || instruction === '') {
    throw new TypeError('PI_INSTRUCTION_INVALID');
  }
  const args = [
    '-p', '--mode', outputMode,
    '--no-session', '--session-dir', sessionDir,
    '--no-extensions', '--no-skills', '--no-prompt-templates',
    '--no-themes', '--no-context-files', '--no-approve',
  ];
  if (provider !== '') args.push('--provider', provider);
  if (model !== '') args.push('--model', model);
  if (thinkingLevel !== '') args.push('--thinking', thinkingLevel);
  args.push('--', instruction);
  return args;
}

// ---- process helpers -------------------------------------------------------

/**
 * Terminate the whole process tree rooted at `pid`. Windows: `taskkill
 * /PID <pid> /T /F` (structured argv, no shell). POSIX: the child is spawned
 * detached (own process group), so a group-wide SIGKILL covers pi and its
 * descendants. Best effort: resolves false when the pid is already gone.
 */
function killTree(pid) {
  if (!pid || pid <= 0) return Promise.resolve(false);
  if (process.platform === 'win32') {
    return new Promise((resolve) => {
      const killer = spawn('taskkill', ['/PID', String(pid), '/T', '/F'], {
        stdio: 'ignore', windowsHide: true,
      });
      const timer = setTimeout(() => {
        try { killer.kill(); } catch {}
        resolve(false);
      }, TREE_KILL_TIMEOUT_MS);
      killer.once('exit', () => { clearTimeout(timer); resolve(true); });
      killer.once('error', () => { clearTimeout(timer); resolve(false); });
    });
  }
  return new Promise((resolve) => {
    try {
      process.kill(-pid, 'SIGKILL'); // detached child = its own process group
      resolve(true);
    } catch (err) {
      resolve(err?.code === 'ESRCH' ? false : false);
    }
  });
}

function limitedBuffer(limit) {
  let buf = '';
  let truncated = false;
  return {
    push(chunk) {
      const text = String(chunk);
      const room = limit - buf.length;
      if (room <= 0) {
        truncated = true;
      } else if (text.length > room) {
        buf += text.slice(0, room);
        truncated = true;
      } else {
        buf += text;
      }
    },
    toString() {
      return truncated ? buf + '\n...(truncated)' : buf;
    },
  };
}

// ---- factory ---------------------------------------------------------------

/**
 * Create the pi AgentProvider/1 (Node). Options:
 *
 *   - binPath (required): explicit path to the pi executable (absolute in
 *     production; pinned-surface discipline forbids implicit discovery).
 *   - programArgs: args inserted between binPath and the pi flags. Empty in
 *     production; the test seam that hosts the fake CLI under
 *     `process.execPath` (and usable for interpreter-hosted CLIs).
 *   - pinnedVersion (default 0.85.1): fail-closed version gate.
 *   - provider / model / thinkingLevel / outputMode: expected configuration
 *     passed via observed flags; never reported as observations.
 *   - credentialEnvVar (default DEEPSEEK_API_KEY): the env var name probe
 *     requires to be explicitly present in the Host context.env.
 *   - inheritedEnv (default snapshot of process.env at factory time): the
 *     environment the spawn env is hardened from (stripped/forced).
 *   - timeoutMs (default 10 min): per-attempt wall clock; expiry tree-kills
 *     the process and classifies UNKNOWN.
 *   - waitDelayMs (default 5s): bounds the wait when the process exited but
 *     orphaned descendants still hold the output pipes.
 *   - tmpParent: parent dir for per-attempt isolation groups (tests).
 */
export function createPiAgentProvider(options = {}) {
  const {
    binPath, programArgs = [], pinnedVersion = PINNED_PI_VERSION,
    provider = '', model = '', thinkingLevel = '',
    outputMode = PI_DEFAULT_OUTPUT_MODE,
    credentialEnvVar = PI_DEFAULT_CREDENTIAL_ENV_VAR,
    inheritedEnv = {...process.env},
    timeoutMs = DEFAULT_TIMEOUT_MS,
    waitDelayMs = DEFAULT_WAIT_DELAY_MS,
    tmpParent,
  } = options;

  if (typeof binPath !== 'string' || binPath.trim() === '') {
    throw new TypeError('PI_BIN_PATH_REQUIRED');
  }
  if (typeof pinnedVersion !== 'string' || pinnedVersion === '') {
    throw new TypeError('PI_PINNED_VERSION_REQUIRED');
  }
  if (!OUTPUT_MODES.has(outputMode)) throw new TypeError('PI_OUTPUT_MODE_INVALID');
  if (thinkingLevel !== '' && !THINKING_LEVELS.has(thinkingLevel)) {
    throw new TypeError('PI_THINKING_LEVEL_INVALID');
  }
  if (provider !== '' && !isSafePiToken(provider)) throw new TypeError('PI_PROVIDER_TOKEN_INVALID');
  if (model !== '' && !isSafePiToken(model)) throw new TypeError('PI_MODEL_TOKEN_INVALID');
  if (typeof credentialEnvVar !== 'string' || credentialEnvVar === '') {
    throw new TypeError('PI_CREDENTIAL_ENV_VAR_INVALID');
  }
  if (!Array.isArray(programArgs) || programArgs.some((a) => typeof a !== 'string')) {
    throw new TypeError('PI_PROGRAM_ARGS_INVALID');
  }
  if (!Number.isFinite(timeoutMs) || timeoutMs <= 0) throw new TypeError('PI_TIMEOUT_INVALID');
  if (!Number.isFinite(waitDelayMs) || waitDelayMs <= 0) throw new TypeError('PI_WAIT_DELAY_INVALID');

  const resolvedBinPath = binPath.trim();

  function resolveBinary() {
    try {
      const info = statSync(resolvedBinPath);
      return info.isFile() ? resolvedBinPath : null;
    } catch {
      return null;
    }
  }

  /**
   * Observe `<bin> --version` once (bare version line, exit 0 — observed
   * shape). Spawn env is the Host-injected env only (mirrors the Go
   * candidate's identity probe). Non-zero exit, spawn failure, timeout or
   * unparseable output resolves null (never cached).
   */
  function observeCliVersion(injectedEnv) {
    return new Promise((resolve) => {
      let settled = false;
      let exitCode = null;
      let stdout = '';
      const child = spawn(resolvedBinPath, [...programArgs, '--version'], {
        env: {...injectedEnv},
        windowsHide: true,
        stdio: ['ignore', 'pipe', 'pipe'],
      });
      const timer = setTimeout(() => {
        killTree(child.pid);
      }, VERSION_PROBE_TIMEOUT_MS);
      const finish = () => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        if (exitCode !== 0) return resolve(null);
        const line = stdout.split('\n').map((s) => s.trim()).find(Boolean);
        resolve(line ? {version: line} : null);
      };
      child.stdout?.on('data', (chunk) => { stdout += chunk; });
      child.once('error', () => { exitCode ??= -1; finish(); });
      child.once('close', (code) => { exitCode = code; finish(); });
    });
  }

  // Successful version observation cache (failures are retried next call —
  // mirrors the Go candidate's caching discipline).
  let versionGate = null;

  async function ensureVersionGate(injectedEnv) {
    if (versionGate) return versionGate;
    if (resolveBinary() === null) return {reason: 'PI_BINARY_MISSING'};
    const observed = await observeCliVersion(injectedEnv);
    if (!observed) return {reason: 'PI_VERSION_UNOBSERVABLE'};
    if (observed.version !== pinnedVersion) {
      return {reason: 'PI_VERSION_PIN_MISMATCH', observedVersion: observed.version};
    }
    versionGate = {version: observed.version};
    return versionGate;
  }

  // In-flight attempt registry: process handles only (pid + cancel flag),
  // NOT an execution journal — no outcome is persisted here. Keyed by
  // `<taskId>::<attemptId>`; an entry exists only while the one-shot process
  // is running.
  const inFlight = new Map();

  return defineAgentProvider({
    id: PI_PROVIDER_ID,
    contractVersion: 1,
    capabilities: [], // honest empty set for a one-shot synchronous process
    async probe(context) {
      if (resolveBinary() === null) {
        return {state: 'UNAVAILABLE', reason: 'PI_BINARY_MISSING'};
      }
      const gate = await ensureVersionGate(context?.env ?? {});
      if (gate.reason) {
        const observation = {state: 'UNAVAILABLE', reason: gate.reason};
        if (gate.observedVersion) {
          observation.evidence = {
            observedVersion: gate.observedVersion, pinnedVersion,
          };
        }
        return observation;
      }
      const injected = context?.env?.[credentialEnvVar];
      if (typeof injected !== 'string' || injected.trim() === '') {
        return {
          state: 'UNAVAILABLE',
          reason: 'PI_CREDENTIAL_MISSING',
          evidence: {credentialEnvVar, cliVersion: gate.version},
        };
      }
      return {
        state: 'AVAILABLE',
        evidence: {
          cliVersion: gate.version,
          pinnedVersion,
          credentialEnvVar,
          canaryTurn: 'NOT_RUN',
        },
      };
    },

    async execute(request, context) {
      const signal = context?.signal ?? null;
      const injectedEnv = context?.env ?? {};
      const key = `${request?.taskId ?? ''}::${request?.attemptId ?? ''}`;
      // Register synchronously (before the first await) so a cancel that
      // races this execute can never miss the process.
      const entry = {pid: 0, cancelRequested: false};
      inFlight.set(key, entry);
      let dirs = null;
      let onAbort = null;
      let timeoutTimer = null;
      try {
        if (signal?.aborted) {
          return {state: 'CANCELLED', reason: 'PI_CANCELLED_BEFORE_START'};
        }
        if (typeof request?.prompt !== 'string' || request.prompt.trim() === '') {
          return {state: 'FAILED', reason: 'PI_PROMPT_INVALID'};
        }
        if (typeof request?.worktreePath !== 'string' || request.worktreePath === '') {
          return {state: 'FAILED', reason: 'PI_WORKSPACE_INVALID'};
        }
        // Fail-closed pinned-version gate (no spawn on mismatch).
        const gate = await ensureVersionGate(injectedEnv);
        if (gate.reason) return {state: 'FAILED', reason: gate.reason};
        if (signal?.aborted) {
          return {state: 'CANCELLED', reason: 'PI_CANCELLED_BEFORE_START'};
        }
        try {
          dirs = createIsolationDirs({parent: tmpParent});
        } catch {
          return {state: 'FAILED', reason: 'PI_ISOLATION_DIRS_FAILED'};
        }
        const env = buildPiSpawnEnv({inheritedEnv, injectedEnv, dirs});
        let cliArgs;
        try {
          cliArgs = buildPiCliArgs({
            outputMode, sessionDir: dirs.sessions,
            provider, model, thinkingLevel, instruction: request.prompt,
          });
        } catch (err) {
          return {state: 'FAILED', reason: err?.message ?? 'PI_ARGV_ASSEMBLY_FAILED'};
        }
        const child = spawn(resolvedBinPath, [...programArgs, ...cliArgs], {
          cwd: request.worktreePath, // workspace = process cwd (observed)
          env,
          windowsHide: true,
          detached: process.platform !== 'win32',
          // stdin MUST be closed for `-p` print mode: the real 0.85.1 CLI
          // reads stdin to EOF before finishing its turn, so a default
          // `pipe` stdin that the adapter never ends makes every real turn
          // hang with zero output until the timeout tree-kills it (found by
          // the real-CLI controller smoke; the fake CLI never reads stdin,
          // which is why the matrix alone could not catch this).
          stdio: ['ignore', 'pipe', 'pipe'],
        });
        entry.pid = child.pid ?? 0;
        if (entry.cancelRequested || signal?.aborted) {
          // Cancel raced the spawn: terminate the fresh tree immediately.
          killTree(entry.pid);
        }
        if (signal) {
          onAbort = () => {
            entry.cancelRequested = true;
            killTree(entry.pid);
          };
          signal.addEventListener('abort', onAbort, {once: true});
        }
        let timedOut = false;
        const effectiveTimeoutMs = request.timeoutMs ?? timeoutMs;
        timeoutTimer = setTimeout(() => {
          timedOut = true;
          killTree(entry.pid);
        }, effectiveTimeoutMs);

        const stdout = limitedBuffer(STDOUT_LIMIT);
        const stderr = limitedBuffer(STDERR_LIMIT);
        child.stdout?.on('data', (chunk) => stdout.push(chunk));
        child.stderr?.on('data', (chunk) => stderr.push(chunk));

        let exitInfo = null;
        let spawnError = null;
        let pipeLeak = false;
        let unresolved = false;
        let waitTimer = null;
        let guardTimer = null;
        await new Promise((resolve) => {
          let settled = false;
          const finish = () => {
            if (settled) return;
            settled = true;
            if (waitTimer) clearTimeout(waitTimer);
            if (guardTimer) clearTimeout(guardTimer);
            resolve();
          };
          child.once('exit', (code, signalName) => {
            exitInfo = {code, signal: signalName ?? null};
            // WaitDelay-style bound: the process exited but orphaned
            // descendants may still hold the output pipes.
            waitTimer = setTimeout(() => {
              pipeLeak = true;
              try { child.stdout?.destroy(); } catch {}
              try { child.stderr?.destroy(); } catch {}
            }, waitDelayMs);
          });
          child.once('error', (err) => {
            spawnError = err;
            if (!exitInfo) finish(); // spawn failure: close may not carry a code
          });
          child.once('close', () => finish());
          // Hard outer guard: classification must always converge even if
          // no exit/error/close ever arrives (defensive; close is expected
          // to always fire per the child_process contract).
          guardTimer = setTimeout(() => {
            unresolved = true;
            finish();
          }, effectiveTimeoutMs + waitDelayMs + 5000);
        });
        if (unresolved) {
          killTree(entry.pid);
          return {state: 'UNKNOWN', reason: 'PI_EXEC_UNRESOLVED'};
        }

        const stderrText = stderr.toString();
        const evidence = {cliVersion: gate.version, outputMode};
        const classification = classifyRun({
          outputMode,
          cancelRequested: entry.cancelRequested,
          timedOut, spawnError, pipeLeak,
          exitCode: exitInfo?.code ?? null,
          exitSignal: exitInfo?.signal ?? null,
          stdoutText: stdout.toString(), stderrText, evidence,
        });
        return classification;
      } finally {
        if (timeoutTimer) clearTimeout(timeoutTimer);
        if (signal && onAbort) {
          try { signal.removeEventListener('abort', onAbort); } catch {}
        }
        inFlight.delete(key);
        removeIsolationDirs(dirs);
      }
    },

    async cancel(attempt) {
      const key = `${attempt?.taskId ?? ''}::${attempt?.attemptId ?? ''}`;
      const entry = inFlight.get(key);
      if (!entry) return {state: 'NOT_RUNNING'};
      entry.cancelRequested = true;
      if (entry.pid > 0) await killTree(entry.pid);
      return {state: 'CANCEL_REQUESTED'};
    },

    async reconcile() {
      // No second persistence layer: an unobserved attempt stays unknown.
      return {state: 'UNKNOWN', reason: 'NOT_RECONCILED'};
    },
  });
}

/**
 * Final-state classification. Priority (mirrors the Go candidate):
 * cancellation (operator intent) > timeout (tree killed, outcome unobserved)
 * > spawn failure > output-pipe non-convergence > non-zero exit (process
 * self-declared incomplete; the raw stderr text is redacted into evidence) >
 * exit 0 without a parseable answer (contract not honored: FAIL-SAFE
 * failed, never guessed success) > completed. json mode completes on a
 * text-bearing assistant message_end; text mode completes on non-empty
 * stdout (usage stays unobserved — text stdout has no usage self-report).
 */
function classifyRun({
  outputMode, cancelRequested, timedOut, spawnError, pipeLeak,
  exitCode, exitSignal, stdoutText, stderrText, evidence,
}) {
  if (cancelRequested) {
    return {
      state: 'CANCELLED',
      reason: 'PI_EXEC_CANCELLED',
      evidence: {...evidence},
    };
  }
  if (timedOut) {
    return {
      state: 'UNKNOWN',
      reason: 'PI_EXEC_TIMEOUT',
      evidence: {...evidence, stderrTail: redactFine(capText(stderrText, EVIDENCE_STDERR_TAIL_CAP))},
    };
  }
  if (spawnError) {
    return {
      state: 'FAILED',
      reason: 'PI_SPAWN_FAILED',
      evidence: {...evidence, message: redactFine(capText(String(spawnError?.message ?? spawnError), EVIDENCE_MESSAGE_CAP))},
    };
  }
  if (pipeLeak) {
    return {
      state: 'FAILED',
      reason: 'PI_OUTPUT_PIPE_UNRESOLVED',
      evidence: {...evidence, stderrTail: redactFine(capText(stderrText, EVIDENCE_STDERR_TAIL_CAP))},
    };
  }
  if (exitCode !== 0) {
    const failure = {
      ...evidence,
      exitCode,
      stderrTail: redactFine(capText(stderrText, EVIDENCE_STDERR_TAIL_CAP)),
    };
    if (exitSignal) failure.exitSignal = exitSignal;
    return {state: 'FAILED', reason: 'PI_EXIT_NONZERO', evidence: failure};
  }
  if (outputMode === 'text') {
    const answer = stdoutText.trim();
    if (answer === '') {
      return {
        state: 'FAILED',
        reason: 'PI_CONTRACT_NO_ANSWER',
        evidence: {...evidence},
      };
    }
    return {
      state: 'COMPLETED',
      evidence: {
        ...evidence,
        summary: capText(answer, EVIDENCE_SUMMARY_CAP),
        summaryTruncated: answer.length > EVIDENCE_SUMMARY_CAP,
      },
    };
  }
  const obs = parsePiStream(stdoutText);
  if (!obs.hasAssistant) {
    const noAnswer = {...evidence};
    if (obs.sessionHandle !== '') noAnswer.sessionId = obs.sessionHandle;
    return {state: 'FAILED', reason: 'PI_CONTRACT_NO_ANSWER', evidence: noAnswer};
  }
  const completed = {
    ...evidence,
    summary: capText(obs.text, EVIDENCE_SUMMARY_CAP),
    summaryTruncated: obs.text.length > EVIDENCE_SUMMARY_CAP,
    sessionId: obs.sessionHandle,
    observedProvider: obs.provider,
    observedModel: obs.model,
  };
  if (obs.stopReason !== '') completed.stopReason = obs.stopReason;
  if (obs.usage) completed.usage = obs.usage;
  return {state: 'COMPLETED', evidence: completed};
}
