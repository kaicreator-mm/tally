/** J04–J06 controller: invokes existing Windows e2e and reads its real result. */
import {spawn} from 'node:child_process';
import {readFile,stat} from 'node:fs/promises';
import {join} from 'node:path';
import {createHash} from 'node:crypto';

const SHA=/^[0-9a-f]{40}$/;
const sha256=value=>createHash('sha256').update(value).digest('hex');
const CLAIM_SINGLE={sourceSha:'d6b32929b481006de4e848eb772fe7dd96c2547c',
  dispatchId:'dispatch:92:alm-write:20261031:1',operatorId:'zcode:i92:alm-write:1',operatorKind:'other',
  writerId:'tally-host:win32:i92:write:1',writerLogin:'kaicreator-mm',commentId:'6538568600',
  registry:'C:/xDev/kAiCreator/_tmp/tally-i92-k3/claim-registry.sqlite',
  writeSet:'d4947ee6cd3c60afee36cc384cb024ee420de5812b11ffd1fdacb2d1801ea100'};
const CONCURRENT={sourceSha:'d6b32929b481006de4e848eb772fe7dd96c2547c',
  registryRoot:'C:/xDev/kAiCreator/_tmp/tally-alm-claims',
  attempts:{A:'attempt:alm:j05:a:1',B:'attempt:alm:j05:b:1',repair:'attempt:alm:j05:repair:1'}};
function concurrentClaim(issue,role,commentId,operatorId,writerId){
  return {dispatchId:`dispatch:${issue}:alm:${role}:20261031:1`,operatorId,operatorKind:'other',
    writerId,writerLogin:'kaicreator-mm',commentId,registry:`${CONCURRENT.registryRoot}/registry-${issue}.sqlite`};
}
const CLAIMS={197:concurrentClaim(197,'builder-a','6539092449','zcode:alm:builder-a:1','tally-host:win32:alm-a:1'),
  198:concurrentClaim(198,'builder-b','6539094169','zcode:alm:builder-b:1','tally-host:win32:alm-b:1'),
  200:concurrentClaim(200,'reviewer','6539702548','zcode:alm:reviewer:1','tally-host:win32:alm-reviewer:1')};
const BUILDER={principal:'nilar\\codexsandboxonline',sid:'S-1-5-21-2839165489-1832998833-1574237796-1004',
  verifierSid:'S-1-5-21-2839165489-1832998833-1574237796-1003'};
const CODEX={cliPath:'C:/Users/15549/AppData/Roaming/fnm/node-versions/v24.21.0/installation/node_modules/@openai/codex/node_modules/@openai/codex-win32-x64/vendor/x86_64-pc-windows-msvc/bin/codex.exe',
  sandboxCliPath:'C:/Users/15549/AppData/Roaming/fnm/node-versions/v24.21.0/installation/node_modules/@openai/codex/node_modules/@openai/codex-win32-x64/vendor/x86_64-pc-windows-msvc/bin/codex.exe'};
export function defaultAlmConfig({dhPath,tallyPath,tallyBPath,githubToken,currentSourceSha,workRoot,
  codexHome}={}){
  const common={dhPath,tallyPath,tallyBPath,githubToken,currentSourceSha,
    builderPrincipal:BUILDER.principal,builderSid:BUILDER.sid,verifierSid:BUILDER.verifierSid,
    builderProfile:'C:/Users/CodexSandboxOnline',builderTemp:'C:/xDev/kAiCreator/_tmp/tally-v2-local-agent/online-builder-temp',
    emptyGitConfig:'C:/xDev/kAiCreator/_tmp/tally-v2-local-agent/empty-gitconfig',operatorHome:'C:/Users/15549',
    hiddenFixture:'C:/xDev/kAiCreator/_tmp/tally-v2-local-agent/hidden/heldout.txt',codexHome};
  return {...common,single:{...CLAIM_SINGLE,attemptId:'attempt:alm:j04:1',
    worktreePath:`${workRoot}/single-worktree`,evidenceDir:`${workRoot}/single-evidence`},
    concurrent:{...CONCURRENT,claims:CLAIMS,worktreeRoot:`${workRoot}/concurrent-worktrees`,
      evidenceDir:`${workRoot}/concurrent-evidence`,hostStorePath:`${workRoot}/concurrent-host.sqlite`},
    timeoutMs:10*60*1000};
}
function singleEnv(config){
  const c=config,s=config.single;
  return {DH:c.dhPath,TALLY:c.tallyPath,TALLY_B:c.tallyBPath,TALLY_GITHUB_READ_TOKEN:c.githubToken,
    TALLY_CLAIM_SOURCE_SHA:s.sourceSha,TALLY_CLAIM_DISPATCH_ID:s.dispatchId,
    TALLY_CLAIM_OPERATOR_ID:s.operatorId,TALLY_CLAIM_OPERATOR_KIND:s.operatorKind,
    TALLY_CLAIM_WRITER_ID:s.writerId,TALLY_CLAIM_WRITER_LOGIN:s.writerLogin,
    TALLY_CLAIM_COMMENT_ID:s.commentId,TALLY_CLAIM_REGISTRY_PATH:s.registry,
    TALLY_TASK_CLAIM_196_DISPATCH_ID:s.dispatchId,TALLY_TASK_CLAIM_196_OPERATOR_ID:s.operatorId,
    TALLY_TASK_CLAIM_196_OPERATOR_KIND:s.operatorKind,TALLY_TASK_CLAIM_196_WRITER_ID:s.writerId,
    TALLY_TASK_CLAIM_196_WRITER_LOGIN:s.writerLogin,TALLY_TASK_CLAIM_196_COMMENT_ID:s.commentId,
    TALLY_TASK_CLAIM_196_REGISTRY_PATH:s.registry,TALLY_TASK_CLAIM_196_WRITESET_SHA256:s.writeSet,
    TALLY_LOCAL_ISSUE_NUMBER:'196',
    TALLY_LOCAL_WRITESET_SHA256:s.writeSet,TALLY_LOCAL_ATTEMPT_ID:s.attemptId,
    TALLY_LOCAL_REPOSITORY_PATH:c.tallyPath,TALLY_LOCAL_WORKTREE_PATH:s.worktreePath,
    TALLY_CODEX_CLI_PATH:CODEX.cliPath,TALLY_CODEX_SANDBOX_PATH:CODEX.sandboxCliPath,
    TALLY_BUILDER_PROFILE:c.builderProfile,TALLY_BUILDER_TEMP:c.builderTemp,
    TALLY_EMPTY_GIT_CONFIG:c.emptyGitConfig,TALLY_OPERATOR_HOME:c.operatorHome,
    TALLY_HIDDEN_FIXTURE:c.hiddenFixture,TALLY_BUILDER_PRINCIPAL:c.builderPrincipal,
    TALLY_BUILDER_SID:c.builderSid,TALLY_VERIFIER_SID:c.verifierSid,
    TALLY_LOCAL_EVIDENCE_DIR:s.evidenceDir,...(c.codexHome?{TALLY_CODEX_HOME:c.codexHome}:{})};
}
function concurrentEnv(config){
  const c=config,s=config.concurrent,env={...singleEnv({...c,single:{...CLAIM_SINGLE,
    attemptId:s.attempts.A,worktreePath:s.worktreeRoot,evidenceDir:s.evidenceDir}}),
    TALLY_CLAIM_SOURCE_SHA:s.sourceSha,TALLY_LOCAL_WORKTREE_ROOT:s.worktreeRoot,
    TALLY_CONCURRENT_ISSUE_A:'197',TALLY_CONCURRENT_ISSUE_B:'198',TALLY_CONCURRENT_ISSUE_REVIEWER:'200',
    TALLY_LOCAL_HOST_STORE_PATH:s.hostStorePath,TALLY_LOCAL_ATTEMPT_ID_A:s.attempts.A,
    TALLY_LOCAL_ATTEMPT_ID_B:s.attempts.B,TALLY_LOCAL_ATTEMPT_ID_REPAIR:s.attempts.repair};
  delete env.TALLY_CLAIM_DISPATCH_ID;delete env.TALLY_CLAIM_OPERATOR_ID;delete env.TALLY_CLAIM_OPERATOR_KIND;
  delete env.TALLY_CLAIM_WRITER_ID;delete env.TALLY_CLAIM_WRITER_LOGIN;delete env.TALLY_CLAIM_COMMENT_ID;
  delete env.TALLY_CLAIM_REGISTRY_PATH;delete env.TALLY_LOCAL_WRITESET_SHA256;delete env.TALLY_LOCAL_ATTEMPT_ID;
  delete env.TALLY_LOCAL_WORKTREE_PATH;
  for(const issue of [197,198,200]){
    const claim=s.claims[issue],prefix=`TALLY_TASK_CLAIM_${issue}_`;
    Object.assign(env,{[prefix+'DISPATCH_ID']:claim.dispatchId,[prefix+'OPERATOR_ID']:claim.operatorId,
      [prefix+'OPERATOR_KIND']:claim.operatorKind,[prefix+'WRITER_ID']:claim.writerId,
      [prefix+'WRITER_LOGIN']:claim.writerLogin,[prefix+'COMMENT_ID']:claim.commentId,
      [prefix+'REGISTRY_PATH']:claim.registry});
    if(issue!==200)env[prefix+'WRITESET_SHA256']=sha256(`frozen exact write set: experiments/v2/selected-business-build/CONCURRENT_${issue===197?'A':'B'}.md`);
  }
  return env;
}
function spawnE2e(script,env,timeoutMs){
  const child=spawn(process.execPath,[script],{env,windowsHide:true});
  child.stdout?.on('data',chunk=>process.stdout.write(chunk));
  child.stderr?.on('data',chunk=>process.stderr.write(chunk));
  const startedAt=Date.now();
  return new Promise(resolve=>{
    let done=false,timer;
    const finish=value=>{if(done)return;done=true;clearTimeout(timer);resolve(value);};
    child.once('error',()=>finish({code:null,signal:null,startedAt,terminatedAt:Date.now()}));
    child.once('close',(code,signal)=>finish({code,signal,startedAt,terminatedAt:Date.now()}));
    timer=setTimeout(()=>{try{child.kill();}catch{}finish({code:null,signal:null,timeout:true,startedAt,terminatedAt:Date.now()});},timeoutMs);
    timer.unref?.();
  });
}
async function resultOf(kind,config,run){
  const env=kind==='start'?singleEnv(config):concurrentEnv(config);
  const evidenceDir=kind==='start'?config.single.evidenceDir:config.concurrent.evidenceDir;
  if(run)return run(kind,env,evidenceDir);
  const script=kind==='start'?'experiments/v2/selected-business-build/local-codex-windows-e2e.mjs':
    'experiments/v2/selected-business-build/concurrent-codex-windows-e2e.mjs';
  const exit=await spawnE2e(join(config.tallyPath,script),env,config.timeoutMs);
  let result=null;
  try{const path=join(evidenceDir,'e2e-result.json'),info=await stat(path);
    if(info.mtimeMs>=exit.startedAt)result=JSON.parse(await readFile(path,'utf8'));}catch{}
  return {status:exit.code===0&&result?.status==='PASS'?'PASS':'FAIL',exitCode:exit.code,
    signal:exit.signal??null,timeout:exit.timeout===true,result,evidenceDir};
}
export function createJourneyController(config,{run}={}){
  if(!SHA.test(config.currentSourceSha??''))throw new TypeError('CURRENT_SOURCE_SHA_REQUIRED');
  const history=[];let pending=Promise.resolve();
  async function act(kind,sourceSha){
    if(sourceSha!==config.currentSourceSha)return {status:'STALE_SOURCE',taskPass:false,releasePass:false};
    const execution=pending.then(()=>resultOf(kind,config,run));pending=execution.catch(()=>{});
    const result=await execution;history.push({kind,...result});
    return result;
  }
  function snapshot(){
    const latest=history.at(-1),single=history.filter(x=>x.kind==='start').at(-1),
      review=history.filter(x=>x.kind==='review-repair').at(-1);
    const task=(number,sourceRef,sourceSha,result)=>({sourceRef,sourceSha,observedAsOf:new Date().toISOString(),
      currentness:'REAL_ACTION_READBACK',status:result?.status??'NOT_RUN',reason:result?.status??'NOT_RUN',
      dependencyState:'CLEAR',claimState:'ACCEPTED',agentAttemptState:result?.result?.realAgentEffect===true&&result.status==='PASS'?'LOCAL_CANDIDATE_READY':result?.status==='FAIL'?'FAILED':'NOT_RUN',
      reviewState:result?.result?.result?.rejoin?.state??'NOT_RUN',ciState:'NOT_VERIFIED',hostState:result?.status??'NOT_RUN',
      releaseState:'NOT_VERIFIED',authorized:false,admitted:false,taskPass:false,releasePass:false});
    return {status:'OBSERVED',project:'tally',version:'v2',sourceSha:config.currentSourceSha,
      observedAsOf:new Date().toISOString(),currentness:'REAL_ACTION_READBACK',tasks:[
        task(196,'https://github.com/kaicreator-mm/tally/issues/196',CLAIM_SINGLE.sourceSha,single),
        task(197,'https://github.com/kaicreator-mm/tally/issues/197',CONCURRENT.sourceSha,review),
        task(198,'https://github.com/kaicreator-mm/tally/issues/198',CONCURRENT.sourceSha,review),
        task(200,'https://github.com/kaicreator-mm/tally/issues/200',CONCURRENT.sourceSha,review)],
      progressSignals:{realCandidates:history.filter(x=>x.status==='PASS').length,reviewed:review?.result?.result?.rejoin?.state==='REVIEW_JOINED'?1:0,released:0,comparison:'COUNTS_NOT_PERCENTAGES'},
      result:latest??null,authorized:false,admitted:false,taskPass:false,releasePass:false};
  }
  return Object.freeze({start:sourceSha=>act('start',sourceSha),reviewRepair:sourceSha=>act('review-repair',sourceSha),snapshot});
}
