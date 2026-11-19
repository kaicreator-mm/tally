/**
 * Tally v2.1 candidate — DeepSeek Harness (DSH) adapter selection/config,
 * ported from PR #467 `packages/provider-dsh/src/dsh-config.mjs` (verbatim
 * semantics; see README.md for the behavior-retention list).
 *
 * Governance change for #473: the env two-gate below is NO LONGER the only
 * selection door. The authoritative door is the AgentProvider/1 registry
 * (`enabledProviderIds` must explicitly include `dsh`, default DISABLED).
 * The env gates are retained purely as defense-in-depth: when a Host sets
 * either env field, both must agree before the adapter acts (see
 * `dshEnvGateState` in dsh-adapter.mjs).
 *
 * CLI facts pinned here were verified against @deepseek-ai/dsh 0.2.2-rc.5
 * npm metadata and source in #467 (as-of 2026-11-19):
 *  - npm latest 0.2.2-rc.5 (2026-11-18), engines.node >=24.9.0
 *  - `dsh --version` prints one bare version line
 *  - headless: `dsh --profile headless [--patch <yml>...] "<task>"`,
 *    workspace root = process cwd, stdout = final answer, exit 0 only on
 *    completed turns
 *  - DSH_PERMISSION_MODE three values `read-only|workspace-write|
 *    danger-full-access`
 *  - SIGTERM drains gracefully and exits 0: cancellation must be classified
 *    from the adapter's own intent, never from exit codes
 */

/** Default pinned CLI version. Any observed mismatch refuses the session. */
export const PINNED_DSH_VERSION='0.2.2-rc.5';

/** SandboxMode values accepted by @deepseek-ai/dsh 0.2.x (source-verified). */
export const DSH_PERMISSION_MODES=Object.freeze(['read-only','workspace-write','danger-full-access']);

const PROVIDER_ID='dsh';

/**
 * #467 two-gate, default-off selection (pure function of env). Retained
 * verbatim as the defense-in-depth layer under the registry gate.
 */
export function resolveDshSelection(env={}){
  if(env.TALLY_ATTEMPT_PROVIDER!==PROVIDER_ID)
    return {selected:false,providerId:PROVIDER_ID,reason:'NOT_SELECTED_DEFAULT_OFF'};
  if(env.TALLY_DSH_ENABLED!=='1')
    return {selected:false,providerId:PROVIDER_ID,reason:'EXPLICIT_ENABLE_GATE_MISSING'};
  return {selected:true,providerId:PROVIDER_ID,reason:'SELECTED'};
}

/**
 * Defense-in-depth gate used by the adapter (#473). The registry's explicit
 * `enabledProviderIds` selection is the authoritative door; this only fires
 * when a Host has *set* one of the legacy env fields — then both must agree.
 * A Host that never touches the env fields is governed by the registry alone.
 */
export function dshEnvGateState(env={}){
  const selection=resolveDshSelection(env);
  const anyGateSet=env.TALLY_ATTEMPT_PROVIDER!==undefined||
    env.TALLY_DSH_ENABLED!==undefined;
  if(anyGateSet&&!selection.selected)return {open:false,reason:selection.reason};
  return {open:true,reason:selection.selected?'SELECTED':'REGISTRY_GATE_AUTHORITATIVE'};
}

function positiveInt(value,fallback){
  if(value===undefined||value==='')return fallback;
  const n=Number(value);
  if(!Number.isSafeInteger(n)||n<=0)throw Object.assign(
    new Error('DSH_CONFIG_INVALID_TIMEOUT'),{code:'DSH_CONFIG_INVALID_TIMEOUT'});
  return n;
}

/**
 * Build the DSH adapter config from the Host environment. The credential
 * value (`DEEPSEEK_API_KEY`) is carried for the execution environment only
 * and is never logged, recorded or echoed by this package.
 */
export function dshConfigFromEnv(env={},overrides={}){
  const permissionMode=env.TALLY_DSH_PERMISSION_MODE??'workspace-write';
  if(!DSH_PERMISSION_MODES.includes(permissionMode))throw Object.assign(
    new Error('DSH_CONFIG_INVALID_PERMISSION_MODE'),{code:'DSH_CONFIG_INVALID_PERMISSION_MODE',
      detail:{permissionMode,legal:[...DSH_PERMISSION_MODES]}});
  if(permissionMode==='danger-full-access'&&env.TALLY_DSH_ALLOW_DANGER_FULL_ACCESS!=='1')
    throw Object.assign(new Error('DSH_CONFIG_DANGER_FULL_ACCESS_NOT_PERMITTED'),
      {code:'DSH_CONFIG_DANGER_FULL_ACCESS_NOT_PERMITTED'});
  const config={
    providerId:PROVIDER_ID,
    binPath:env.TALLY_DSH_BIN_PATH??undefined,
    // Test/launcher seam: when the binary cannot be spawned directly (e.g.
    // Windows .cmd shims), point binPath at node and prefix the real entry.
    binArgsPrefix:env.TALLY_DSH_BIN_ARGS_PREFIX?env.TALLY_DSH_BIN_ARGS_PREFIX.split('|'):undefined,
    pinnedVersion:env.TALLY_DSH_PINNED_VERSION??PINNED_DSH_VERSION,
    permissionMode,
    allowDangerousFullAccess:env.TALLY_DSH_ALLOW_DANGER_FULL_ACCESS==='1',
    timeoutMs:positiveInt(env.TALLY_DSH_TIMEOUT_MS,300000),
    pipeGraceMs:positiveInt(env.TALLY_DSH_PIPE_GRACE_MS,5000),
    deepseekApiKey:env.DEEPSEEK_API_KEY??null,
    sandboxCliPath:env.TALLY_DSH_SANDBOX_CLI_PATH??env.TALLY_CODEX_SANDBOX_PATH??undefined,
    // Scratch CODEX_HOME selecting the sandbox wrapper's elevated backend.
    sandboxCodexHome:env.TALLY_DSH_SANDBOX_CODEX_HOME??env.TALLY_CODEX_HOME??undefined,
    gitPath:env.TALLY_GIT_PATH??undefined,
    // Per-attempt throwaway DSH_HOME override (default: fresh mkdtemp).
    dshHome:env.TALLY_DSH_HOME??undefined,
    systemRoot:env.TALLY_SYSTEM_ROOT??undefined,
    systemDrive:env.TALLY_SYSTEM_DRIVE??undefined,
    builderProfile:env.TALLY_BUILDER_PROFILE??undefined,
    builderTemp:env.TALLY_BUILDER_TEMP??undefined,
    emptyGitConfig:env.TALLY_EMPTY_GIT_CONFIG??undefined,
    modelProxy:env.TALLY_MODEL_PROXY??undefined,
    caBundle:env.TALLY_CA_BUNDLE??undefined
  };
  return {...config,...overrides};
}
