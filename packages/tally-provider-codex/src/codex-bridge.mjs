/**
 * Codex AgentProvider/1 bridge adapter (#473).
 *
 * Responsibility is EXACTLY: host terminal-state mapping, request/context
 * adaptation and observation normalization/redaction. The bridge owns no
 * authorization, no CAS store, no git and no candidate qualification — all of
 * that stays inside experiments/v2/selected-business-build/local-codex-host.mjs
 * and the existing selected B/K/Host/C chain.
 *
 * The host module imports `@kaicreator/domain-harness-host-node` at its top
 * level, so the bridge loads it lazily via an injected `loadHost` and never
 * statically. Every operation fails closed to a contract-legal observation
 * when the host module cannot be loaded.
 */
import {normalizeProviderObservation} from '../../tally-agent-provider/src/index.mjs';

export const CODEX_PROVIDER_ID = 'codex';

/** Resolved relative to packages/tally-provider-codex/src/index.mjs. */
export const DEFAULT_HOST_MODULE_RELATIVE_PATH =
  '../../../experiments/v2/selected-business-build/local-codex-host.mjs';

/**
 * Contract-legal reason vocabulary produced by the bridge itself. Host reasons
 * are passed through verbatim whenever they already satisfy the contract
 * reason shape; they never come from this table.
 */
export const BRIDGE_REASONS = Object.freeze({
  HOST_MODULE_UNAVAILABLE: 'HOST_MODULE_UNAVAILABLE',
  HOST_MODULE_SURFACE_INVALID: 'HOST_MODULE_SURFACE_INVALID',
  HOST_STATE_MAPPING_MISS: 'HOST_STATE_MAPPING_MISS',
  HOST_EXECUTION_ERROR: 'HOST_EXECUTION_ERROR',
  HOST_REASON_UNSAFE: 'HOST_REASON_UNSAFE',
  PROBE_EXECUTION_FAILED: 'PROBE_EXECUTION_FAILED',
  PROBE_UNAVAILABLE: 'PROBE_UNAVAILABLE',
  PROBE_OBSERVATION_INVALID: 'PROBE_OBSERVATION_INVALID',
  PROVIDER_CONFIG_UNAVAILABLE: 'PROVIDER_CONFIG_UNAVAILABLE',
  EXECUTION_REQUEST_INVALID: 'EXECUTION_REQUEST_INVALID',
  ATTEMPT_PIN_INVALID: 'ATTEMPT_PIN_INVALID',
  ATTEMPT_PIN_MISMATCH: 'ATTEMPT_PIN_MISMATCH',
  NO_INFLIGHT_ATTEMPT: 'NO_INFLIGHT_ATTEMPT',
  CANCELLED_BEFORE_LAUNCH: 'CANCELLED_BEFORE_LAUNCH',
  RECONCILE_STATE_UNAVAILABLE: 'RECONCILE_STATE_UNAVAILABLE',
});

/**
 * Host attempt terminal state -> contract execute state. AGENT_NOT_EXECUTED
 * and NOT_ADMITTED are deterministic terminal non-successes durably recorded
 * by the host, so they map to FAILED (never UNKNOWN) with the host reason
 * preserved verbatim; UNKNOWN is reserved for genuinely undeterminable
 * outcomes that need reconciliation.
 */
export const EXECUTE_STATE_MAP = Object.freeze({
  LOCAL_CANDIDATE_READY: 'COMPLETED',
  LOCAL_ATTEMPT_FAILED: 'FAILED',
  LOCAL_ATTEMPT_CANCELLED: 'CANCELLED',
  LOCAL_ATTEMPT_UNKNOWN: 'UNKNOWN',
  AGENT_NOT_EXECUTED: 'FAILED',
  NOT_ADMITTED: 'FAILED',
});

/**
 * Host replay -> contract reconcile state. The host's replay collapses
 * FAILED/CANCELLED/UNKNOWN/RUNNING records into LOCAL_ATTEMPT_UNKNOWN and
 * preserves only process.status; a RUNNING process.status is reported as
 * RUNNING. The bridge never fabricates FAILED/CANCELLED for a replay that the
 * host could not classify, and NOT_ADMITTED on a replay means the attempt's
 * own record state could not be established, so it fails closed to UNKNOWN.
 */
export const RECONCILE_STATE_MAP = Object.freeze({
  LOCAL_CANDIDATE_READY: 'COMPLETED',
  LOCAL_ATTEMPT_FAILED: 'FAILED',
  LOCAL_ATTEMPT_CANCELLED: 'CANCELLED',
  LOCAL_ATTEMPT_UNKNOWN: 'UNKNOWN',
  AGENT_NOT_EXECUTED: 'FAILED',
  NOT_ADMITTED: 'UNKNOWN',
});

const REASON = /^[A-Z][A-Z0-9_]{0,95}$/;

/** Pass host reasons through only when contract-legal, else fall back. */
export function sanitizeReason(value, fallback) {
  return typeof value === 'string' && REASON.test(value) ? value : fallback;
}

/**
 * Adapter-redacted execution observations ONLY: launch/exit/session/duration.
 * Candidate git facts (candidateTree/candidateCommit/candidateRef) are
 * deliberately dropped — the Host independently reads them back from the
 * durable CAS record via verifyCandidateReadback; provider results are
 * untrusted observations and must not carry them at all.
 */
export function redactExecutionEvidence(result) {
  const process = result && typeof result === 'object' ? result.process : null;
  const evidence = {};
  if (result?.agentLaunch === true) evidence.agentLaunch = true;
  for (const key of ['exitCode', 'pid', 'startedAt', 'terminatedAt']) {
    const value = process?.[key];
    if (value !== undefined && value !== null) evidence[key] = value;
  }
  const startedAt = Date.parse(process?.startedAt ?? '');
  const terminatedAt = Date.parse(process?.terminatedAt ?? '');
  if (Number.isFinite(startedAt) && Number.isFinite(terminatedAt) && terminatedAt >= startedAt)
    evidence.durationMs = terminatedAt - startedAt;
  return Object.keys(evidence).length > 0 ? Object.freeze(evidence) : null;
}

/** probeProvider observation -> contract probe observation. */
export function mapProbeOutcome(probe) {
  if (probe && typeof probe === 'object') {
    if (probe.available === true)
      return normalizeProviderObservation('probe', {state: 'AVAILABLE', reason: null,
        evidence: Object.freeze({provider:
          typeof probe.provider === 'string' ? probe.provider : 'codex-cli'})});
    if (probe.available === false) {
      const evidence = {};
      if (typeof probe.stage === 'string') evidence.stage = probe.stage;
      if (probe.exitCode !== undefined && probe.exitCode !== null) evidence.exitCode = probe.exitCode;
      return normalizeProviderObservation('probe', {state: 'UNAVAILABLE',
        reason: sanitizeReason(probe.reason, BRIDGE_REASONS.PROBE_UNAVAILABLE),
        evidence: Object.keys(evidence).length > 0 ? Object.freeze(evidence) : null});
    }
  }
  return normalizeProviderObservation('probe', {state: 'UNKNOWN',
    reason: BRIDGE_REASONS.PROBE_OBSERVATION_INVALID, evidence: null});
}

/** Host attempt outcome -> contract execute/reconcile observation. */
export function mapAttemptOutcome(operation, result) {
  const table = operation === 'reconcile' ? RECONCILE_STATE_MAP : EXECUTE_STATE_MAP;
  const hostState = result?.state;
  let state = Object.hasOwn(table, hostState) ? table[hostState] : null;
  if (state === null)
    return normalizeProviderObservation(operation, {state: 'UNKNOWN',
      reason: BRIDGE_REASONS.HOST_STATE_MAPPING_MISS, evidence: redactExecutionEvidence(result)});
  if (operation === 'reconcile' && hostState === 'LOCAL_ATTEMPT_UNKNOWN' &&
      result?.process?.status === 'RUNNING')
    state = 'RUNNING';
  const reason = state === 'COMPLETED' ? null
    : sanitizeReason(result?.reason,
        state === 'UNKNOWN' ? BRIDGE_REASONS.HOST_REASON_UNSAFE : null);
  return normalizeProviderObservation(operation, {state, reason,
    evidence: redactExecutionEvidence(result)});
}

/**
 * Bridge core. Holds only: one memoized host-module load promise, AbortController
 * handles for execute calls that are currently in flight, and the per-attempt
 * host configs needed to ask the host for a replay. No results, no journal, no
 * scheduler — durable attempt state lives solely in the host CAS store.
 */
export function createCodexBridge({loadHost, config = null, now} = {}) {
  if (typeof loadHost !== 'function') throw new TypeError('PROVIDER_LOAD_HOST_INVALID');
  if (now !== undefined && typeof now !== 'function') throw new TypeError('PROVIDER_NOW_INVALID');
  let hostPromise = null;
  const inflight = new Map();
  const replayConfigs = new Map();

  const loadHostOnce = () => {
    if (!hostPromise) {
      hostPromise = (async () => {
        const module = await loadHost();
        const factory = module?.createLocalCodexHost;
        if (typeof factory !== 'function')
          throw Object.assign(new Error(BRIDGE_REASONS.HOST_MODULE_SURFACE_INVALID),
            {code: BRIDGE_REASONS.HOST_MODULE_SURFACE_INVALID});
        return factory(now ? {now} : {});
      })().catch(error => { hostPromise = null; throw error; });
    }
    return hostPromise;
  };

  async function probe() {
    if (!config || typeof config !== 'object')
      return normalizeProviderObservation('probe', {state: 'UNAVAILABLE',
        reason: BRIDGE_REASONS.PROVIDER_CONFIG_UNAVAILABLE, evidence: null});
    let host;
    try { host = await loadHostOnce(); }
    catch { return normalizeProviderObservation('probe', {state: 'UNAVAILABLE',
      reason: BRIDGE_REASONS.HOST_MODULE_UNAVAILABLE, evidence: null}); }
    try { return mapProbeOutcome(await host.probeProvider(config)); }
    catch { return normalizeProviderObservation('probe', {state: 'UNKNOWN',
      reason: BRIDGE_REASONS.PROBE_EXECUTION_FAILED, evidence: null}); }
  }

  async function execute(request, context) {
    if (!request || typeof request !== 'object' ||
        typeof request.taskId !== 'string' || typeof request.attemptId !== 'string' ||
        typeof request.sourceSha !== 'string' || typeof request.worktreePath !== 'string')
      return normalizeProviderObservation('execute', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.EXECUTION_REQUEST_INVALID, evidence: null});
    if (!config || typeof config !== 'object')
      return normalizeProviderObservation('execute', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.PROVIDER_CONFIG_UNAVAILABLE, evidence: null});
    // request.prompt is intentionally NOT forwarded: the local Codex host owns
    // the candidate-note prompt bound to its write-set protocol.
    // context.env is intentionally NOT merged: the host builder environment
    // allowlist is the only authority for what the agent process sees.
    const attemptConfig = {...config, attemptId: request.attemptId,
      sourceSha: request.sourceSha, worktreePath: request.worktreePath};
    if (typeof request.timeoutMs === 'number') attemptConfig.timeoutMs = request.timeoutMs;
    // Register the cancel handle before the first await so cancel() can target
    // an attempt even while the host module is still being lazily loaded.
    const controller = new AbortController();
    const forward = () => controller.abort();
    if (context?.signal) {
      if (context.signal.aborted) controller.abort();
      else context.signal.addEventListener('abort', forward, {once: true});
    }
    attemptConfig.signal = controller.signal;
    inflight.set(request.attemptId, {taskId: request.taskId,
      sourceSha: request.sourceSha, controller});
    replayConfigs.set(request.attemptId, {config: attemptConfig,
      taskId: request.taskId});
    let launched = false;
    try {
      let host;
      try { host = await loadHostOnce(); }
      catch { host = null; }
      if (!host) return normalizeProviderObservation('execute', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.HOST_MODULE_UNAVAILABLE, evidence: null});
      // The host abort path is event-driven and cannot observe a signal that
      // was already aborted before runAttempt; the bridge owns the corner and
      // refuses to launch a cancelled attempt at all. Nothing durable was
      // recorded, so the replay config is dropped to keep reconcile honest.
      if (controller.signal.aborted) {
        replayConfigs.delete(request.attemptId);
        return normalizeProviderObservation('execute', {state: 'CANCELLED',
          reason: BRIDGE_REASONS.CANCELLED_BEFORE_LAUNCH, evidence: null});
      }
      launched = true;
      return mapAttemptOutcome('execute', await host.runAttempt(attemptConfig));
    } catch (error) {
      return normalizeProviderObservation('execute', {state: 'UNKNOWN',
        reason: sanitizeReason(error?.code, BRIDGE_REASONS.HOST_EXECUTION_ERROR),
        evidence: null});
    } finally {
      if (inflight.get(request.attemptId)?.controller === controller)
        inflight.delete(request.attemptId);
      // a config whose attempt never reached the host must not be replayable,
      // otherwise a later reconcile could trigger a launch side effect
      if (!launched) replayConfigs.delete(request.attemptId);
      context?.signal?.removeEventListener('abort', forward);
    }
  }

  /**
   * The host has no dedicated cancel API: cancellation is the AbortSignal path
   * of runAttempt. Aborting the in-flight controller makes the host terminate
   * the process tree and durably record LOCAL_ATTEMPT_CANCELLED; execute then
   * resolves CANCELLED. With no in-flight handle the bridge cannot know the
   * durable record state and reports NOT_RUNNING rather than guessing.
   */
  async function cancel(attempt) {
    if (!attempt || typeof attempt !== 'object' ||
        typeof attempt.taskId !== 'string' || typeof attempt.attemptId !== 'string' ||
        typeof attempt.sourceSha !== 'string')
      return normalizeProviderObservation('cancel', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.ATTEMPT_PIN_INVALID, evidence: null});
    const tracked = inflight.get(attempt.attemptId);
    if (!tracked)
      return normalizeProviderObservation('cancel', {state: 'NOT_RUNNING',
        reason: BRIDGE_REASONS.NO_INFLIGHT_ATTEMPT,
        evidence: Object.freeze({attemptId: attempt.attemptId})});
    if (tracked.taskId !== attempt.taskId || tracked.sourceSha !== attempt.sourceSha ||
        attempt.providerId !== CODEX_PROVIDER_ID)
      return normalizeProviderObservation('cancel', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.ATTEMPT_PIN_MISMATCH, evidence: null});
    tracked.controller.abort();
    return normalizeProviderObservation('cancel', {state: 'CANCEL_REQUESTED', reason: null,
      evidence: Object.freeze({attemptId: attempt.attemptId})});
  }

  /**
   * Reconcile is STRICTLY READ-ONLY (R106 ABI review P1 / R107 repair 1):
   * runAttempt can still LAUNCH when no durable row exists, so it may never
   * be called from reconcile. The bridge uses the host's dedicated
   * readAttemptRecord port — same state vocabulary, zero side effects — and
   * answers UNKNOWN when the port is absent (older host) or when no
   * vouchered config names the store to read (the one recorded when execute
   * launched here, or one supplied by the trusted Host with the exact pin).
   */
  async function reconcile(attempt, context) {
    if (!attempt || typeof attempt !== 'object' ||
        typeof attempt.attemptId !== 'string' || typeof attempt.sourceSha !== 'string')
      return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.ATTEMPT_PIN_INVALID, evidence: null});
    let tracked = replayConfigs.get(attempt.attemptId) ?? null;
    if (tracked) {
      if (tracked.config.sourceSha !== attempt.sourceSha ||
          tracked.taskId !== attempt.taskId)
        return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
          reason: BRIDGE_REASONS.ATTEMPT_PIN_MISMATCH, evidence: null});
    }
    let attemptConfig = tracked?.config ?? null;
    if (!attemptConfig) {
      const supplied = context?.attemptConfig;
      if (supplied && typeof supplied === 'object' &&
          supplied.attemptId === attempt.attemptId &&
          supplied.sourceSha === attempt.sourceSha)
        attemptConfig = supplied;
    }
    if (!attemptConfig)
      return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE, evidence: null});
    let host;
    try { host = await loadHostOnce(); }
    catch { return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
      reason: BRIDGE_REASONS.HOST_MODULE_UNAVAILABLE, evidence: null}); }
    if (typeof host.readAttemptRecord !== 'function')
      return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
        reason: BRIDGE_REASONS.RECONCILE_STATE_UNAVAILABLE, evidence: null});
    try {
      return mapAttemptOutcome('reconcile',
        await host.readAttemptRecord(attemptConfig));
    } catch (error) {
      return normalizeProviderObservation('reconcile', {state: 'UNKNOWN',
        reason: sanitizeReason(error?.code, BRIDGE_REASONS.HOST_EXECUTION_ERROR),
        evidence: null});
    }
  }

  return Object.freeze({probe, execute, cancel, reconcile});
}
