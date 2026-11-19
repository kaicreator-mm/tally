/**
 * Tally v2.1 candidate — optional DeepSeek Harness (DSH) provider selection
 * and Host-environment configuration.
 *
 * Governance: #392 freezes exactly ONE selected real Agent provider (Local
 * Codex) for the v2 installed execution path; this package is therefore an
 * isolated, default-off candidate outside that path. Selection requires TWO
 * explicit Host environment gates and can never be on by default.
 *
 * CLI facts pinned here were verified against @deepseek-ai/dsh npm metadata
 * and source (see DSH_PROVIDER_SEAMS.md, as-of 2026-11-19):
 *  - npm latest 0.2.2-rc.5 (2026-11-18), engines.node >=24.9.0
 *  - `dsh --version` prints one bare version line
 *  - headless: `dsh --profile headless [--patch <yml>...] "<task>"`,
 *    workspace root = process cwd, stdout = final answer, exit 0 only on
 *    completed turns
 *  - DSH_PERMISSION_MODE adds a third legal value `read-only` in 0.2.x
 *  - SIGTERM now drains gracefully and exits 0: cancellation must be
 *    classified from the adapter's own intent, never from exit codes
 */

/** Default pinned CLI version. Any observed mismatch refuses the session. */
export const PINNED_DSH_VERSION='0.2.2-rc.5';

/** SandboxMode values accepted by @deepseek-ai/dsh 0.2.x (source-verified). */
export const DSH_PERMISSION_MODES=Object.freeze(['read-only','workspace-write','danger-full-access']);

const PROVIDER_ID='dsh';

/** Two-gate, default-off provider selection (pure function of env). */
export function resolveDshSelection(env={}){
  if(env.TALLY_ATTEMPT_PROVIDER!==PROVIDER_ID)
    return {selected:false,providerId:PROVIDER_ID,reason:'NOT_SELECTED_DEFAULT_OFF'};
  if(env.TALLY_DSH_ENABLED!=='1')
    return {selected:false,providerId:PROVIDER_ID,reason:'EXPLICIT_ENABLE_GATE_MISSING'};
  return {selected:true,providerId:PROVIDER_ID,reason:'SELECTED'};
}

function positiveInt(value,fallback){
  if(value===undefined||value==='')return fallback;
  const n=Number(value);
  if(!Number.isSafeInteger(n)||n<=0)throw Object.assign(
    new Error('DSH_CONFIG_INVALID_TIMEOUT'),{code:'DSH_CONFIG_INVALID_TIMEOUT'});
  return n;
}

/**
 * Build the DSH Host config from the Host environment. The credential value
 * (`DEEPSEEK_API_KEY`) is carried for the execution environment only and is
 * never logged, recorded or echoed by this package.
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
    repositoryPath:env.TALLY_REPOSITORY_PATH??undefined,
    worktreePath:env.TALLY_LOCAL_WORKTREE_PATH??undefined,
    dshHome:env.TALLY_DSH_HOME??undefined,
    issueRef:env.TALLY_ISSUE_REF??undefined,
    sourceSha:env.TALLY_SOURCE_SHA??undefined,
    attemptId:env.TALLY_LOCAL_ATTEMPT_ID??undefined,
    candidateRelativePath:env.TALLY_CANDIDATE_RELATIVE_PATH??undefined,
    hostStorePath:env.TALLY_LOCAL_HOST_STORE_PATH??undefined,
    builderProfile:env.TALLY_BUILDER_PROFILE??undefined,
    builderTemp:env.TALLY_BUILDER_TEMP??undefined,
    builderPrincipal:env.TALLY_BUILDER_PRINCIPAL??undefined,
    builderSid:env.TALLY_BUILDER_SID??undefined,
    verifierSid:env.TALLY_VERIFIER_SID??undefined,
    operatorHome:env.TALLY_OPERATOR_HOME??undefined,
    hiddenFixture:env.TALLY_HIDDEN_FIXTURE??undefined,
    emptyGitConfig:env.TALLY_EMPTY_GIT_CONFIG??undefined,
    modelProxy:env.TALLY_MODEL_PROXY??undefined,
    caBundle:env.TALLY_CA_BUNDLE??undefined,
    powershellPath:env.TALLY_POWERSHELL_PATH??undefined,
    sshAddPath:env.TALLY_SSH_ADD_PATH??undefined,
    systemRoot:env.TALLY_SYSTEM_ROOT??undefined,
    systemDrive:env.TALLY_SYSTEM_DRIVE??undefined
  };
  return {...config,...overrides};
}
