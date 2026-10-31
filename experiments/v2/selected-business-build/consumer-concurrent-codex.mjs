/** T06n concurrent real/replay consumer: ONE K facade, two overlapping
 *  external Builder effects, reviewer Join, bounded Repair and re-Join. */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {mkdtemp,rm,mkdir} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
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
console.log('T06N_PROVIDER_INTEGRITY_BEFORE_K_LOAD=PASS');

const replay=process.env.TALLY_LOCAL_REPLAY==='1';
const token=process.env.TALLY_GITHUB_READ_TOKEN;
const sourceSha=process.env.TALLY_CLAIM_SOURCE_SHA;
const issueRefFor=n=>'https://github.com/kaicreator-mm/tally/issues/'+n;
const issueA=Number(process.env.TALLY_CONCURRENT_ISSUE_A??177);
const issueB=Number(process.env.TALLY_CONCURRENT_ISSUE_B??178);
const issueReviewer=Number(process.env.TALLY_CONCURRENT_ISSUE_REVIEWER??179);
const CLAIMS=[
  {issueNumber:issueA,actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_A.md'},
  {issueNumber:issueB,actorRole:'builder',candidateRelativePath:'experiments/v2/selected-business-build/CONCURRENT_B.md'},
  {issueNumber:issueReviewer,actorRole:'reviewer'},
];
for(const claim of CLAIMS){
  const p='TALLY_TASK_CLAIM_'+claim.issueNumber+'_';
  claim.dispatchId=process.env[p+'DISPATCH_ID'];
  claim.operatorId=process.env[p+'OPERATOR_ID'];
  claim.operatorKind=process.env[p+'OPERATOR_KIND'];
  claim.writerId=process.env[p+'WRITER_ID'];
  claim.writerLogin=process.env[p+'WRITER_LOGIN'];
  claim.commentId=Number(process.env[p+'COMMENT_ID']);
  claim.registryPath=process.env[p+'REGISTRY_PATH'];
}
assert.ok(typeof token==='string'&&token.length>=8);
assert.match(sourceSha??'',/^[0-9a-f]{40}$/);
for(const claim of CLAIMS){
  assert.ok(claim.dispatchId&&claim.operatorId&&claim.operatorKind&&
    claim.writerId&&claim.writerLogin,'claim env required for #'+claim.issueNumber);
  assert.ok(Number.isSafeInteger(claim.commentId)&&claim.commentId>0,
    'comment id required for #'+claim.issueNumber);
  if(!replay)assert.ok(typeof claim.registryPath==='string'&&claim.registryPath,
    'real run requires the Host-prepared per-issue registry path for #'+claim.issueNumber);
}
const attemptA=process.env.TALLY_LOCAL_ATTEMPT_ID_A??'attempt:t06n:177:1';
const attemptB=process.env.TALLY_LOCAL_ATTEMPT_ID_B??'attempt:t06n:178:1';
const attemptRepair=process.env.TALLY_LOCAL_ATTEMPT_ID_REPAIR??'attempt:t06n:repair:1';
const claimKeyFor=claim=>'kaicreator-mm/tally#'+claim.issueNumber+':'+claim.actorRole+':__default__';
const registryKeyFor=claim=>'tally:v2:claim-admission:'+sha(claimKeyFor(claim));
const writeSetKeyFor=issue=>'tally:v2:task-writeset:'+issue;
const rowValue=claim=>({phase:'PUBLISHED_VERIFIED',generation:1,
  protectedClaimKey:claimKeyFor(claim),repository:'kaicreator-mm/tally',
  issueRef:issueRefFor(claim.issueNumber),sourceSha,dispatchId:claim.dispatchId,
  actorRole:claim.actorRole,operatorKind:claim.operatorKind,operatorId:claim.operatorId,
  writerId:claim.writerId,commentId:claim.commentId,
  commentRef:`${issueRefFor(claim.issueNumber)}#issuecomment-${claim.commentId}`});
const evidenceBase=process.env.TALLY_LOCAL_EVIDENCE_DIR;
if(evidenceBase)await mkdir(evidenceBase,{recursive:true});
const tmp=await mkdtemp(join(evidenceBase??tmpdir(),'tally-t06n-concurrent-'));
const db=join(tmp,'kernel.sqlite');
// Each accepted Claim lives in its own Host-prepared registry; safe replay
// seeds one isolated registry per issue, never a synthetic shared authority.
for(const claim of CLAIMS)
  claim.registryDb=replay?join(tmp,`claim-registry-${claim.issueNumber}.sqlite`):claim.registryPath;
let registry=null;
if(replay){
  for(const claim of CLAIMS){
    registry=await openSqliteCasStore({path:claim.registryDb,busyTimeoutMs:1000});
    const key=registryKeyFor(claim);
    const reservation={...rowValue(claim),phase:'PENDING_GITHUB_PUBLISH'};
    delete reservation.commentId;delete reservation.commentRef;
    await registry.put(key,reservation,{mustBeAbsent:true});
    await registry.put(key,rowValue(claim),{mustMatchRevision:0});
    if(claim.candidateRelativePath){
      const writeSetDocument='frozen exact write set: '+claim.candidateRelativePath;
      process.env['TALLY_TASK_CLAIM_'+claim.issueNumber+'_WRITESET_SHA256']=sha(writeSetDocument);
      await registry.put(writeSetKeyFor(claim.issueNumber),{phase:'FROZEN_DAG_WRITESET',
        repository:'kaicreator-mm/tally',issueNumber:claim.issueNumber,sourceSha,
        writeSetSha256:sha(writeSetDocument),document:writeSetDocument},{mustBeAbsent:true});
    }
    await registry.close();registry=null;
  }
}else{
  for(const claim of CLAIMS){
    registry=await openSqliteCasStore({path:claim.registryDb,busyTimeoutMs:1000});
    const existing=await registry.get(registryKeyFor(claim));
    assert.equal(existing?.value?.phase,'PUBLISHED_VERIFIED',
      'real run requires a Host-prepared accepted registry row for #'+claim.issueNumber);
    await registry.close();registry=null;
  }
}
if(replay){
  const date='2026-10-30T00:00:00Z';
  const commentFor=claim=>{
    const body=`<!-- ads:github -->\n\`\`\`yaml\nschema: ads/github/v3\nkind: event\ndata:\n  schema: ai-dev/event-v2\n  event: DISPATCH_CLAIMED\n  actor_role: ${claim.actorRole}\n  operator_kind: ${claim.operatorKind}\n  operator_id: ${claim.operatorId}\n  task: '#${claim.issueNumber}'\n  issue: '#${claim.issueNumber}'\n  sha: ${sourceSha}\n  dispatch_id: ${claim.dispatchId}\n  dispatch_state: CLAIMED\n  protected_claim_key: ${claimKeyFor(claim)}\n  admission_generation: 1\n\`\`\`\n`;
    return {id:claim.commentId,body,html_url:`${issueRefFor(claim.issueNumber)}#issuecomment-${claim.commentId}`,
      issue_url:`https://api.github.com/repos/kaicreator-mm/tally/issues/${claim.issueNumber}`,
      created_at:date,updated_at:date,user:{login:claim.writerLogin}};
  };
  const issueFor=claim=>({id:claim.issueNumber*10,number:claim.issueNumber,
    html_url:issueRefFor(claim.issueNumber),state:'open',updated_at:date,
    labels:[{name:'type:task'},{name:'state:ready'}]});
  globalThis.fetch=async input=>{
    const url=new URL(input);let value,status=200;
    const byComment=CLAIMS.find(c=>url.pathname===
      `/repos/kaicreator-mm/tally/issues/comments/${c.commentId}`);
    const issueMatch=url.pathname.match(/\/issues\/(\d+)(?:\/comments|\/dependencies|$)/);
    const claim=byComment??CLAIMS.find(c=>c.issueNumber===Number(issueMatch?.[1]));
    if(claim&&url.pathname===`/repos/kaicreator-mm/tally/issues/${claim.issueNumber}`)value=issueFor(claim);
    else if(url.pathname===`/repos/kaicreator-mm/tally/commits/${sourceSha}`)value={sha:sourceSha};
    else if(claim&&url.pathname===`/repos/kaicreator-mm/tally/issues/${claim.issueNumber}/dependencies/blocked_by`)value=[];
    else if(claim&&url.pathname===`/repos/kaicreator-mm/tally/issues/${claim.issueNumber}/comments`)value=[commentFor(claim)];
    else if(claim&&url.pathname===`/repos/kaicreator-mm/tally/issues/comments/${claim.commentId}`)value=commentFor(claim);
    else if(url.pathname==='/user')value={login:CLAIMS[0].writerLogin};
    else{status=404;value={message:'fixture-not-found'};}
    return new Response(JSON.stringify(value),{status,headers:{'content-type':'application/json',
      'x-github-request-id':'t06n-replay-fixture'}});
  };
}
for(const claim of CLAIMS)
  process.env['TALLY_TASK_CLAIM_'+claim.issueNumber+'_REGISTRY_PATH']=claim.registryDb;
process.env.TALLY_LOCAL_HOST_STORE_PATH=db;
const deployment={deployedAppId:'deploy:tally-v2-t06n',tenantId:'tenant:tally-v2',appId:sealed.appId};
const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
  authorize:(identity,target)=>identity?.tenantId===target.tenantId&&
    target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
const condition=c=>c?.kind==='create'?{mustBeAbsent:true}:
  c?.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
  (()=>{throw Error('INVALID_K_CONDITION');})();
const options=sqlite=>{
  let pending=Promise.resolve();
  const serial=fn=>{const result=pending.then(fn,fn);pending=result.catch(()=>{});return result;};
  return {store:{get:key=>serial(()=>sqlite.get(key)),
    put:(key,v,c)=>serial(()=>sqlite.put(key,v,condition(c))),
    remove:(key,c)=>serial(()=>sqlite.remove(key,condition(c)))},deploymentIdentity:deployment,
    deploymentRoot:process.cwd(),tenantId:deployment.tenantId,queryAuthority:authority,
    now:()=>Date.UTC(2026,9,30)};
};
const selectorFor=instanceKey=>({workflowId:'tally-readiness',instanceKey,
  businessPackageId:'business-tally-task-readiness@v2.0.0'});
const attemptMessage=(claim,attemptId,instanceKey,messageId)=>({
  intentType:'runLocalCodexAttempt',instanceKey,messageId,
  input:{issueRef:issueRefFor(claim.issueNumber),issueNumber:claim.issueNumber,
    sourceSha,attemptId}});
const joinMessage=(joinId,candidates,messageId)=>({
  intentType:'joinReviewedCandidates',instanceKey:`task-concurrent-codex-${issueReviewer}`,
  messageId,
  input:{issueRef:issueRefFor(issueReviewer),issueNumber:issueReviewer,sourceSha,joinId,candidates}});
const candidatesFrom=effects=>effects.map((effect,ix)=>({
  issueNumber:CLAIMS[ix].issueNumber,attemptId:effect.attemptId,
  candidateTree:effect.candidateTree??'a'.repeat(40),
  candidateCommit:effect.candidateCommit??'b'.repeat(40)}));
let sqlite=null,facade=null;
try{
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  // ONE K facade; TWO distinct instance keys/messages so the two real
  // external Builder effects genuinely overlap in process lifetimes.
  const instanceKeyA=`task-concurrent-codex-${issueA}`,instanceKeyB=`task-concurrent-codex-${issueB}`;
  for(const [claim,instanceKey] of [[CLAIMS[0],instanceKeyA],[CLAIMS[1],instanceKeyB]]){
    const initialized=await facade.send({intentType:'inspectTaskReadiness',instanceKey,
      messageId:'message-t06n-init-'+claim.issueNumber,input:{issueRef:issueRefFor(claim.issueNumber),
        issueNumber:claim.issueNumber,sourceSha,claimRef:'t06n-init',dependenciesComplete:true}});
    assert.equal(initialized.status,'admitted',JSON.stringify(initialized).slice(0,1200));
  }
  const [receiptA,receiptB]=await Promise.all([
    facade.send(attemptMessage(CLAIMS[0],attemptA,instanceKeyA,'message-t06n-177-1')),
    facade.send(attemptMessage(CLAIMS[1],attemptB,instanceKeyB,'message-t06n-178-1')),
  ]);
  assert.equal(receiptA.status,'admitted',JSON.stringify(receiptA).slice(0,1500));
  assert.equal(receiptB.status,'admitted',JSON.stringify(receiptB).slice(0,1500));
  const effectA=receiptA.admitted.effects[0].output;
  const effectB=receiptB.admitted.effects[0].output;
  if(replay){
    for(const effect of [effectA,effectB]){
      assert.equal(effect.state,'REPLAY_COMPLETE',JSON.stringify(effect));
      assert.equal(effect.replayFixture,true);assert.equal(effect.agentLaunch,false);
    }
  }else if(process.env.TALLY_LOCAL_REQUIRE_CANDIDATE==='1'){
    for(const effect of [effectA,effectB]){
      assert.equal(effect.state,'LOCAL_CANDIDATE_READY',JSON.stringify(effect));
      assert.equal(effect.agentLaunch,true);assert.equal(effect.taskPass,false);
      assert.match(effect.candidateTree,/^[0-9a-f]{40}$/);
      assert.match(effect.candidateCommit,/^[0-9a-f]{40}$/);
    }
    assert.notEqual(effectA.candidateCommit,effectB.candidateCommit,'candidates must be distinct');
    const a0=Date.parse(effectA.process.startedAt),a1=Date.parse(effectA.process.terminatedAt);
    const b0=Date.parse(effectB.process.startedAt),b1=Date.parse(effectB.process.terminatedAt);
    assert.ok(Number.isFinite(a0)&&Number.isFinite(a1)&&Number.isFinite(b0)&&Number.isFinite(b1),
      'Host must return actual started/terminated timestamps for overlap evidence');
    assert.ok(a0<b1&&b0<a1,'REAL_PROCESS_OVERLAP_REQUIRED: two Builder lifetimes must overlap');
  }
  // Reviewer refusal must precede any valid Join. In replay the sealed C
  // short-circuits to REPLAY_FIXTURE_ONLY, so denials are real-mode only.
  const validCandidates=candidatesFrom([effectA,effectB]);
  const denialChecks=[];
  if(!replay){
    denialChecks.push(
      ['wrong',joinMessage('join:t06n:deny:wrong',
        [{...validCandidates[0],candidateCommit:'e'.repeat(40)},validCandidates[1]],
        'message-t06n-deny-wrong'),['CANDIDATE_STALE','CANDIDATE_READBACK_MISMATCH']],
      ['stale',joinMessage('join:t06n:deny:stale',
        [{...validCandidates[0],candidateTree:'f'.repeat(40)},validCandidates[1]],
        'message-t06n-deny-stale'),['CANDIDATE_STALE']],
      ['missing',joinMessage('join:t06n:deny:missing',
        [{...validCandidates[0],attemptId:'attempt:t06n:missing:1'},validCandidates[1]],
        'message-t06n-deny-missing'),['CANDIDATE_NOT_FOUND']],
      ['overlap',joinMessage('join:t06n:deny:overlap',
        [validCandidates[0],validCandidates[0]],'message-t06n-deny-overlap'),
        ['CANDIDATE_WRITESET_OVERLAP','CANDIDATE_STALE']],
    );
  }
  const denials=[];
  for(const [name,message,reasons] of denialChecks){
    const receipt=await facade.send(message);
    assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1500));
    const output=receipt.admitted.effects[0].output;
    assert.equal(output.state,'NOT_ADMITTED',name+':'+JSON.stringify(output));
    assert.ok(reasons.includes(output.reason),name+':'+output.reason);
    assert.equal(output.admitted,false);assert.equal(output.taskPass,false);
    denials.push({name,reason:output.reason});
  }
  const joinReceipt=await facade.send(joinMessage('join:t06n:179:1',validCandidates,
    'message-t06n-join-1'));
  assert.equal(joinReceipt.status,'admitted',JSON.stringify(joinReceipt).slice(0,1500));
  const joinEffect=joinReceipt.admitted.effects[0].output;
  if(replay){
    assert.equal(joinEffect.state,'REPLAY_COMPLETE',JSON.stringify(joinEffect));
    assert.equal(joinEffect.replayFixture,true);
  }else{
    assert.equal(joinEffect.state,'REVIEW_JOINED',JSON.stringify(joinEffect));
    assert.equal(joinEffect.taskPass,false);assert.equal(joinEffect.authorized,false);
    assert.equal(joinEffect.candidates.length,2);
    for(const c of joinEffect.candidates){
      assert.equal(c.candidateCommit,validCandidates.find(x=>x.attemptId===c.attemptId)
        .candidateCommit);
    }
  }
  // Bounded Repair: ONE new attempt ID/generation; prior evidence preserved.
  const repairReceipt=await facade.send(attemptMessage(CLAIMS[1],attemptRepair,
    'task-concurrent-codex-178-repair','message-t06n-178-repair-1'));
  assert.equal(repairReceipt.status,'admitted',JSON.stringify(repairReceipt).slice(0,1500));
  const repairEffect=repairReceipt.admitted.effects[0].output;
  if(replay){
    assert.equal(repairEffect.state,'REPLAY_COMPLETE',JSON.stringify(repairEffect));
  }else if(process.env.TALLY_LOCAL_REQUIRE_CANDIDATE==='1'){
    assert.equal(repairEffect.state,'LOCAL_CANDIDATE_READY',JSON.stringify(repairEffect));
    assert.notEqual(repairEffect.candidateCommit,effectB.candidateCommit,
      'repair must produce a fresh candidate, never rewrite the prior one');
  }
  const rejoined=await facade.send(joinMessage('join:t06n:179:2',
    candidatesFrom([effectA,repairEffect]),'message-t06n-join-2'));
  assert.equal(rejoined.status,'admitted',JSON.stringify(rejoined).slice(0,1500));
  const rejoinEffect=rejoined.admitted.effects[0].output;
  assert.equal(replay?rejoinEffect.state==='REPLAY_COMPLETE':rejoinEffect.state==='REVIEW_JOINED',
    true,JSON.stringify(rejoinEffect));
  const before=(await facade.query(selectorFor(instanceKeyA))).snapshot;
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
  assert.deepEqual((await facade.query(selectorFor(instanceKeyA))).snapshot,before);
  const replayed=await facade.send(joinMessage('join:t06n:179:2',
    candidatesFrom([effectA,repairEffect]),'message-t06n-join-2'));
  assert.equal(replayed.status,'admitted');assert.equal(replayed.idempotentReplay,true);
  assert.deepEqual(await sqlite.list(''),recordsBefore);
  const overlapMs=!replay&&process.env.TALLY_LOCAL_REQUIRE_CANDIDATE==='1'?
    Math.min(Date.parse(effectA.process.terminatedAt),Date.parse(effectB.process.terminatedAt))-
    Math.max(Date.parse(effectA.process.startedAt),Date.parse(effectB.process.startedAt)):null;
  const summary={provider:'local-codex-host',via:'real-sealed-k-b-c',sourceSha,
    attemptA,attemptB,attemptRepair,
    resultA:effectA.state,resultB:effectB.state,repairResult:repairEffect.state,
    agentLaunchA:effectA.agentLaunch,agentLaunchB:effectB.agentLaunch,
    candidateA:{tree:effectA.candidateTree??null,commit:effectA.candidateCommit??null},
    candidateB:{tree:effectB.candidateTree??null,commit:effectB.candidateCommit??null},
    overlap:{processA:replay?null:effectA.process??null,processB:replay?null:effectB.process??null,
      overlapMs,verified:!replay&&overlapMs!==null&&overlapMs>0},
    join:{joinId:'join:t06n:179:1',state:joinEffect.state,taskPass:false},
    denials,rejoin:{joinId:'join:t06n:179:2',state:rejoinEffect.state,taskPass:false},
    repair:{attemptId:attemptRepair,state:repairEffect.state,
      priorCandidatePreserved:!replay?effectB.candidateCommit??null:null},
    replayFixture:replay,reopened:true,idempotentReplay:replayed.idempotentReplay,
    taskPass:false,hiddenPass:false,docCount:recordsBefore.length,hostStorePath:db,
    worktreeRoot:process.env.TALLY_LOCAL_WORKTREE_ROOT??null};
  if(process.env.TALLY_LOCAL_RESULT_PATH)
    writeFileSync(process.env.TALLY_LOCAL_RESULT_PATH,JSON.stringify(summary,null,2)+'\n',{mode:0o600});
  console.log(JSON.stringify(summary));
  if(!replay)console.log('T06N_HOST_EVIDENCE_DIR='+tmp);
  console.log(replay?
    'T06N_SAFE_REPLAY_ONLY_NO_REAL_OVERLAP_NO_AGENT_EFFECT=PASS':
    'T06N_REAL_CONCURRENT_LOCAL_CODEX_K_HOST_JOIN_REPAIR=PASS');
}finally{
  if(facade)facade.stop();if(sqlite)await sqlite.close();if(registry)await registry.close();
  if(replay||process.env.TALLY_LOCAL_CLEAN==='1')await rm(tmp,{recursive:true,force:true});
}
