import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync} from 'node:fs';
import {mkdir} from 'node:fs/promises';
import {dirname,resolve} from 'node:path';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';

const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const SID=/^S-1-5-21(?:-\d+){4}$/;
const CANDIDATE='experiments/v2/selected-business-build/LOCAL_CODEX_L3.md';
const keyFor=attemptId=>'tally:v2:local-agent-attempt:'+attemptId;
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
  return done;
}
function encodedPowerShell(script){return Buffer.from(script,'utf16le').toString('base64');}
function quotePs(value){return "'"+value.replaceAll("'","''")+"'";}

const sandboxPrefix=['-c','sandbox_mode="workspace-write"','-c',
  'sandbox_workspace_write.network_access=true','sandbox','--'];
function createDefaultOperations(spawnFn){
  const sandbox=(config,command,args,options={})=>short(spawnFn,config.sandboxCliPath,
    [...sandboxPrefix,command,...args],{env:builderEnvironment(config),...options});
  const git=(config,args,options={})=>short(spawnFn,config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe',
    args,{env:builderEnvironment(config),...options});
  async function probeProvider(config){
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
    const remote=await git(config,['-C',worktreePath,'push','--dry-run','origin'],{timeoutMs:30000});
    if(remote.code===0)return {verified:false,check:'REMOTE_WRITE',reason:'BUILDER_CONFINEMENT_UNAVAILABLE'};
    return {verified:true,builderPrincipal:principal,builderSid:config.builderSid,
      hiddenVerifierStatus:'HIDDEN_NOT_QUALIFIED'};
  }
  async function prepareWorktree(config){
    await mkdir(dirname(config.worktreePath),{recursive:true});
    if(existsSync(config.worktreePath))throw safeError('CANDIDATE_WORKTREE_ALREADY_EXISTS');
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const added=await short(spawnFn,gitPath,['-C',config.repositoryPath,'worktree','add','--detach',
      config.worktreePath,config.sourceSha],{timeoutMs:60000});
    if(added.code!==0)throw safeError('CANDIDATE_WORKTREE_UNAVAILABLE');
    const head=await short(spawnFn,gitPath,['-C',config.worktreePath,'rev-parse','HEAD']);
    if(head.code!==0||head.stdout.trim()!==config.sourceSha)throw safeError('CANDIDATE_BASE_MISMATCH');
    for(const [path,permission] of [[dirname(config.worktreePath),'RX'],[config.worktreePath,'M'],
      ...(config.codexHome?[[config.codexHome,'F']]:[])]){
      const acl=await short(spawnFn,'C:\\Windows\\System32\\icacls.exe',[path,'/grant',
        `*${config.builderSid}:(OI)(CI)(${permission})`],{timeoutMs:15000});
      if(acl.code!==0)throw safeError('BUILDER_ACL_UNAVAILABLE');
    }
    return {worktreePath:config.worktreePath,baseSha:head.stdout.trim()};
  }
  async function finalizeCandidate(config){
    const gitPath=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    const run=(args,options={})=>short(spawnFn,gitPath,['-C',config.worktreePath,...args],options);
    const [tracked,untracked]=await Promise.all([
      run(['diff','--name-only',config.sourceSha]),
      run(['ls-files','--others','--exclude-standard','--no-directory'])
    ]);
    if(tracked.code!==0||untracked.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const paths=[...new Set([...tracked.stdout.split(/\r?\n/),...untracked.stdout.split(/\r?\n/)]
      .map(x=>x.trim()).filter(Boolean))];
    if(paths.length!==1||paths[0]!==CANDIDATE)return {writeSetValid:false,paths};
    const added=await run(['add','--',CANDIDATE]);
    if(added.code!==0)throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const tree=await run(['write-tree']);
    if(tree.code!==0||!SHA.test(tree.stdout.trim()))throw safeError('CANDIDATE_TREE_UNAVAILABLE');
    const treeSha=tree.stdout.trim();
    const message=`Tally local candidate ${config.attemptId}`;
    const commit=await run(['commit-tree',treeSha,'-p',config.sourceSha,'-m',message],{
      env:{...process.env,GIT_AUTHOR_NAME:'Tally Host',GIT_AUTHOR_EMAIL:'tally-host@localhost',
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
  return {probeProvider,proveConfinement,prepareWorktree,finalizeCandidate,sandbox};
}

export function createLocalCodexHost({spawnFn=spawn,operations,now=()=>Date.now(),
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
    const base={issueRef:config.issueRef,sourceSha:config.sourceSha,attemptId:config.attemptId,...flags};
    if(!SHA.test(config.sourceSha??'')||!ATTEMPT.test(config.attemptId??'')||
      !SID.test(config.builderSid??''))return {...base,state:'AGENT_NOT_EXECUTED',
      reason:'BUILDER_CONFINEMENT_UNAVAILABLE',agentLaunch:false,replayFixture:false,
      candidateTree:null,candidateCommit:null,process:null};
    let store=await openSqliteCasStore({path:config.hostStorePath,busyTimeoutMs:5000});
    const key=keyFor(config.attemptId);
    try{
      const existing=await store.get(key);
      if(existing){
        const value=existing.value;
        if(value?.phase==='CANDIDATE_READY')return {...base,state:'LOCAL_CANDIDATE_READY',
          reason:'REAL_CODEX_CANDIDATE_RECORDED',agentLaunch:true,replayFixture:false,
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
      const startedAt=iso(now);
      await store.put(key,{phase:'STARTING',attemptId:config.attemptId,sourceSha:config.sourceSha,
        issueRef:config.issueRef,claimGeneration:1,worktreePath:config.worktreePath,
        candidateRelativePath:CANDIDATE,startedAt},{mustBeAbsent:true});
      let revision=0;
      const update=async value=>{const result=await store.put(key,value,{mustMatchRevision:revision});
        revision=result.revision;};
      const notExecuted=async (reason,detail={})=>{
        const current=await store.get(key);revision=current.revision;
        await update({...current.value,phase:'NOT_EXECUTED',reason,...detail,terminatedAt:iso(now)});
        return {...base,state:'AGENT_NOT_EXECUTED',reason,agentLaunch:false,replayFixture:false,
          candidateTree:null,candidateCommit:null,process:null};
      };
      const probe=await ops.probeProvider(config);
      if(!probe.available)return await notExecuted('LOCAL_AGENT_UNAVAILABLE',{providerProbe:probe});
      const worktree=await ops.prepareWorktree(config);
      let current=await store.get(key);revision=current.revision;
      await update({...current.value,worktreePath:worktree.worktreePath,baseSha:worktree.baseSha});
      const confinement=await ops.proveConfinement(config,worktree.worktreePath);
      if(!confinement.verified)return await notExecuted(confinement.reason,{confinement});
      const env=builderEnvironment(config);
      const prompt=`Work only in the current disposable Git worktree. Create exactly ${CANDIDATE} with a short Markdown candidate note containing attempt ${config.attemptId}. Do not modify any other path, do not commit, do not use GitHub or network except the model provider, and stop after writing the file.`;
      const argv=[config.sandboxCliPath,...sandboxPrefix,config.cliPath,'exec',
        '--dangerously-bypass-approvals-and-sandbox','--ignore-user-config','--ignore-rules',
        '--ephemeral','--json','-C',worktree.worktreePath,prompt];
      current=await store.get(key);revision=current.revision;
      await update({...current.value,phase:'PERMITTED',argv,cwd:worktree.worktreePath,
        envAllowlist:Object.keys(env),builderPrincipal:confinement.builderPrincipal,
        builderSid:confinement.builderSid,hiddenVerifierStatus:confinement.hiddenVerifierStatus});
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
      const candidate=await ops.finalizeCandidate(config).catch(error=>({error:error.code}));
      const process={status:terminal.spawnError||terminal.probeTimeout?'UNKNOWN':'EXITED',
        exitCode:Number.isSafeInteger(terminal.code)?terminal.code:null,pid:running.child.pid,
        builderPrincipal:confinement.builderPrincipal,builderSid:confinement.builderSid,
        startedAt,terminatedAt:iso(now)};
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
      return {...base,state,reason,agentLaunch:true,replayFixture:false,
        candidateTree:candidate.candidateTree??null,candidateCommit:candidate.candidateCommit??null,process};
    }finally{await store.close();}
  }
  return Object.freeze({runAttempt,probeProvider:config=>ops.probeProvider(config),
    proveConfinement:(config,worktreePath)=>ops.proveConfinement(config,worktreePath)});
}

export const LOCAL_CODEX_CONSTANTS=Object.freeze({candidateRelativePath:CANDIDATE,
  attemptKeyPrefix:'tally:v2:local-agent-attempt:',builderEnvironment});
