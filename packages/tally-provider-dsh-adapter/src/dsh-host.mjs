/**
 * Tally v2.1 candidate — DeepSeek Harness (DSH) process-level machinery,
 * ported from PR #467 `packages/provider-dsh/src/dsh-host.mjs`.
 *
 * Ported verbatim (the #467-verified behaviors; see README.md):
 *  - `spawnCapture` with the Go `exec.Cmd.WaitDelay` equivalent: when the
 *    process has exited but orphan descendants keep the output pipes open
 *    past `pipeGraceMs`, our pipe ends are destroyed so the wait converges
 *    and the run is classified `DSH_OUTPUT_PIPE_NOT_CONVERGED` (fail-closed,
 *    never a guessed success);
 *  - the allowlisted builder environment (no credential, no GitHub/SSH
 *    material) plus the execution-environment overlay: per-attempt throwaway
 *    `DSH_HOME`, `DSH_TELEMETRY_MODE=DISABLED`, `DSH_PERMISSION_MODE`, and
 *    `DEEPSEEK_API_KEY` injected only there — never logged or recorded;
 *  - the session-log upload disable `--patch` overlay (top-level YAML array
 *    form, live-verified in #467 against 0.2.2-rc.5);
 *  - fail-closed version probe: `dsh --version` must equal the pinned
 *    version exactly; credential presence probe: `DEEPSEEK_API_KEY`;
 *  - default tree kill (Windows `taskkill /T /F`; POSIX detached process
 *    group `SIGKILL`);
 *  - cancellation is classified from the adapter's own intent because DSH
 *    SIGTERM drains gracefully and exits 0 (#467-verified trap);
 *  - stdout must carry a final answer: exit 0 with empty stdout is FAILED.
 *
 * NOT ported (deliberately): the #467 Host admission chain — worktree
 * acquisition, OS-principal confinement proof, CAS-durable attempt records,
 * atomic resource grants, candidate write-tree/commit-tree/readback. Under
 * the AgentProvider/1 contract those remain solely with the existing
 * selected B/K/Host/C chain: a provider owns only its CLI/protocol
 * execution and reports untrusted observations. The Host supplies an
 * already admitted workspace (`request.worktreePath`).
 */
import {spawn} from 'node:child_process';
import {mkdir,mkdtemp,writeFile} from 'node:fs/promises';
import {existsSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {dirname,join} from 'node:path';
import {DSH_PERMISSION_MODES,PINNED_DSH_VERSION} from './dsh-config.mjs';

const SHA=/^[0-9a-f]{40}$/;
/** Contract-width attempt id (matches createProviderAttemptPin). */
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const TASK=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/;
const IS_WIN=process.platform==='win32';
const DEFAULT_GIT=IS_WIN?'C:\\Program Files\\Git\\cmd\\git.exe':'git';
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();
const tail=(text,runeCap)=>[...String(text??'')].slice(-runeCap).join('');
const redact=(text,secrets=[])=>{
  let out=String(text??'');
  for(const secret of secrets){
    if(typeof secret==='string'&&secret.length>=8)out=out.split(secret).join('[REDACTED]');
  }
  return out;
};
const sanitizeId=id=>String(id).replace(/[^A-Za-z0-9._-]/g,'-');

/**
 * Patch overlay written into the per-attempt DSH_HOME. `DSH_TELEMETRY_MODE=
 * DISABLED` only covers OTel; the default-on DeepSeek session-log upload is
 * a separate contributor whose `enabled` config must be turned off
 * explicitly. Format: 0.2.x overlays are a top-level YAML ARRAY of loader
 * patch entries (live-verified 2026-11-19 in #467: a mapping form is
 * rejected with "must be a top-level YAML array"; the array form is accepted
 * and the session-log row is source-derived from the 0.2.2-rc.5 config
 * catalog).
 */
const SESSION_LOG_DISABLE_PATCH=`# Tally provider-dsh-adapter per-attempt overlay:\n`+
  `# disable the default-on DeepSeek session-log upload (dsh_session_log).\n`+
  `- id: session-log-deepseek\n`+
  `  name: '@deepseek-ai/dsh-session-log-deepseek'\n`+
  `  config:\n`+
  `    enabled: false\n`;

function dshBuilderEnvironment(config){
  const root=config.systemRoot??(IS_WIN?'C:\\Windows':'/usr');
  const builderProfile=config.builderProfile??(IS_WIN?'C:\\Users\\CodexSandboxOnline':undefined);
  const gitDir=dirname(config.gitPath??DEFAULT_GIT);
  // Node's directory joins the PATH when the CLI runs through an explicit
  // interpreter entry (binPath = node + binArgsPrefix), so tool subprocesses
  // spawned by the harness can still resolve node.
  const binDir=typeof config.binPath==='string'&&config.binPath.includes('\\')?
    dirname(config.binPath):null;
  const path=IS_WIN?[...(binDir?[binDir]:[]),gitDir,root+'\\System32','C:\\Program Files\\Git\\cmd']
      .join(';')
    :['/usr/local/bin','/usr/bin','/bin'].join(':');
  const env={
    SystemRoot:root,SystemDrive:config.systemDrive??'C:',
    ComSpec:IS_WIN?root+'\\System32\\cmd.exe':'/bin/sh',
    PATH:path,
    USERPROFILE:builderProfile,HOME:builderProfile,
    GIT_CONFIG_NOSYSTEM:'1',
    GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never',
    GIT_AUTHOR_NAME:'Tally Builder',GIT_AUTHOR_EMAIL:'tally-builder@localhost',
    GIT_COMMITTER_NAME:'Tally Builder',GIT_COMMITTER_EMAIL:'tally-builder@localhost'
  };
  if(IS_WIN){
    env.PATHEXT='.COM;.EXE;.BAT;.CMD';
    env.APPDATA=builderProfile+'\\AppData\\Roaming';
    env.LOCALAPPDATA=builderProfile+'\\AppData\\Local';
    env.TEMP=config.builderTemp??builderProfile+'\\AppData\\Local\\Temp';
    env.TMP=env.TEMP;
    env.SSL_CERT_FILE=config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt';
    env.CURL_CA_BUNDLE=config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt';
  }else{
    env.TMPDIR=config.builderTemp??'/tmp';
  }
  // Proxy is opt-in: the DeepSeek model endpoint does not inherit the Codex
  // provider's proxy assumption (#467 live finding: api.deepseek.com is
  // directly reachable with no proxy).
  if(config.modelProxy){env.HTTPS_PROXY=config.modelProxy;env.HTTP_PROXY=config.modelProxy;}
  if(config.emptyGitConfig)env.GIT_CONFIG_GLOBAL=config.emptyGitConfig;
  // Scratch CODEX_HOME for the Host sandbox wrapper (the elevated backend
  // selection rides this home's config only — never the operator's).
  if(config.sandboxCodexHome)env.CODEX_HOME=config.sandboxCodexHome;
  // DEEPSEEK_API_KEY is intentionally NOT part of the base allowlist; it is
  // added only to the execution environment when explicitly configured.
  return env;
}

function dshExecutionEnvironment(config,prepared){
  const env={...dshBuilderEnvironment(config),
    DSH_HOME:prepared.dshHome,
    DSH_TELEMETRY_MODE:'DISABLED',
    DSH_PERMISSION_MODE:config.permissionMode};
  if(config.deepseekApiKey)env.DEEPSEEK_API_KEY=config.deepseekApiKey;
  return env;
}

/**
 * spawnCapture with a Go `WaitDelay` equivalent: `close` waits for stdio
 * convergence; when the process already exited and orphan descendants keep
 * the pipes open past pipeGraceMs, our pipe ends are destroyed (and the
 * orphan recorded) so the promise always converges.
 */
function spawnCapture(spawnFn,command,args,{cwd,env,input,timeoutMs=0,pipeGraceMs=0,detached=false}={}){
  const child=spawnFn(command,args,{cwd,env,windowsHide:true,detached,
    stdio:['pipe','pipe','pipe']});
  let stdout='',stderr='',settled=false,exited=false,orphanPipe=false;
  const append=(which,chunk)=>{
    const text=chunk.toString('utf8');
    if(which==='stdout')stdout=(stdout+text).slice(-65536);
    else stderr=(stderr+text).slice(-65536);
  };
  child.stdout?.on('data',chunk=>append('stdout',chunk));
  child.stderr?.on('data',chunk=>append('stderr',chunk));
  const startedAt=Date.now();
  const done=new Promise(resolveDone=>{
    let timer,graceTimer;
    const finish=value=>{if(!settled){settled=true;clearTimeout(timer);clearTimeout(graceTimer);
      resolveDone(value);}};
    child.once('error',()=>finish({code:null,signal:null,spawnError:true,pid:child.pid??null,
      stdout,stderr,startedAt,terminatedAt:Date.now()}));
    child.once('exit',()=>{
      if(settled)return;
      exited=true;
      if(pipeGraceMs>0){
        graceTimer=setTimeout(()=>{
          if(settled)return;
          orphanPipe=true;
          // Destroying our read ends releases `close` below; the tree itself
          // is killed by the host's terminate() path.
          child.stdout?.destroy?.();
          child.stderr?.destroy?.();
        },pipeGraceMs);
        graceTimer.unref?.();
      }
    });
    child.once('close',(code,signal)=>finish({code,signal,spawnError:false,orphanPipe,
      pid:child.pid??null,stdout,stderr,startedAt,terminatedAt:Date.now()}));
    if(input!==undefined){child.stdin?.end(input);}else child.stdin?.end();
    if(timeoutMs>0){
      timer=setTimeout(()=>{
        try{child.kill();}catch{}
        finish({code:null,signal:null,spawnError:false,probeTimeout:true,orphanPipe,
          pid:child.pid??null,stdout,stderr,startedAt,terminatedAt:Date.now()});
      },timeoutMs);
      timer.unref?.();
    }
  });
  return {child,done};
}

const sandboxPrefix=['-c','sandbox_mode="workspace-write"','-c',
  'sandbox_workspace_write.network_access=true','sandbox','--'];

function resolveBin(config,{checkExists=false}={}){
  if(typeof config.binPath==='string'&&config.binPath.length>0){
    if(checkExists&&(config.binPath.includes('/')||config.binPath.includes('\\'))&&
      !existsSync(config.binPath))
      return {bin:config.binPath,binArgs:config.binArgsPrefix??[],source:'config',
        error:'DSH_BIN_PATH_NOT_FOUND'};
    return {bin:config.binPath,binArgs:config.binArgsPrefix??[],source:'config'};
  }
  return {bin:'dsh',binArgs:[],source:'path'};
}

/**
 * Fail-closed capability probe, ported from #467 `probeProvider`: VERSION
 * (exact pinned-version match on `dsh --version`) then CREDENTIALS
 * (`DEEPSEEK_API_KEY` presence). Never spawns a headless turn.
 */
async function probeProvider(spawnFn,config){
  const resolved=resolveBin(config,{checkExists:true});
  const {bin,binArgs,source}=resolved;
  const pinned=config.pinnedVersion??PINNED_DSH_VERSION;
  if(resolved.error)
    return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
      provider:'dsh',detail:resolved.error,pinnedVersion:pinned,binSource:source};
  const run=(command,args)=>spawnCapture(spawnFn,command,args,
    {env:dshBuilderEnvironment(config),timeoutMs:15000}).done;
  const version=await (typeof config.sandboxCliPath==='string'&&config.sandboxCliPath?
    spawnCapture(spawnFn,config.sandboxCliPath,[...sandboxPrefix,bin,...binArgs,'--version'],
      {env:dshBuilderEnvironment(config),timeoutMs:15000}).done
    :run(bin,[...binArgs,'--version']));
  const observed=version.stdout.trim().split(/\r?\n/)[0]?.trim()??'';
  if(version.code!==0||version.spawnError||observed!==pinned)
    return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
      provider:'dsh',exitCode:version.code,spawnError:version.spawnError??false,
      observedVersion:observed||null,pinnedVersion:pinned,binSource:source,outputMatched:false};
  if(!config.deepseekApiKey)
    return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',
      provider:'dsh',credentialEnv:'DEEPSEEK_API_KEY',credentialPresent:false,
      pinnedVersion:pinned,binSource:source};
  return {available:true,provider:'dsh',providerVersion:observed,
    pinnedVersion:pinned,binSource:source};
}

/**
 * Default process-tree kill, ported verbatim from #467 `terminate`.
 */
function createDefaultTerminate(spawnFn){
  return async pid=>{
    if(IS_WIN){
      // Best-effort tree kill; a dead root (pid already reaped) is an
      // expected outcome, so this stays quiet instead of logging a failure.
      const {done}=spawnCapture(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
        ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
      const result=await done;
      return result.code===0;
    }
    // POSIX: the exec spawn is detached (own process group), so the negative
    // pid kills the whole group including orphaned tool descendants.
    try{process.kill(-pid,'SIGKILL');return true;}
    catch{try{process.kill(pid,'SIGKILL');return true;}catch{return false;}}
  };
}

/**
 * Headless argv, ported from #467 runAttempt: `--profile headless --patch
 * <overlay> "<task>"` (workspace root = process cwd; stdout = final answer).
 * When a Host sandbox wrapper is configured it fronts the CLI, exactly as
 * in #467.
 */
function headlessArgv(config,bin,binArgs,prompt,patchPath){
  return typeof config.sandboxCliPath==='string'&&config.sandboxCliPath?
    [config.sandboxCliPath,...sandboxPrefix,bin,...binArgs,'--profile','headless',
      '--patch',patchPath,prompt]:
    [bin,...binArgs,'--profile','headless','--patch',patchPath,prompt];
}

async function prepareDshHome(config,attemptId){
  const dshHome=config.dshHome??await mkdtemp(join(tmpdir(),'tally-dsh-home-'));
  await mkdir(dshHome,{recursive:true});
  const patchPath=join(dshHome,'tally-dsh.patch.yml');
  await writeFile(patchPath,SESSION_LOG_DISABLE_PATCH,'utf8');
  return {dshHome,patchPath};
}

/**
 * One-shot headless DSH turn with #467's exact outcome classification.
 * Returns `{classification, terminal, timedOut, cancelled, attempt}` where
 * `classification` is `{state, reason}` in AgentProvider/1 execute states.
 *
 * The classification order is the documented #467 deviation from the Codex
 * Host (nonzero exit is authoritative about the run itself, so process
 * failure and empty-final-answer surface before workspace complaints):
 * timeout/spawn-error -> UNKNOWN; adapter cancellation intent -> CANCELLED
 * (even when the graceful SIGTERM drain exits 0); orphan-pipe ->
 * FAILED; nonzero exit -> FAILED; empty stdout -> FAILED; otherwise
 * COMPLETED.
 */
function classifyDshTurnOutcome({terminal,timedOut,cancelled,finalAnswer}){
  if(timedOut)return {state:'UNKNOWN',reason:'ATTEMPT_TIMEOUT_RECONCILED'};
  if(terminal.spawnError)return {state:'UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN'};
  if(cancelled)return {state:'CANCELLED',reason:'ATTEMPT_CANCELLED'};
  if(terminal.orphanPipe)return {state:'FAILED',reason:'DSH_OUTPUT_PIPE_NOT_CONVERGED'};
  if(terminal.code!==0)return {state:'FAILED',reason:'ATTEMPT_PROCESS_FAILED'};
  if(String(finalAnswer??'').trim()==='')
    return {state:'FAILED',reason:'DSH_EMPTY_FINAL_ANSWER'};
  return {state:'COMPLETED',reason:'REAL_DSH_ATTEMPT_COMPLETED'};
}

/**
 * Run one `dsh --profile headless` turn as a one-shot process and classify
 * it per `classifyDshTurnOutcome`. `signal` abort maps to the adapter's own
 * cancellation intent (never to exit codes); `timeoutMs` kills the process
 * tree and yields UNKNOWN. `intent` is a mutable `{cancelled:boolean}` flag
 * shared with `cancel()`.
 */
async function runDshHeadlessTurn({spawnFn,terminate,config,env,cwd,prompt,
  patchPath,timeoutMs,signal,intent}){
  const {bin,binArgs}=resolveBin(config);
  const argv=headlessArgv(config,bin,binArgs,prompt,patchPath);
  const running=spawnCapture(spawnFn,argv[0],argv.slice(1),
    {cwd,env,timeoutMs:0,pipeGraceMs:config.pipeGraceMs??5000,detached:!IS_WIN});
  // Publish the pid immediately so a racing cancel() can target the tree.
  intent.pid=Number.isSafeInteger(running.child.pid)?running.child.pid:null;
  let timedOut=false,timer=null;
  const abort=()=>{intent.cancelled=true;
    terminate(running.child.pid).catch(()=>{});};
  if(Number.isSafeInteger(running.child.pid)){
    if(signal){
      if(signal.aborted)abort();
      else signal.addEventListener('abort',abort,{once:true});
    }
    timer=setTimeout(()=>{timedOut=true;
      terminate(running.child.pid).catch(()=>{});},timeoutMs);
    timer.unref?.();
  }
  const terminal=await running.done;
  clearTimeout(timer);
  signal?.removeEventListener('abort',abort);
  if((timedOut||intent.cancelled||terminal.orphanPipe)&&terminal.terminatedAt&&
    Number.isSafeInteger(running.child.pid)){
    // Orphan non-convergence also gets a best-effort tree kill; on Windows
    // taskkill /T cannot attach an already-dead root, so a surviving
    // descendant is bounded by the disposable worktree/DSH_HOME and kept
    // visible via process.orphanPipe evidence (ported #467 note).
    await terminate(running.child.pid).catch(()=>{});
  }
  const classification=classifyDshTurnOutcome({
    terminal,timedOut,cancelled:intent.cancelled===true,
    finalAnswer:terminal.spawnError?'':terminal.stdout});
  return {classification,terminal,timedOut,cancelled:intent.cancelled===true,
    pid:Number.isSafeInteger(running.child.pid)?running.child.pid:null,
    argv,startedAt:terminal.startedAt,terminatedAt:terminal.terminatedAt};
}

export {SHA,ATTEMPT,TASK,IS_WIN,DEFAULT_GIT,SESSION_LOG_DISABLE_PATCH,
  safeError,iso,tail,redact,sanitizeId,dshBuilderEnvironment,
  dshExecutionEnvironment,spawnCapture,resolveBin,probeProvider,
  createDefaultTerminate,headlessArgv,prepareDshHome,runDshHeadlessTurn,
  classifyDshTurnOutcome};
export const DSH_HOST_CONSTANTS=Object.freeze({
  pinnedDshVersion:PINNED_DSH_VERSION,permissionModes:DSH_PERMISSION_MODES,
  builderEnvironment:dshBuilderEnvironment,executionEnvironment:dshExecutionEnvironment,
  sessionLogDisablePatch:SESSION_LOG_DISABLE_PATCH});
