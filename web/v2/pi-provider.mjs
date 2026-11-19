/** Pi Agent Provider adapter (#64, v2.1 candidate; NOT a v2 MVP requirement).
 *
 * Unified TaskProvider for the pi coding agent CLI, ported from the Go worker
 * runtime contract (kaicreator-mm/tally#469, runtimes/pi) into a thin Node
 * adapter shaped like the zcode provider (#470): same injectable spawn shape,
 * same state/reason vocabulary, same git-derived candidate evidence. The Go
 * code is a CONTRACT REFERENCE ONLY — v2 never runs through the old worker.
 * Durability is git-anchored only (candidate ref + worktree occupancy); there
 * is no second scheduler, kernel, permission system or event log.
 *
 * LIVE CLI CONTRACT EMPIRICS (pi 0.85.1 @ C:\Application\pi\pi, Windows,
 * 2026-11-19; every argv flag cross-checked against `pi --help`; nothing
 * invented beyond what the Go contract observed and this session re-ran):
 *   - `pi --version` → bare version line "0.85.1", exit 0 (observed).
 *   - Non-interactive one-shot form (help text + live run):
 *       pi -p --mode json --no-session --session-dir <dir>
 *          --no-extensions --no-skills --no-prompt-templates --no-themes
 *          --no-context-files --no-approve [--provider <p>] [--model <m>]
 *          [--thinking <t>] -- <instruction>
 *     workspace = process cwd (pi has no cwd flag; its file tools write in
 *     cwd), so the spawn cwd IS the disposable worktree. `--api-key` exists
 *     but is never used: credentials never ride argv.
 *   - exit 0 = the process claims success; exit != 0 = not completed
 *     (keyless provider: exit 1 + stderr "No API key found for <provider>.").
 *   - --mode json stdout is an NDJSON event stream: first line
 *     {"type":"session","version":3,"id":"<uuid>","cwd":...} (emitted even
 *     under --no-session), then agent_start/turn_start/message_start|
 *     message_end/message_update/turn_end/agent_end/agent_settled. The
 *     assistant message_end carries content[] (may hold "thinking" blocks
 *     before the text block; only type:"text" blocks count), provider, model,
 *     usage{input,output,...} and stopReason. Live canary: assistant
 *     message_end text exactly "OK", provider "deepseek", model
 *     "deepseek-v4-pro" (observed actual model, never backfilled from the
 *     requested flags). Truncated tail lines (cancel/crash mid-stream) fail
 *     JSON.parse and are skipped; stream completeness is judged by whether a
 *     text-bearing assistant message_end was observed, never by "exit 0".
 *   - Auth model: pi reads per-provider env vars (DEEPSEEK_API_KEY et al.,
 *     all listed in pi --help). Read-only readiness check observed live:
 *     `pi auth check --provider deepseek --json --no-refresh` →
 *     {"status":"ready","provider":"deepseek","authType":"api_key"}, exit 0,
 *     no credential material printed. The ONLY credential channel in this
 *     adapter is providerConfig.env (never argv, never inherited).
 *   - Isolation keys forced on every spawn (isolation semantics from the Go
 *     contract, re-observed live): PI_CODING_AGENT_DIR (config dir redirect —
 *     pi scaffolds auth.json/models-store.json as "{}" into it, owner ~/.pi
 *     untouched), PI_CODING_AGENT_SESSION_DIR + --session-dir (session
 *     storage; --no-session keeps it empty, the dir is defense in depth),
 *     HOME/USERPROFILE (~ redirect), PI_OFFLINE=1 (disables startup network
 *     ops; live canary proves real model turns still work), PI_TELEMETRY=0.
 *     Forced keys override same-named providerConfig.env values.
 *   - Cancellation = process-tree termination (Go observed taskkill /T /F
 *     cleanly terminates the pi tree on Windows). The AbortSignal is the only
 *     cancellation channel: execute() terminates on input.signal abort, and
 *     cancel(input) aborts the attempt's internal AbortController — it never
 *     kills a process directly.
 *
 * Evidence rules (inherited from the v2 provider contract): the agent's own
 * output is a claim, never evidence (INV-AUTH-01) — candidates are derived
 * exclusively from git readback with explicit `-C`; taskPass/releasePass are
 * always false; there is no failure-driven fallback to another provider; pi
 * is DISABLED by default and selectable only through resolvePiProviderSelection
 * with an explicit request, an enabled config row and a passed capability
 * probe. Candidate refs live under refs/tally-pi-candidates/. */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {mkdir,mkdtemp,rm} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {tmpdir} from 'node:os';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const OUTPUT_MODES=new Set(['json','text']);
const THINKING_LEVELS=new Set(['off','minimal','low','medium','high','xhigh','max']);
const MAX_INSTRUCTION_LENGTH=30000; // Windows command-line ceiling is 32767; keep margin.
const safeRef=attemptId=>attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();

export const PI_PROVIDER_CONSTANTS=Object.freeze({providerId:'pi',
  candidateRefPrefix:'refs/tally-pi-candidates/',
  canaryPrompt:'Reply with exactly: OK',
  pinnedVersion:'0.85.1',
  defaultOutputMode:'json',
  defaultCanaryTimeoutMs:180000,
  defaultTimeoutMs:600000,
  shaPattern:SHA,attemptIdPattern:ATTEMPT});

/** A provider/model/thinking token may ride argv bare: only alphanumerics and
 * . _ / - : + (the Go adapter's observed "deepseek-v4-flash"-class token set).
 * Anything else is rejected, never escaped. */
export function piConfigToken(value){
  return typeof value==='string'&&value!==''&&
    /^[A-Za-z0-9._/:+-]+$/.test(value);
}

/** Builder environment is an explicit allowlist — nothing is inherited. pi
 * credentials live in per-provider env vars, so every credential variable
 * (DEEPSEEK_API_KEY, ANTHROPIC_API_KEY, ...) is absent by default; the only
 * credential channel is providerConfig.env layered by piSpawnEnvironment. No
 * GitHub/CI credentials either — the builder never pushes; the Host finalizes
 * the candidate ref itself. Forced isolation keys (PI_CODING_AGENT_DIR,
 * PI_CODING_AGENT_SESSION_DIR, HOME, USERPROFILE, PI_OFFLINE=1,
 * PI_TELEMETRY=0) are applied last and override same-named config.env values
 * (case-sensitive here; Node spawn passes them verbatim). */
export function piBuilderEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  const home=config.builderProfile??process.env.USERPROFILE??'C:\\Users\\15549';
  const path=[root+'\\System32','C:\\Program Files\\Git\\cmd'];
  if(typeof config.cliPath==='string'&&config.cliPath)path.unshift(dirname(config.cliPath));
  const env={SystemRoot:root,SystemDrive:config.systemDrive??'C:',
    ComSpec:root+'\\System32\\cmd.exe',PATH:path.join(';'),
    PATHEXT:'.COM;.EXE;.BAT;.CMD',
    USERPROFILE:home,HOME:home,
    APPDATA:home+'\\AppData\\Roaming',LOCALAPPDATA:home+'\\AppData\\Local',
    TEMP:config.builderTemp??home+'\\AppData\\Local\\Temp',
    TMP:config.builderTemp??home+'\\AppData\\Local\\Temp',
    GIT_CONFIG_NOSYSTEM:'1',GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never',
    NO_COLOR:'1'};
  if(config.modelProxy){env.HTTPS_PROXY=config.modelProxy;env.HTTP_PROXY=config.modelProxy;}
  if(config.caBundle){env.SSL_CERT_FILE=config.caBundle;env.CURL_CA_BUNDLE=config.caBundle;}
  if(typeof config.emptyGitConfig==='string')env.GIT_CONFIG_GLOBAL=config.emptyGitConfig;
  return env;
}

/** Per-attempt isolation directory group (Go sessionDirs semantics): a fresh
 * temp root with agent-config/sessions/home children, removed after the
 * attempt's process settles. If the process dies hard the OS temp dir
 * cleanup is the backstop — by design there is no durable state here. */
async function newIsolationDirs(){
  const root=await mkdtemp(join(tmpdir(),'tally-pi-attempt-'));
  const dirs={root,agentConfig:join(root,'agent-config'),
    sessions:join(root,'sessions'),home:join(root,'home')};
  await mkdir(dirs.agentConfig,{recursive:true});
  await mkdir(dirs.sessions,{recursive:true});
  await mkdir(dirs.home,{recursive:true});
  return dirs;
}
const closeIsolationDirs=dirs=>rm(dirs.root,{recursive:true,force:true}).catch(()=>{});

function forcedIsolationEnv(dirs){
  return {HOME:dirs.home,USERPROFILE:dirs.home,
    PI_CODING_AGENT_DIR:dirs.agentConfig,
    PI_CODING_AGENT_SESSION_DIR:dirs.sessions,
    PI_OFFLINE:'1',PI_TELEMETRY:'0'};
}

/** Spawn env for the pi process: explicit builder allowlist + providerConfig.env
 * (the ONLY credential channel) + forced isolation keys last. */
export function piSpawnEnvironment(config,dirs){
  return {...piBuilderEnvironment(config),...(config.env??{}),
    ...forcedIsolationEnv(dirs)};
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
  if(result.code!==0)console.error('PI_HOST_COMMAND_FAILED',command,args,
    JSON.stringify(result).slice(0,4000));
  return result;
}

/** Parse the pi --mode json NDJSON stream (Go parseJSONStream semantics):
 * one JSON object per line, truncated tail lines skipped. Captures the
 * session line's UUID handle and the LAST text-bearing assistant message_end
 * (content "thinking" blocks are ignored; only type:"text" blocks join the
 * response). Unparseable output yields hasAssistant:false — a null here never
 * fabricates success, because the git-derived candidate is the only evidence. */
export function parsePiJsonStream(stdout){
  const observation={sessionHandle:null,model:null,provider:null,response:null,
    usage:null,hasAssistant:false};
  if(typeof stdout!=='string')return observation;
  for(const line of stdout.split(/\r?\n/)){
    if(!line.trim())continue;
    let event;
    try{event=JSON.parse(line);}catch{continue;}
    if(!event||typeof event!=='object')continue;
    if(event.type==='session'){
      if(typeof event.id==='string'&&event.id)observation.sessionHandle=event.id;
    }else if(event.type==='message_end'&&event.message?.role==='assistant'){
      const content=Array.isArray(event.message.content)?event.message.content:[];
      const text=content.filter(block=>block?.type==='text'&&typeof block.text==='string'&&block.text)
        .map(block=>block.text).join('\n');
      if(!text)continue;
      observation.hasAssistant=true;
      observation.response=text;
      observation.model=typeof event.message.model==='string'&&event.message.model?
        event.message.model:observation.model;
      observation.provider=typeof event.message.provider==='string'&&event.message.provider?
        event.message.provider:observation.provider;
      const usage=event.message.usage;
      if(usage&&Number.isSafeInteger(usage.input)&&Number.isSafeInteger(usage.output))
        observation.usage={inputTokens:usage.input,outputTokens:usage.output};
    }
  }
  return observation;
}

/** The exact non-interactive argv contract (see module header). The
 * instruction is always the single positional argument after "--"; the
 * workspace rides the spawn cwd, never a flag. */
function piArgv(config,{instruction,sessionDir}){
  const argv=[config.cliPath,'-p','--mode',
    config.outputMode??PI_PROVIDER_CONSTANTS.defaultOutputMode,
    '--no-session','--session-dir',sessionDir,
    '--no-extensions','--no-skills','--no-prompt-templates','--no-themes',
    '--no-context-files','--no-approve'];
  if(config.provider)argv.push('--provider',config.provider);
  if(config.model)argv.push('--model',config.model);
  if(config.thinkingLevel)argv.push('--thinking',config.thinkingLevel);
  argv.push('--',instruction);
  return argv;
}

export function piTaskPrompt({allowedWriteSet,taskPrompt}){
  const paths=(allowedWriteSet??[]).map(x=>`  - ${x}`).join('\n');
  return `Work only in the current disposable Git worktree. Modify or create exactly the allowed paths:\n${paths}\nDo not modify, create or delete any other path. Do not run git commit, git push or any GitHub or network operation beyond the model provider API. Stop after the file changes are complete.\n\nTask: ${taskPrompt}`;
}

function normalizeConfig(config){
  // Accept either the unified execute input (providerConfig nested) or an
  // already-flat provider config; contract fields always win from the input.
  const input=config??{};
  return input.providerConfig&&typeof input.providerConfig==='object'?
    {...input.providerConfig,...input}:input;
}

function validateAttemptInput(input){
  if(!input||typeof input!=='object')return false;
  if(input.providerId!==undefined&&input.providerId!==PI_PROVIDER_CONSTANTS.providerId)return false;
  if(!ATTEMPT.test(input.attemptId??''))return false;
  if(!SHA.test(input.sourceSha??''))return false;
  if(typeof input.taskPrompt!=='string'||!input.taskPrompt)return false;
  if(!Array.isArray(input.allowedWriteSet)||
    input.allowedWriteSet.some(path=>typeof path!=='string'||!path))return false;
  if(typeof input.worktreePath!=='string'||!input.worktreePath)return false;
  if(input.taskId!==undefined&&(typeof input.taskId!=='string'||!input.taskId))return false;
  if(input.signal!==undefined&&typeof input.signal?.addEventListener!=='function')return false;
  const config=input.providerConfig;
  if(!config||typeof config!=='object')return false;
  if(typeof config.cliPath!=='string'||!config.cliPath)return false;
  if(typeof config.repositoryPath!=='string'||!config.repositoryPath)return false;
  if(config.outputMode!==undefined&&!OUTPUT_MODES.has(config.outputMode))return false;
  if(config.thinkingLevel!==undefined&&!THINKING_LEVELS.has(config.thinkingLevel))return false;
  for(const token of [config.provider,config.model])
    if(token!==undefined&&!piConfigToken(token))return false;
  if(config.env!==undefined&&(config.env===null||typeof config.env!=='object'))return false;
  return true;
}

function createDefaultOperations(spawnFn){
  const gitPath=config=>config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
  const git=(config,args,options={})=>short(spawnFn,gitPath(config),
    args,{env:hostEnvironment(config),...options});
  async function probeProvider(config){
    console.error('PI_PROBE_PROVIDER_START');
    if(typeof config.cliPath!=='string'||!config.cliPath||!existsSync(config.cliPath))
      return {available:false,provider:PI_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:null,outputMatched:false,
        detail:'MISSING_CLI'};
    // Tier VERSION: bare version line, exact pin (Go discipline: no version
    // anchor, no contract — pi iterates fast and the argv shape is pinned to
    // the observed release).
    const version=await spawnCapture(spawnFn,config.cliPath,['--version'],
      {env:piBuilderEnvironment(config),timeoutMs:30000}).done
      .catch(()=>({code:null,stdout:'',stderr:'',spawnError:true}));
    const versionText=(version.stdout.trim().split(/\r?\n/).find(line=>line.trim())??'').trim();
    const pinned=config.pinnedVersion??PI_PROVIDER_CONSTANTS.pinnedVersion;
    if(version.code!==0||versionText!==pinned)
      return {available:false,provider:PI_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
        outputMatched:false,detail:version.spawnError?'SPAWN_ERROR':'VERSION_MISMATCH',
        observedVersion:versionText||null};
    // Tier CREDENTIALS: pi's own read-only readiness check per provider.
    // No --credentials flag → no credential material is printed; --no-refresh
    // prevents OAuth refresh side effects. The provider must be configured:
    // readiness is defined per provider, so an unconfigured one is an honest
    // probe failure, not a pass-by-default.
    if(!piConfigToken(config.provider))
      return {available:false,provider:PI_PROVIDER_CONSTANTS.providerId,version:pinned,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',exitCode:null,
        outputMatched:false,detail:'PROVIDER_UNCONFIGURED'};
    const dirs=await newIsolationDirs();
    try{
      const spawnEnv=piSpawnEnvironment(config,dirs);
      const auth=await spawnCapture(spawnFn,config.cliPath,
        ['auth','check','--provider',config.provider,'--json','--no-refresh'],
        {env:spawnEnv,timeoutMs:30000}).done
        .catch(()=>({code:null,stdout:'',stderr:'',spawnError:true}));
      let ready=false;
      try{ready=JSON.parse(auth.stdout.trim()).status==='ready';}catch{}
      if(auth.code!==0||!ready)
        return {available:false,provider:PI_PROVIDER_CONSTANTS.providerId,version:pinned,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',exitCode:auth.code,
          outputMatched:false,detail:auth.spawnError?'SPAWN_ERROR':'CREDENTIALS_ABSENT'};
      // Tier CANARY: one bounded real turn. Auth failures stall silently in
      // real CLIs; the canary is the only definitive auth+provider check and
      // a bounded stall is reported honestly instead of retried into a fake
      // PASS. The canary cwd is a disposable dir under the isolation root.
      const canaryCwd=join(dirs.root,'canary-cwd');
      await mkdir(canaryCwd,{recursive:true});
      const canaryArgv=piArgv({...config,outputMode:PI_PROVIDER_CONSTANTS.defaultOutputMode},
        {instruction:PI_PROVIDER_CONSTANTS.canaryPrompt,sessionDir:dirs.sessions});
      const canary=await spawnCapture(spawnFn,canaryArgv[0],canaryArgv.slice(1),
        {cwd:canaryCwd,env:spawnEnv,
          timeoutMs:config.canaryTimeoutMs??PI_PROVIDER_CONSTANTS.defaultCanaryTimeoutMs}).done
        .catch(()=>({code:null,stdout:'',stderr:'',spawnError:true,probeTimeout:true}));
      const canaryObservation=parsePiJsonStream(canary.stdout);
      if(canary.code!==0||!canaryObservation.hasAssistant||
        !/OK/i.test(canaryObservation.response??''))
        return {available:false,provider:PI_PROVIDER_CONSTANTS.providerId,version:pinned,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',exitCode:canary.code,
          outputMatched:false,
          detail:canary.probeTimeout?'CREDENTIALS_OR_PROVIDER_STALL':'CANARY_UNSATISFIED'};
      return {available:true,provider:PI_PROVIDER_CONSTANTS.providerId,version:pinned,
        canary:{sessionHandle:canaryObservation.sessionHandle,
          response:canaryObservation.response,model:canaryObservation.model,
          provider:canaryObservation.provider}};
    }finally{await closeIsolationDirs(dirs);}
  }
  async function attemptRecord(config){
    // Git-anchored durable state only: the candidate ref and worktree
    // occupancy ARE the attempt record (no second store by design).
    const ref=PI_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
    const commit=await git(config,['-C',config.repositoryPath,'rev-parse','--verify',
      '--quiet',ref],{timeoutMs:15000});
    const commitSha=commit.stdout.trim();
    if(commit.code===0&&SHA.test(commitSha)){
      const [tree,parent,diff]=await Promise.all([
        git(config,['-C',config.repositoryPath,'rev-parse',commitSha+'^{tree}']),
        git(config,['-C',config.repositoryPath,'rev-parse',commitSha+'^']),
        git(config,['-C',config.repositoryPath,'diff','--name-only',config.sourceSha,commitSha])]);
      if(tree.code!==0||parent.code!==0||diff.code!==0)return {kind:'STALE'};
      if(parent.stdout.trim()!==config.sourceSha)return {kind:'STALE'};
      return {kind:'CANDIDATE',candidateCommit:commitSha,candidateTree:tree.stdout.trim(),
        paths:diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean),ref};
    }
    if(existsSync(config.worktreePath))return {kind:'WORKTREE_OCCUPIED'};
    return null;
  }
  async function prepareWorktree(config){
    console.error('PI_PREPARE_WORKTREE_START',config.worktreePath);
    await mkdir(dirname(config.worktreePath),{recursive:true});
    const added=await git(config,['-C',config.repositoryPath,'worktree','add','--detach',
      config.worktreePath,config.sourceSha],{timeoutMs:60000});
    if(added.code!==0){
      const listed=await git(config,['-C',config.repositoryPath,'worktree','list','--porcelain']);
      const occupied=listed.code===0&&listed.stdout.includes(config.worktreePath);
      throw safeError(occupied?'CANDIDATE_WORKTREE_OVERLAP':'CANDIDATE_WORKTREE_UNAVAILABLE');
    }
    const head=await git(config,['-C',config.worktreePath,'rev-parse','HEAD']);
    if(head.code!==0||head.stdout.trim()!==config.sourceSha)throw safeError('CANDIDATE_BASE_MISMATCH');
    return {worktreePath:config.worktreePath,baseSha:head.stdout.trim()};
  }
  async function finalizeCandidate(config){
    const allowed=new Set(config.allowedWriteSet??[]);
    const run=(args,options={})=>git(config,['-C',config.worktreePath,...args],options);
    const [tracked,untracked]=await Promise.all([
      run(['diff','--name-only',config.sourceSha]),
      run(['ls-files','--others','--exclude-standard','--no-directory'])]);
    if(tracked.code!==0||untracked.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const paths=[...new Set([...tracked.stdout.split(/\r?\n/),...untracked.stdout.split(/\r?\n/)]
      .map(x=>x.trim()).filter(Boolean))];
    // Post-hoc write-set fence: every changed path must be inside the allowed
    // set and the diff must be non-empty. Violations are FAILED, never PASS.
    if(paths.length===0||paths.some(path=>!allowed.has(path)))
      return {writeSetValid:false,paths,
        detail:paths.length===0?'NO_CHANGES_RECORDED':'PATHS_OUTSIDE_ALLOWED_SET'};
    const added=await run(['add','--',...paths]);
    if(added.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const tree=await run(['write-tree']);
    if(tree.code!==0||!SHA.test(tree.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const treeSha=tree.stdout.trim();
    const issueNumber=/\/issues\/([1-9][0-9]*)$/.exec(config.issueRef??'')?.[1];
    const message=`Tally local candidate ${config.attemptId}\n\nRefs: #${issueNumber??'64'}`;
    const commit=await run(['commit-tree',treeSha,'-p',config.sourceSha,'-m',message],{
      env:{...hostEnvironment(config),GIT_AUTHOR_NAME:'Tally Host',GIT_AUTHOR_EMAIL:'tally-host@localhost',
        GIT_COMMITTER_NAME:'Tally Host',GIT_COMMITTER_EMAIL:'tally-host@localhost'}});
    if(commit.code!==0||!SHA.test(commit.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const commitSha=commit.stdout.trim();
    const ref=PI_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
    const updated=await run(['update-ref',ref,commitSha]);
    if(updated.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const readback=await run(['rev-parse',commitSha+'^{tree}']);
    if(readback.code!==0||readback.stdout.trim()!==treeSha)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    return {writeSetValid:true,paths,candidateTree:treeSha,candidateCommit:commitSha,candidateRef:ref};
  }
  async function verifyCandidateReadback(config,candidate){
    // Independent git readback of one exact candidate: commit parent, tree
    // and changed paths are re-derived from the repository, never from
    // agent-asserted values.
    const commit=candidate.candidateCommit;
    if(!SHA.test(commit??''))return {verified:false};
    const [tree,parent,diff]=await Promise.all([
      git(config,['-C',config.repositoryPath,'rev-parse',commit+'^{tree}']),
      git(config,['-C',config.repositoryPath,'rev-parse',commit+'^']),
      git(config,['-C',config.repositoryPath,'diff','--name-only',config.sourceSha,commit])]);
    if(tree.code!==0||parent.code!==0||diff.code!==0)return {verified:false};
    return {verified:true,candidateTree:tree.stdout.trim(),parent:parent.stdout.trim(),
      paths:diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean)};
  }
  return {probeProvider,attemptRecord,prepareWorktree,finalizeCandidate,verifyCandidateReadback};
}

/** Unified TaskProvider factory. Injectable for tests: spawnFn, operations,
 * now, killTree. The frozen object exposes providerId, probe (VERSION →
 * CREDENTIALS → CANARY tiers; any failure ⇒ available:false and execute never
 * spawns), execute (one-shot worktree + spawn + finalize + state mapping),
 * cancel (signal-only channel) and readback (independent git readback). */
export function createPiTaskProvider({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const result=await short(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    return result.code===0;
  });
  const activeAttempts=new Map(); // attemptId -> AbortController
  const nulls={candidateTree:null,candidateCommit:null,candidateRef:null,
    candidatePaths:null,process:null};
  const notAdmitted=(reason,detail={})=>({state:'NOT_ADMITTED',reason,
    claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
    ...nulls,...detail});
  async function execute(input){
    const config=normalizeConfig(input);
    const base={issueRef:input?.issueRef,
      taskId:typeof input?.taskId==='string'?input.taskId:undefined,
      provider:PI_PROVIDER_CONSTANTS.providerId,
      providerId:PI_PROVIDER_CONSTANTS.providerId,
      sourceSha:input?.sourceSha,attemptId:input?.attemptId,
      claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const notExecuted=(reason,detail={})=>({...base,state:'AGENT_NOT_EXECUTED',reason,
      agentLaunch:false,replayFixture:false,...nulls,...detail});
    if(!validateAttemptInput(input))return notExecuted('ATTEMPT_INPUT_INVALID');
    // Admission gate: an agent process is launched only under an explicit
    // host authorization record. Without it nothing is probed, prepared or
    // spawned — NOT_ADMITTED precedes every observation.
    if(!input.hostAuthorization)return {...base,...notAdmitted('HOST_AUTHORIZATION_REQUIRED')};
    // Durable replay first: a completed attempt (candidate ref present) is
    // read back from git and NEVER relaunched; an occupied worktree without a
    // ref may belong to a running or crashed attempt and stays UNKNOWN.
    const record=await ops.attemptRecord(config);
    if(record?.kind==='CANDIDATE')return {...base,state:'LOCAL_CANDIDATE_READY',
      reason:'REAL_PI_CANDIDATE_RECORDED',agentLaunch:true,replayed:true,
      replayFixture:false,candidateTree:record.candidateTree,
      candidateCommit:record.candidateCommit,candidateRef:record.ref,
      candidatePaths:record.paths,process:null};
    if(record?.kind==='STALE')return {...base,...notAdmitted('CANDIDATE_STALE')};
    if(record?.kind==='WORKTREE_OCCUPIED')return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',
      reason:'ATTEMPT_STATUS_UNKNOWN',agentLaunch:false,replayFixture:false,...nulls};
    const probe=await ops.probeProvider(config);
    if(!probe.available)return notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
    const worktree=await ops.prepareWorktree(config).catch(error=>error.code);
    if(typeof worktree==='string')
      return {...base,...notAdmitted(worktree,{providerProbe:probe})};
    const dirs=await newIsolationDirs();
    try{
      const env=piSpawnEnvironment(config,dirs);
      const instruction=piTaskPrompt(input);
      if(instruction.length>MAX_INSTRUCTION_LENGTH)
        return notExecuted('ATTEMPT_INPUT_INVALID',{detail:'PROMPT_TOO_LONG_FOR_ARGV'});
      const argv=piArgv(config,{instruction,sessionDir:dirs.sessions});
      // A signal already aborted during setup is a cancellation (Go: ctx done
      // before start) — never launch. This also closes the registration race:
      // a listener added to an already-aborted signal would never fire.
      if(input.signal?.aborted)
        return {...base,state:'LOCAL_ATTEMPT_CANCELLED',reason:'ATTEMPT_CANCELLED',
          agentLaunch:false,replayFixture:false,...nulls};
      const running=spawnCapture(spawnFn,argv[0],argv.slice(1),
        {cwd:config.worktreePath,env,timeoutMs:0});
      if(!Number.isSafeInteger(running.child.pid))
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
          agentLaunch:true,replayFixture:false,...nulls,
          process:{status:'UNKNOWN',exitCode:null,pid:null,startedAt:iso(now),
            terminatedAt:iso(now)},providerProbe:probe};
      const controller=new AbortController();
      activeAttempts.set(config.attemptId,controller);
      let timedOut=false,cancelled=false;
      const timeoutMs=input.timeoutMs??PI_PROVIDER_CONSTANTS.defaultTimeoutMs;
      const timer=setTimeout(()=>{timedOut=true;
        terminate(running.child.pid).catch(()=>{});},timeoutMs);
      timer.unref?.();
      const abort=()=>{cancelled=true;terminate(running.child.pid).catch(()=>{});};
      controller.signal.addEventListener('abort',abort,{once:true});
      input.signal?.addEventListener('abort',abort,{once:true});
      if(input.signal?.aborted)abort(); // aborted between spawn and registration
      const terminal=await running.done;
      clearTimeout(timer);
      controller.signal.removeEventListener('abort',abort);
      input.signal?.removeEventListener('abort',abort);
      activeAttempts.delete(config.attemptId);
      if((timedOut||cancelled)&&terminal.terminatedAt)await terminate(running.child.pid);
      // The agent's stdout is a claim, recorded as an observation only — the
      // candidate below comes from git, never from this parse.
      const agentObservation=parsePiJsonStream(terminal.stdout);
      // The worktree is left in place: occupancy is the physical
      // single-writer lock for crash reconciliation. The candidate ref is
      // the durable record.
      const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,
        pid:running.child.pid,startedAt:new Date(terminal.startedAt).toISOString(),
        terminatedAt:iso(now)};
      let state,reason;
      if(timedOut||terminal.spawnError||terminal.probeTimeout){
        state='LOCAL_ATTEMPT_UNKNOWN';
        reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
      }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';
      }else if(!candidate.writeSetValid){
        state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';
      }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';
      }else{state='LOCAL_CANDIDATE_READY';reason='REAL_PI_CANDIDATE_RECORDED';}
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,
        candidateCommit:candidate.candidateCommit??null,
        candidateRef:candidate.candidateRef??null,
        candidatePaths:candidate.paths??null,
        writeSetDetail:candidate.writeSetValid?null:candidate.detail??null,
        error:candidate.error??null,process,
        agent:{sessionHandle:agentObservation.sessionHandle,
          response:agentObservation.hasAssistant?
            agentObservation.response.slice(0,2000):null,
          model:agentObservation.model,provider:agentObservation.provider,
          usage:agentObservation.usage},
        stdoutTail:terminal.stdout.slice(-4000),stderrTail:terminal.stderr.slice(-2000),
        providerProbe:probe};
    }finally{await closeIsolationDirs(dirs);}
  }
  /** Signal-only cancellation: cancel() aborts the attempt's internal
   * AbortController; execute()'s abort listener performs the process-tree
   * termination. cancel() never kills anything by itself, and cancelling an
   * unknown/inactive attempt is reported honestly instead of guessed. */
  async function cancel(input){
    const attemptId=input?.attemptId;
    if(!ATTEMPT.test(attemptId??''))return {cancelled:false,reason:'CANCEL_INPUT_INVALID'};
    const controller=activeAttempts.get(attemptId);
    if(!controller)return {cancelled:false,reason:'ATTEMPT_NOT_ACTIVE'};
    controller.abort(safeError('ATTEMPT_CANCELLED'));
    return {cancelled:true,reason:'ATTEMPT_CANCELLED'};
  }
  /** Independent git readback: re-derives tree, parent and changed paths from
   * the repository, then applies the write-set fence (non-empty diff, every
   * path inside the allowed set, parent exactly the attempt's sourceSha). */
  async function readback(config,candidate){
    const normalized=normalizeConfig(config);
    const result=await ops.verifyCandidateReadback(normalized,candidate??{});
    const allowed=new Set(normalized.allowedWriteSet??[]);
    const verified=result.verified===true&&result.parent===normalized.sourceSha&&
      result.paths.length>0&&result.paths.every(path=>allowed.has(path));
    return {...result,verified,
      state:verified?'CANDIDATE_READBACK_VERIFIED':'CANDIDATE_READBACK_MISMATCH'};
  }
  return Object.freeze({providerId:PI_PROVIDER_CONSTANTS.providerId,
    probe:config=>ops.probeProvider(normalizeConfig(config)),
    execute,cancel,readback});
}

/** Selection policy. The default provider is codex; pi requires an explicit
 * request AND an enabled config row AND a locally validated capability probe.
 * There is deliberately NO failure-driven fallback: a codex failure never
 * switches provider, and an unvalidated pi config never silently selects pi. */
export function resolvePiProviderSelection({requested,pi,probe}){
  if(requested!=='pi')
    return {selected:'codex',piAdmitted:false,
      reason:requested==='codex'?'CODEX_EXPLICIT_DEFAULT':'CODEX_DEFAULT',probe:null};
  if(!pi||pi.enabled!==true||typeof pi.cliPath!=='string'||!pi.cliPath)
    return {selected:'codex',piAdmitted:false,reason:'PI_NOT_CONFIGURED',probe:null};
  if(!probe||probe.available!==true)
    return {selected:'codex',piAdmitted:false,reason:'PI_CAPABILITY_UNVERIFIED',
      probe:probe??null};
  return {selected:'pi',piAdmitted:true,reason:'PI_VALIDATED',probe};
}
