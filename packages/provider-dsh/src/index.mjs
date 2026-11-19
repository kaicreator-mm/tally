/**
 * Tally v2.1 candidate — optional DeepSeek Harness (DSH) Agent Provider.
 *
 * Isolated, default-off package: no v2 MVP dependency, no second scheduler,
 * no GitHub writer, no R101 change. See README.md and DSH_PROVIDER_SEAMS.md.
 */
export {createLocalDshHost,createLocalDshHostWithDhStore,LOCAL_DSH_CONSTANTS}
  from './dsh-host.mjs';
export {PINNED_DSH_VERSION,DSH_PERMISSION_MODES,resolveDshSelection,dshConfigFromEnv}
  from './dsh-config.mjs';
