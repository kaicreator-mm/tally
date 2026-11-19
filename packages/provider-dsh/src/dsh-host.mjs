/**
 * Tally v2.1 candidate — optional DeepSeek Harness (DSH) local Agent Host.
 *
 * Mirrors the Local Codex Host contract (experiments/v2/selected-business-
 * build/local-codex-host.mjs on version/v2): one disposable detached Git
 * worktree per attempt, Host-owned candidate tree/commit/ref, CAS-durable
 * attempt records with atomic resource grants, honest UNKNOWN on timeout /
 * lost-ACK / ambiguous restart, and no Task/Review/Release PASS semantics.
 *
 * DSH-specific behavior (source-verified against @deepseek-ai/dsh 0.2.2-rc.5,
 * see DSH_PROVIDER_SEAMS.md):
 *  - fail-closed version pin: `dsh --version` must equal the pinned version
 *    exactly; developer-preview breaking changes surface as a refusal, never
 *    as guessed compatibility;
 *  - credential = DEEPSEEK_API_KEY presence in the (allowlisted) execution
 *    environment; the value never reaches records or logs;
 *  - per-attempt throwaway DSH_HOME so persistent sessions/storages/profiles
 *    never cross attempts, plus DSH_TELEMETRY_MODE=DISABLED and a patch
 *    overlay that turns the default-on DeepSeek session-log upload off;
 *  - cancellation is classified from the adapter's own intent because SIGTERM
 *    now drains gracefully and exits 0;
 *  - stdout must carry a final answer: exit 0 with empty stdout is FAILED
 *    (fail-safe), never a guessed success;
 *  - a Go `exec.Cmd.WaitDelay` equivalent: when the process has exited but
 *    orphan descendants keep the output pipes open past the grace window,
 *    the host destroys its pipe ends so wait converges, kills the tree and
 *    classifies DSH_OUTPUT_PIPE_NOT_CONVERGED (the P4-04 "declared but never
 *    wired" failure mode is asserted by tests).
 *
 * Differences from the Codex Host by design:
 *  - the CAS store is injected (`openStore`) so this package stays standalone
 *    (no non-npm dependency); production wiring passes the same public
 *    `openSqliteCasStore` from @kaicreator/domain-harness-host-node via
 *    `createLocalDshHostWithDhStore`;
 *  - attempt records live under their own key prefix, while resource grants
 *    use the byte-identical `tally:v2:local-agent-grant:` derivation so a
 *    DSH attempt and a Codex attempt can never hold the same worktree or
 *    candidate path concurrently;
 *  - `sandboxCliPath` is optional only in the sense that omitting it selects
 *    direct spawn for contract tests/smokes; on a real Host the default
 *    confinement proof (builder principal check) then fails closed.
 */
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync} from 'node:fs';
import {mkdir,writeFile} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {PINNED_DSH_VERSION,DSH_PERMISSION_MODES} from './dsh-config.mjs';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const SID=/^S-1-5-21(?:-\d+){4}$/;
const IS_WIN=process.platform==='win32';
const DEFAULT_GIT=IS_WIN?'C:\\Program Files\\Git\\cmd\\git.exe':'git';
/** Default candidate note written by the agent inside its worktree. */
const CANDIDATE='packages/provider-dsh/DSH_CANDIDATE_NOTE.md';
const ATTEMPT_KEY_PREFIX='tally:v2:local-dsh-attempt:';
// Byte-identical to local-codex-host.mjs: grants guard PHYSICAL resources and
// must exclude across providers, not just within one.
const GRANT_KEY_PREFIX='tally:v2:local-agent-grant:';
const grantKeyFor=(kind,value)=>GRANT_KEY_PREFIX+kind+':'+
  createHash('sha256').update(value).digest('hex');
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
 * patch entries (live-verified 2026-11-19: a mapping form is rejected with
 * "must be a top-level YAML array"; the array form is accepted and the
 * session-log row is source-derived from the 0.2.2-rc.5 config catalog).
 */
const SESSION_LOG_DISABLE_PATCH=`# Tally provider-dsh per-attempt overlay:\n`+
  `# disable the default-on DeepSeek session-log upload (dsh_session_log).\n`+
  `- id: session-log-deepseek\n`+
  `  name: '@deepseek-ai/dsh-session-log-deepseek'\n`+
  `  config:\n`+
  `    enabled: false\n`;

function dshBuilderEnvironment(config){
  const root=config.systemRoot??(IS_WIN?'C:\\Windows':'/usr');
  const builderProfile=config.builderProfile??(IS_WIN?'C:\\Users\\CodexSandboxOnline':undefined);
  const gitDir=dirname(config.gitPath??DEFAULT_GIT);
  const path=IS_WIN?[gitDir,root+'\\System32','C:\\Program Files\\Git\\cmd'].join(';')
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
  // provider's proxy assumption.
  if(config.modelProxy){env.HTTPS_PROXY=config.modelProxy;env.HTTP_PROXY=config.modelProxy;}
  if(config.emptyGitConfig)env.GIT_CONFIG_GLOBAL=config.emptyGitConfig;
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

function hostEnvironment(config){
  const root=config.systemRoot??(IS_WIN?'C:\\Windows':'/usr');
  return {SystemRoot:root,SystemDrive:config.systemDrive??'C:',
    ComSpec:IS_WIN?root+'\\System32\\cmd.exe':'/bin/sh',
    PATH:IS_WIN?root+'\\System32;C:\\Program Files\\Git\\cmd':'/usr/local/bin:/usr/bin:/bin',
    GIT_CONFIG_NOSYSTEM:'1',GIT_TERMINAL_PROMPT:'0',
    ...(config.emptyGitConfig?{GIT_CONFIG_GLOBAL:config.emptyGitConfig}:{})};
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

async function short(spawnFn,command,args,options={},secrets=[]){
  const {done}=spawnCapture(spawnFn,command,args,options);
  const result=await done;
  if(result.code!==0)console.error('HOST_COMMAND_FAILED',command,args.join(' '),
    JSON.stringify({...result,stdout:redact(result.stdout,secrets).slice(0,2000),
      stderr:redact(result.stderr,secrets).slice(0,2000)}));
  return result;
}
function encodedPowerShell(script){return Buffer.from(script,'utf16le').toString('base64');}
function quotePs(value){return "'"+value.replaceAll("'","''")+"'";}

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

function createDefaultOperations(spawnFn){
  // Host isolation wrapper (Codex Windows sandbox CLI) when configured. When
  // absent, commands spawn directly: on a real Host the default confinement
  // proof below then fails closed on the principal check. This exists for
  // contract tests and explicit CLI smokes, never as an isolation mechanism.
  const sandbox=(config,command,args,options={})=>{
    if(typeof config.sandboxCliPath==='string'&&config.sandboxCliPath){
      return short(spawnFn,config.sandboxCliPath,
        [...sandboxPrefix,command,...args],{env:dshBuilderEnvironment(config),...options},
        config.deepseekApiKey?[config.deepseekApiKey]:[]);
    }
    return short(spawnFn,command,args,{env:dshBuilderEnvironment(config),...options},
      config.deepseekApiKey?[config.deepseekApiKey]:[]);
  };
  const git=(config,args,options={})=>short(spawnFn,config.gitPath??DEFAULT_GIT,
    args,{env:hostEnvironment(config),...options});
  async function probeProvider(config){
    console.error('HOST_PROBE_PROVIDER_START','dsh');
    const resolved=resolveBin(config,{checkExists:true});
    const {bin,binArgs,source}=resolved;
    const pinned=config.pinnedVersion??PINNED_DSH_VERSION;
    if(resolved.error)
      return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
        provider:'dsh',detail:resolved.error,pinnedVersion:pinned,binSource:source};
    const version=await sandbox(config,bin,[...binArgs,'--version'],{timeoutMs:15000});
    const observed=version.stdout.trim().split(/\r?\n/)[0]?.trim()??'';
    if(version.code!==0||version.spawnError||observed!==pinned)
      return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
        provider:'dsh',exitCode:version.code,spawnError:version.spawnError??false,
        observedVersion:observed||null,pinnedVersion:pinned,binSource:source,outputMatched:false};
    if(!config.deepseekApiKey)
      return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH',
        provider:'dsh',credentialEnv:'DEEPSEEK_API_KEY',credentialPresent:false,
        pinnedVersion:pinned,binSource:source};
    return {available:true,provider:'dsh',providerVersion:observed,
      pinnedVersion:pinned,binSource:source};
  }
  async function readDenied(config,path,isDirectory=false){
    const access=isDirectory?
      `[System.IO.Directory]::EnumerateFileSystemEntries(${quotePs(path)}).GetEnumerator().MoveNext() | Out-Null`:
      `$s=[System.IO.File]::OpenRead(${quotePs(path)});$s.Dispose()`;
    const script=`try { ${access}; exit 0 } catch { exit 1 }`;
    const result=await sandbox(config,config.powershellPath??
      'C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe',
      ['-NoProfile','-NonInteractive','-EncodedCommand',encodedPowerShell(script)]);
    return result.code!==0;
  }
  async function proveConfinement(config,worktreePath){
    if(!IS_WIN)return {verified:false,check:'PLATFORM',reason:'BUILDER_CONFINEMENT_UNAVAILABLE',
      detail:'default confinement proof is Windows-principal based; POSIX Host wiring pending'};
    const identity=await sandbox(config,'C:\\Windows\\System32\\whoami.exe',[]);
    const principal=identity.stdout.trim().toLowerCase();
    if(identity.code!==0||principal!==config.builderPrincipal.toLowerCase())
      return {verified:false,check:'PRINCIPAL',reason:'BUILDER_CONFINEMENT_UNAVAILABLE',
        observedPrincipal:principal,expectedPrincipal:config.builderPrincipal.toLowerCase(),exitCode:identity.code};
    const user=await sandbox(config,'C:\\Windows\\System32\\whoami.exe',['/user']);
    if(user.code!==0||!user.stdout.includes(config.builderSid))
      return {verified:false,check:'SID',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    const groups=await sandbox(config,'C:\\Windows\\System32\\whoami.exe',['/groups']);
    if(groups.code!==0||groups.stdout.includes('S-1-5-32-544')||
      groups.stdout.includes('S-1-16-12288'))
      return {verified:false,check:'ADMIN',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    if(!SID.test(config.builderSid)||config.builderSid===config.verifierSid)
      return {verified:false,check:'VERIFIER_PRINCIPAL',reason:'HIDDEN_VERIFIER_UNAVAILABLE'};
    if(typeof config.hiddenFixture!=='string'||!existsSync(config.hiddenFixture))
      return {verified:false,check:'HIDDEN_MISSING',reason:'HIDDEN_VERIFIER_UNAVAILABLE'};
    const operatorHome=config.operatorHome;
    const deniedPaths=[
      ['GH',operatorHome+'\\.config\\gh\\hosts.yml',false],
      ['GCM_FILE',operatorHome+'\\.git-credentials',false],
      ['OPERATOR_DSH',operatorHome+'\\.dsh',true],
      ['SSH_KEYS',operatorHome+'\\.ssh',true],
      ['HIDDEN',config.hiddenFixture,false]
    ];
    for(const [check,path,directory] of deniedPaths){
      if(!existsSync(path))continue;
      if(!await readDenied(config,path,directory))return {verified:false,check,
        reason:check==='HIDDEN'?'HIDDEN_VERIFIER_UNAVAILABLE':'BUILDER_CONFINEMENT_UNAVAILABLE'};
    }
    const credential=await git(config,['credential','fill'],{input:'protocol=https\nhost=github.com\n\n'});
    if(credential.code===0&&/(?:^|\r?\n)(?:username|password)=/.test(credential.stdout))
      return {verified:false,check:'GCM',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    if(config.sshAddPath&&existsSync(config.sshAddPath)){
      const agent=await sandbox(config,config.sshAddPath,['-l']);
      if(agent.code===0)return {verified:false,check:'SSH_AGENT',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    }
    const probeRef='HEAD:refs/_tally_probe/'+sanitizeId(config.attemptId);
    const remote=await git(config,['-C',worktreePath,'push','--dry-run','origin',probeRef],{timeoutMs:30000});
    if(remote.code===0)return {verified:false,check:'REMOTE_WRITE',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    return {verified:true,builderPrincipal:principal,builderSid:config.builderSid,
      hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'};
  }
  async function prepareWorktree(config){
    console.error('HOST_PREPARE_WORKTREE_START',config.worktreePath);
    await mkdir(dirname(config.worktreePath),{recursive:true});
    if(existsSync(config.worktreePath))throw safeError('CANDIDATE_WORKTREE_ALREADY_EXISTS');
    const dshHome=config.dshHome??join(dirname(config.worktreePath),
      '.tally-dsh-home-'+sanitizeId(config.attemptId));
    if(existsSync(dshHome))throw safeError('DSH_HOME_ALREADY_EXISTS');
    const added=await short(spawnFn,config.gitPath??DEFAULT_GIT,['-C',config.repositoryPath,
      'worktree','add','--detach',config.worktreePath,config.sourceSha],
      {env:hostEnvironment(config),timeoutMs:60000});
    if(added.code!==0)throw safeError('CANDIDATE_WORKTREE_UNAVAILABLE');
    const head=await short(spawnFn,config.gitPath??DEFAULT_GIT,['-C',config.worktreePath,
      'rev-parse','HEAD'],{env:hostEnvironment(config)});
    if(head.code!==0||head.stdout.trim()!==config.sourceSha)throw safeError('CANDIDATE_BASE_MISMATCH');
    await mkdir(dshHome,{recursive:true});
    const patchPath=join(dshHome,'tally-dsh.patch.yml');
    await writeFile(patchPath,SESSION_LOG_DISABLE_PATCH,'utf8');
    if(IS_WIN){
      for(const [path,permission] of [[dirname(config.worktreePath),'RX'],
        [config.worktreePath,'M'],[dshHome,'F']]){
        const acl=await short(spawnFn,'C:\\Windows\\System32\\icacls.exe',[path,'/grant',
          `*${config.builderSid}:(OI)(CI)(${permission})`],{env:hostEnvironment(config),timeoutMs:15000});
        if(acl.code!==0)throw safeError('BUILDER_ACL_UNAVAILABLE');
      }
    }
    return {worktreePath:config.worktreePath,baseSha:head.stdout.trim(),dshHome,patchPath};
  }
  async function finalizeCandidate(config){
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    const run=(args,options={})=>short(spawnFn,config.gitPath??DEFAULT_GIT,
      ['-C',config.worktreePath,...args],{env:hostEnvironment(config),...options});
    const [tracked,untracked]=await Promise.all([
      run(['diff','--name-only',config.sourceSha]),
      run(['ls-files','--others','--exclude-standard','--no-directory'])
    ]);
    if(tracked.code!==0||untracked.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const paths=[...new Set([...tracked.stdout.split(/\r?\n/),...untracked.stdout.split(/\r?\n/)]
      .map(x=>x.trim()).filter(Boolean))];
    if(paths.length!==1||paths[0]!==candidatePath)return {writeSetValid:false,paths};
    const added=await run(['add','--',candidatePath]);
    if(added.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const tree=await run(['write-tree']);
    if(tree.code!==0||!SHA.test(tree.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const treeSha=tree.stdout.trim();
    const issueNumber=/\/issues\/([1-9][0-9]*)$/.exec(config.issueRef??'')?.[1];
    const message=`Tally local DSH candidate ${config.attemptId}\n\nRefs: #${issueNumber??'64'}`;
    const commit=await run(['commit-tree',treeSha,'-p',config.sourceSha,'-m',message],{
      env:{...hostEnvironment(config),GIT_AUTHOR_NAME:'Tally Host',GIT_AUTHOR_EMAIL:'tally-host@localhost',
        GIT_COMMITTER_NAME:'Tally Host',GIT_COMMITTER_EMAIL:'tally-host@localhost'}});
    if(commit.code!==0||!SHA.test(commit.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const commitSha=commit.stdout.trim();
    const ref='refs/tally-candidates/'+sanitizeId(config.attemptId);
    const updated=await run(['update-ref',ref,commitSha]);
    if(updated.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const readback=await run(['rev-parse',commitSha+'^{tree}']);
    if(readback.code!==0||readback.stdout.trim()!==treeSha)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    return {writeSetValid:true,paths,candidateTree:treeSha,candidateCommit:commitSha,candidateRef:ref};
  }
  async function verifyCandidateReadback(config,candidate){
    // Independent git readback of one exact candidate: commit parent, tree and
    // the single changed path are re-derived from the repository, never from
    // Agent-asserted values.
    const run=(args,options={})=>short(spawnFn,config.gitPath??DEFAULT_GIT,
      ['-C',config.repositoryPath,...args],{env:hostEnvironment(config),...options});
    const commit=candidate.candidateCommit;
    if(!SHA.test(commit??''))return {verified:false};
    const [tree,parent,diff]=await Promise.all([
      run(['rev-parse',commit+'^{tree}']),
      run(['rev-parse',commit+'^']),
      run(['diff','--name-only',config.sourceSha,commit])
    ]);
    if(tree.code!==0||parent.code!==0||diff.code!==0)return {verified:false};
    return {verified:true,candidateTree:tree.stdout.trim(),parent:parent.stdout.trim(),
      paths:diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean)};
  }
  return {probeProvider,proveConfinement,prepareWorktree,finalizeCandidate,
    verifyCandidateReadback,sandbox};
}

const ACTIVE_PHASES=new Set(['STARTING','PERMITTED','RUNNING']);
const overlapDenied=(base,reason)=>({...base,state:'NOT_ADMITTED',reason,
  claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
  candidateTree:null,candidateCommit:null,process:null});

/** Default operations ports, exported for composability in tests/wiring. */
export const createDefaultDshOperations=createDefaultOperations;

export function createLocalDshHost({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree,openStore}={}){
  if(typeof openStore!=='function')throw safeError('DSH_HOST_STORE_FACTORY_REQUIRED');
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
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
  });
  async function runAttempt(config){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={providerId:'dsh',issueRef:config.issueRef,sourceSha:config.sourceSha,
      attemptId:config.attemptId,...flags};
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    const secrets=config.deepseekApiKey?[config.deepseekApiKey]:[];
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.attemptId??'')||
      !SID.test(config.builderSid??'')||!DSH_PERMISSION_MODES.includes(config.permissionMode??'')||
      (config.permissionMode==='danger-full-access'&&!config.allowDangerousFullAccess))
      return {...base,state:'AGENT_NOT_EXECUTED',reason:'DSH_CONFIG_INVALID',
        agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,process:null};
    let store=await openStore({path:config.hostStorePath,busyTimeoutMs:5000});
    const key=ATTEMPT_KEY_PREFIX+config.attemptId;
    try{
      const existing=await store.get(key);
      if(existing){
        const value=existing.value;
        if(value?.phase==='CANDIDATE_READY')return {...base,state:'LOCAL_CANDIDATE_READY',
          reason:'REAL_DSH_CANDIDATE_RECORDED',agentLaunch:true,replayFixture:false,
          candidateTree:value.candidateTree,candidateCommit:value.candidateCommit,process:value.process};
        if(value?.phase==='NOT_EXECUTED')return {...base,state:'AGENT_NOT_EXECUTED',
          reason:value.reason,agentLaunch:false,replayFixture:false,candidateTree:null,
          candidateCommit:null,process:null};
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
          agentLaunch:true,replayFixture:false,candidateTree:value?.candidateTree??null,
          candidateCommit:value?.candidateCommit??null,
          process:value?.process??{status:'UNKNOWN',exitCode:null,pid:null,
            builderPrincipal:config.builderPrincipal,builderSid:config.builderSid,
            startedAt:null,terminatedAt:null}};
      }
      const grants=[];
      const releaseGrants=async()=>{
        for(const grant of grants.reverse())await store.remove(grant.key,{mustMatchRevision:grant.revision});
        grants.length=0;
      };
      const acquireGrant=async(kind,value,reason)=>{
        const grantKey=grantKeyFor(kind,value);
        try{
          await store.put(grantKey,{phase:'ACTIVE',kind,value,attemptId:config.attemptId},
            {mustBeAbsent:true});
          grants.push({key:grantKey,revision:0});return null;
        }catch{await releaseGrants();return overlapDenied(base,reason);}
      };
      // Shared-derivation grants close the cross-provider (Codex/DSH) race on
      // physical worktree and candidate paths; released once a terminal
      // attempt record exists, retained for operator reconciliation on crash.
      const worktreeGrant=await acquireGrant('worktree',config.worktreePath,'CANDIDATE_WORKTREE_OVERLAP');
      if(worktreeGrant)return worktreeGrant;
      const writeGrant=await acquireGrant('candidate',candidatePath,'CANDIDATE_WRITE_GRANT_OVERLAP');
      if(writeGrant)return writeGrant;
      for(const otherKey of await store.list(ATTEMPT_KEY_PREFIX)){
        if(otherKey===key)continue;
        const other=(await store.get(otherKey))?.value;
        if(!other||!ACTIVE_PHASES.has(other.phase))continue;
        if(other.worktreePath&&other.worktreePath===config.worktreePath){
          await releaseGrants();return overlapDenied(base,'CANDIDATE_WORKTREE_OVERLAP');
        }
        if((other.candidateRelativePath??CANDIDATE)===candidatePath){
          await releaseGrants();return overlapDenied(base,'CANDIDATE_WRITE_GRANT_OVERLAP');
        }
      }
      const startedAt=iso(now);
      await store.put(key,{phase:'STARTING',attemptId:config.attemptId,sourceSha:config.sourceSha,
        issueRef:config.issueRef,providerId:'dsh',claimGeneration:1,worktreePath:config.worktreePath,
        candidateRelativePath:candidatePath,permissionMode:config.permissionMode,
        pinnedVersion:config.pinnedVersion??PINNED_DSH_VERSION,startedAt},{mustBeAbsent:true});
      let revision=0;
      const update=async value=>{const result=await store.put(key,value,{mustMatchRevision:revision});
        revision=result.revision;};
      const notExecuted=async (reason,detail={})=>{
        const current=await store.get(key);revision=current.revision;
        await update({...current.value,phase:'NOT_EXECUTED',reason,...detail,terminatedAt:iso(now)});
        await releaseGrants();
        return {...base,state:'AGENT_NOT_EXECUTED',reason,agentLaunch:false,replayFixture:false,
          candidateTree:null,candidateCommit:null,process:null};
      };
      const probe=await ops.probeProvider(config);
      if(!probe.available)return await notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
      const worktree=await ops.prepareWorktree(config);
      let current=await store.get(key);revision=current.revision;
      await update({...current.value,worktreePath:worktree.worktreePath,baseSha:worktree.baseSha,
        dshHome:worktree.dshHome,patchPath:worktree.patchPath});
      const confinement=await ops.proveConfinement(config,worktree.worktreePath);
      if(!confinement.verified)return await notExecuted(confinement.reason,{confinement});
      const env=dshExecutionEnvironment(config,worktree);
      const prompt=`Work only in the current disposable Git worktree. Create exactly ${candidatePath} with a short Markdown candidate note containing attempt ${config.attemptId}. Do not modify any other path, do not commit, do not use GitHub or network except the model provider, and stop after writing the file.`;
      const {bin,binArgs,source}=resolveBin(config);
      const argv=typeof config.sandboxCliPath==='string'&&config.sandboxCliPath?
        [config.sandboxCliPath,...sandboxPrefix,bin,...binArgs,'--profile','headless',
          '--patch',worktree.patchPath,prompt]:
        [bin,...binArgs,'--profile','headless','--patch',worktree.patchPath,prompt];
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase:'PERMITTED',argv,cwd:worktree.worktreePath,
        envAllowlist:Object.keys(env),binSource:source,
        sandboxWrapper:typeof config.sandboxCliPath==='string'&&config.sandboxCliPath?
          'HOST_SANDBOX':'DIRECT_SPAWN_CONFINEMENT_REFUSED_ON_REAL_HOST',
        builderPrincipal:confinement.builderPrincipal,
        builderSid:confinement.builderSid,hiddenVerifierStatus:confinement.hiddenVerifierStatus});
      const running=spawnCapture(spawnFn,argv[0],argv.slice(1),{cwd:worktree.worktreePath,env,
        timeoutMs:0,pipeGraceMs:config.pipeGraceMs??5000,detached:!IS_WIN});
      if(!Number.isSafeInteger(running.child.pid)){
        const reason='ATTEMPT_STATUS_UNKNOWN';
        current=await store.get(key);revision=current.revision;
        await update({...current.value,phase:'UNKNOWN',reason,process:{status:'UNKNOWN',pid:null}});
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason,agentLaunch:true,replayFixture:false,
          candidateTree:null,candidateCommit:null,process:{status:'UNKNOWN',exitCode:null,pid:null,
            builderPrincipal:confinement.builderPrincipal,builderSid:confinement.builderSid,
            startedAt,terminatedAt:iso(now)}};
      }
      const processBase={pid:running.child.pid,builderPrincipal:confinement.builderPrincipal,
        builderSid:confinement.builderSid,startedAt};
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase:'RUNNING',process:{status:'RUNNING',...processBase}});
      let timedOut=false,cancelled=false;
      const timeoutMs=config.timeoutMs??300000;
      const timer=setTimeout(()=>{timedOut=true;terminate(running.child.pid).catch(()=>{});},timeoutMs);
      timer.unref?.();
      const abort=()=>{cancelled=true;terminate(running.child.pid).catch(()=>{});};
      config.signal?.addEventListener('abort',abort,{once:true});
      const terminal=await running.done;
      clearTimeout(timer);config.signal?.removeEventListener('abort',abort);
      if((timedOut||cancelled||terminal.orphanPipe)&&terminal.terminatedAt){
        // Orphan non-convergence also gets a best-effort tree kill; on
        // Windows taskkill /T cannot attach an already-dead root, so a
        // surviving descendant is bounded by the disposable worktree/DSH_HOME
        // and kept visible via process.orphanPipe evidence.
        await terminate(running.child.pid).catch(()=>{});
      }
      // Overlap evidence uses the ACTUAL process lifetime (spawn-ack time),
      // not the earlier reservation/probe time.
      const processStartedAt=new Date(terminal.startedAt).toISOString();
      const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
        orphanPipe:terminal.orphanPipe===true,
        builderPrincipal:confinement.builderPrincipal,builderSid:confinement.builderSid,
        startedAt:processStartedAt,terminatedAt:iso(now)};
      const finalAnswer=terminal.stdout??'';
      let state,reason,phase;
      // Reason order (documented deviation from the Codex Host, which checks
      // the write set before the exit code): a nonzero exit is authoritative
      // about the run itself — e.g. the MISSING_CREDENTIAL stderr shape — so
      // ATTEMPT_PROCESS_FAILED and DSH_EMPTY_FINAL_ANSWER surface before the
      // workspace write-set complaint. Every branch still fails closed.
      if(timedOut||terminal.spawnError||terminal.probeTimeout){
        state='LOCAL_ATTEMPT_UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';phase='UNKNOWN';
      }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';phase='CANCELLED';
      }else if(terminal.orphanPipe){state='LOCAL_ATTEMPT_FAILED';reason='DSH_OUTPUT_PIPE_NOT_CONVERGED';phase='FAILED';
      }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';phase='FAILED';
      }else if(finalAnswer.trim()===''){state='LOCAL_ATTEMPT_FAILED';reason='DSH_EMPTY_FINAL_ANSWER';phase='FAILED';
      }else if(!candidate.writeSetValid){
        state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';phase='FAILED';
      }else{state='LOCAL_CANDIDATE_READY';reason='REAL_DSH_CANDIDATE_RECORDED';phase='CANDIDATE_READY';}
      const recordExtras=state==='LOCAL_ATTEMPT_FAILED'?
        {stderrTail:redact(tail(terminal.stderr,500),secrets)}:{};
      if(state==='LOCAL_CANDIDATE_READY'){
        recordExtras.finalAnswerSha256=createHash('sha256').update(finalAnswer).digest('hex');
        recordExtras.finalAnswerBytes=Buffer.byteLength(finalAnswer);
        recordExtras.finalAnswerPreview=tail(finalAnswer.trim(),200);
      }
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase,reason,process,candidateTree:candidate.candidateTree??null,
        candidateCommit:candidate.candidateCommit??null,candidateRef:candidate.candidateRef??null,
        ...recordExtras,terminatedAt:process.terminatedAt});
      await releaseGrants();
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,
        process,...(state==='LOCAL_CANDIDATE_READY'?
          {finalAnswerSha256:recordExtras.finalAnswerSha256,
            finalAnswerPreview:recordExtras.finalAnswerPreview}:{})};
    }finally{await store.close();}
  }
  return Object.freeze({runAttempt,
    probeProvider:config=>ops.probeProvider(config),
    proveConfinement:(config,worktreePath)=>ops.proveConfinement(config,worktreePath)});
}

/**
 * Convenience factory for the real wiring context. The public DomainHarness
 * SQLite CAS store can be passed explicitly (`openSqliteCasStore`) — ESM
 * dynamic import resolves from THIS module's location, so the import
 * fallback only works when this source runs inside the consumer install
 * lane (e.g. the sealed bundle copied next to the installed
 * @kaicreator/domain-harness-host-node, as run-clean.sh does for the Codex
 * provider).
 */
export async function createLocalDshHostWithDhStore({openSqliteCasStore,...options}={}){
  const store=openSqliteCasStore??(await import('@kaicreator/domain-harness-host-node'))
    .openSqliteCasStore;
  return createLocalDshHost({...options,
    openStore:({path,busyTimeoutMs})=>store({path,busyTimeoutMs})});
}

export const LOCAL_DSH_CONSTANTS=Object.freeze({candidateRelativePath:CANDIDATE,
  attemptKeyPrefix:ATTEMPT_KEY_PREFIX,grantKeyPrefix:GRANT_KEY_PREFIX,
  pinnedDshVersion:PINNED_DSH_VERSION,permissionModes:DSH_PERMISSION_MODES,
  builderEnvironment:dshBuilderEnvironment,executionEnvironment:dshExecutionEnvironment});
