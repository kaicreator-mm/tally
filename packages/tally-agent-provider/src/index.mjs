/**
 * Tally AgentProvider/1: process-only, untrusted agent observations.
 * Task admission, worktree grants, git candidate readback, review and Outcome
 * remain solely with the existing selected B/K/Host/C chain.
 */
export const AGENT_PROVIDER_CONTRACT_VERSION = 1;
export const DEFAULT_AGENT_PROVIDER = 'codex';

const ID = /^[a-z][a-z0-9-]{0,63}$/;
const SHA = /^[a-f0-9]{40}$/;
const REASON = /^[A-Z][A-Z0-9_]{0,95}$/;
const STATES = Object.freeze({
  probe: new Set(['AVAILABLE', 'UNAVAILABLE', 'UNKNOWN']),
  execute: new Set(['COMPLETED', 'FAILED', 'CANCELLED', 'UNKNOWN']),
  cancel: new Set(['CANCEL_REQUESTED', 'NOT_RUNNING', 'UNKNOWN']),
  reconcile: new Set(['RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED', 'UNKNOWN']),
});
const RESULT_KEYS = new Set(['state', 'reason', 'evidence']);

function fail(reason) { throw new TypeError(reason); }
function validId(value) { return typeof value === 'string' && ID.test(value); }

/** A Provider result is an observation, NEVER a Task/Review/Release fact. */
export function normalizeProviderObservation(operation, observation) {
  if (!Object.hasOwn(STATES, operation)) fail('PROVIDER_OPERATION_UNKNOWN');
  if (!observation || typeof observation !== 'object' || Array.isArray(observation))
    fail('PROVIDER_OBSERVATION_INVALID');
  if (Reflect.ownKeys(observation).some(key => !RESULT_KEYS.has(key)))
    fail('PROVIDER_OBSERVATION_UNEXPECTED_FIELD');
  const {state, reason = null, evidence = null} = observation;
  if (!STATES[operation].has(state)) fail('PROVIDER_OBSERVATION_STATE_INVALID');
  if (reason !== null && (typeof reason !== 'string' || !REASON.test(reason)))
    fail('PROVIDER_OBSERVATION_REASON_INVALID');
  if (state === 'UNKNOWN' && reason === null)
    fail('PROVIDER_UNKNOWN_REASON_REQUIRED');
  // Evidence is deliberately untrusted and must be redacted by the adapter;
  // Host may inspect it but cannot treat any field as authority or qualification.
  if (evidence !== null && (typeof evidence !== 'object' || Array.isArray(evidence)))
    fail('PROVIDER_OBSERVATION_EVIDENCE_INVALID');
  return Object.freeze({state, reason, evidence});
}

/**
 * A provider owns only its CLI/protocol execution and reporting. The Host
 * supplies an already admitted workspace; this is NOT an authority API.
 */
export function defineAgentProvider(specification) {
  if (!specification || !validId(specification.id)) fail('PROVIDER_ID_INVALID');
  if (specification.contractVersion !== AGENT_PROVIDER_CONTRACT_VERSION)
    fail('PROVIDER_CONTRACT_VERSION_UNSUPPORTED');
  const capabilities = specification.capabilities ?? [];
  if (!Array.isArray(capabilities) ||
      capabilities.some(c => typeof c !== 'string' || !/^[a-z][a-z0-9.-]*$/.test(c)) ||
      new Set(capabilities).size !== capabilities.length)
    fail('PROVIDER_CAPABILITIES_INVALID');
  for (const operation of Object.keys(STATES))
    if (typeof specification[operation] !== 'function')
      fail('PROVIDER_OPERATION_MISSING_' + operation.toUpperCase());
  const operations = Object.fromEntries(Object.keys(STATES).map(operation => [
    operation,
    async (...args) => normalizeProviderObservation(
      operation, await specification[operation](...args)),
  ]));
  return Object.freeze({id: specification.id,
    contractVersion: AGENT_PROVIDER_CONTRACT_VERSION,
    capabilities: Object.freeze([...capabilities]), ...operations});
}

/** Selection pins do not grant admission; the selected Host validates them. */
export function createProviderAttemptPin({taskId, attemptId, sourceSha, providerId}) {
  if (!validId(providerId)) fail('PROVIDER_ID_INVALID');
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(taskId ?? ''))
    fail('PROVIDER_TASK_ID_INVALID');
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(attemptId ?? ''))
    fail('PROVIDER_ATTEMPT_ID_INVALID');
  if (typeof sourceSha !== 'string' || !SHA.test(sourceSha))
    fail('PROVIDER_SOURCE_SHA_INVALID');
  return Object.freeze({taskId, attemptId, sourceSha, providerId});
}

/** No fallback, no scheduler, no in-memory execution journal. */
export function createAgentProviderRegistry(providers, {defaultProviderId = DEFAULT_AGENT_PROVIDER} = {}) {
  if (!Array.isArray(providers) || !validId(defaultProviderId))
    fail('PROVIDER_REGISTRY_INVALID');
  const installed = new Map();
  for (const provider of providers) {
    const checked = defineAgentProvider({...provider,
      probe: provider?.probe, execute: provider?.execute,
      cancel: provider?.cancel, reconcile: provider?.reconcile});
    if (installed.has(checked.id)) fail('PROVIDER_DUPLICATE_ID');
    installed.set(checked.id, checked);
  }
  return Object.freeze({
    list: () => Object.freeze([...installed.values()].map(p => Object.freeze({
      id: p.id, contractVersion: p.contractVersion, capabilities: p.capabilities,
    }))),
    resolve: ({providerId, enabledProviderIds = [defaultProviderId], pin} = {}) => {
      const selected = providerId ?? pin?.providerId ?? defaultProviderId;
      if (!validId(selected)) fail('PROVIDER_ID_INVALID');
      if (pin && pin.providerId !== selected) fail('PROVIDER_ATTEMPT_PIN_MISMATCH');
      if (!Array.isArray(enabledProviderIds) || !enabledProviderIds.includes(selected))
        fail('PROVIDER_DISABLED');
      const provider = installed.get(selected);
      if (!provider) fail('PROVIDER_NOT_INSTALLED');
      return provider;
    },
  });
}
