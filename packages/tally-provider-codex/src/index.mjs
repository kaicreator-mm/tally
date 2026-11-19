/**
 * @kaicreator/tally-provider-codex — Codex AgentProvider/1 bridge (#473).
 *
 * v2.1 candidate, default-off. Wraps the existing real Codex execution Host
 * (experiments/v2/selected-business-build/local-codex-host.mjs) as an
 * AgentProvider/1 provider WITHOUT re-implementing any host authority:
 * admission, confinement, CAS, git candidate readback, review and Outcome all
 * remain with the existing selected chain. This module's outputs are process
 * observations only and are never Task/Review/Release facts.
 */
import {AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider} from
  '../../tally-agent-provider/src/index.mjs';
import {
  BRIDGE_REASONS, CODEX_PROVIDER_ID, DEFAULT_HOST_MODULE_RELATIVE_PATH,
  EXECUTE_STATE_MAP, RECONCILE_STATE_MAP, createCodexBridge, mapAttemptOutcome,
  mapProbeOutcome, redactExecutionEvidence, sanitizeReason,
} from './codex-bridge.mjs';

export {
  BRIDGE_REASONS, CODEX_PROVIDER_ID, DEFAULT_HOST_MODULE_RELATIVE_PATH,
  EXECUTE_STATE_MAP, RECONCILE_STATE_MAP, mapAttemptOutcome, mapProbeOutcome,
  redactExecutionEvidence, sanitizeReason,
};

/**
 * Default lazy loader for the host module. Deliberately a dynamic import():
 * the host module imports `@kaicreator/domain-harness-host-node` at its top
 * level, which is NOT installed in every workspace; loading it must therefore
 * never happen at factory or module-graph time, only on first operation, and
 * a failed load must degrade to a contract-legal observation.
 */
export function defaultLoadHost() {
  return import(new URL(DEFAULT_HOST_MODULE_RELATIVE_PATH, import.meta.url).href);
}

/**
 * Create the Codex AgentProvider/1 provider.
 *
 * @param {object} [options]
 * @param {() => Promise<{createLocalCodexHost: Function}>} [options.loadHost]
 *   Lazy host module loader; tests inject a fake module. Defaults to
 *   {@link defaultLoadHost}.
 * @param {object|null} [options.config]
 *   Static host attempt configuration (hostStorePath, repositoryPath, cliPath,
 *   sandboxCliPath, builder identity fields, ...). Per-attempt fields
 *   (attemptId/sourceSha/worktreePath/timeoutMs/signal) are supplied per
 *   execute() request by the trusted Host.
 * @param {() => number} [options.now] Clock forwarded to the host factory.
 */
export function createCodexAgentProvider({loadHost = defaultLoadHost, config = null, now} = {}) {
  const bridge = createCodexBridge({loadHost, config, now});
  return defineAgentProvider({
    id: CODEX_PROVIDER_ID,
    contractVersion: AGENT_PROVIDER_CONTRACT_VERSION,
    capabilities: [],
    ...bridge,
  });
}
