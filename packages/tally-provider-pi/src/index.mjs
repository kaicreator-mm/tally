/**
 * Tally v2.1 candidate: AgentProvider/1 Node adapter for the pi coding agent
 * CLI (pinned 0.85.1), ported from the PR #469 Go candidate's observed CLI
 * contract. Source-only candidate for kaicreator-mm/tally#473.
 *
 * DEFAULT-OFF: this adapter is not mounted into the selected v2 Host. It only
 * participates through an explicit `enabledProviderIds` selection and actual
 * Host qualification; there is no fallback path. A provider COMPLETED result
 * is an untrusted observation, never a Task/Review/Release fact.
 *
 * Live acceptance (a real pi turn against the pinned CLI) is NOT run here —
 * it is the controller's acceptance gate. Probe reports canaryTurn:'NOT_RUN'.
 */

export {
  PI_PROVIDER_ID,
  PI_ADAPTER_VERSION,
  PINNED_PI_VERSION,
  PI_DEFAULT_CREDENTIAL_ENV_VAR,
  PI_DEFAULT_OUTPUT_MODE,
  createPiAgentProvider,
  buildPiCliArgs,
  isSafePiToken,
  parsePiStream,
  redactFine,
} from './pi-agent-provider.mjs';

export {
  PI_CREDENTIAL_ENV_VARS,
  PI_CONFIG_ENV_VARS,
  PI_FORCED_ENV_KEYS,
  PI_OFFLINE_FORCED,
  PI_TELEMETRY_FORCED,
  isStrippedEnvName,
  isForcedEnvName,
  buildPiSpawnEnv,
  createIsolationDirs,
  removeIsolationDirs,
} from './pi-env.mjs';
