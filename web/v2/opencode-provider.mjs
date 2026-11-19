/** OpenCode Agent Provider adapter (#64, v2.1 candidate; NOT a v2 MVP requirement).
 * Ports the opencode CLI contract validated in the Go Worker (PR #471,
 * runtimes/opencode) into the v2 Node thin-adapter shape — the same injectable
 * spawn shape, state/reason vocabulary, git-derived candidate evidence and
 * selection policy as zcode-provider.mjs. The Go code is a CONTRACT REFERENCE
 * ONLY: v2 never runs through the old worker.
 *
 * CLI contract (opencode-ai pinned 1.19.6; observed, never invented):
 *  - `opencode run [flags] <message>`: one execution = one disposable process;
 *    `--format json` stdout = NDJSON event stream (step_start / text /
 *    tool_use / step_finish envelopes; the answer = the LAST text event's
 *    part.text, usage = sum of step_finish tokens input/output). exit 0 =
 *    completed and stdout carries the answer; exit != 0 = not completed.
 *  - argv: [bin,'run','--format','json','--pure', [--model p/m][--agent a],
 *    <message>]. The workspace is pinned via spawn cwd (a --dir flag exists
 *    but is redundant — never passed). `--attach` (shared server surface) is
 *    NEVER passed; no --standalone flag exists on 1.19.6. The instruction is
 *    always the last positional argv element, verbatim.
 *  - isolation (per attempt): a fresh temp tree is redirected through
 *    XDG_CONFIG_HOME / XDG_DATA_HOME / XDG_CACHE_HOME / XDG_STATE_HOME
 *    (+ TMP/TEMP); `opencode debug paths` verified live 2026-11-19 on 1.19.6:
 *    every path follows the matching XDG root + "opencode" subdir, so the
 *    owner's ~/.config/opencode and ~/.local/share/opencode (auth.json, logs,
 *    sessions) are never read or written. PWD/OLDPWD are pinned to the
 *    workspace (live-calibrated in #471: opencode resolves the project
 *    directory from the PWD env var, not from the process cwd).
 *  - auth: provider API keys are recognized from environment variables only
 *    (observed: "Environment: DeepSeek DEEPSEEK_API_KEY"); credentials are
 *    injected exclusively through the explicit config.env surface and are
 *    never printed, never inherited from the provider process env, and never
 *    written to disk (the isolated auth.json stays empty). Missing credentials
 *    would NOT fail the CLI (1.19.6 silently falls back to a credential-free
 *    default model) — the v2 candidate surface deliberately refuses that
 *    fallback at the CREDENTIALS probe stage, fail-closed: an uncontrolled
 *    model fallback must never produce candidates.
 *  - version gate: `opencode --version` prints a bare version line, exit 0;
 *    a mismatch with the pinned version is fail-closed (#471: run-surface
 *    flags drift between upstream versions).
 *  Live verification (this workstation, 2026-11-19, opencode 1.19.6):
 *  - `opencode --version` → "1.19.6", exit 0.
 *  - `opencode run --help` → confirms --format {default,json}, --pure,
 *    --model provider/model, --agent, message positional, --attach exists.
 *  - `opencode debug paths` under an XDG redirect tree → all paths redirected.
 *  - canary with an unregistered model (deepseek/deepseek-chat) → exit 1 in
 *    ~3.9s, stdout carries {"type":"error",...} NDJSON, stderr empty (delta
 *    vs the Go note "structured error JSON on stderr": on this machine the
 *    error event arrives as a stdout NDJSON line; the exit!=0 classification
 *    is unaffected — the "error" envelope type carries no answer and is
 *    skipped by the parser exactly like the other unobserved types).
 *  - canary with the registered model deepseek/deepseek-v4-pro → exit 0 in
 *    8.4s; NDJSON step_start → text("OK") → step_finish (tokens input=8846,
 *    output=2); session handle ses_e0c4422c3ffeI6T4xtqHqAk5xP observed but
 *    deliberately NOT reported (conservative #471 mapping: ses_* is a
 *    per-run fact, not a session-establishment fact).
 *
 * Durability is git-anchored only (candidate ref + worktree occupancy);
 * there is no second scheduler, kernel, permission system or event log.
 * taskPass/releasePass stay false — agent output is diagnostics, the
 * candidate comes exclusively from independent git readback. The default
 * provider stays codex; opencode is DISABLED unless explicitly requested
 * through resolveOpencodeProviderSelection with an enabled config row plus a
 * locally validated probe. There is no failure-driven fallback. */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {mkdir,mkdtemp,rm} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {tmpdir} from 'node:os';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const FLAG_TOKEN=/^[A-Za-z0-9._-]+$/;
const DEFAULT_CANDIDATE='experiments/v2/selected-business-build/LOCAL_OPENCODE_L3.md';
const safeRef=attemptId=>attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();

export const OPENCODE_PROVIDER_CONSTANTS=Object.freeze({
  providerId:'opencode',
  // #471 pins the argv/output contract to this exact upstream version; a
  // different observed --version fails the probe closed (VERSION stage).
  pinnedCliVersion:'1.19.6',
  candidateRelativePath:DEFAULT_CANDIDATE,
  candidateRefPrefix:'refs/tally-opencode-candidates/',
  canaryPrompt:'Reply with exactly: OK',
  canaryTimeoutMs:180000,
  providerApiKeyVar:'DEEPSEEK_API_KEY'});

/** Builder environment is an explicit allowlist. Credentials are NEVER
 * inherited from the provider process env: the only credential path is the
 * explicit config.env injection (the #471 auth model — opencode reads provider
 * API keys from environment variables, e.g. DEEPSEEK_API_KEY). No GitHub/CI
 * credentials either — the builder never pushes; the Host finalizes the
 * candidate ref itself. The per-attempt XDG/TMP/TEMP isolation keys and the
 * PWD/OLDPWD workspace pins are applied by the caller AFTER this allowlist,
 * last-wins over config.env (the owner deployment surface must not leak in). */
export function opencodeBuilderEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  const home=config.opencodeUserProfile??process.env.USERPROFILE??'C:\\Users\\15549';
  const path=[root+'\\System32','C:\\Program Files\\Git\\cmd'].join(';');
  const env={
    SystemRoot:root,SystemDrive:config.systemDrive??'C:',ComSpec:root+'\\System32\\cmd.exe',
    PATH:path,PATHEXT:'.COM;.EXE;.BAT;.CMD',
    USERPROFILE:home,HOME:home,
    APPDATA:home+'\\AppData\\Roaming',LOCALAPPDATA:home+'\\AppData\\Local',
    TEMP:config.builderTemp??home+'\\AppData\\Local\\Temp',
    TMP:config.builderTemp??home+'\\AppData\\Local\\Temp',
    HTTPS_PROXY:config.modelProxy??'http://127.0.0.1:10808',
    HTTP_PROXY:config.modelProxy??'http://127.0.0.1:10808',
    GIT_CONFIG_NOSYSTEM:'1',
    SSL_CERT_FILE:config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt',
    CURL_CA_BUNDLE:config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt',
    GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never',
    GIT_AUTHOR_NAME:'Tally Builder',GIT_AUTHOR_EMAIL:'tally-builder@localhost',
    GIT_COMMITTER_NAME:'Tally Builder',GIT_COMMITTER_EMAIL:'tally-builder@localhost',
    NO_COLOR:'1'};
  if(typeof config.emptyGitConfig==='string')env.GIT_CONFIG_GLOBAL=config.emptyGitConfig;
  // Explicit credential/config injection surface (#471 Config.Env analog):
  // only literal strings from the provider config, never process.env.
  if(config.env)for(const [name,value] of Object.entries(config.env))
    if(typeof value==='string')env[name]=value;
  return env;
}

/** Per-attempt isolation overrides (#471 sessionDirs analog): the six forced
 * keys are applied last-wins over the builder allowlist AND over config.env
 * same-names — the owner deployment surface cannot silently redirect
 * opencode's config/data/cache/state or the OS temp. PWD/OLDPWD pin to the
 * workspace (live-calibrated in #471; opencode resolves the project directory
 * from PWD, and a stale inherited OLDPWD is a cross-workspace leak). */
export function opencodeSessionEnvironment(sessionDirs,worktreePath){
  return {...sessionDirs.env(),PWD:worktreePath,OLDPWD:worktreePath};
}

async function newSessionDirs(){
  const root=await mkdtemp(join(tmpdir(),'tally-opencode-attempt-'));
  for(const sub of ['config','data','cache','state','tmp'])
    await mkdir(join(root,sub),{recursive:true});
  return {root,
    env:()=>({XDG_CONFIG_HOME:join(root,'config'),XDG_DATA_HOME:join(root,'data'),
      XDG_CACHE_HOME:join(root,'cache'),XDG_STATE_HOME:join(root,'state'),
      TMP:join(root,'tmp'),TEMP:join(root,'tmp')}),
    remove:async()=>{await rm(root,{recursive:true,force:true});}};
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
  if(result.code!==0)console.error('OPENCODE_HOST_COMMAND_FAILED',command,args,
    JSON.stringify(result).slice(0,4000));
  return result;
}

/** Parse `--format json` stdout (NDJSON, one event per line). The answer is
 * the LAST text event's part.text; usage sums every step_finish tokens
 * input/output. Single unparseable lines are skipped silently (tolerates
 * noise and the observed {"type":"error"} envelope); only the observed
 * envelope types carry the answer — unobserved types are ignored but still
 * mark sawJSON. Agent output is diagnostics only; it never becomes evidence. */
export function parseOpencodeRunOutput(stdout){
  const out={answer:null,usage:{inputTokens:0,outputTokens:0},sawJSON:false};
  if(typeof stdout!=='string')return out;
  for(const rawLine of stdout.split(/\r?\n/)){
    const line=rawLine.trim();
    if(!line.startsWith('{'))continue;
    let event;try{event=JSON.parse(line);}catch{continue;}
    if(!event||typeof event!=='object'||typeof event.type!=='string')continue;
    const part=event.part&&typeof event.part==='object'?event.part:null;
    if(event.type==='text'&&part&&typeof part.text==='string'){
      out.answer=part.text;out.sawJSON=true;
    }else if(event.type==='step_finish'){
      if(part&&part.tokens&&typeof part.tokens==='object'){
        if(Number.isFinite(part.tokens.input))out.usage.inputTokens+=part.tokens.input;
        if(Number.isFinite(part.tokens.output))out.usage.outputTokens+=part.tokens.output;
      }
      out.sawJSON=true;
    }else out.sawJSON=true;
  }
  return out;
}

/** #471 argv shape: [bin,'run','--format','json','--pure', [--model p/m]
 * [--agent a] <message>]. Flag values are validated against the observed
 * token charset (model = exactly one '/', both sides flag-safe; agent = one
 * flag-safe token); invalid values are rejected instead of escaped. */
export function validateOpencodeFlagConfig(config){
  const model=config.model??'';
  const agent=config.agent??'';
  if(model){
    const slash=model.indexOf('/');
    if(slash<=0||slash===model.length-1||model.includes('/',slash+1))
      return {valid:false,detail:'MODEL_FLAG_INVALID'};
    if(!FLAG_TOKEN.test(model.slice(0,slash))||!FLAG_TOKEN.test(model.slice(slash+1)))
      return {valid:false,detail:'MODEL_FLAG_INVALID'};
  }
  if(agent&&!FLAG_TOKEN.test(agent))return {valid:false,detail:'AGENT_FLAG_INVALID'};
  return {valid:true};
}

export function opencodeArgv(config,{prompt}){
  const argv=[config.cliPath,'run','--format','json','--pure'];
  if(config.model)argv.push('--model',config.model);
  if(config.agent)argv.push('--agent',config.agent);
  argv.push(prompt);
  return argv;
}

export function opencodeTaskPrompt({allowedWriteSet,taskPrompt}){
  const paths=(allowedWriteSet??[DEFAULT_CANDIDATE]).map(x=>`  - ${x}`).join('\n');
  return `Work only in the current disposable Git worktree. Modify or create exactly the allowed paths:\n${paths}\nDo not modify, create or delete any other path. Do not run git commit, git push or any GitHub or network operation. Stop after the file changes are complete.\n\nTask: ${taskPrompt}`;
}

function createDefaultOperations(spawnFn){
  const gitPath=config=>config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
  const git=(config,args,options={})=>short(spawnFn,gitPath(config),
    args,{env:hostEnvironment(config),...options});
  /** Staged probe: VERSION (CLI exists + bare version line matches the pinned
   * version) → CREDENTIALS (read-only env-injection presence check per the
   * #471 auth model) → CANARY (one bounded isolated real turn). Any stage
   * failing reports {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage,
   * detail} and the caller never spawns. */
  async function probeProvider(config){
    console.error('OPENCODE_PROBE_PROVIDER_START');
    if(typeof config.cliPath!=='string'||!config.cliPath||!existsSync(config.cliPath))
      return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:null,outputMatched:false,
        detail:'MISSING_CLI'};
    const version=await spawnCapture(spawnFn,config.cliPath,['--version'],
      {env:opencodeBuilderEnvironment(config),timeoutMs:30000}).done
      .catch(()=>({code:null,stdout:'',stderr:'',spawnError:true}));
    const versionText=version.spawnError?'':(version.stdout.trim().split(/\r?\n/)[0]??'').trim();
    if(version.spawnError||version.code!==0||!versionText)
      return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
        outputMatched:false,detail:version.spawnError?'SPAWN_ERROR':'VERSION_UNOBSERVABLE'};
    const pinned=config.pinnedVersion??OPENCODE_PROVIDER_CONSTANTS.pinnedCliVersion;
    if(versionText!==pinned)
      return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
        outputMatched:false,detail:'VERSION_MISMATCH',observedVersion:versionText,
        pinnedVersion:pinned};
    // Read-only credential presence check (#471 auth model): provider API
    // keys arrive exclusively through the explicit config.env injection; the
    // value itself is never printed and never read from process.env here.
    const apiKeyVar=config.providerApiKeyVar??OPENCODE_PROVIDER_CONSTANTS.providerApiKeyVar;
    const injected=config.env?.[apiKeyVar];
    if(typeof injected!=='string'||!injected)
      return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',exitCode:null,outputMatched:false,
        detail:'CREDENTIALS_ABSENT'};
    // The canary is the only definitive auth+model check; a bounded stall is
    // reported honestly instead of being retried into a fake PASS.
    const dirs=await newSessionDirs();
    const canaryWs=await mkdtemp(join(tmpdir(),'tally-opencode-canary-'));
    try{
      const canaryArgv=opencodeArgv(config,{prompt:OPENCODE_PROVIDER_CONSTANTS.canaryPrompt});
      const canary=await spawnCapture(spawnFn,canaryArgv[0],canaryArgv.slice(1),
        {cwd:canaryWs,timeoutMs:config.canaryTimeoutMs??OPENCODE_PROVIDER_CONSTANTS.canaryTimeoutMs,
          env:{...opencodeBuilderEnvironment(config),...opencodeSessionEnvironment(dirs,canaryWs)}}).done;
      const parsed=parseOpencodeRunOutput(canary.stdout);
      if(canary.probeTimeout)
        return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',exitCode:null,outputMatched:false,
          detail:'CREDENTIALS_OR_PROVIDER_STALL'};
      if(canary.code!==0||!parsed.answer||!/OK/i.test(parsed.answer.trim()))
        return {available:false,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
          reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',exitCode:canary.code,
          outputMatched:false,detail:'CANARY_UNSATISFIED'};
      return {available:true,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
        canary:{answer:parsed.answer.trim(),usage:parsed.usage}};
    }finally{
      await dirs.remove();
      await rm(canaryWs,{recursive:true,force:true});
    }
  }
  async function attemptRecord(config){
    // Git-anchored durable state only: the candidate ref and worktree
    // occupancy ARE the attempt record (no second store by design).
    const ref=OPENCODE_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
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
    console.error('OPENCODE_PREPARE_WORKTREE_START',config.worktreePath);
    await mkdir(dirname(config.worktreePath),{recursive:true});
    // One disposable detached worktree per attempt; occupancy is the
    // physical single-writer lock, so an add failure is resolved against the
    // registered worktree list instead of being blindly retried.
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
    const allowed=new Set(config.allowedWriteSet??[DEFAULT_CANDIDATE]);
    const run=(args,options={})=>git(config,['-C',config.worktreePath,...args],options);
    const [tracked,untracked]=await Promise.all([
      run(['diff','--name-only',config.sourceSha]),
      run(['ls-files','--others','--exclude-standard','--no-directory'])]);
    if(tracked.code!==0||untracked.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const paths=[...new Set([...tracked.stdout.split(/\r?\n/),...untracked.stdout.split(/\r?\n/)]
      .map(x=>x.trim()).filter(Boolean))];
    // Post-hoc write-set fence: every changed path must be inside the allowed
    // set and the diff must be non-empty. Violations are FAILED, not PASS.
    if(paths.length===0||paths.some(path=>!allowed.has(path)))
      return {writeSetValid:false,paths};
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
    const ref=OPENCODE_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
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

/** Unified TaskProvider (#64). createOpencodeTaskProvider({spawnFn,operations,
 * now,killTree}) → frozen {providerId,probe,execute,cancel,readback}. */
export function createOpencodeTaskProvider({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const result=await short(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    return result.code===0;
  });
  async function execute(input){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:input.issueRef,provider:OPENCODE_PROVIDER_CONSTANTS.providerId,
      providerId:OPENCODE_PROVIDER_CONSTANTS.providerId,taskId:input.taskId??null,
      sourceSha:input.sourceSha,attemptId:input.attemptId,...flags};
    const notExecuted=(reason,detail={})=>({...base,state:'AGENT_NOT_EXECUTED',reason,
      agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
      candidateRef:null,candidatePaths:null,error:null,process:null,...detail});
    // Earliest gate: shape-invalid attempts never spawn and never probe.
    if((input.providerId&&input.providerId!==OPENCODE_PROVIDER_CONSTANTS.providerId)||
      !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.attemptId??'')||
      typeof input.taskPrompt!=='string'||!input.taskPrompt||
      !Array.isArray(input.allowedWriteSet)||input.allowedWriteSet.some(x=>typeof x!=='string')||
      typeof input.providerConfig!=='object'||input.providerConfig===null||
      typeof input.providerConfig.cliPath!=='string'||!input.providerConfig.cliPath||
      !validateOpencodeFlagConfig(input.providerConfig).valid)
      return notExecuted('ATTEMPT_INPUT_INVALID');
    const config={...input,...input.providerConfig};
    // A pre-aborted signal is a normal cancelled terminal state — never a
    // failed attempt and never a spawn (#471: cancel before start).
    if(input.signal?.aborted)return {...base,state:'LOCAL_ATTEMPT_CANCELLED',
      reason:'ATTEMPT_CANCELLED',agentLaunch:false,replayFixture:false,candidateTree:null,
      candidateCommit:null,candidateRef:null,candidatePaths:null,error:null,process:null};
    // Durable replay first: a completed attempt (candidate ref present) is
    // read back from git and NEVER relaunched; an occupied worktree without a
    // ref may belong to a running or crashed attempt and stays UNKNOWN.
    const record=await ops.attemptRecord(config);
    if(record?.kind==='CANDIDATE')return {...base,state:'LOCAL_CANDIDATE_READY',
      reason:'REAL_OPENCODE_CANDIDATE_RECORDED',agentLaunch:true,replayed:true,
      replayFixture:false,candidateTree:record.candidateTree,
      candidateCommit:record.candidateCommit,candidateRef:record.ref,
      candidatePaths:record.paths,error:null,process:null};
    if(record?.kind==='STALE')return {...base,state:'NOT_ADMITTED',reason:'CANDIDATE_STALE',
      claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
      error:null,process:null};
    if(record?.kind==='WORKTREE_OCCUPIED')return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',
      reason:'ATTEMPT_STATUS_UNKNOWN',agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
      error:null,process:null};
    const probe=await ops.probeProvider(config);
    if(!probe.available)return notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
    const worktree=await ops.prepareWorktree(config).catch(error=>error.code);
    if(typeof worktree==='string')return {...base,state:'NOT_ADMITTED',reason:worktree,
      claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,candidateRef:null,candidatePaths:null,
      error:null,process:null,providerProbe:probe};
    const dirs=await newSessionDirs();
    try{
      const prompt=opencodeTaskPrompt(input);
      if(prompt.length>7500)return notExecuted('ATTEMPT_INPUT_INVALID',
        {detail:'PROMPT_TOO_LONG_FOR_ARGV',providerProbe:probe});
      const argv=opencodeArgv(config,{prompt});
      const env={...opencodeBuilderEnvironment(config),
        ...opencodeSessionEnvironment(dirs,worktree.worktreePath)};
      const running=spawnCapture(spawnFn,argv[0],argv.slice(1),
        {cwd:worktree.worktreePath,env,timeoutMs:0});
      if(!Number.isSafeInteger(running.child.pid))
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
          agentLaunch:true,replayFixture:false,candidateTree:null,candidateCommit:null,
          candidateRef:null,candidatePaths:null,error:null,providerProbe:probe,
          process:{status:'UNKNOWN',exitCode:null,pid:null,startedAt:iso(now),terminatedAt:iso(now)}};
      const startedAt=iso(now);
      let timedOut=false,cancelled=false;
      const timeoutMs=input.timeoutMs??600000;
      const timer=setTimeout(()=>{timedOut=true;terminate(running.child.pid).catch(()=>{});},timeoutMs);
      timer.unref?.();
      const abort=()=>{cancelled=true;terminate(running.child.pid).catch(()=>{});};
      input.signal?.addEventListener('abort',abort,{once:true});
      // The signal may have aborted while the attempt was being prepared
      // (record/probe/worktree/session dirs): addEventListener does not fire
      // retroactively on an aborted signal, so re-check once and cancel now.
      if(input.signal?.aborted)abort();
      const terminal=await running.done;
      clearTimeout(timer);input.signal?.removeEventListener('abort',abort);
      if((timedOut||cancelled)&&terminal.terminatedAt)await terminate(running.child.pid);
      const agentOutput=parseOpencodeRunOutput(terminal.stdout);
      // The worktree is left in place: occupancy is the physical single-writer
      // lock for crash reconciliation. The candidate ref is the durable record.
      const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
        startedAt:new Date(terminal.startedAt).toISOString(),terminatedAt:iso(now)};
      // #471 fail-safe: exit 0 with an EMPTY stdout violates the observed
      // contract (exit 0 = completed AND stdout carries the answer) — judged
      // failed, never guessed into success. Non-JSON but non-empty stdout
      // degrades to the raw text (answer extraction only; evidence stays git).
      const contractUnfulfilled=terminal.code===0&&!terminal.spawnError&&
        !terminal.probeTimeout&&agentOutput.answer===null&&terminal.stdout.trim()==='';
      let state,reason;
      if(timedOut||terminal.spawnError||terminal.probeTimeout){
        state='LOCAL_ATTEMPT_UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
      }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';
      }else if(contractUnfulfilled){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_CONTRACT_UNFULFILLED';
      }else if(!candidate.writeSetValid){
        state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';
      }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';
      }else{state='LOCAL_CANDIDATE_READY';reason='REAL_OPENCODE_CANDIDATE_RECORDED';}
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,
        candidateRef:candidate.candidateRef??null,candidatePaths:candidate.paths??null,
        error:candidate.error??null,process,providerProbe:probe,
        agent:{answer:agentOutput.answer!==null?String(agentOutput.answer).slice(0,2000):null,
          usage:agentOutput.usage},
        stdoutTail:terminal.stdout.slice(-4000),stderrTail:terminal.stderr.slice(-2000)};
    }finally{
      await dirs.remove();
    }
  }
  /** The cancel channel is controller-shaped: the R105 dispatch passes the
   * AbortController under the controller key; direct callers may pass a
   * signal-like object under signal (with the abort face on input.abort or
   * on the signal itself). Not-aborted → abort() and true; already-aborted
   * → false; missing channel (or no abort surface) → false. */
  function cancel(input){
    const channel=input?.controller??input?.signal;
    if(!channel||typeof channel!=='object')return false;
    if(channel.signal?.aborted===true||channel.aborted===true)return false;
    const abort=typeof channel.abort==='function'?()=>channel.abort():
      typeof input?.abort==='function'?input.abort:null;
    if(!abort)return false;
    abort();
    return true;
  }
  /** Independent git readback of one candidate. Accepts either the full
   * attempt input (providerConfig merged in, like execute) or a flat config
   * carrying repositoryPath + sourceSha. */
  async function readback(config,candidate){
    const merged=config&&typeof config==='object'&&config.providerConfig?
      {...config,...config.providerConfig}:config;
    return ops.verifyCandidateReadback(merged,candidate);
  }
  return Object.freeze({providerId:OPENCODE_PROVIDER_CONSTANTS.providerId,
    execute,cancel,readback,
    probe:config=>ops.probeProvider(config)});
}

/** Selection policy. The default provider is codex; opencode requires an
 * explicit request AND an enabled config row AND a locally validated
 * capability probe. There is deliberately NO failure-driven fallback: a codex
 * failure never switches provider, and an unvalidated opencode config never
 * silently selects opencode. */
export function resolveOpencodeProviderSelection({requested,opencode,probe}){
  if(requested!=='opencode')
    return {selected:'codex',opencodeAdmitted:false,
      reason:requested==='codex'?'CODEX_EXPLICIT_DEFAULT':'CODEX_DEFAULT',probe:null};
  if(!opencode||opencode.enabled!==true)
    return {selected:'codex',opencodeAdmitted:false,reason:'OPENCODE_NOT_CONFIGURED',probe:null};
  if(!probe||probe.available!==true)
    return {selected:'codex',opencodeAdmitted:false,reason:'OPENCODE_CAPABILITY_UNVERIFIED',
      probe:probe??null};
  return {selected:'opencode',opencodeAdmitted:true,reason:'OPENCODE_VALIDATED',probe};
}
