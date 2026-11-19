/** ZCode AgentProvider/1 adapter (v2.1 candidate; #473 bridge from #470).
 * Ports the #470 Windows-proven process machinery verbatim — staged
 * capability probe (VERSION → CREDENTIALS → CANARY), the exact CLI argv
 * contract, noise-tolerant final-JSON parsing, explicit env allowlist,
 * self-enforced timeout with process-tree kill — and reshapes the results
 * into AgentProvider/1 observations ({state,reason,evidence} only).
 *
 * Contract split (AgentProvider/1): the provider owns ONLY its CLI/protocol
 * execution and reporting. Task admission, worktree grants, git candidate
 * commit/finalize/readback, review and Outcome remain solely with the
 * existing selected B/K/Host/C chain — so unlike #470 this adapter never
 * creates a worktree (it requires the Host-admitted worktreePath to exist)
 * and never creates the candidate ref; reconcile only READS
 * refs/tally-zcode-candidates/<attemptId> for honest status reporting.
 *
 * Default-off: nothing registers this provider anywhere; selection is the
 * contract registry's resolve() with an explicit enabledProviderIds entry
 * (the default provider stays codex). There is deliberately NO fallback.
 */
import {spawn} from 'node:child_process';
import {existsSync} from 'node:fs';
import {readFile} from 'node:fs/promises';
import {dirname,join} from 'node:path';
import {defineAgentProvider} from '../../tally-agent-provider/src/index.mjs';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const SEMVER=/^\d+\.\d+\.\d+/;

export const ZCODE_PROVIDER_CONSTANTS=Object.freeze({providerId:'zcode',
  candidateRefPrefix:'refs/tally-zcode-candidates/',
  canaryPrompt:'Reply with exactly: OK',
  defaultTimeoutMs:600000});

/** Builder environment is an explicit allowlist (verbatim port from #470).
 * No ZCODE_* inheritance: the desktop-inherited provider pointers are a
 * proven stale-path trap and the API-key override turns auth failures into
 * silent stalls; ZCode credentials live in ~/.zcode/v2/credentials.json, not
 * the environment. No GitHub/CI credentials either — the builder never
 * pushes; the Host chain finalizes the candidate ref itself. */
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

/** Parse the agent's final JSON from stdout (verbatim port from #470). With
 * --json the turn ends with one object, but noise lines (AI SDK warnings) can
 * precede it — never JSON.parse the whole stream. Unparseable output yields
 * null; the agent output is never evidence of success by itself. */
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
  const paths=(allowedWriteSet??[]).map(x=>`  - ${x}`).join('\n');
  return `Work only in the current disposable Git worktree. Modify or create exactly the allowed paths:\n${paths}\nDo not modify, create or delete any other path. Do not run git commit, git push or any GitHub or network operation. Stop after the file changes are complete.\n\nTask: ${taskPrompt}`;
}

function createDefaultOperations(spawnFn){
  const gitPath=config=>config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
  const git=(config,args,options={})=>{
    const {done}=spawnCapture(spawnFn,gitPath(config),args,
      {env:hostEnvironment(config),...options});
    return done;
  };
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
    // Read-only git readback of the durable attempt record: the candidate ref
    // (created by the Host chain) and worktree occupancy ARE the attempt
    // record; the provider never writes it.
    const ref=ZCODE_PROVIDER_CONSTANTS.candidateRefPrefix+String(config.attemptId)
      .replace(/[^A-Za-z0-9._-]/g,'-');
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
  function runAgentProcess(config){
    // Spawn one bounded, unbounded-by-CLI-timeout agent turn; the adapter
    // owns timeout + tree-kill (the CLI has no built-in timeout). The Host
    // task prompt is wrapped in the #470 write-set fence text; actual
    // enforcement is Host-side (post-hoc, inside the disposable worktree).
    const argv=zcodeArgv(config,{prompt:zcodeTaskPrompt(
      {allowedWriteSet:config.allowedWriteSet,taskPrompt:config.prompt}),
      worktreePath:config.worktreePath,attach:config.attach});
    const {child,done}=spawnCapture(spawnFn,argv[0],argv.slice(1),
      {cwd:config.worktreePath,env:zcodeBuilderEnvironment(config),timeoutMs:0});
    return {child,done,argv};
  }
  return {probeProvider,attemptRecord,runAgentProcess};
}

/** Build the contract-compliant zcode provider. `config` carries the local
 * CLI/repo facts (cliPath, nodePath, gitPath, credentialsPath, repositoryPath,
 * canaryTimeoutMs, ...); `spawnFn`/`operations`/`killTree` are injectable for
 * the fake-process test matrix (same seams as #470). */
export function createZcodeAgentProvider({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree,config={}}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const {done}=spawnCapture(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    const result=await done;
    return result.code===0;
  });
  // Control-plane handle for the single in-flight attempt per attemptId —
  // NOT an execution journal: entries hold only {pid,abort}, are registered
  // at spawn and deleted when the terminal observation is produced. The
  // durable attempt record remains the git candidate ref (read by reconcile).
  const live=new Map();
  async function probe(){
    const probeResult=await ops.probeProvider(config);
    if(probeResult?.available===true)
      return {state:'AVAILABLE',reason:null,
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
          version:probeResult.version??null,canary:probeResult.canary??null}};
    return {state:'UNAVAILABLE',reason:'LOCAL_AGENT_UNAVAILABLE',
      evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,providerProbe:probeResult??null}};
  }
  async function execute(request,context={}){
    const agentLaunch=()=>({agentLaunch:false});
    const notRun=(reason,detail={})=>({state:'FAILED',reason,
      evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
        taskId:request?.taskId??null,attemptId:request?.attemptId??null,
        sourceSha:request?.sourceSha??null,process:null,...agentLaunch(),...detail}});
    if(!SHA.test(request?.sourceSha??'')||!ATTEMPT.test(request?.attemptId??'')||
      typeof request?.prompt!=='string'||!request.prompt||
      typeof request?.worktreePath!=='string'||!request.worktreePath)
      return notRun('ATTEMPT_INPUT_INVALID');
    if(context?.signal?.aborted)
      // Interface adaptation of the AbortSignal contract: a Host signal that
      // was already aborted before dispatch never spawns the CLI and reads
      // CANCELLED (the attempt did not fail — it was cancelled up front).
      return {state:'CANCELLED',reason:'ATTEMPT_CANCELLED',
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
          taskId:request.taskId,attemptId:request.attemptId,
          sourceSha:request.sourceSha,process:null,agentLaunch:false}};
    const probeResult=await ops.probeProvider(config);
    if(probeResult?.available!==true)
      // Staged probe failed ⇒ execute NEVER spawns (auth failures stall
      // silently in the real CLI; unavailability is reported honestly).
      return notRun('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probeResult});
    // The Host supplies an already admitted workspace: never create it.
    if(!existsSync(request.worktreePath))
      return notRun('WORKTREE_ABSENT',{providerProbe:probeResult});
    const running=ops.runAgentProcess({...config,attemptId:request.attemptId,
      sourceSha:request.sourceSha,prompt:request.prompt,
      worktreePath:request.worktreePath,attach:request.attach??config.attach});
    const pid=running.child.pid;
    const controller={cancelled:false};
    const abort=()=>{controller.cancelled=true;
      if(Number.isSafeInteger(pid))terminate(pid).catch(()=>{});};
    context?.signal?.addEventListener('abort',abort,{once:true});
    if(Number.isSafeInteger(pid))
      live.set(request.attemptId,{pid,abort,startedAt:iso(now)});
    const startedAt=iso(now);
    let timedOut=false;
    const timeoutMs=Number.isSafeInteger(request.timeoutMs)&&request.timeoutMs>0?
      request.timeoutMs:(config.timeoutMs??ZCODE_PROVIDER_CONSTANTS.defaultTimeoutMs);
    const timer=setTimeout(()=>{timedOut=true;abort();},timeoutMs);
    timer.unref?.();
    const terminal=await running.done;
    clearTimeout(timer);
    context?.signal?.removeEventListener('abort',abort);
    live.delete(request.attemptId);
    if((timedOut||controller.cancelled)&&terminal.terminatedAt&&Number.isSafeInteger(pid))
      await terminate(pid).catch(()=>{});
    const agentResult=parseZcodeAgentResult(terminal.stdout);
    const processEvidence={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
      exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,
      pid:Number.isSafeInteger(pid)?pid:null,startedAt,
      terminatedAt:iso(now)};
    let state,reason;
    if(timedOut||terminal.spawnError||terminal.probeTimeout){
      state='UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
    }else if(controller.cancelled){state='CANCELLED';reason='ATTEMPT_CANCELLED';
    }else if(terminal.code!==0){
      // Nonzero exit is FAILED from strictness: a healthy turn was once seen
      // exiting 127 after printing its answer; stdout/git stay as evidence
      // for Host reconciliation, but a nonzero exit never reads COMPLETED.
      state='FAILED';reason='ATTEMPT_PROCESS_FAILED';
    }else{state='COMPLETED';reason='AGENT_TURN_COMPLETED';}
    return {state,reason,evidence:{
      provider:ZCODE_PROVIDER_CONSTANTS.providerId,taskId:request.taskId,
      attemptId:request.attemptId,sourceSha:request.sourceSha,agentLaunch:true,
      process:processEvidence,
      agent:agentResult?{sessionId:agentResult.sessionId??null,
        response:String(agentResult.response??'').slice(0,2000),
        usage:agentResult.usage??null}:null,
      stdoutTail:terminal.stdout.slice(-4000),stderrTail:terminal.stderr.slice(-2000),
      providerProbe:probeResult}};
  }
  async function cancel(attempt){
    const slot=live.get(attempt?.attemptId);
    if(!slot)
      return {state:'NOT_RUNNING',reason:null,
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
          attemptId:attempt?.attemptId??null}};
    try{slot.abort();}catch{
      return {state:'UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
          attemptId:attempt.attemptId,pid:slot.pid}};
    }
    return {state:'CANCEL_REQUESTED',reason:null,
      evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,
        attemptId:attempt.attemptId,pid:slot.pid}};
  }
  async function reconcile(attempt){
    const attemptId=attempt?.attemptId;
    if(live.has(attemptId))
      return {state:'RUNNING',reason:null,
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,attemptId,
          live:true,startedAt:live.get(attemptId).startedAt}};
    const record=await ops.attemptRecord({...config,attemptId,
      sourceSha:attempt?.sourceSha??config.sourceSha,
      worktreePath:attempt?.worktreePath??config.worktreePath});
    if(record?.kind==='CANDIDATE')
      return {state:'COMPLETED',reason:'REAL_ZCODE_CANDIDATE_RECORDED',
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,attemptId,
          candidateCommit:record.candidateCommit,candidateTree:record.candidateTree,
          paths:record.paths,ref:record.ref}};
    if(record?.kind==='STALE')
      return {state:'FAILED',reason:'CANDIDATE_STALE',
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,attemptId}};
    if(record?.kind==='WORKTREE_OCCUPIED')
      // Occupied worktree without a ref: possibly running or crashed —
      // honest UNKNOWN, never a blind relaunch (port of the #470 rule).
      return {state:'UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
        evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,attemptId}};
    return {state:'UNKNOWN',reason:'ATTEMPT_RECORD_ABSENT',
      evidence:{provider:ZCODE_PROVIDER_CONSTANTS.providerId,attemptId}};
  }
  return defineAgentProvider({id:ZCODE_PROVIDER_CONSTANTS.providerId,
    contractVersion:1,
    capabilities:['agent-turn','probe-canary','git-ref-reconcile'],
    probe,execute,cancel,reconcile});
}

function iso(now){return new Date(now()).toISOString();}

export {createDefaultOperations};
