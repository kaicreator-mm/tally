/** Grok Agent Provider adapter (#64, v2.1 candidate; NOT a v2 MVP requirement).
 * Mirrors the zcode-provider/local-codex-host execution contract — same
 * injectable spawn shape, same state/reason vocabulary, same git-derived
 * candidate evidence — without touching the selected v2 Codex path: the
 * default provider stays codex and grok is only usable through
 * resolveGrokProviderSelection with explicit config plus a locally validated
 * capability probe. Durability is git-anchored only (candidate ref + worktree
 * occupancy); there is no second scheduler, kernel, permission system or
 * event log. taskPass/releasePass are constant false: agent output is a
 * claim, never evidence.
 *
 * CLI CONTRACT — anchored to the PR #468 Go adapter (runtimes/grok, live
 * observations of the official xAI Grok Build CLI @xai-official/grok native
 * binary 1.0.87, commit 929e6dd187b1, win32-x64). No flag is invented; every
 * argv element below traces to an observed surface. Live re-verification on
 * this host, 2026-11-19: `grok --version` → stdout `grok 1.0.87
 * (929e6dd187b1)`, EXIT=0; keyless `--no-auto-update --single=<p>` → EXIT=1,
 * empty stdout, stderr `Not signed in. ...`; empty temp GROK_HOME accepted,
 * directory kept empty, no ~/.grok writes.
 *
 * PR #468 FAILURE / TRAP LEDGER AND THIS ADAPTER'S DEFENSES:
 * 1. Bare `-p "-leading dash task"` is mis-parsed by clap as an unknown flag
 *    (exit 2, usage error) — DEFENSE: instruction is ALWAYS attached to the
 *    last argv element as `--single=<instruction>` (one argv element, no
 *    shell parsing; safe for arbitrary prompt text).
 * 2. npm dist-tag `latest` is stale at 0.1.4 while the real line is 1.0.x —
 *    DEFENSE: binPath is explicit-required (no PATH discovery) and the probe
 *    VERSION tier parses `--version` and fail-closed rejects any version
 *    other than the pinned one (config.pinnedVersion, default 1.0.87).
 * 3. Keyless / bad-key runs exit 1 with empty stdout and stderr
 *    `Not signed in. ...` — DEFENSE: probe CREDENTIALS tier checks the
 *    explicitly injected XAI_API_KEY FIRST and fails honestly
 *    (CREDENTIALS_ABSENT) without ever spawning the canary or the task; an
 *    auth failure can therefore never be retried or masked into a PASS.
 * 4. Auto-update would write a new binary into GROK_HOME/bin — DEFENSE:
 *    `--no-auto-update` is forced on every spawn AND each attempt/probe runs
 *    with a fresh temp GROK_HOME (isolation discipline of runtimes/grok
 *    isolation.go), so no state or binary ever lands in ~/.grok.
 * 5. `--cwd` is declared by --help but its effect was never observed in a
 *    real turn — DEFENSE: the flag is never sent; workspace pinning is done
 *    purely via the spawn cwd (disposable git worktree), matching the
 *    observed `inspect` behavior (workspaceRoot = process cwd).
 * 6. Cancel was never observed in-flight (#468 host had no credentials) —
 *    DEFENSE: cancellation is implemented only as process-tree termination
 *    (taskkill /T /F default on Windows), reached EXCLUSIVELY through the
 *    AbortSignal channel (input.signal or cancel(input) aborting the same
 *    internal controller); there is no second cancel path.
 * 7. Owner environment secrets (DEEPSEEK_API_KEY etc.) must never reach the
 *    agent or its tool subprocesses — DEFENSE: spawn env is a strict
 *    allowlist (OS basics + PATH + home-resolution keys only, isolation.go)
 *    plus explicit config.env injection; GROK_HOME is force-overridden so a
 *    hostile config.env value cannot re-open the owner home.
 * 8. Flags declared by --help but never observed in a real turn (-r/--resume,
 *    -s/--session-id, --json-schema, --max-turns, --output-format json) —
 *    DEFENSE: none of them is ever sent; stdout is treated as the plain
 *    final answer only, and session/model identity is never fabricated
 *    (no sessionId/usage fields in the agent claim).
 * 9. exit 0 with empty stdout means the CLI contract was not fulfilled
 *    (contract: exit 0 = completed AND stdout = final answer) — DEFENSE:
 *    fail-safe judged as LOCAL_ATTEMPT_FAILED (ATTEMPT_CONTRACT_UNFULFILLED),
 *    never guessed into a candidate.
 * 10. Agent stdout is a claim (INV-AUTH-01) — DEFENSE: the answer is only
 *    recorded (bounded, redacted) and used as a NEGATIVE contract signal;
 *    the only positive evidence is the independent git readback of the
 *    candidate ref / tree / write-set.
 *
 * HONESTY STATUS: this adapter is contract-verified against a fake grok
 * binary only. Host credentials are ABSENT (no XAI_API_KEY; ~/.grok/
 * config.toml contains only `[cli] installer`), so the CREDENTIALS probe
 * tier honestly fails, no real-model canary has been run, and the provider
 * ships DISABLED (resolveGrokProviderSelection returns codex unless grok is
 * explicitly requested, enabled, and probe-validated). No claim of a real
 * model run is made anywhere. */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {mkdir,mkdtemp,rm} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {tmpdir} from 'node:os';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const MAX_PROMPT_LENGTH=30000; // single argv element must stay inside CreateProcess limits
const GROK_HOME_ENV='GROK_HOME';
const safeRef=attemptId=>attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();

export const GROK_PROVIDER_CONSTANTS=Object.freeze({
  providerId:'grok',
  candidateRefPrefix:'refs/tally-grok-candidates/',
  canaryPrompt:'Reply with exactly: OK',
  pinnedVersion:'1.0.87',
  probeVersionTimeoutMs:30000,
  probeCanaryTimeoutMs:180000,
  defaultAttemptTimeoutMs:600000,
  // --permission-mode enum exactly as enumerated by grok 1.0.87 --help; the
  // runtime semantics of the values were never observed (no credentials in
  // #468), so the adapter only validates and passes them through.
  permissionModes:Object.freeze(['default','acceptEdits','auto','dontAsk',
    'bypassPermissions','plan']),
  // Strict spawn-env allowlist (runtimes/grok isolation.go): OS/process
  // basics + PATH + home resolution. Everything else in the owner
  // environment — including other providers' credentials — is structurally
  // excluded from the agent process.
  envAllowlist:Object.freeze(['PATH','OS','LANG','LC_ALL','NUMBER_OF_PROCESSORS',
    'PROCESSOR_ARCHITECTURE','PROCESSOR_IDENTIFIER','PROCESSOR_LEVEL',
    'PROCESSOR_REVISION','SYSTEMROOT','SYSTEMDRIVE','WINDIR','COMSPEC','PATHEXT',
    'TEMP','TMP','HOME','USERPROFILE','HOMEDRIVE','HOMEPATH','TMPDIR'])});

/** Parse `grok --version` output (observed shape: `grok 1.0.87
 * (929e6dd187b1)`): first non-empty line, whitespace-split, version field
 * must start with a digit so TUI banners are never misread as identity.
 * Returns null when unparseable (unobserved, never guessed). */
export function parseGrokVersionIdentity(stdout){
  if(typeof stdout!=='string')return null;
  for(const line of stdout.split(/\r?\n/)){
    const trimmed=line.trim();
    if(!trimmed)continue;
    const fields=trimmed.split(/\s+/);
    if(fields.length<2||!/^[0-9]/.test(fields[1]))return null;
    return {name:fields[0],version:fields[1],
      commit:fields.length>2?fields[2].replace(/^\(+|\)+$/g,''):null};
  }
  return null;
}

/** Spawn environment for the agent process: allowlist copy from the owner
 * environment, then explicit config.env injection, then GROK_HOME forced to
 * the per-attempt temp home supplied as config.grokHome (a config.env
 * GROK_HOME value can never win). config.env values are never logged. */
export function grokSpawnEnvironment(config,ownerEnv=process.env){
  const env={};
  for(const key of GROK_PROVIDER_CONSTANTS.envAllowlist){
    const value=ownerEnv[key];
    if(typeof value==='string'&&value!=='')env[key]=value;
  }
  for(const [key,value] of Object.entries(config.env??{}))
    if(typeof value==='string')env[key]=value;
  if(typeof config.grokHome!=='string'||!config.grokHome)
    throw safeError('GROK_HOME_REQUIRED');
  env[GROK_HOME_ENV]=config.grokHome;
  return env;
}

/** Single-turn task prompt: the disposable-worktree fence plus the write-set
 * list, mirroring the zcode provider wording. */
export function grokTaskPrompt({allowedWriteSet,taskPrompt}){
  const paths=(allowedWriteSet??[]).map(x=>`  - ${x}`).join('\n');
  return `Work only in the current disposable Git worktree. Modify or create exactly the allowed paths:\n${paths}\nDo not modify, create or delete any other path. Do not run git commit, git push or any GitHub or network operation. Stop after the file changes are complete.\n\nTask: ${taskPrompt}`;
}

/** Structured argv, runtimes/grok process.go buildSpec: [bin] --no-auto-update
 * [-m <model>] [--permission-mode <mode>] --single=<instruction>. config.binArgs
 * (default empty) is an adapter-level launcher prefix for test doubles of the
 * native binary; production grok keeps the #468 argv byte-for-byte. */
function grokArgv(config,{prompt}){
  const argv=[config.binPath,...(config.binArgs??[]),'--no-auto-update'];
  if(typeof config.model==='string'&&config.model.trim()!=='')
    argv.push('-m',config.model.trim());
  if(typeof config.permissionMode==='string'&&config.permissionMode!=='')
    argv.push('--permission-mode',config.permissionMode);
  argv.push('--single='+prompt);
  return argv;
}

function hostEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  const env={SystemRoot:root,SystemDrive:config.systemDrive??'C:',
    ComSpec:root+'\\System32\\cmd.exe',PATH:root+'\\System32;C:\\Program Files\\Git\\cmd',
    PATHEXT:'.COM;.EXE;.BAT;.CMD',TEMP:root+'\\Temp',TMP:root+'\\Temp',
    GIT_CONFIG_NOSYSTEM:'1',GIT_TERMINAL_PROMPT:'0'};
  if(typeof config.emptyGitConfig==='string')env.GIT_CONFIG_GLOBAL=config.emptyGitConfig;
  return env;
}

function spawnCapture(spawnFn,command,args,{cwd,env,timeoutMs=15000}={}){
  const child=spawnFn(command,args,{cwd,env,windowsHide:true,stdio:['pipe','pipe','pipe']});
  let stdout='',stderr='',settled=false;
  const append=(which,chunk)=>{
    const text=chunk.toString('utf8');
    if(which==='stdout')stdout=(stdout+text).slice(-65536);
    else stderr=(stderr+text).slice(-65536);
  };
  child.stdout?.on('data',chunk=>append('stdout',chunk));
  child.stderr?.on('data',chunk=>append('stderr',chunk));
  const startedAt=Date.now();
  const done=new Promise(resolveDone=>{
    let timer;
    const finish=value=>{if(!settled){settled=true;clearTimeout(timer);resolveDone(value);}};
    child.once('error',()=>finish({code:null,signal:null,spawnError:true,pid:child.pid??null,
      stdout,stderr,startedAt,terminatedAt:Date.now()}));
    child.once('close',(code,signal)=>finish({code,signal,spawnError:false,pid:child.pid??null,
      stdout,stderr,startedAt,terminatedAt:Date.now()}));
    child.stdin?.end();
    if(timeoutMs>0){
      timer=setTimeout(()=>{
        try{child.kill();}catch{}
        finish({code:null,signal:null,spawnError:false,probeTimeout:true,pid:child.pid??null,
          stdout,stderr,startedAt,terminatedAt:Date.now()});
      },timeoutMs);
      timer.unref?.();
    }
  });
  return {child,done};
}

async function short(spawnFn,command,args,options={}){
  const {done}=spawnCapture(spawnFn,command,args,options);
  const result=await done;
  if(result.code!==0)console.error('GROK_HOST_COMMAND_FAILED',command,args,
    JSON.stringify(result).slice(0,4000));
  return result;
}

/** Defense against secret leakage through failure tails (parity with the Go
 * adapter's logpipe.RedactFine): explicitly injected KEY/TOKEN/SECRET values
 * are replaced before any tail is recorded. Values are never logged. */
function redactSecrets(text,config){
  let out=text;
  for(const [key,value] of Object.entries(config.env??{})){
    if(typeof value==='string'&&value.length>=8&&
      /KEY|TOKEN|SECRET|PASSWORD/i.test(key))
      out=out.split(value).join('[REDACTED]');
  }
  return out;
}

async function newSessionHome(prefix='tally-grok-home-'){
  return mkdtemp(join(tmpdir(),prefix));
}

function createDefaultOperations(spawnFn){
  const git=(input,args,options={})=>short(spawnFn,
    input.providerConfig?.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe',
    args,{env:hostEnvironment(input.providerConfig??{}),...options});
  /** Tiered availability probe: VERSION (binary + pinned identity gate) →
   * CREDENTIALS (explicit XAI_API_KEY injection) → CANARY (one bounded real
   * prompt round-trip). Any tier failing yields available:false and the
   * caller must never spawn the task. No retry, no fallback. */
  async function probeProvider(input){
    console.error('GROK_PROBE_PROVIDER_START');
    const config=input?.providerConfig??{};
    if(typeof config.binPath!=='string'||!config.binPath||!existsSync(config.binPath))
      return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:null,outputMatched:false,
        detail:'MISSING_CLI'};
    const versionHome=await newSessionHome();
    try{
      const version=await spawnCapture(spawnFn,config.binPath,
        [...(config.binArgs??[]),'--version'],
        {env:grokSpawnEnvironment({...config,grokHome:versionHome}),
          timeoutMs:config.versionTimeoutMs??GROK_PROVIDER_CONSTANTS.probeVersionTimeoutMs})
        .done.catch(()=>({code:null,stdout:'',stderr:'',spawnError:true}));
      if(version.code!==0||version.spawnError)
        return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
          outputMatched:false,detail:version.spawnError?'SPAWN_ERROR':'VERSION_PROBE_FAILED'};
      const identity=parseGrokVersionIdentity(version.stdout);
      const pinned=typeof config.pinnedVersion==='string'&&config.pinnedVersion?
        config.pinnedVersion:GROK_PROVIDER_CONSTANTS.pinnedVersion;
      if(!identity)
        return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
          outputMatched:false,detail:'VERSION_UNPARSEABLE'};
      if(identity.version!==pinned)
        return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
          outputMatched:false,version:identity.version,detail:'VERSION_MISMATCH'};
      // Credentials only via the explicit injection boundary. The owner
      // process environment is deliberately NOT consulted here: inheriting it
      // would silently couple provider availability to whatever key happens
      // to sit in the operator shell (the stale-pointer trap the zcode
      // adapter documented). Absence is an honest unavailable, never a retry.
      const apiKey=config.env?.XAI_API_KEY;
      if(typeof apiKey!=='string'||!apiKey.trim())
        return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
          version:identity.version,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',
          exitCode:null,outputMatched:false,detail:'CREDENTIALS_ABSENT'};
      // Canary: one bounded real round-trip (default 180s budget). The
      // canary argv carries zero expectation overlay (no -m /
      // --permission-mode): it proves auth+provider reachability, not model
      // configuration. A stall is reported honestly — invalid credentials
      // stall silently in the real CLI and must never be retried into a
      // fake PASS.
      const canaryConfig={binPath:config.binPath,binArgs:config.binArgs};
      const canaryArgv=grokArgv(canaryConfig,{prompt:GROK_PROVIDER_CONSTANTS.canaryPrompt});
      const scratch=await newSessionHome('tally-grok-canary-');
      const canaryHome=await newSessionHome();
      try{
        const canary=await spawnCapture(spawnFn,canaryArgv[0],canaryArgv.slice(1),
          {cwd:scratch,env:grokSpawnEnvironment({...config,grokHome:canaryHome}),
            timeoutMs:config.canaryTimeoutMs??GROK_PROVIDER_CONSTANTS.probeCanaryTimeoutMs}).done;
        const answer=canary.stdout.trim();
        if(canary.code!==0||!/^OK\b/i.test(answer))
          return {available:false,provider:GROK_PROVIDER_CONSTANTS.providerId,
            version:identity.version,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',
            exitCode:canary.code,outputMatched:false,
            detail:canary.probeTimeout?'CREDENTIALS_OR_PROVIDER_STALL':'CANARY_UNSATISFIED'};
        return {available:true,provider:GROK_PROVIDER_CONSTANTS.providerId,
          version:identity.version,canary:{answer:answer.slice(0,200)}};
      }finally{
        await rm(canaryHome,{recursive:true,force:true}).catch(()=>{});
        await rm(scratch,{recursive:true,force:true}).catch(()=>{});
      }
    }finally{
      await rm(versionHome,{recursive:true,force:true}).catch(()=>{});
    }
  }
  /** Git-anchored durable state only: the candidate ref and worktree
   * occupancy ARE the attempt record (no second store by design). */
  async function attemptRecord(input){
    const config=input.providerConfig??{};
    const ref=GROK_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(input.attemptId);
    const commit=await git(input,['-C',config.repositoryPath,'rev-parse','--verify',
      '--quiet',ref],{timeoutMs:15000});
    const commitSha=commit.stdout.trim();
    if(commit.code===0&&SHA.test(commitSha)){
      const [tree,parent,diff]=await Promise.all([
        git(input,['-C',config.repositoryPath,'rev-parse',commitSha+'^{tree}']),
        git(input,['-C',config.repositoryPath,'rev-parse',commitSha+'^']),
        git(input,['-C',config.repositoryPath,'diff','--name-only',input.sourceSha,commitSha])]);
      if(tree.code!==0||parent.code!==0||diff.code!==0)return {kind:'STALE'};
      if(parent.stdout.trim()!==input.sourceSha)return {kind:'STALE'};
      return {kind:'CANDIDATE',candidateCommit:commitSha,candidateTree:tree.stdout.trim(),
        paths:diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean),ref};
    }
    if(existsSync(input.worktreePath))return {kind:'WORKTREE_OCCUPIED'};
    return null;
  }
  async function prepareWorktree(input){
    const config=input.providerConfig??{};
    console.error('GROK_PREPARE_WORKTREE_START',input.worktreePath);
    await mkdir(dirname(input.worktreePath),{recursive:true});
    const added=await git(input,['-C',config.repositoryPath,'worktree','add','--detach',
      input.worktreePath,input.sourceSha],{timeoutMs:60000});
    if(added.code!==0){
      const listed=await git(input,['-C',config.repositoryPath,'worktree','list','--porcelain']);
      const occupied=listed.code===0&&listed.stdout.includes(input.worktreePath);
      throw safeError(occupied?'CANDIDATE_WORKTREE_OVERLAP':'CANDIDATE_WORKTREE_UNAVAILABLE');
    }
    const head=await git(input,['-C',input.worktreePath,'rev-parse','HEAD']);
    if(head.code!==0||head.stdout.trim()!==input.sourceSha)throw safeError('CANDIDATE_BASE_MISMATCH');
    return {worktreePath:input.worktreePath,baseSha:head.stdout.trim()};
  }
  async function finalizeCandidate(input){
    const allowed=new Set(input.allowedWriteSet??[]);
    const run=(args,options={})=>git(input,['-C',input.worktreePath,...args],options);
    const [tracked,untracked]=await Promise.all([
      run(['diff','--name-only',input.sourceSha]),
      run(['ls-files','--others','--exclude-standard','--no-directory'])]);
    if(tracked.code!==0||untracked.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const paths=[...new Set([...tracked.stdout.split(/\r?\n/),...untracked.stdout.split(/\r?\n/)]
      .map(x=>x.trim()).filter(Boolean))];
    // Post-hoc write-set fence: every changed path must be inside the allowed
    // set and the diff must be non-empty. Violations are FAILED, never PASS.
    if(paths.length===0||paths.some(path=>!allowed.has(path)))
      return {writeSetValid:false,paths};
    const added=await run(['add','--',...paths]);
    if(added.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const tree=await run(['write-tree']);
    if(tree.code!==0||!SHA.test(tree.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const treeSha=tree.stdout.trim();
    const issueNumber=/\/issues\/([1-9][0-9]*)$/.exec(input.issueRef??'')?.[1];
    const message=`Tally local candidate ${input.attemptId}\n\nRefs: #${issueNumber??'64'}`;
    const commit=await run(['commit-tree',treeSha,'-p',input.sourceSha,'-m',message],{
      env:{...hostEnvironment(input.providerConfig??{}),
        GIT_AUTHOR_NAME:'Tally Host',GIT_AUTHOR_EMAIL:'tally-host@localhost',
        GIT_COMMITTER_NAME:'Tally Host',GIT_COMMITTER_EMAIL:'tally-host@localhost'}});
    if(commit.code!==0||!SHA.test(commit.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const commitSha=commit.stdout.trim();
    const ref=GROK_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(input.attemptId);
    const updated=await run(['update-ref',ref,commitSha]);
    if(updated.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const readback=await run(['rev-parse',commitSha+'^{tree}']);
    if(readback.code!==0||readback.stdout.trim()!==treeSha)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    return {writeSetValid:true,paths,candidateTree:treeSha,candidateCommit:commitSha,candidateRef:ref};
  }
  /** Independent git readback of one exact candidate: commit parent, tree and
   * changed paths are re-derived from the repository, never from
   * agent-asserted values. */
  async function verifyCandidateReadback(input,candidate){
    const config=input.providerConfig??{};
    const commit=candidate?.candidateCommit;
    if(!SHA.test(commit??''))return {verified:false};
    const run=(args,options={})=>git(input,['-C',config.repositoryPath,...args],options);
    const [tree,parent,diff]=await Promise.all([
      run(['rev-parse',commit+'^{tree}']),
      run(['rev-parse',commit+'^']),
      run(['diff','--name-only',input.sourceSha,commit])]);
    if(tree.code!==0||parent.code!==0||diff.code!==0)return {verified:false};
    const paths=diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean);
    const allowed=new Set(input.allowedWriteSet??[]);
    return {verified:true,candidateTree:tree.stdout.trim(),parent:parent.stdout.trim(),
      paths,writeSetValid:paths.length>0&&paths.every(path=>allowed.has(path))};
  }
  return {probeProvider,attemptRecord,prepareWorktree,finalizeCandidate,verifyCandidateReadback};
}

/** Unified TaskProvider for the grok CLI. Inputs are the unified attempt
 * input {providerId:'grok',taskId,attemptId,sourceSha,worktreePath,
 * allowedWriteSet,taskPrompt,issueRef?,hostAuthorization,providerConfig,
 * signal?,timeoutMs?}; outputs are the unified attempt receipt with
 * taskPass/releasePass constant false. hostAuthorization is recorded
 * verbatim (pass-through audit field, not interpreted here); admission is
 * the selection function's job. */
export function createGrokTaskProvider({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const result=await short(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    return result.code===0;
  });
  const inFlight=new Map();
  async function runAttempt(input){
    const config=input?.providerConfig??{};
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:input?.issueRef,provider:GROK_PROVIDER_CONSTANTS.providerId,
      providerId:GROK_PROVIDER_CONSTANTS.providerId,taskId:input?.taskId??null,
      sourceSha:input?.sourceSha,attemptId:input?.attemptId,
      hostAuthorization:input?.hostAuthorization??null,...flags};
    const notExecuted=(reason,extra={})=>({...base,state:'AGENT_NOT_EXECUTED',reason,
      agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
      candidateRef:null,candidatePaths:null,process:null,error:null,...extra});
    const cancelledReceipt=(extra={})=>({...base,state:'LOCAL_ATTEMPT_CANCELLED',
      reason:'ATTEMPT_CANCELLED',agentLaunch:false,replayFixture:false,candidateTree:null,
      candidateCommit:null,candidateRef:null,candidatePaths:null,process:null,error:null,...extra});
    // Cancellation arriving before any work is a normal cancelled outcome —
    // never disguised as a failure (runtimes/grok classifyRun priority).
    if(input?.signal?.aborted)return cancelledReceipt();
    if(input.providerId!==undefined&&input.providerId!==GROK_PROVIDER_CONSTANTS.providerId)
      return notExecuted('ATTEMPT_INPUT_INVALID',{detail:'PROVIDER_ID_MISMATCH'});
    if(!SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.attemptId??'')||
      typeof input.taskPrompt!=='string'||!input.taskPrompt||
      !Array.isArray(input.allowedWriteSet)||input.allowedWriteSet.length===0||
      !input.allowedWriteSet.every(path=>typeof path==='string'&&path.length>0)||
      typeof config.repositoryPath!=='string'||!config.repositoryPath)
      return notExecuted('ATTEMPT_INPUT_INVALID',{detail:'INPUT_FIELDS_INVALID'});
    if(config.permissionMode!==undefined&&config.permissionMode!==''&&
      !GROK_PROVIDER_CONSTANTS.permissionModes.includes(config.permissionMode))
      return notExecuted('ATTEMPT_INPUT_INVALID',{detail:'PERMISSION_MODE_INVALID'});
    // Cancellation plumbing is registered before any probe/worktree work so
    // cancel(input) and input.signal share ONE channel into this attempt.
    const controller=new AbortController();
    const propagate=()=>controller.abort(new Error('ATTEMPT_CANCELLED'));
    input.signal?.addEventListener('abort',propagate,{once:true});
    inFlight.set(input.attemptId,{controller});
    let cancelled=false,pid=null,timer=null,home=null;
    const onAbort=()=>{cancelled=true;if(pid!==null)terminate(pid).catch(()=>{});};
    controller.signal.addEventListener('abort',onAbort);
    try{
      if(controller.signal.aborted)return cancelledReceipt();
      // Durable replay first: a completed attempt (candidate ref present) is
      // read back from git and NEVER relaunched; an occupied worktree without
      // a ref may belong to a running or crashed attempt and stays UNKNOWN.
      const record=await ops.attemptRecord(input);
      if(record?.kind==='CANDIDATE')return {...base,state:'LOCAL_CANDIDATE_READY',
        reason:'REAL_GROK_CANDIDATE_RECORDED',agentLaunch:true,replayed:true,
        candidateTree:record.candidateTree,candidateCommit:record.candidateCommit,
        candidateRef:record.ref,candidatePaths:record.paths,process:null,error:null};
      if(record?.kind==='STALE')return {...base,state:'NOT_ADMITTED',reason:'CANDIDATE_STALE',
        claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
        candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
        process:null,error:null};
      if(record?.kind==='WORKTREE_OCCUPIED')return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',
        reason:'ATTEMPT_STATUS_UNKNOWN',agentLaunch:false,replayFixture:false,
        candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
        process:null,error:null};
      const probe=await ops.probeProvider(input);
      if(!probe.available)return notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
      const prompt=grokTaskPrompt(input);
      if(prompt.length>MAX_PROMPT_LENGTH)
        return notExecuted('ATTEMPT_INPUT_INVALID',{detail:'PROMPT_TOO_LONG_FOR_ARGV'});
      const worktree=await ops.prepareWorktree(input).catch(error=>error.code);
      if(typeof worktree==='string')return {...base,state:'NOT_ADMITTED',reason:worktree,
        claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
        candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
        process:null,error:null,providerProbe:probe};
      if(controller.signal.aborted)return cancelledReceipt();
      home=await newSessionHome();
      // Fresh per-attempt GROK_HOME (defense #4/#7): no auth hand-over, no
      // ~/.grok state, auto-update output would land in a directory deleted
      // below. config.env cannot override it (forced key).
      const env=grokSpawnEnvironment({...config,grokHome:home});
      const argv=grokArgv(config,{prompt});
      const running=spawnCapture(spawnFn,argv[0],argv.slice(1),
        {cwd:worktree.worktreePath,env,timeoutMs:0});
      pid=Number.isSafeInteger(running.child.pid)?running.child.pid:null;
      const timeoutMs=input.timeoutMs??GROK_PROVIDER_CONSTANTS.defaultAttemptTimeoutMs;
      let timedOut=false;
      timer=setTimeout(()=>{timedOut=true;terminate(running.child.pid).catch(()=>{});},timeoutMs);
      timer.unref?.();
      const terminal=await running.done;
      clearTimeout(timer);timer=null;
      if((timedOut||cancelled)&&terminal.terminatedAt)await terminate(running.child.pid);
      // The worktree is left in place: occupancy is the physical single-writer
      // lock for crash reconciliation. The candidate ref is the durable record.
      const candidate=await ops.finalizeCandidate(input).catch(error=>({error:error.code}));
      const answer=terminal.stdout.trim();
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
        startedAt:new Date(terminal.startedAt).toISOString(),terminatedAt:iso(now)};
      let state,reason;
      if(timedOut||terminal.spawnError||terminal.probeTimeout){
        state='LOCAL_ATTEMPT_UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
      }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';
      }else if(terminal.code===0&&answer===''){
        // Contract unfulfilled (exit 0 must carry the final answer): fail-safe,
        // never guessed into a candidate (defense #9).
        state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_CONTRACT_UNFULFILLED';
      }else if(!candidate.writeSetValid){
        state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';
      }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';
      }else{state='LOCAL_CANDIDATE_READY';reason='REAL_GROK_CANDIDATE_RECORDED';}
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,
        candidateRef:candidate.candidateRef??null,candidatePaths:candidate.paths??null,process,
        error:candidate.error??null,
        agent:{answer:answer?answer.slice(0,2000):null},
        stdoutTail:redactSecrets(terminal.stdout.slice(-4000),config),
        stderrTail:redactSecrets(terminal.stderr.slice(-2000),config),
        providerProbe:probe};
    }finally{
      if(timer!==null)clearTimeout(timer);
      controller.signal.removeEventListener('abort',onAbort);
      input.signal?.removeEventListener('abort',propagate);
      inFlight.delete(input.attemptId);
      if(home!==null)await rm(home,{recursive:true,force:true}).catch(()=>{});
    }
  }
  /** Cancellation request for one in-flight attempt. The AbortSignal is the
   * ONLY termination channel: cancel aborts the attempt's controller and the
   * execute path performs the (single) process-tree termination. Cancel
   * never touches processes directly and never launches anything. */
  function cancel(input){
    const attemptId=input?.attemptId;
    if(!ATTEMPT.test(attemptId??''))
      return {cancelled:false,attemptId:attemptId??null,reason:'ATTEMPT_INPUT_INVALID'};
    const entry=inFlight.get(attemptId);
    if(!entry)return {cancelled:false,attemptId,reason:'ATTEMPT_NOT_IN_FLIGHT'};
    entry.controller.abort(new Error('ATTEMPT_CANCELLED'));
    return {cancelled:true,attemptId,channel:'signal'};
  }
  return Object.freeze({providerId:GROK_PROVIDER_CONSTANTS.providerId,
    probe:input=>ops.probeProvider(input),execute:runAttempt,cancel,
    readback:(input,candidate)=>ops.verifyCandidateReadback(input,candidate)});
}

/** Selection policy. The default provider is codex; grok ships DISABLED and
 * requires an explicit request AND an enabled config row with an explicit
 * binPath AND a locally validated capability probe. There is deliberately NO
 * failure-driven fallback: a codex failure never switches provider, and an
 * unvalidated grok config never silently selects grok. */
export function resolveGrokProviderSelection({requested,grok,probe}){
  if(requested!=='grok')
    return {selected:'codex',grokAdmitted:false,
      reason:requested==='codex'?'CODEX_EXPLICIT_DEFAULT':'CODEX_DEFAULT',probe:null};
  if(!grok||grok.enabled!==true||typeof grok.binPath!=='string'||!grok.binPath)
    return {selected:'codex',grokAdmitted:false,reason:'GROK_NOT_CONFIGURED',probe:null};
  if(!probe||probe.available!==true)
    return {selected:'codex',grokAdmitted:false,reason:'GROK_CAPABILITY_UNVERIFIED',
      probe:probe??null};
  return {selected:'grok',grokAdmitted:true,reason:'GROK_VALIDATED',probe};
}
