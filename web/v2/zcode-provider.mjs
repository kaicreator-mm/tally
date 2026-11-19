/** ZCode Agent Provider adapter (#64, v2.1 candidate; NOT a v2 MVP requirement).
 * Mirrors the local-codex-host execution contract — same injectable spawn
 * shape, same state/reason vocabulary, same git-derived candidate evidence —
 * without touching the selected v2 Codex path: the default provider stays
 * codex and zcode is only usable through resolveZcodeProviderSelection with
 * explicit config plus a locally validated capability probe. Durability is
 * git-anchored only (candidate ref + worktree occupancy); there is no second
 * scheduler, kernel, permission system or event log. */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {mkdir,readFile} from 'node:fs/promises';
import {dirname,join} from 'node:path';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const SEMVER=/^\d+\.\d+\.\d+/;
const DEFAULT_CANDIDATE='experiments/v2/selected-business-build/LOCAL_ZCODE_L3.md';
const safeRef=attemptId=>attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();

export const ZCODE_PROVIDER_CONSTANTS=Object.freeze({providerId:'zcode',
  candidateRelativePath:DEFAULT_CANDIDATE,
  candidateRefPrefix:'refs/tally-zcode-candidates/',
  canaryPrompt:'Reply with exactly: OK'});

/** Builder environment is an explicit allowlist. No ZCODE_* inheritance: the
 * desktop-inherited provider pointers are a proven stale-path trap and the
 * API-key override turns auth failures into silent stalls; ZCode credentials
 * live in ~/.zcode/v2/credentials.json, not the environment. No GitHub/CI
 * credentials either — the builder never pushes; the Host finalizes the
 * candidate ref itself. */
export function zcodeBuilderEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  const home=config.zcodeUserProfile??process.env.USERPROFILE??'C:\\Users\\15549';
  const path=[dirname(config.nodePath??process.execPath),root+'\\System32',
    'C:\\Program Files\\Git\\cmd'].join(';');
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
  if(config.zcodeLocale)env.ZCODE_LOCALE=config.zcodeLocale;
  return env;
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
  if(result.code!==0)console.error('ZCODE_HOST_COMMAND_FAILED',command,args,
    JSON.stringify(result).slice(0,4000));
  return result;
}

/** Parse the agent's final JSON from stdout. With --json the turn ends with
 * one object, but noise lines (AI SDK warnings) can precede it — never
 * JSON.parse the whole stream. Unparseable output yields null; the git-derived
 * candidate stays the only evidence, so a null here never fabricates success. */
export function parseZcodeAgentResult(stdout){
  if(typeof stdout!=='string'||!stdout.includes('{'))return null;
  const direct=stdout.trim();
  try{const value=JSON.parse(direct);return value&&typeof value==='object'?value:null;}
  catch{}
  const first=direct.indexOf('{'),last=direct.lastIndexOf('}');
  if(first<0||last<=first)return null;
  try{const value=JSON.parse(direct.slice(first,last+1));
    return value&&typeof value==='object'?value:null;}catch{}
  return null;
}

function zcodeArgv(config,{prompt,worktreePath,attach}){
  const argv=[config.nodePath??process.execPath,config.cliPath,'-p',prompt,
    '--cwd',worktreePath,'--mode',config.mode??'yolo','--json','--no-color'];
  if(attach)argv.push('--attach',attach);
  return argv;
}

export function zcodeTaskPrompt({allowedWriteSet,taskPrompt}){
  const paths=(allowedWriteSet??[DEFAULT_CANDIDATE]).map(x=>`  - ${x}`).join('\n');
  return `Work only in the current disposable Git worktree. Modify or create exactly the allowed paths:\n${paths}\nDo not modify, create or delete any other path. Do not run git commit, git push or any GitHub or network operation. Stop after the file changes are complete.\n\nTask: ${taskPrompt}`;
}

function createDefaultOperations(spawnFn){
  const gitPath=config=>config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
  const git=(config,args,options={})=>short(spawnFn,gitPath(config),
    args,{env:hostEnvironment(config),...options});
  async function probeProvider(config){
    console.error('ZCODE_PROBE_PROVIDER_START');
    if(typeof config.cliPath!=='string'||!config.cliPath||!existsSync(config.cliPath))
      return {available:false,provider:ZCODE_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:null,outputMatched:false,
        detail:'MISSING_CLI'};
    const version=await spawnCapture(spawnFn,config.nodePath??process.execPath,
      [config.cliPath,'version'],{env:zcodeBuilderEnvironment(config),timeoutMs:30000}).done
      .catch(()=>({code:null,stdout:'',stderr:'',spawnError:true}));
    // `--version` lies (bundle stub); the `version` subcommand is the CLI core.
    if(version.code!==0||!SEMVER.test(version.stdout.trim().split(/\r?\n/)[0]??''))
      return {available:false,provider:ZCODE_PROVIDER_CONSTANTS.providerId,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',exitCode:version.code,
        outputMatched:false,detail:version.spawnError?'SPAWN_ERROR':'VERSION_MISMATCH'};
    const versionText=version.stdout.trim().split(/\r?\n/)[0];
    const credentialsPath=config.credentialsPath??join(process.env.USERPROFILE??'.',
      '.zcode','v2','credentials.json');
    try{
      const credentials=JSON.parse(await readFile(credentialsPath,'utf8'));
      if(typeof credentials['oauth:active_provider']!=='string'||
        !credentials['oauth:active_provider'])
        throw new Error('empty');
    }catch{
      return {available:false,provider:ZCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CREDENTIALS',exitCode:null,outputMatched:false,
        detail:'CREDENTIALS_ABSENT'};
    }
    // Invalid credentials do NOT fail fast — they stall silently. The canary
    // is the only definitive auth+provider check; a bounded stall is reported
    // honestly instead of being retried into a fake PASS.
    const canaryArgv=zcodeArgv(config,{prompt:ZCODE_PROVIDER_CONSTANTS.canaryPrompt,
      worktreePath:dirname(credentialsPath)});
    const canary=await spawnCapture(spawnFn,canaryArgv[0],canaryArgv.slice(1),
      {env:zcodeBuilderEnvironment(config),timeoutMs:config.canaryTimeoutMs??180000}).done;
    const canaryResult=parseZcodeAgentResult(canary.stdout);
    if(canary.code!==0||!canaryResult||!/OK/i.test(String(canaryResult.response??'')))
      return {available:false,provider:ZCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
        reason:'LOCAL_AGENT_UNAVAILABLE',stage:'CANARY',exitCode:canary.code,
        outputMatched:false,detail:canary.probeTimeout?'CREDENTIALS_OR_PROVIDER_STALL':'CANARY_UNSATISFIED'};
    return {available:true,provider:ZCODE_PROVIDER_CONSTANTS.providerId,version:versionText,
      canary:{sessionId:canaryResult.sessionId??null,response:String(canaryResult.response)}};
  }
  async function attemptRecord(config){
    // Git-anchored durable state only: the candidate ref and worktree
    // occupancy ARE the attempt record (no second store by design).
    const ref=ZCODE_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
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
    console.error('ZCODE_PREPARE_WORKTREE_START',config.worktreePath);
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
    const ref=ZCODE_PROVIDER_CONSTANTS.candidateRefPrefix+safeRef(config.attemptId);
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

export function createZcodeProviderHost({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const result=await short(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    return result.code===0;
  });
  async function runAttempt(config){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:config.issueRef,provider:ZCODE_PROVIDER_CONSTANTS.providerId,
      sourceSha:config.sourceSha,attemptId:config.attemptId,...flags};
    const notExecuted=(reason,detail={})=>({...base,state:'AGENT_NOT_EXECUTED',reason,
      agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
      process:null,...detail});
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.attemptId??'')||
      typeof config.taskPrompt!=='string'||!config.taskPrompt)
      return notExecuted('ATTEMPT_INPUT_INVALID');
    // Durable replay first: a completed attempt (candidate ref present) is
    // read back from git and NEVER relaunched; an occupied worktree without a
    // ref may belong to a running or crashed attempt and stays UNKNOWN.
    const record=await ops.attemptRecord(config);
    if(record?.kind==='CANDIDATE')return {...base,state:'LOCAL_CANDIDATE_READY',
      reason:'REAL_ZCODE_CANDIDATE_RECORDED',agentLaunch:true,replayed:true,
      candidateTree:record.candidateTree,candidateCommit:record.candidateCommit,
      process:null};
    if(record?.kind==='STALE')return {...base,state:'NOT_ADMITTED',reason:'CANDIDATE_STALE',
      claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,process:null};
    if(record?.kind==='WORKTREE_OCCUPIED')return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',
      reason:'ATTEMPT_STATUS_UNKNOWN',agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,process:null};
    const probe=await ops.probeProvider(config);
    if(!probe.available)return notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
    const worktree=await ops.prepareWorktree(config).catch(error=>error.code);
    if(typeof worktree==='string')return {...base,state:'NOT_ADMITTED',reason:worktree,
      claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,process:null,providerProbe:probe};
    const env=zcodeBuilderEnvironment(config);
    const prompt=zcodeTaskPrompt(config);
    if(prompt.length>7500&&!config.briefPath)return notExecuted('ATTEMPT_INPUT_INVALID',
      {detail:'PROMPT_TOO_LONG_FOR_ARGV'});
    const argv=zcodeArgv(config,{prompt:config.briefPath?
      'Execute the task described in the attached brief file exactly. Stay inside the allowed write set.':prompt,
      worktreePath:worktree.worktreePath,attach:config.briefPath});
    const running=spawnCapture(spawnFn,argv[0],argv.slice(1),
      {cwd:worktree.worktreePath,env,timeoutMs:0});
    if(!Number.isSafeInteger(running.child.pid))
      return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
        agentLaunch:true,replayFixture:false,candidateTree:null,candidateCommit:null,
        process:{status:'UNKNOWN',exitCode:null,pid:null,startedAt:iso(now),terminatedAt:iso(now)}};
    const startedAt=iso(now);
    let timedOut=false,cancelled=false;
    const timeoutMs=config.timeoutMs??600000;
    const timer=setTimeout(()=>{timedOut=true;terminate(running.child.pid).catch(()=>{});},timeoutMs);
    timer.unref?.();
    const abort=()=>{cancelled=true;terminate(running.child.pid).catch(()=>{});};
    config.signal?.addEventListener('abort',abort,{once:true});
    const terminal=await running.done;
    clearTimeout(timer);config.signal?.removeEventListener('abort',abort);
    if((timedOut||cancelled)&&terminal.terminatedAt)await terminate(running.child.pid);
    const agentResult=parseZcodeAgentResult(terminal.stdout);
    // The worktree is left in place: occupancy is the physical single-writer
    // lock for crash reconciliation. The candidate ref is the durable record.
    const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
    const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
      exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
      startedAt:new Date(terminal.startedAt).toISOString(),terminatedAt:iso(now)};
    let state,reason;
    if(timedOut||terminal.spawnError||terminal.probeTimeout){
      state='LOCAL_ATTEMPT_UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
    }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';
    }else if(!candidate.writeSetValid){
      state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';
    }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';
    }else{state='LOCAL_CANDIDATE_READY';reason='REAL_ZCODE_CANDIDATE_RECORDED';}
    return {...base,state,reason,agentLaunch:true,replayFixture:false,
      candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,
      candidateRef:candidate.candidateRef??null,candidatePaths:candidate.paths??null,process,
      agent:{sessionId:agentResult?.sessionId??null,
        response:agentResult?String(agentResult.response??'').slice(0,2000):null,
        usage:agentResult?.usage??null},
      stdoutTail:terminal.stdout.slice(-4000),stderrTail:terminal.stderr.slice(-2000),
      providerProbe:probe};
  }
  async function joinVerifyCandidate(config,candidate){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const readback=await ops.verifyCandidateReadback(config,candidate);
    const allowed=new Set(config.allowedWriteSet??[DEFAULT_CANDIDATE]);
    const verified=readback.verified===true&&readback.parent===config.sourceSha&&
      readback.paths.length>0&&readback.paths.every(path=>allowed.has(path));
    return {...flags,issueRef:config.issueRef,provider:ZCODE_PROVIDER_CONSTANTS.providerId,
      sourceSha:config.sourceSha,attemptId:candidate.attemptId,
      state:verified?'CANDIDATE_READBACK_VERIFIED':'CANDIDATE_READBACK_MISMATCH',
      reason:verified?'ZCODE_CANDIDATE_VERIFIED_FROM_GIT':'CANDIDATE_READBACK_MISMATCH',
      agentLaunch:false,candidateTree:readback.candidateTree??null,
      candidatePaths:readback.paths??null,readback};
  }
  return Object.freeze({runAttempt,joinVerifyCandidate,
    probeProvider:config=>ops.probeProvider(config)});
}

/** Selection policy. The default provider is codex; zcode requires an explicit
 * request AND an enabled config row AND a locally validated capability probe.
 * There is deliberately NO failure-driven fallback: a codex failure never
 * switches provider, and an unvalidated zcode config never silently selects
 * zcode. */
export function resolveZcodeProviderSelection({requested,zcode,probe}){
  if(requested!=='zcode')
    return {selected:'codex',zcodeAdmitted:false,
      reason:requested==='codex'?'CODEX_EXPLICIT_DEFAULT':'CODEX_DEFAULT',probe:null};
  if(!zcode||zcode.enabled!==true||typeof zcode.cliPath!=='string'||!zcode.cliPath)
    return {selected:'codex',zcodeAdmitted:false,reason:'ZCODE_NOT_CONFIGURED',probe:null};
  if(!probe||probe.available!==true)
    return {selected:'codex',zcodeAdmitted:false,reason:'ZCODE_CAPABILITY_UNVERIFIED',
      probe:probe??null};
  return {selected:'zcode',zcodeAdmitted:true,reason:'ZCODE_VALIDATED',probe};
}

/** Adapter to the journey-controller run(kind,env,evidenceDir) seam so the
 * zcode provider plugs into the EXISTING execution contract unchanged:
 * {status,reason,exitCode,signal,timeout,observedAt,result,evidenceDir}. */
export function createZcodeJourneyRun({host,attemptConfig}){
  if(typeof host?.runAttempt!=='function'||typeof attemptConfig!=='function')
    throw new TypeError('ZCODE_JOURNEY_RUN_INPUT_REQUIRED');
  return Object.freeze({async run(kind,env,evidenceDir){
    if(kind!=='start')return {status:'UNKNOWN',reason:'ZCODE_REVIEW_CONSUMER_NOT_MOUNTED',
      exitCode:null,signal:null,timeout:false,observedAt:new Date().toISOString(),
      result:null};
    const attempt=await host.runAttempt(attemptConfig(kind,env));
    const status=attempt.state==='LOCAL_CANDIDATE_READY'?'PASS':
      ['LOCAL_ATTEMPT_UNKNOWN','LOCAL_ATTEMPT_CANCELLED'].includes(attempt.state)?'UNKNOWN':'FAIL';
    return {status,reason:attempt.state==='LOCAL_CANDIDATE_READY'?undefined:attempt.reason,
      exitCode:attempt.process?.exitCode??null,signal:null,
      timeout:attempt.reason==='ATTEMPT_TIMEOUT_RECONCILED',observedAt:new Date().toISOString(),
      result:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,...attempt},evidenceDir};
  }});
}
