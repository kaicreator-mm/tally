/**
 * Tally v2.1 candidate — @kaicreator/tally-provider-dsh-adapter.
 *
 * DeepSeek Harness (DSH) AgentProvider/1 adapter, adapted from the #467
 * Node DSH Provider package. Isolated, DEFAULT-OFF (registry-gated), zero
 * npm dependencies: no v2 MVP dependency, no second scheduler, no GitHub
 * writer, no R101 change. See README.md.
 */
export {defineDshAgentProviderSpecification,createDshAgentProvider}
  from './dsh-adapter.mjs';
export {PINNED_DSH_VERSION,DSH_PERMISSION_MODES,resolveDshSelection,
  dshEnvGateState,dshConfigFromEnv} from './dsh-config.mjs';
export {DSH_HOST_CONSTANTS,classifyDshTurnOutcome,spawnCapture,
  dshBuilderEnvironment,dshExecutionEnvironment,resolveBin,probeProvider,
  createDefaultTerminate} from './dsh-host.mjs';
