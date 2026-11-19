import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync} from 'node:fs';
import {mkdir} from 'node:fs/promises';
import {dirname,resolve} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const PROVIDER_ID=/^[a-z][a-z0-9-]{0,63}$/;
const SID=/^S-1-5-21(?:-\d+){4}$/;
const CANDIDATE='experiments/v2/selected-business-build/LOCAL_CODEX_L3.md';
const keyFor=attemptId=>'tally:v2:local-agent-attempt:'+attemptId;
const grantKeyFor=(kind,value)=>'tally:v2:local-agent-grant:'+kind+':'+
  createHash('sha256').update(value).digest('hex');
// #264 J05 single-Builder review lane: one stable decision idempotency key
// binds repository/issue/source/attempt/candidate/repair/reviewer identity;
// decision and outcome rows are create-only with exact readback.
const REVIEW_DECISIONS=new Set(['ACCEPT','REWORK','DEFER']);
// Task-required CI contexts for the single-candidate qualification gate: the
// exact-SHA combined status must carry EACH of these terminal-success
// contexts. A single unrelated green context (or missing/pending/failing
// required ones) must never qualify the task CI gate (R103 review P1-03).
const REVIEW_REQUIRED_CI_CONTEXTS=Object.freeze([
  'ci/woodpecker/pr/v2-alm-workspace',
  'ci/woodpecker/pr/v2-candidate-publication',
  'ci/woodpecker/pr/v2-t06l-kernel-admission',
  'ci/woodpecker/pr/v2-t06m-local-agent',
  'ci/woodpecker/pr/v2-t06n-concurrent-agent']);
const reviewDecisionKeyFor=decisionKey=>
  'tally:v2:single-review-decision:'+decisionKey;
const reviewOutcomeKeyFor=decisionKey=>
  'tally:v2:single-task-outcome:'+decisionKey;
const reviewDecisionKeyOf=config=>createHash('sha256')
  .update(JSON.stringify([config.issueRef,config.sourceSha,config.attemptId,
    config.candidateCommit,config.repairCommit,
    config.reviewer?.operatorId??null])).digest('hex');
const safeError=code=>Object.assign(new Error(code),{code});
const iso=now=>new Date(now()).toISOString();

function builderEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  const builderProfile=config.builderProfile??'C:\\Users\\CodexSandboxOnline';
  const path=[dirname(config.cliPath),root+'\\System32','C:\\Program Files\\Git\\cmd'].join(';');
  const env={
    SystemRoot:root,SystemDrive:config.systemDrive??'C:',ComSpec:root+'\\System32\\cmd.exe',
    PATH:path,PATHEXT:'.COM;.EXE;.BAT;.CMD',
    USERPROFILE:builderProfile,HOME:builderProfile,
    APPDATA:builderProfile+'\\AppData\\Roaming',LOCALAPPDATA:builderProfile+'\\AppData\\Local',
    TEMP:config.builderTemp??builderProfile+'\\AppData\\Local\\Temp',
    TMP:config.builderTemp??builderProfile+'\\AppData\\Local\\Temp',
    HTTPS_PROXY:config.modelProxy??'http://127.0.0.1:10808',
    HTTP_PROXY:config.modelProxy??'http://127.0.0.1:10808',
    GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:config.emptyGitConfig,
    SSL_CERT_FILE:config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt',
    CURL_CA_BUNDLE:config.caBundle??'C:\\Program Files\\Git\\mingw64\\etc\\ssl\\certs\\ca-bundle.crt',
    GIT_TERMINAL_PROMPT:'0',GCM_INTERACTIVE:'never',
    GIT_AUTHOR_NAME:'Tally Builder',GIT_AUTHOR_EMAIL:'tally-builder@localhost',
    GIT_COMMITTER_NAME:'Tally Builder',GIT_COMMITTER_EMAIL:'tally-builder@localhost'
  };
  if(config.codexHome)env.CODEX_HOME=config.codexHome;
  return env;
}

function hostEnvironment(config){
  const root=config.systemRoot??'C:\\Windows';
  return {SystemRoot:root,SystemDrive:config.systemDrive??'C:',
    ComSpec:root+'\\System32\\cmd.exe',PATH:root+'\\System32;C:\\Program Files\\Git\\cmd',
    PATHEXT:'.COM;.EXE;.BAT;.CMD',TEMP:root+'\\Temp',TMP:root+'\\Temp',
    GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:config.emptyGitConfig,GIT_TERMINAL_PROMPT:'0'};
}

function spawnCapture(spawnFn,command,args,{cwd,env,input,timeoutMs=15000}={}){
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
    if(input!==undefined){child.stdin?.end(input);}else child.stdin?.end();
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
  if(result.code!==0)console.error('HOST_COMMAND_FAILED',command,args,JSON.stringify(result).slice(0,4000));
  return result;
}
function encodedPowerShell(script){return Buffer.from(script,'utf16le').toString('base64');}
function quotePs(value){return "'"+value.replaceAll("'","''")+"'";}

const sandboxPrefix=['-c','sandbox_mode="workspace-write"','-c',
  'sandbox_workspace_write.network_access=true','sandbox','--'];
export function createDefaultOperations(spawnFn){
  const sandbox=(config,command,args,options={})=>short(spawnFn,config.sandboxCliPath,
    [...sandboxPrefix,command,...args],{env:builderEnvironment(config),...options});
  const git=(config,args,options={})=>short(spawnFn,config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe',
    args,{env:builderEnvironment(config),...options});
  async function probeProvider(config){
    console.error('HOST_PROBE_PROVIDER_START');
    const version=await sandbox(config,config.cliPath,['--version']);
    if(version.code!==0||!/^codex-cli\s+\d+\./.test(version.stdout.trim()))
      return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'VERSION',
        exitCode:version.code,outputMatched:false};
    const login=await sandbox(config,config.cliPath,['login','status'],{timeoutMs:30000});
    if(login.code!==0||!(login.stdout+'\n'+login.stderr).includes('Logged in'))
      return {available:false,reason:'LOCAL_AGENT_UNAVAILABLE',stage:'AUTH',
        exitCode:login.code,outputMatched:false};
    return {available:true,provider:'codex-cli'};
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
    const deniedPaths=[
      ['GH',config.operatorHome+'\\.config\\gh\\hosts.yml',false],
      ['GCM_FILE',config.operatorHome+'\\.git-credentials',false],
      ['OPERATOR_CODEX',config.operatorHome+'\\.codex\\auth.json',false],
      ['SSH_KEYS',config.operatorHome+'\\.ssh',true],
      ['HIDDEN',config.hiddenFixture,false]
    ];
    for(const [check,path,directory] of deniedPaths){
      if(check!=='HIDDEN'&&!existsSync(path))continue;
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
    const probeRef='HEAD:refs/_tally_probe/'+config.attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
    const remote=await git(config,['-C',worktreePath,'push','--dry-run','origin',probeRef],{timeoutMs:30000});
    if(remote.code===0)return {verified:false,check:'REMOTE_WRITE',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    return {verified:true,builderPrincipal:principal,builderSid:config.builderSid,
      hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'};
  }
  async function prepareWorktree(config){
    console.error('HOST_PREPARE_WORKTREE_START',config.worktreePath);
    await mkdir(dirname(config.worktreePath),{recursive:true});
    if(existsSync(config.worktreePath))throw safeError('CANDIDATE_WORKTREE_ALREADY_EXISTS');
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const added=await short(spawnFn,gitPath,['-C',config.repositoryPath,'worktree','add','--detach',
      config.worktreePath,config.sourceSha],{env:hostEnvironment(config),timeoutMs:60000});
    if(added.code!==0)throw safeError('CANDIDATE_WORKTREE_UNAVAILABLE');
    const head=await short(spawnFn,gitPath,['-C',config.worktreePath,'rev-parse','HEAD'],
      {env:hostEnvironment(config)});
    if(head.code!==0||head.stdout.trim()!==config.sourceSha)throw safeError('CANDIDATE_BASE_MISMATCH');
    for(const [path,permission] of [[dirname(config.worktreePath),'RX'],[config.worktreePath,'M'],
      ...(config.codexHome?[[config.codexHome,'F']]:[])]){
      const acl=await short(spawnFn,'C:\\Windows\\System32\\icacls.exe',[path,'/grant',
        `*${config.builderSid}:(OI)(CI)(${permission})`],{env:hostEnvironment(config),timeoutMs:15000});
      if(acl.code!==0)throw safeError('BUILDER_ACL_UNAVAILABLE');
    }
    return {worktreePath:config.worktreePath,baseSha:head.stdout.trim()};
  }
  async function finalizeCandidate(config){
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const run=(args,options={})=>short(spawnFn,gitPath,['-C',config.worktreePath,...args],
      {env:hostEnvironment(config),...options});
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
    const message=`Tally local candidate ${config.attemptId}\n\nRefs: #${issueNumber??'172'}`;
    const commit=await run(['commit-tree',treeSha,'-p',config.sourceSha,'-m',message],{
      env:{...hostEnvironment(config),GIT_AUTHOR_NAME:'Tally Host',GIT_AUTHOR_EMAIL:'tally-host@localhost',
        GIT_COMMITTER_NAME:'Tally Host',GIT_COMMITTER_EMAIL:'tally-host@localhost'}});
    if(commit.code!==0||!SHA.test(commit.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const commitSha=commit.stdout.trim();
    const ref='refs/tally-candidates/'+config.attemptId.replace(/[^A-Za-z0-9._-]/g,'-');
    const updated=await run(['update-ref',ref,commitSha]);
    if(updated.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const readback=await run(['rev-parse',commitSha+'^{tree}']);
    if(readback.code!==0||readback.stdout.trim()!==treeSha)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    return {writeSetValid:true,paths,candidateTree:treeSha,candidateCommit:commitSha,candidateRef:ref};
  }
  async function verifyCandidateReadback(config,candidate){
    // Independent git readback of one exact candidate: commit parent, tree and
    // the single changed path are re-derived from the repository, never from
    // Agent-asserted values. Used by the Reviewer Join only.
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const run=(args,options={})=>short(spawnFn,gitPath,['-C',config.repositoryPath,...args],
      {env:hostEnvironment(config),...options});
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
  async function candidatePathOccupied(config){
    // Sealed-prompt path-occupancy guard (#264 J04 lesson): the Builder prompt
    // says "Create exactly <path>"; if that path is already tracked at the
    // pinned source SHA the attempt would silently overwrite tracked history.
    // Refuse unless an explicit, separately authorized replacement flag is
    // set by the trusted Host configuration — never by the Agent or caller.
    // Returns {occupied:true|false} only when git PROVED the answer; every
    // other nonzero result (invalid ref/repo, launch failure, timeout) is
    // {occupied:null} and the caller must fail closed without spawning — a
    // git error may never read as "not occupied" (R103 review P1-02).
    if(config.candidateReplacementAuthorized===true)return {occupied:false};
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    const check=await short(spawnFn,gitPath,['-C',config.repositoryPath,'cat-file','-e',
      config.sourceSha+':'+candidatePath],{env:hostEnvironment(config)});
    if(check.code===0)return {occupied:true};
    const stderr=typeof check.stderr==='string'?check.stderr:'';
    // git's proven-absent signature for `cat-file -e <ref>:<path>`; any other
    // failure shape (bad ref, bad repository, no git binary, timeout) stays
    // UNKNOWN instead of guessing absence.
    if(/(?:^|\r?\n)fatal: path '.*' does not exist in '.*'(?:\r?\n|$)/.test(stderr))
      return {occupied:false};
    return {occupied:null,detail:stderr.slice(0,400)};
  }
  async function verifyRepairReadback(config){
    // Independent git readback of the repaired candidate PR head: the repair
    // commit must inherit the original candidate (no history rewrite), carry
    // the expected tree and change exactly the frozen single path.
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const run=(args,options={})=>short(spawnFn,gitPath,['-C',config.repositoryPath,...args],
      {env:hostEnvironment(config),...options});
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    const [tree,parent,diff]=await Promise.all([
      run(['rev-parse',config.repairCommit+'^{tree}']),
      run(['rev-parse',config.repairCommit+'^']),
      run(['diff','--name-only',config.sourceSha,config.repairCommit])
    ]);
    if(tree.code!==0||parent.code!==0||diff.code!==0)return {verified:false};
    return {verified:true,repairTree:tree.stdout.trim(),parent:parent.stdout.trim(),
      paths:diff.stdout.split(/\r?\n/).map(x=>x.trim()).filter(Boolean),
      candidatePath};
  }
  return {probeProvider,proveConfinement,prepareWorktree,finalizeCandidate,
    verifyCandidateReadback,candidatePathOccupied,verifyRepairReadback,sandbox};
}

const ACTIVE_PHASES=new Set(['STARTING','PERMITTED','RUNNING']);
const overlapDenied=(base,reason)=>({...base,state:'NOT_ADMITTED',reason,
  claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
  candidateTree:null,candidateCommit:null,process:null});

export function createLocalCodexHost({spawnFn=spawn,operations,now=()=>Date.now(),
  killTree}={}){
  const ops=operations??createDefaultOperations(spawnFn);
  const terminate=killTree??(async pid=>{
    const result=await short(spawnFn,'C:\\Windows\\System32\\taskkill.exe',
      ['/PID',String(pid),'/T','/F'],{timeoutMs:15000});
    return result.code===0;
  });
  async function runAttempt(config){
    const providerId=config.providerId??'codex';
    const agentProvider=config.agentProvider??null;
    const taskId=config.taskId??/\/issues\/([1-9][0-9]*)$/.exec(config.issueRef??'')?.[1];
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:config.issueRef,sourceSha:config.sourceSha,attemptId:config.attemptId,
      ...(providerId==='codex'?{}:{providerId}),...flags};
    const candidatePath=config.candidateRelativePath??CANDIDATE;
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.attemptId??'')||
      !SID.test(config.builderSid??''))return {...base,state:'AGENT_NOT_EXECUTED',
      reason:'BUILDER_CONFINEMENT_UNAVAILABLE',agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,process:null};
    if(!PROVIDER_ID.test(providerId)||(providerId!=='codex'&&!ATTEMPT.test(taskId??''))||
      (providerId!=='codex'&&
        (!agentProvider||agentProvider.id!==providerId||
          typeof agentProvider.probe!=='function'||typeof agentProvider.execute!=='function'||
          typeof agentProvider.cancel!=='function'||typeof agentProvider.reconcile!=='function'))||
      (config.providerEnv!==undefined&&(!config.providerEnv||
        typeof config.providerEnv!=='object'||Array.isArray(config.providerEnv))))
      return {...base,state:'AGENT_NOT_EXECUTED',reason:'PROVIDER_CONFIG_INVALID',
        agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,process:null};
    let store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:5000});
    const key=keyFor(config.attemptId);
    try{
      const existing=await store.get(key);
      if(existing){
        const value=existing.value;
        if((value?.providerId??'codex')!==providerId||
          (value?.sourceSha&&value.sourceSha!==config.sourceSha)||
          (value?.issueRef&&value.issueRef!==config.issueRef))
          return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_IDENTITY_MISMATCH',
            agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
            process:value?.process??null,readOnly:true};
        if(value?.phase==='CANDIDATE_READY')return {...base,state:'LOCAL_CANDIDATE_READY',
          reason:'REAL_CODEX_CANDIDATE_RECORDED',agentLaunch:true,replayFixture:false,
          candidateTree:value.candidateTree,candidateCommit:value.candidateCommit,process:value.process,
          readOnly:true};
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
        const key=grantKeyFor(kind,value);
        try{
          await store.put(key,{phase:'ACTIVE',kind,value,attemptId:config.attemptId},{mustBeAbsent:true});
          grants.push({key,revision:0});return null;
        }catch{await releaseGrants();return overlapDenied(base,reason);}
      };
      // Atomic resource grants close the cross-attempt read/scan/spawn race.
      // They are released once a terminal attempt record exists; a crash with
      // no terminal record retains them for operator reconciliation.
      const worktreeGrant=await acquireGrant('worktree',config.worktreePath,'CANDIDATE_WORKTREE_OVERLAP');
      if(worktreeGrant)return worktreeGrant;
      const writeGrant=await acquireGrant('candidate',candidatePath,'CANDIDATE_WRITE_GRANT_OVERLAP');
      if(writeGrant)return writeGrant;
      // T06n: two Builders may execute concurrently, but never in one worktree
      // or under one write grant. An active record holding the same worktree
      // path or candidate path denies the new attempt before any reservation.
      for(const otherKey of await store.list(LOCAL_CODEX_CONSTANTS.attemptKeyPrefix)){
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
        issueRef:config.issueRef,claimGeneration:1,worktreePath:config.worktreePath,
        candidateRelativePath:candidatePath,startedAt,
        ...(providerId==='codex'?{}:{providerId})},{mustBeAbsent:true});
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
      const probe=agentProvider?
        await agentProvider.probe({env:config.providerEnv??{}}).catch(()=>null):
        await ops.probeProvider(config);
      if(agentProvider?probe?.state!=='AVAILABLE':!probe.available)
        return await notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
      // Pre-spawn path-occupancy refusal (#264 J04 overwrite lesson): refuse
      // before any worktree/ACL/spawn side effect, never overwrite tracked
      // history through the sealed "Create exactly <path>" prompt. Both a
      // PROVEN occupied path and an UNPROVEN occupancy (git error, bad ref,
      // bad repository, launch failure, timeout) refuse without spawning;
      // only a proven-absent path (or the separately authorized replacement)
      // may proceed (R103 review P1-02 fail-closed repair).
      if(ops.candidatePathOccupied){
        const occupancy=await ops.candidatePathOccupied(config);
        if(occupancy?.occupied===true)
          return await notExecuted('CANDIDATE_PATH_OCCUPIED');
        if(occupancy==null||typeof occupancy!=='object'||occupancy.occupied==null)
          return await notExecuted('CANDIDATE_PATH_OCCUPANCY_UNKNOWN',
            {occupancyDetail:occupancy?.detail??null});
      }
      const worktree=await ops.prepareWorktree(config);
      let current=await store.get(key);revision=current.revision;
      await update({...current.value,worktreePath:worktree.worktreePath,baseSha:worktree.baseSha});
      const confinement=await ops.proveConfinement(config,worktree.worktreePath);
      if(!confinement.verified)return await notExecuted(confinement.reason,{confinement});
      const env=builderEnvironment(config);
      const prompt=`Work only in the current disposable Git worktree. Create exactly ${candidatePath} with a short Markdown candidate note containing attempt ${config.attemptId}. Do not modify any other path, do not commit, do not use GitHub or network except the model provider, and stop after writing the file.`;
      const providerEnv=config.providerEnv??{};
      const argv=agentProvider?['AgentProvider/1',providerId]:
        [config.sandboxCliPath,...sandboxPrefix,config.cliPath,'exec',
          '--dangerously-bypass-approvals-and-sandbox','--ignore-user-config','--ignore-rules',
          '--ephemeral','--json','-C',worktree.worktreePath,prompt];
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase:'PERMITTED',argv,cwd:worktree.worktreePath,
        envAllowlist:Object.keys(agentProvider?providerEnv:env),
        builderPrincipal:agentProvider?null:confinement.builderPrincipal,
        builderSid:agentProvider?null:confinement.builderSid,
        hiddenVerifierStatus:confinement.hiddenVerifierStatus});
      if(agentProvider){
        // AgentProvider/1 is ONLY the process-execution substitution point.
        // Claim/FSW admission, the CAS row/grants, worktree, candidate tree,
        // write-set fence and git readback above/below remain this one Host.
        const controller=new AbortController();
        let timedOut=false,cancelled=false;
        const abort=()=>{cancelled=true;controller.abort();};
        if(config.signal?.aborted)abort();
        else config.signal?.addEventListener('abort',abort,{once:true});
        const timeoutMs=Number.isSafeInteger(config.timeoutMs)&&config.timeoutMs>0?
          config.timeoutMs:300000;
        const timer=setTimeout(()=>{timedOut=true;controller.abort();},timeoutMs);
        timer.unref?.();
        const providerStartedAt=iso(now);
        current=await store.get(key);revision=current.revision;
        await update({...current.value,phase:'RUNNING',process:{status:'RUNNING',pid:null,
          builderPrincipal:null,builderSid:null,startedAt:providerStartedAt}});
        let hardTimer;
        const hardBound=new Promise(resolve=>{
          hardTimer=setTimeout(()=>resolve(null),timeoutMs+60000);
          hardTimer.unref?.();
        });
        let observation;
        try{
          observation=await Promise.race([agentProvider.execute(
            {taskId,attemptId:config.attemptId,sourceSha:config.sourceSha,
              prompt,worktreePath:worktree.worktreePath,timeoutMs},
            {signal:controller.signal,env:providerEnv}),hardBound]).catch(()=>null);
        }finally{
          clearTimeout(timer);clearTimeout(hardTimer);
          config.signal?.removeEventListener('abort',abort);
        }
        if(!observation||!['COMPLETED','FAILED','CANCELLED','UNKNOWN'].includes(observation.state)){
          const reason=observation?'PROVIDER_OBSERVATION_INVALID':
            timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';
          if(!observation)await agentProvider.cancel({taskId,attemptId:config.attemptId,
            sourceSha:config.sourceSha,providerId}).catch(()=>null);
          const process={status:'UNKNOWN',exitCode:null,pid:null,builderPrincipal:null,
            builderSid:null,startedAt:providerStartedAt,terminatedAt:iso(now)};
          current=await store.get(key);revision=current.revision;
          await update({...current.value,phase:'UNKNOWN',reason,process,
            providerObservation:observation?{state:observation.state??null,
              reason:observation.reason??null}:null,terminatedAt:process.terminatedAt});
          // A provider that never settled may still own a live process. Keep
          // the CAS resource grants for reconciliation; never open an overlap.
          if(observation)await releaseGrants();
          return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason,agentLaunch:true,
            replayFixture:false,candidateTree:null,candidateCommit:null,process};
        }
        const evidence=observation.evidence??{};
        const processEvidence=evidence.process??evidence;
        const launched=observation.state==='COMPLETED'||
          (evidence.agentLaunch!==false&&evidence.spawned!==false);
        if(!launched){
          const reason=observation.state==='CANCELLED'?'ATTEMPT_CANCELLED':
            'LOCAL_AGENT_UNAVAILABLE';
          current=await store.get(key);revision=current.revision;
          await update({...current.value,phase:'NOT_EXECUTED',reason,process:null,
            providerObservation:{state:observation.state,reason:observation.reason??null},
            terminatedAt:iso(now)});
          await releaseGrants();
          return {...base,state:'AGENT_NOT_EXECUTED',reason,agentLaunch:false,
            replayFixture:false,candidateTree:null,candidateCommit:null,process:null};
        }
        const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
        const process={status:observation.state==='UNKNOWN'?'UNKNOWN':'EXITED',
          // AgentProvider/1 COMPLETED is the normalized successful-process
          // observation. Some adapters (Pi) intentionally omit optional PID /
          // exitCode evidence; map that contract state to the sealed EXITED/0
          // shape instead of rejecting an otherwise valid Host-read-back
          // candidate. This is process evidence only, never Task authority.
          exitCode:Number.isSafeInteger(processEvidence.exitCode)?processEvidence.exitCode:
            observation.state==='COMPLETED'?0:null,
          pid:Number.isSafeInteger(processEvidence.pid)?processEvidence.pid:null,
          builderPrincipal:null,builderSid:null,startedAt:providerStartedAt,
          terminatedAt:iso(now)};
        let state,reason,phase;
        if(timedOut||observation.state==='UNKNOWN'){
          state='LOCAL_ATTEMPT_UNKNOWN';
          reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';phase='UNKNOWN';
        }else if(cancelled||observation.state==='CANCELLED'){
          state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';phase='CANCELLED';
        }else if(!candidate.writeSetValid&&candidate.paths?.length){
          state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';phase='FAILED';
        }else if(observation.state==='FAILED'){
          state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';phase='FAILED';
        }else if(!candidate.writeSetValid){
          state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';phase='FAILED';
        }else{
          state='LOCAL_CANDIDATE_READY';reason='REAL_CODEX_CANDIDATE_RECORDED';
          phase='CANDIDATE_READY';
        }
        current=await store.get(key);revision=current.revision;
        await update({...current.value,phase,reason,process,
          providerObservation:{state:observation.state,reason:observation.reason??null},
          candidateTree:candidate.candidateTree??null,
          candidateCommit:candidate.candidateCommit??null,candidateRef:candidate.candidateRef??null,
          terminatedAt:process.terminatedAt});
        await releaseGrants();
        return {...base,state,reason,agentLaunch:true,replayFixture:false,
          candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,
          process};
      }
      const running=spawnCapture(spawnFn,argv[0],argv.slice(1),{cwd:worktree.worktreePath,env,
        timeoutMs:0});
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
      if((timedOut||cancelled)&&terminal.terminatedAt){
        await terminate(running.child.pid);
      }
      // Overlap evidence must use the ACTUAL process lifetime: startedAt is
      // the spawn-ack time captured by spawnCapture, not the earlier attempt
      // reservation/probe time, which would overstate the overlap window.
      const processStartedAt=new Date(terminal.startedAt).toISOString();
      const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
        builderPrincipal:confinement.builderPrincipal,builderSid:confinement.builderSid,
        startedAt:processStartedAt,terminatedAt:iso(now)};
      let state,reason,phase;
      if(timedOut||terminal.spawnError||terminal.probeTimeout){
        state='LOCAL_ATTEMPT_UNKNOWN';reason=timedOut?'ATTEMPT_TIMEOUT_RECONCILED':'ATTEMPT_STATUS_UNKNOWN';phase='UNKNOWN';
      }else if(cancelled){state='LOCAL_ATTEMPT_CANCELLED';reason='ATTEMPT_CANCELLED';phase='CANCELLED';
      }else if(!candidate.writeSetValid){
        state='LOCAL_ATTEMPT_FAILED';reason=candidate.error??'WRITE_SET_VIOLATION';phase='FAILED';
      }else if(terminal.code!==0){state='LOCAL_ATTEMPT_FAILED';reason='ATTEMPT_PROCESS_FAILED';phase='FAILED';
      }else{state='LOCAL_CANDIDATE_READY';reason='REAL_CODEX_CANDIDATE_RECORDED';phase='CANDIDATE_READY';}
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase,reason,process,candidateTree:candidate.candidateTree??null,
        candidateCommit:candidate.candidateCommit??null,candidateRef:candidate.candidateRef??null,
        terminatedAt:process.terminatedAt});
      await releaseGrants();
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,process};
    }finally{await store.close();}
  }
  // T06n Reviewer Join: consume durable Builder attempt records plus an
  // independent git readback. The Reviewer never launches a process, never
  // writes source and never upgrades candidates into Task/Release PASS.
  async function joinReviewedCandidates(config){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:config.issueRef,sourceSha:config.sourceSha,joinId:config.joinId,...flags};
    const joinKey=LOCAL_CODEX_CONSTANTS.joinKeyPrefix+config.joinId;
    const denied=(reason)=>({...base,state:'NOT_ADMITTED',reason,claimAccepted:false,
      admitted:false,agentLaunch:false,replayFixture:false,joinedAt:null,candidates:[]});
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.joinId??'')||
      !Array.isArray(config.candidates)||config.candidates.length<2)
      return denied('JOIN_INPUT_INVALID');
    const seen={attemptId:new Set(),path:new Set(),worktree:new Set(),ref:new Set()};
    let store=null;
    try{
      store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:5000});
      const existing=await store.get(joinKey);
      if(existing?.value?.phase==='JOINED'){
        if(existing.value.sourceSha!==config.sourceSha)return denied('CANDIDATE_STALE');
        const v=existing.value;
        return {...base,state:'REVIEW_JOINED',reason:'REVIEWED_CANDIDATES_VERIFIED',
          agentLaunch:false,replayFixture:false,joinedAt:v.joinedAt,candidates:v.candidates};
      }
      const joined=[];
      for(const candidate of config.candidates){
        if(!ATTEMPT.test(candidate?.attemptId??'')||!SHA.test(candidate?.candidateCommit??'')||
          !SHA.test(candidate?.candidateTree??''))return denied('JOIN_INPUT_INVALID');
        const record=(await store.get(keyFor(candidate.attemptId)))?.value;
        if(!record)return denied('CANDIDATE_NOT_FOUND');
        if(record.phase!=='CANDIDATE_READY')return denied('CANDIDATE_NOT_READY');
        if(record.sourceSha!==config.sourceSha)return denied('CANDIDATE_STALE');
        if(record.candidateCommit!==candidate.candidateCommit||
          record.candidateTree!==candidate.candidateTree)return denied('CANDIDATE_STALE');
        // Disjointness is computed from the durable record, never from
        // caller-asserted values: attempt, path, worktree and ref must all
        // be distinct across the reviewed candidate set.
        const recordFields={attemptId:record.attemptId,
          path:record.candidateRelativePath??CANDIDATE,worktree:record.worktreePath,
          ref:record.candidateRef};
        for(const [field,value] of Object.entries(recordFields)){
          if(value===undefined||value===null)continue;
          if(seen[field].has(value))return denied('CANDIDATE_WRITESET_OVERLAP');
          seen[field].add(value);
        }
        const readback=await ops.verifyCandidateReadback(config,record);
        const candidatePath=record.candidateRelativePath??CANDIDATE;
        if(!readback.verified||readback.candidateTree!==record.candidateTree||
          readback.parent!==config.sourceSha||readback.paths.length!==1||
          readback.paths[0]!==candidatePath)return denied('CANDIDATE_READBACK_MISMATCH');
        joined.push(Object.freeze({attemptId:record.attemptId,issueRef:record.issueRef,
          candidateRelativePath:candidatePath,candidateTree:record.candidateTree,
          candidateCommit:record.candidateCommit,candidateRef:record.candidateRef??null,
          startedAt:record.startedAt??null,terminatedAt:record.process?.terminatedAt??null}));
      }
      const joinedAt=iso(now);
      await store.put(joinKey,{phase:'JOINED',joinId:config.joinId,sourceSha:config.sourceSha,
        issueRef:config.issueRef,candidates:joined,joinedAt},{mustBeAbsent:true});
      return {...base,state:'REVIEW_JOINED',reason:'REVIEWED_CANDIDATES_VERIFIED',
        agentLaunch:false,replayFixture:false,joinedAt,candidates:joined};
    }finally{if(store)await store.close();}
  }
  // #264 J05 — single-Builder ReviewDecision + minimal qualified Task Outcome.
  // The Reviewer never launches a process and never writes source. Every fact
  // is re-derived: the Builder attempt record from the durable Builder CAS
  // store, candidate/repair provenance from independent git readback, the PR
  // head and exact-SHA CI from live GitHub reads. The decision row is
  // create-only under one stable idempotency key with exact readback; a lost
  // or ambiguous CAS ACK stays UNKNOWN for read-only reconciliation, never a
  // blind second write. Task PASS requires the full single-candidate
  // qualification chain AND an ACCEPT decision; REWORK/DEFER and every
  // denial keep taskPass=false. This lane never touches the T06N Join.
  async function reviewSingleCandidate(config){
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:config.issueRef,sourceSha:config.sourceSha,
      decisionId:config.decisionId,...flags};
    const denied=(reason,extra={})=>({...base,state:'NOT_ADMITTED',reason,
      claimAccepted:false,admitted:false,agentLaunch:false,replayFixture:false,
      decision:null,decisionRecord:null,taskOutcome:null,...extra});
    const unknown=(reason,extra={})=>({...base,state:'REVIEW_DECISION_UNKNOWN',reason,
      agentLaunch:false,replayFixture:false,reconciliationRequired:true,
      decision:null,decisionRecord:null,taskOutcome:null,...extra});
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.decisionId??'')||
      !ATTEMPT.test(config.attemptId??'')||!SHA.test(config.candidateCommit??'')||
      !SHA.test(config.candidateTree??'')||!SHA.test(config.repairCommit??'')||
      !SHA.test(config.repairTree??'')||!REVIEW_DECISIONS.has(config.decision??'')||
      !Number.isSafeInteger(config.prNumber)||config.prNumber<1)
      return denied('REVIEW_INPUT_INVALID');
    let store=null;
    const decisionKey=reviewDecisionKeyOf(config);
    try{
      // Gate 1 — the Builder's own durable attempt record: CANDIDATE_READY at
      // the exact pinned source, with the exact candidate commit and tree.
      let builderStore=null;
      try{
        builderStore=await openSqliteCasStore({path:config.builderStorePath,
          busyTimeoutMs:5000});
      }catch{return denied('BUILDER_STORE_UNAVAILABLE');}
      let builderRow=null;
      try{
        builderRow=await builderStore.get(keyFor(config.attemptId));
      }catch{builderRow=null;}
      finally{await builderStore.close();}
      const record=builderRow?.value;
      if(!record)return denied('BUILDER_RECORD_NOT_FOUND');
      if(record.phase!=='CANDIDATE_READY')return denied('BUILDER_RECORD_NOT_READY');
      if(record.sourceSha!==config.sourceSha)return denied('BUILDER_RECORD_STALE');
      if(record.candidateCommit!==config.candidateCommit||
        record.candidateTree!==config.candidateTree)
        return denied('BUILDER_RECORD_MISMATCH');
      const candidatePath=record.candidateRelativePath??CANDIDATE;
      // Gate 2 — independent git readback: candidate parent/tree/single path,
      // then the repair commit inheriting the candidate with no history
      // rewrite and the same single changed path.
      const candidate=await ops.verifyCandidateReadback(
        {...config,sourceSha:config.sourceSha},{candidateCommit:config.candidateCommit});
      if(!candidate.verified||candidate.candidateTree!==config.candidateTree||
        candidate.parent!==config.sourceSha||candidate.paths.length!==1||
        candidate.paths[0]!==candidatePath)
        return denied('CANDIDATE_READBACK_MISMATCH');
      const repair=await ops.verifyRepairReadback(
        {...config,candidateRelativePath:candidatePath});
      if(!repair.verified||repair.repairTree!==config.repairTree||
        repair.parent!==config.candidateCommit||repair.paths.length!==1||
        repair.paths[0]!==candidatePath)
        return denied('REPAIR_READBACK_MISMATCH');
      // Gate 3 — live GitHub facts: the candidate PR head must still be the
      // reviewed repair commit on the selected base, and the exact-SHA CI
      // must be green at that commit (never at any other SHA).
      let pull=null,status=null;
      try{
        pull=await config.githubReads?.readPullRequest?.(config.prNumber);
        status=await config.githubReads?.readCombinedStatus?.(config.repairCommit);
      }catch{return denied('GITHUB_READBACK_UNAVAILABLE');}
      if(!pull||pull.state!=='open'||pull.headSha!==config.repairCommit||
        pull.baseRef!=='version/v2')
        return denied('PR_READBACK_MISMATCH');
      if(!status||status.state!=='success')
        return denied('CI_EXACT_SHA_NOT_VERIFIED');
      // R103 review P1-03: one unrelated green context must never qualify
      // this gate. Every task-required named context must be present and
      // terminal-success at the exact repair SHA; missing, pending, failing
      // or unrelated-only contexts all deny with the observed evidence.
      const observedContexts=new Map(Array.isArray(status.contexts)?
        status.contexts.filter(x=>x&&typeof x.context==='string')
          .map(x=>[x.context,x.state]):[]);
      const ciMissingContexts=[],ciNotTerminalContexts=[];
      for(const context of REVIEW_REQUIRED_CI_CONTEXTS){
        const state=observedContexts.get(context);
        if(state===undefined)ciMissingContexts.push(context);
        else if(state!=='success')ciNotTerminalContexts.push({context,state});
      }
      if(ciMissingContexts.length||ciNotTerminalContexts.length)
        return denied('CI_EXACT_SHA_NOT_VERIFIED',{ciMissingContexts,
          ciNotTerminalContexts,observedContexts:[...observedContexts]});
      store=await openSqliteCasStore({path:config.decisionStorePath,busyTimeoutMs:5000});
      const decisionKeyFull=reviewDecisionKeyFor(decisionKey);
      const decidedAt=iso(now);
      const decisionValue={schema:'tally/single-review-decision/v1',
        phase:'REVIEW_DECISION_RECORDED',decision:config.decision,
        repository:'kaicreator-mm/tally',issueRef:config.issueRef,
        sourceSha:config.sourceSha,attemptId:config.attemptId,
        candidateCommit:config.candidateCommit,candidateTree:config.candidateTree,
        candidateRelativePath:candidatePath,repairCommit:config.repairCommit,
        repairTree:config.repairTree,prNumber:config.prNumber,
        reviewer:Object.freeze({operatorId:config.reviewer?.operatorId??null,
          writerId:config.reviewer?.writerId??null,
          writerLogin:config.reviewer?.writerLogin??null,
          claimRef:config.reviewer?.claimRef??null,
          commentId:config.reviewer?.commentId??null}),
        builder:Object.freeze({operatorId:config.builder?.operatorId??null,
          writerId:config.builder?.writerId??null}),
        reviewRef:config.reviewRef??null,decidedAt};
      let recorded;
      const existing=await store.get(decisionKeyFull);
      if(existing){
        const v=existing.value;
        if(v?.phase!=='REVIEW_DECISION_RECORDED'||
          v.decision!==decisionValue.decision||
          v.sourceSha!==decisionValue.sourceSha||
          v.candidateCommit!==decisionValue.candidateCommit||
          v.repairCommit!==decisionValue.repairCommit||
          v.reviewer?.operatorId!==decisionValue.reviewer.operatorId)
          return denied('REVIEW_DECISION_CONFLICT');
        recorded={key:decisionKeyFull,revision:existing.revision,
          value:v,alreadyRecorded:true};
      }else{
        let reservation;
        try{
          reservation=await store.put(decisionKeyFull,decisionValue,{mustBeAbsent:true});
        }catch(error){
          if(error&&error.message==='CAS_EXISTS'){
            const reread=await store.get(decisionKeyFull);
            const v=reread?.value;
            if(v?.phase==='REVIEW_DECISION_RECORDED'&&
              v.decision===decisionValue.decision&&
              v.reviewer?.operatorId===decisionValue.reviewer.operatorId)
              recorded={key:decisionKeyFull,revision:reread.revision,value:v,
                alreadyRecorded:true};
            else return denied('REVIEW_DECISION_CONFLICT');
          }else return unknown('REVIEW_DECISION_ACK_LOST',
            {decisionKey:decisionKeyFull,
             retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_SECOND_WRITE'});
        }
        if(!recorded){
          if(!Number.isSafeInteger(reservation?.revision))
            return unknown('REVIEW_DECISION_ACK_LOST',{decisionKey:decisionKeyFull});
          const proof=await store.get(decisionKeyFull);
          if(proof?.revision!==reservation.revision||
            proof?.value?.phase!=='REVIEW_DECISION_RECORDED'||
            proof.value.decision!==decisionValue.decision||
            proof.value.repairCommit!==decisionValue.repairCommit)
            return unknown('REVIEW_DECISION_READBACK_UNVERIFIED',
              {decisionKey:decisionKeyFull});
          recorded={key:decisionKeyFull,revision:proof.revision,value:proof.value,
            alreadyRecorded:false};
        }
      }
      const decisionRecord=Object.freeze({decisionKey:decisionKeyFull,
        revision:recorded.revision,decision:recorded.value.decision,
        decidedAt:recorded.value.decidedAt,alreadyRecorded:recorded.alreadyRecorded});
      // Gate 4 — qualified Task Outcome: ONLY an ACCEPT decision with every
      // gate above verified may publish; REWORK/DEFER hold publication.
      if(config.decision!=='ACCEPT')
        return {...base,state:'REVIEW_DECISION_RECORDED',
          reason:recorded.alreadyRecorded?
            'REVIEW_DECISION_ALREADY_RECORDED':'REVIEW_DECISION_CAS_READBACK_VERIFIED',
          agentLaunch:false,replayFixture:false,decision:config.decision,
          decisionRecord,taskOutcome:Object.freeze({status:'HOLD',
            reason:'DECISION_'+config.decision+'_NOT_TASK_PASS',publication:'HOLD',
            taskPass:false,releasePass:false})};
      const outcomeKeyFull=reviewOutcomeKeyFor(decisionKey);
      const outcomeValue={schema:'tally/single-task-outcome/v1',
        phase:'QUALIFIED_TASK_OUTCOME_PUBLISHED',repository:'kaicreator-mm/tally',
        issueRef:config.issueRef,sourceSha:config.sourceSha,
        attemptId:config.attemptId,candidateCommit:config.candidateCommit,
        candidateTree:config.candidateTree,repairCommit:config.repairCommit,
        repairTree:config.repairTree,prNumber:config.prNumber,
        decision:{decision:config.decision,decisionKey:decisionKeyFull,
          revision:recorded.revision,decidedAt:recorded.value.decidedAt,
          reviewRef:config.reviewRef??null},
        reviewer:decisionValue.reviewer,builder:decisionValue.builder,
        ci:{commit:config.repairCommit,combinedState:'success',
          contextCount:status.totalCount},
        publisherWriterId:config.reviewer?.writerId??null,publishedAt:iso(now),
        distinctFrom:Object.freeze(['DISPATCH_CLAIMED_COMMENT_ACK',
          'NATIVE_ISSUE_CLOSED','NATIVE_DONE_LABEL','JOURNEY_PROJECTION',
          'PR466_INDEPENDENT_SOURCE_REVIEW'])};
      let outcome;
      const existingOutcome=await store.get(outcomeKeyFull);
      if(existingOutcome){
        const v=existingOutcome.value;
        if(v?.phase!=='QUALIFIED_TASK_OUTCOME_PUBLISHED'||
          v.decision?.decisionKey!==decisionKeyFull)
          return {...base,state:'REVIEW_DECISION_RECORDED',
            reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',
            agentLaunch:false,replayFixture:false,decision:config.decision,
            decisionRecord,taskOutcome:Object.freeze({status:'HOLD',
              reason:'OUTCOME_ROW_CONFLICT',publication:'HOLD',
              taskPass:false,releasePass:false})};
        outcome={key:outcomeKeyFull,revision:existingOutcome.revision,
          alreadyPublished:true};
      }else{
        let reservation;
        try{reservation=await store.put(outcomeKeyFull,outcomeValue,{mustBeAbsent:true});}
        catch(error){
          if(error&&error.message==='CAS_EXISTS'){
            const reread=await store.get(outcomeKeyFull);
            if(reread?.value?.phase==='QUALIFIED_TASK_OUTCOME_PUBLISHED')
              outcome={key:outcomeKeyFull,revision:reread.revision,alreadyPublished:true};
            else return {...base,state:'REVIEW_DECISION_RECORDED',
              reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',agentLaunch:false,
              replayFixture:false,decision:config.decision,decisionRecord,
              taskOutcome:Object.freeze({status:'HOLD',reason:'OUTCOME_ROW_CONFLICT',
                publication:'HOLD',taskPass:false,releasePass:false})};
          }else return {...base,state:'REVIEW_DECISION_RECORDED',
            reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',agentLaunch:false,
            replayFixture:false,decision:config.decision,decisionRecord,
            taskOutcome:Object.freeze({status:'UNKNOWN',reason:'OUTCOME_ACK_LOST',
              publication:'HOLD',reconciliationRequired:true,taskPass:false,
              releasePass:false,
              retryPolicy:'READ_BACK_BEFORE_RETRY_NO_BLIND_SECOND_PUBLISH'})};
        }
        if(!outcome){
          if(!Number.isSafeInteger(reservation?.revision))
            return {...base,state:'REVIEW_DECISION_RECORDED',
              reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',agentLaunch:false,
              replayFixture:false,decision:config.decision,decisionRecord,
              taskOutcome:Object.freeze({status:'UNKNOWN',reason:'OUTCOME_ACK_LOST',
                publication:'HOLD',reconciliationRequired:true,taskPass:false,
                releasePass:false})};
          const proof=await store.get(outcomeKeyFull);
          if(proof?.revision!==reservation.revision||
            proof?.value?.phase!=='QUALIFIED_TASK_OUTCOME_PUBLISHED'||
            proof.value.sourceSha!==config.sourceSha||
            proof.value.repairCommit!==config.repairCommit)
            return {...base,state:'REVIEW_DECISION_RECORDED',
              reason:'REVIEW_DECISION_CAS_READBACK_VERIFIED',agentLaunch:false,
              replayFixture:false,decision:config.decision,decisionRecord,
              taskOutcome:Object.freeze({status:'UNKNOWN',
                reason:'OUTCOME_READBACK_UNVERIFIED',publication:'HOLD',
                reconciliationRequired:true,taskPass:false,releasePass:false})};
          outcome={key:outcomeKeyFull,revision:proof.revision,alreadyPublished:false};
        }
      }
      return {...base,state:'REVIEW_DECISION_RECORDED',
        reason:recorded.alreadyRecorded?
          'REVIEW_DECISION_ALREADY_RECORDED':'REVIEW_DECISION_CAS_READBACK_VERIFIED',
        agentLaunch:false,replayFixture:false,decision:config.decision,
        decisionRecord,
        // Top-level Task PASS exists ONLY together with the published
        // qualified outcome; the decision alone never passes the task.
        taskPass:true,releasePass:false,
        taskOutcome:Object.freeze({
          status:'QUALIFIED_TASK_OUTCOME_PUBLISHED',
          reason:'SINGLE_CANDIDATE_FULL_CHAIN_CAS_READBACK_VERIFIED',
          outcomeKey:outcomeKeyFull,revision:outcome.revision,
          alreadyPublished:outcome.alreadyPublished,publication:'PUBLISHED',
          taskPass:true,releasePass:false})};
    }catch{return unknown('REVIEW_DECISION_ACK_LOST');}
    finally{if(store)await store.close();}
  }
  /** #473 R107 repair 1 — READ-ONLY attempt-record reconciliation port.
   * Mirrors runAttempt's durable-replay branch exactly (same state
   * vocabulary and field shapes) but can never create, launch, mutate or
   * grant anything: it opens the host store, reads the one attempt row and
   * closes. A missing row is the honest read-only answer UNKNOWN — adapters
   * must use this for reconcile instead of runAttempt, which can still
   * launch when no durable row exists (R106 ABI review P1). */
  async function readAttemptRecord(config){
    const providerId=config.providerId??'codex';
    const flags={claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      taskPass:false,releasePass:false};
    const base={issueRef:config.issueRef,sourceSha:config.sourceSha,
      attemptId:config.attemptId,...(providerId==='codex'?{}:{providerId}),...flags};
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.attemptId??'')||
      !PROVIDER_ID.test(providerId))
      return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_INPUT_INVALID',
        agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
        process:null,readOnly:true};
    let db;
    try{
      // better-sqlite3 READONLY + fileMustExist: unlike openSqliteCasStore,
      // this cannot create a missing DB, install schema, enable WAL or write
      // any attempt/grant row. Reconcile uses the same v8_host_docs grammar,
      // but strictly as a read-only durable query (R108 independent P1).
      const Database=(await import('better-sqlite3')).default;
      db=new Database(config.hostStorePath,{readonly:true,fileMustExist:true});
      db.pragma('busy_timeout = 5000');
    }catch{
      return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_RECORD_STORE_UNAVAILABLE',
        agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
        process:null,readOnly:true};
    }
    try{
      const existing=db.prepare('SELECT revision, value FROM v8_host_docs WHERE key = ?')
        .get(keyFor(config.attemptId));
      if(!existing)
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_RECORD_NOT_FOUND',
          agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
          process:null,readOnly:true};
      const value=JSON.parse(existing.value);
      if((value?.providerId??'codex')!==providerId||
        value?.sourceSha!==config.sourceSha||value?.issueRef!==config.issueRef)
        return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_IDENTITY_MISMATCH',
          agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
          process:value?.process??null,readOnly:true};
      if(value?.phase==='CANDIDATE_READY')
        return {...base,state:'LOCAL_CANDIDATE_READY',
          reason:'REAL_CODEX_CANDIDATE_RECORDED',agentLaunch:true,replayFixture:false,
          candidateTree:value.candidateTree,candidateCommit:value.candidateCommit,
          process:value.process,readOnly:true};
      if(value?.phase==='NOT_EXECUTED')
        return {...base,state:'AGENT_NOT_EXECUTED',reason:value.reason,agentLaunch:false,
          replayFixture:false,candidateTree:null,candidateCommit:null,process:null,
          readOnly:true};
      return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_STATUS_UNKNOWN',
        agentLaunch:true,replayFixture:false,candidateTree:value?.candidateTree??null,
        candidateCommit:value?.candidateCommit??null,
        process:value?.process??{status:'UNKNOWN',exitCode:null,pid:null,
          builderPrincipal:config.builderPrincipal,builderSid:config.builderSid,
          startedAt:null,terminatedAt:null},readOnly:true};
    }catch{
      return {...base,state:'LOCAL_ATTEMPT_UNKNOWN',reason:'ATTEMPT_RECORD_READ_FAILED',
        agentLaunch:false,replayFixture:false,candidateTree:null,candidateCommit:null,
        process:null,readOnly:true};
    }finally{try{db.close();}catch{}}
  }
  return Object.freeze({runAttempt,joinReviewedCandidates,reviewSingleCandidate,
    readAttemptRecord,
    probeProvider:config=>ops.probeProvider(config),
    proveConfinement:(config,worktreePath)=>ops.proveConfinement(config,worktreePath)});
}

export const LOCAL_CODEX_CONSTANTS=Object.freeze({candidateRelativePath:CANDIDATE,
  attemptKeyPrefix:'tally:v2:local-agent-attempt:',joinKeyPrefix:'tally:v2:reviewer-join:',
  reviewDecisionKeyPrefix:'tally:v2:single-review-decision:',
  reviewOutcomeKeyPrefix:'tally:v2:single-task-outcome:',
  reviewDecisions:Object.freeze(['ACCEPT','REWORK','DEFER']),
  builderEnvironment});
