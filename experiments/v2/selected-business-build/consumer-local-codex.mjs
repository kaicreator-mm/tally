/** T06m real/replay consumer: public K/Host, sealed provider, durable candidate. */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {mkdtemp,rm,mkdir} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const execFileAsync=promisify(execFile);
const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-local-codex@v1.0.0');
assert.ok(selectedC?.moduleSource,'sealed Local Codex C required');
const sha=x=>createHash('sha256').update(x).digest('hex');
assert.equal(sha(selectedC.moduleSource),selectedC.moduleSha256);
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'Local Codex provider digest required');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
const originalProvider=readFileSync(providerPath);
const assertProvider=bytes=>assert.equal(sha(bytes),digestMatch[1],'PROVIDER_INTEGRITY_MISMATCH');
assertProvider(originalProvider);
try{
  writeFileSync(providerPath,Buffer.concat([originalProvider,Buffer.from('\n// substitution')]));
  assert.throws(()=>assertProvider(readFileSync(providerPath)),/PROVIDER_INTEGRITY_MISMATCH/);
}finally{writeFileSync(providerPath,originalProvider);}
assertProvider(readFileSync(providerPath));
console.log('T06M_PROVIDER_INTEGRITY_BEFORE_K_LOAD=PASS');

const replay=process.env.TALLY_LOCAL_REPLAY==='1';
const token=process.env.TALLY_GITHUB_READ_TOKEN;
const sourceSha=process.env.TALLY_CLAIM_SOURCE_SHA;
const dispatchId=process.env.TALLY_CLAIM_DISPATCH_ID;
const operatorId=process.env.TALLY_CLAIM_OPERATOR_ID;
const operatorKind=process.env.TALLY_CLAIM_OPERATOR_KIND;
const writerId=process.env.TALLY_CLAIM_WRITER_ID;
const writerLogin=process.env.TALLY_CLAIM_WRITER_LOGIN;
const commentId=Number(process.env.TALLY_CLAIM_COMMENT_ID);
const attemptId=process.env.TALLY_LOCAL_ATTEMPT_ID??'attempt:t06m:172:1';
assert.ok(typeof token==='string'&&token.length>=8);
assert.match(sourceSha??'',/^[0-9a-f]{40}$/);
assert.ok(dispatchId&&operatorId&&operatorKind&&writerId&&writerLogin);
assert.ok(Number.isSafeInteger(commentId)&&commentId>0);
const issueNumber=Number(process.env.TALLY_LOCAL_ISSUE_NUMBER??172);
const issueRef=`https://github.com/kaicreator-mm/tally/issues/${issueNumber}`;
const claimKey=`kaicreator-mm/tally#${issueNumber}:builder:__default__`;
const registryKey='tally:v2:claim-admission:'+sha(claimKey);
const writeSetKey=`tally:v2:task-writeset:${issueNumber}`;
const rowValue=()=>({phase:'PUBLISHED_VERIFIED',generation:1,protectedClaimKey:claimKey,
  repository:'kaicreator-mm/tally',issueRef,sourceSha,dispatchId,actorRole:'builder',
  operatorKind,operatorId,writerId,commentId,commentRef:`${issueRef}#issuecomment-${commentId}`});
const evidenceBase=process.env.TALLY_LOCAL_EVIDENCE_DIR;
if(evidenceBase)await mkdir(evidenceBase,{recursive:true});
const tmp=await mkdtemp(join(evidenceBase??tmpdir(),'tally-t06m-local-'));
const db=join(tmp,'kernel.sqlite');
const registryDb=replay?join(tmp,'claim-registry.sqlite'):process.env.TALLY_CLAIM_REGISTRY_PATH;
assert.ok(registryDb);
let registry=await openSqliteCasStore({path:registryDb,busyTimeoutMs:1000});
if(replay){
  const writeSetDocument='frozen exact write set: experiments/v2/selected-business-build/LOCAL_CODEX_L3.md';
  const writeSetSha256=sha(writeSetDocument);
  process.env.TALLY_LOCAL_WRITESET_SHA256=writeSetSha256;
  const reservation={...rowValue(),phase:'PENDING_GITHUB_PUBLISH'};
  delete reservation.commentId;delete reservation.commentRef;
  await registry.put(registryKey,reservation,{mustBeAbsent:true});
  await registry.put(registryKey,rowValue(),{mustMatchRevision:0});
  await registry.put(writeSetKey,{phase:'FROZEN_DAG_WRITESET',repository:'kaicreator-mm/tally',
    issueNumber:172,sourceSha,writeSetSha256,document:writeSetDocument},{mustBeAbsent:true});
  const date='2026-10-30T00:00:00Z';
  const body=`<!-- ads:github -->\n\`\`\`yaml\nschema: ads/github/v3\nkind: event\ndata:\n  schema: ai-dev/event-v2\n  event: DISPATCH_CLAIMED\n  actor_role: builder\n  operator_kind: ${operatorKind}\n  operator_id: ${operatorId}\n  task: '#172'\n  issue: '#172'\n  sha: ${sourceSha}\n  dispatch_id: ${dispatchId}\n  dispatch_state: CLAIMED\n  protected_claim_key: ${claimKey}\n  admission_generation: 1\n\`\`\`\n`;
  const comment={id:commentId,body,html_url:`${issueRef}#issuecomment-${commentId}`,
    issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/172',
    created_at:date,updated_at:date,user:{login:writerLogin}};
  const issue={id:issueNumber*10,number:issueNumber,html_url:issueRef,state:'open',updated_at:date,
    labels:[{name:'type:task'},{name:'state:ready'}]};
  globalThis.fetch=async input=>{
    const url=new URL(input);let value,status=200;
    if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}`)value=issue;
    else if(url.pathname===`/repos/kaicreator-mm/tally/commits/${sourceSha}`)value={sha:sourceSha};
    else if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}/dependencies/blocked_by`)value=[];
    else if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}/comments`)value=[comment];
    else if(url.pathname===`/repos/kaicreator-mm/tally/issues/comments/${commentId}`)value=comment;
    else if(url.pathname==='/user')value={login:writerLogin};
    else{status=404;value={message:'fixture-not-found'};}
    return new Response(JSON.stringify(value),{status,headers:{'content-type':'application/json',
      'x-github-request-id':'t06m-replay-fixture'}});
  };
}else{
  const existing=await registry.get(registryKey);
  assert.equal(existing?.value?.phase,'PUBLISHED_VERIFIED');
  assert.ok(process.env.TALLY_LOCAL_WRITESET_SHA256);
}
await registry.close();registry=null;
process.env.TALLY_CLAIM_REGISTRY_PATH=registryDb;
process.env.TALLY_LOCAL_HOST_STORE_PATH=db;
const deployment={deployedAppId:'deploy:tally-v2-t06m',tenantId:'tenant:tally-v2',appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>identity?.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c?.kind==='create'?{mustBeAbsent:true}:
  c?.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw Error('INVALID_K_CONDITION');})();
const options=sqlite=>({store:{get:key=>sqlite.get(key),put:(key,v,c)=>sqlite.put(key,v,condition(c)),
  remove:(key,c)=>sqlite.remove(key,condition(c))},deploymentIdentity:deployment,
  deploymentRoot:process.cwd(),tenantId:deployment.tenantId,queryAuthority:authority,
  now:()=>Date.UTC(2026,9,30)});
const input={issueRef,issueNumber,sourceSha,attemptId};
const instanceKey=`task-local-codex-${issueNumber}`;
const selector={workflowId:'tally-readiness',instanceKey,
  businessPackageId:'business-tally-task-readiness@v2.0.0'};
const message={intentType:'runLocalCodexAttempt',instanceKey,messageId:'message-t06m-172-1',input};
let sqlite=null,facade=null;
try{
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  const receipt=await facade.send(message);
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1500));
  assert.equal(receipt.admitted?.effects?.length,1);
  const effect=receipt.admitted.effects[0].output;
  if(replay){
    assert.equal(effect.state,'REPLAY_COMPLETE',JSON.stringify(effect));
    assert.equal(effect.replayFixture,true);assert.equal(effect.agentLaunch,false);
  }else if(process.env.TALLY_LOCAL_REQUIRE_CANDIDATE==='1'){
    assert.equal(effect.state,'LOCAL_CANDIDATE_READY',JSON.stringify(effect));
    assert.equal(effect.agentLaunch,true);assert.equal(effect.taskPass,false);
    assert.match(effect.candidateTree,/^[0-9a-f]{40}$/);
    assert.match(effect.candidateCommit,/^[0-9a-f]{40}$/);
    const {stdout}=await execFileAsync(process.env.TALLY_GIT_PATH??'C:\\Program Files\\Git\\cmd\\git.exe',
      ['-C',process.env.TALLY_LOCAL_WORKTREE_PATH,'rev-parse',effect.candidateCommit+'^{tree}']);
    assert.equal(stdout.trim(),effect.candidateTree);
  }
  const before=(await facade.query(selector)).snapshot;
  const recordsBefore=await sqlite.list('');
  for(const key of recordsBefore){
    const row=await sqlite.get(key);const text=JSON.stringify(row);
    assert.ok(!text.includes(token),'Host token may not enter K documents');
    assert.ok(!text.includes('DISPATCH_CLAIMED'),'raw Claim body may not enter K documents');
  }
  facade.stop();facade=null;await sqlite.close();sqlite=null;
  delete process.env.TALLY_GITHUB_READ_TOKEN;
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  assert.deepEqual((await facade.query(selector)).snapshot,before);
  const replayed=await facade.send(message);
  assert.equal(replayed.status,'admitted');assert.equal(replayed.idempotentReplay,true);
  assert.deepEqual(await sqlite.list(''),recordsBefore);
  const summary={provider:'local-codex-host',via:'real-sealed-k-b-c',sourceSha,attemptId,
    result:effect.state,agentLaunch:effect.agentLaunch,candidateTree:effect.candidateTree??null,
    candidateCommit:effect.candidateCommit??null,replayFixture:replay,reopened:true,
    idempotentReplay:replayed.idempotentReplay,taskPass:false,hiddenPass:false,docCount:recordsBefore.length,
    hostStorePath:db,worktreePath:process.env.TALLY_LOCAL_WORKTREE_PATH??null};
  if(process.env.TALLY_LOCAL_RESULT_PATH)
    writeFileSync(process.env.TALLY_LOCAL_RESULT_PATH,JSON.stringify(summary,null,2)+'\n',{mode:0o600});
  console.log(JSON.stringify(summary));
  if(!replay)console.log('T06M_HOST_EVIDENCE_DIR='+tmp);
  console.log(replay?'T06M_REPLAY_ONLY_NO_AGENT_EFFECT=PASS':'T06M_REAL_LOCAL_CODEX_K_HOST_CANDIDATE=PASS');
}finally{
  if(facade)facade.stop();if(sqlite)await sqlite.close();if(registry)await registry.close();
  if(replay||process.env.TALLY_LOCAL_CLEAN==='1')await rm(tmp,{recursive:true,force:true});
}
