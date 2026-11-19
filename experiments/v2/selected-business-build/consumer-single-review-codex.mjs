/** #264 J05 — single-Builder ReviewDecision/TaskOutcome consumer.
 *
 * Real/replay public K consumer over the SAME sealed B→K→C composition as
 * the J04 Builder lane: one K send of intent reviewSingleCandidate, bounded
 * output assertions, and a durable consumer result. The Reviewer never
 * launches a process; the sealed C verifies the reviewer Claim, the Builder
 * Claim (independence), the current source pair, the frozen write-set, the
 * Builder CAS CANDIDATE_READY record, the candidate/repair git readback, the
 * PR head and the exact-SHA CI — then records the ReviewDecision create-only
 * and (ACCEPT + full qualification only) the qualified Task Outcome.
 */
import assert from 'node:assert/strict';
import {readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {mkdtemp,rm,mkdir} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {load} from '@kaicreator/domain-harness-microkernel';
import {openSqliteCasStore} from '@kaicreator/domain-harness-host-node';
const sha=x=>createHash('sha256').update(x).digest('hex');
const sealed=JSON.parse(readFileSync(new URL('./sealed-app.json',import.meta.url),'utf8'));
const selectedC=sealed.modules.find(x=>x.role==='capability'&&
  x.packageId==='capability-tally-local-codex@v1.0.0');
assert.ok(selectedC?.moduleSource,'sealed Local Codex C required');
assert.equal(sha(selectedC.moduleSource),selectedC.moduleSha256);
const digestMatch=/NATIVE_PROVIDER_SHA256='([0-9a-f]{64})'/.exec(selectedC.moduleSource);
assert.ok(digestMatch,'Local Codex provider digest required');
const providerPath=fileURLToPath(import.meta.resolve('@kaicreator/tally-local-codex-host-bundle'));
const originalProvider=readFileSync(providerPath);
assert.equal(sha(originalProvider),digestMatch[1],'PROVIDER_INTEGRITY_MISMATCH');
console.log('T264_PROVIDER_INTEGRITY_BEFORE_K_LOAD=PASS');
assert.ok(sealed.routes.some(r=>r.intentId==='reviewSingleCandidate'),
  'reviewSingleCandidate route required in the sealed BUILD');

const replay=process.env.TALLY_LOCAL_REPLAY==='1';
const token=process.env.TALLY_GITHUB_READ_TOKEN;
const sourceSha=process.env.TALLY_CLAIM_SOURCE_SHA;
assert.ok(typeof token==='string'&&token.length>=8);
assert.match(sourceSha??'',/^[0-9a-f]{40}$/);
const issueNumber=Number(process.env.TALLY_LOCAL_ISSUE_NUMBER??264);
assert.ok(Number.isSafeInteger(issueNumber)&&issueNumber>0);
// The sealed shim's issue allowlist evaluates at sealed-module load; pin the
// lane's issue before the K facade loads the sealed app.
process.env.TALLY_LOCAL_ISSUE_NUMBER=String(issueNumber);
const issueRef=`https://github.com/kaicreator-mm/tally/issues/${issueNumber}`;
const decisionId=process.env.TALLY_REVIEW_DECISION_ID??'decision:t264:j5:1';
const decision=process.env.TALLY_REVIEW_DECISION??'ACCEPT';
assert.ok(['ACCEPT','REWORK','DEFER'].includes(decision));
const attemptId=process.env.TALLY_LOCAL_ATTEMPT_ID??'attempt:264:e1:r91:4';
const candidateCommit=process.env.TALLY_REVIEW_CANDIDATE_COMMIT??'023db3aa9a6c30ef5975c6f085ba207ec315cb09';
const candidateTree=process.env.TALLY_REVIEW_CANDIDATE_TREE??'ff7cd2b35ea65d6c7ab316b963eed4066af455db';
const repairCommit=process.env.TALLY_REVIEW_REPAIR_COMMIT??'d9fbc0239a8c38daae0c5637451f1e6b10fce983';
const repairTree=process.env.TALLY_REVIEW_REPAIR_TREE??'cb761f739b2fd2bc737f68ad19e4a0b41c63c023';
const prNumber=Number(process.env.TALLY_REVIEW_PR_NUMBER??466);
assert.ok(Number.isSafeInteger(prNumber)&&prNumber>0);
const SHA_RE=/^[0-9a-f]{40}$/;
assert.match(candidateCommit,SHA_RE);assert.match(candidateTree,SHA_RE);
assert.match(repairCommit,SHA_RE);assert.match(repairTree,SHA_RE);

const claimKey=(role)=>`kaicreator-mm/tally#${issueNumber}:${role}:__default__`;
const registryKeyOf=(claim)=>'tally:v2:claim-admission:'+sha(claim);
const writeSetKey=`tally:v2:task-writeset:${issueNumber}`;
const claimRow=(pins,role)=>({phase:'PUBLISHED_VERIFIED',generation:1,
  protectedClaimKey:claimKey(role),repository:'kaicreator-mm/tally',issueRef,
  sourceSha,actorRole:role,operatorKind:pins.operatorKind,operatorId:pins.operatorId,
  dispatchId:pins.dispatchId,writerId:pins.writerId,commentId:pins.commentId,
  commentRef:`${issueRef}#issuecomment-${pins.commentId}`});
const builderPins=()=>({dispatchId:process.env.TALLY_TASK_CLAIM_264_DISPATCH_ID,
  operatorId:process.env.TALLY_TASK_CLAIM_264_OPERATOR_ID,
  operatorKind:process.env.TALLY_TASK_CLAIM_264_OPERATOR_KIND,
  writerId:process.env.TALLY_TASK_CLAIM_264_WRITER_ID,
  writerLogin:process.env.TALLY_TASK_CLAIM_264_WRITER_LOGIN,
  commentId:Number(process.env.TALLY_TASK_CLAIM_264_COMMENT_ID),
  registryPath:process.env.TALLY_TASK_CLAIM_264_REGISTRY_PATH});
const reviewerPins=()=>({dispatchId:process.env.TALLY_TASK_CLAIM_264_REVIEWER_DISPATCH_ID,
  operatorId:process.env.TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_ID,
  operatorKind:process.env.TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_KIND,
  writerId:process.env.TALLY_TASK_CLAIM_264_REVIEWER_WRITER_ID,
  writerLogin:process.env.TALLY_TASK_CLAIM_264_REVIEWER_WRITER_LOGIN,
  commentId:Number(process.env.TALLY_TASK_CLAIM_264_REVIEWER_COMMENT_ID),
  registryPath:process.env.TALLY_TASK_CLAIM_264_REVIEWER_REGISTRY_PATH});

const evidenceBase=process.env.TALLY_LOCAL_EVIDENCE_DIR;
if(evidenceBase)await mkdir(evidenceBase,{recursive:true});
const tmp=await mkdtemp(join(evidenceBase??tmpdir(),'tally-t264-review-'));
const db=join(tmp,'kernel.sqlite');
let registries=[];
try{
  if(replay){
    // Seed BOTH accepted Claims (builder + independent reviewer) and the
    // frozen write-set in fixture registries, plus the native fetch fixture.
    const writeSetDocument='frozen exact write set: experiments/v2/selected-business-build/LOCAL_CODEX_L3.md';
    process.env.TALLY_LOCAL_WRITESET_SHA256=sha(writeSetDocument);
    process.env.TALLY_TASK_CLAIM_264_DISPATCH_ID ??= 'dispatch:264:e1:integration:host:20261119:5';
    process.env.TALLY_TASK_CLAIM_264_OPERATOR_ID ??= 'zcode:i264:e1:integration:1';
    process.env.TALLY_TASK_CLAIM_264_OPERATOR_KIND ??= 'other';
    process.env.TALLY_TASK_CLAIM_264_WRITER_ID ??= 'tally-host:win32:i264:e1:integration:1';
    process.env.TALLY_TASK_CLAIM_264_WRITER_LOGIN ??= 'kaicreator-mm';
    process.env.TALLY_TASK_CLAIM_264_COMMENT_ID ??= '6996696149';
    process.env.TALLY_TASK_CLAIM_264_REGISTRY_PATH=join(tmp,'builder-claim.sqlite');
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_DISPATCH_ID ??= 'dispatch:264:e1:reviewer:20261119:1';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_ID ??= 'zcode:i264:e1:reviewer:1';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_OPERATOR_KIND ??= 'other';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_WRITER_ID ??= 'tally-host:win32:i264:e1:reviewer:1';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_WRITER_LOGIN ??= 'kaicreator-mm';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_COMMENT_ID ??= '7004539047';
    process.env.TALLY_TASK_CLAIM_264_REVIEWER_REGISTRY_PATH=join(tmp,'reviewer-claim.sqlite');
    for(const [path,row] of [
        [process.env.TALLY_TASK_CLAIM_264_REGISTRY_PATH,claimRow(builderPins(),'builder')],
        [process.env.TALLY_TASK_CLAIM_264_REVIEWER_REGISTRY_PATH,claimRow(reviewerPins(),'reviewer')]]){
      const registry=await openSqliteCasStore({path,busyTimeoutMs:1000});
      await registry.put(registryKeyOf(row.protectedClaimKey),row,{mustBeAbsent:true});
      registries.push(registry);
    }
    const builderRegistry=registries[0];
    await builderRegistry.put(writeSetKey,{phase:'FROZEN_DAG_WRITESET',
      repository:'kaicreator-mm/tally',issueNumber,sourceSha,
      writeSetSha256:process.env.TALLY_LOCAL_WRITESET_SHA256,
      document:writeSetDocument},{mustBeAbsent:true});
    const commentFor=pins=>{const date='2026-11-19T00:00:00Z';
      const body=`<!-- ads:github -->\n\`\`\`yaml\nschema: ads/github/v3\nkind: event\ndata:\n  schema: ai-dev/event-v2\n  event: DISPATCH_CLAIMED\n  actor_role: ${pins.operatorKind==='other'&&pins.operatorId.includes('reviewer')?'reviewer':'builder'}\n  operator_kind: ${pins.operatorKind}\n  operator_id: ${pins.operatorId}\n  task: '#${issueNumber}'\n  issue: '#${issueNumber}'\n  sha: ${sourceSha}\n  dispatch_id: ${pins.dispatchId}\n  dispatch_state: CLAIMED\n  protected_claim_key: ${claimKey(pins.operatorId.includes('reviewer')?'reviewer':'builder')}\n  admission_generation: 1\n\`\`\`\n`;
      return {id:pins.commentId,body,html_url:`${issueRef}#issuecomment-${pins.commentId}`,
        issue_url:`https://api.github.com/repos/kaicreator-mm/tally/issues/${issueNumber}`,
        created_at:date,updated_at:date,user:{login:pins.writerLogin}};};
    const comments=[commentFor(builderPins()),commentFor(reviewerPins())];
    const issue={id:issueNumber*10,number:issueNumber,html_url:issueRef,state:'open',
      updated_at:'2026-11-19T00:00:00Z',
      labels:[{name:'type:task'},{name:'state:ready'}]};
    globalThis.fetch=async input=>{
      const url=new URL(input);let value,status=200;
      if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}`)value=issue;
      else if(url.pathname===`/repos/kaicreator-mm/tally/commits/${sourceSha}`)value={sha:sourceSha};
      else if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}/dependencies/blocked_by`)value=[];
      else if(url.pathname===`/repos/kaicreator-mm/tally/issues/${issueNumber}/comments`)value=comments;
      else if(url.pathname===`/repos/kaicreator-mm/tally/issues/comments/${builderPins().commentId}`)value=comments[0];
      else if(url.pathname===`/repos/kaicreator-mm/tally/issues/comments/${reviewerPins().commentId}`)value=comments[1];
      else if(url.pathname==='/user')value={login:'kaicreator-mm'};
      else{status=404;value={message:'fixture-not-found'};}
      return new Response(JSON.stringify(value),{status,headers:{'content-type':'application/json',
        'x-github-request-id':'t264-review-fixture'}});
    };
  }else{
    // Real lane: both accepted Claim registries must already exist with
    // PUBLISHED_VERIFIED rows (Host-process-only T06d publications).
    for(const pins of [builderPins(),reviewerPins()]){
      assert.ok(pins.dispatchId&&pins.operatorId&&pins.operatorKind&&
        pins.writerId&&pins.writerLogin,'T264_CLAIM_PINS_REQUIRED');
      assert.ok(Number.isSafeInteger(pins.commentId)&&pins.commentId>0,
        'T264_CLAIM_PINS_REQUIRED');
      assert.ok(typeof pins.registryPath==='string'&&pins.registryPath,
        'T264_CLAIM_REGISTRY_PATH_REQUIRED');
      const registry=await openSqliteCasStore({path:pins.registryPath,busyTimeoutMs:1000});
      const row=await registry.get(registryKeyOf(claimKey(
        pins===reviewerPins()?'reviewer':'builder')));
      assert.equal(row?.value?.phase,'PUBLISHED_VERIFIED','T264_CLAIM_NOT_PUBLISHED');
      registries.push(registry);
    }
    assert.ok(process.env.TALLY_LOCAL_BUILDER_STORE_PATH,
      'TALLY_LOCAL_BUILDER_STORE_PATH_REQUIRED');
  }
  process.env.TALLY_LOCAL_HOST_STORE_PATH=db;

  const deployment={deployedAppId:'deploy:tally-v2-t264-review',
    tenantId:'tenant:tally-v2',appId:sealed.appId};
  const authority={principal:{source:'host-authenticated',tenantId:deployment.tenantId},
    authorize:(identity,target)=>identity?.tenantId===target.tenantId&&
      target.businessPackageId==='business-tally-task-readiness@v2.0.0'};
  const condition=c=>c?.kind==='create'?{mustBeAbsent:true}:
    c?.kind==='cas'&&Number.isSafeInteger(c.revision)?{mustMatchRevision:c.revision}:
    (()=>{throw Error('INVALID_K_CONDITION');})();
  const options=sqlite=>({store:{get:key=>sqlite.get(key),
    put:(key,v,c)=>sqlite.put(key,v,condition(c)),
    remove:(key,c)=>sqlite.remove(key,condition(c))},deploymentIdentity:deployment,
    deploymentRoot:process.cwd(),tenantId:deployment.tenantId,queryAuthority:authority,
    now:()=>Date.UTC(2026,10,19)});
  const input={issueRef,issueNumber,sourceSha,decisionId,decision,attemptId,
    candidateCommit,candidateTree,repairCommit,repairTree,prNumber,
    // Always-present scalar (null when unused): the S SDK refuses to project
    // absent payload keys across the K event boundary.
    reviewRef:process.env.TALLY_REVIEW_EVIDENCE_REF??null};
  const message={intentType:'reviewSingleCandidate',
    instanceKey:`task-single-review-${issueNumber}`,
    messageId:'message-t264-review-1',input};
  let sqlite=null,facade=null;
  sqlite=await openSqliteCasStore({path:db,busyTimeoutMs:1000});
  facade=await load(sealed,options(sqlite));
  const receipt=await facade.send(message);
  assert.equal(receipt.status,'admitted',JSON.stringify(receipt).slice(0,1500));
  assert.equal(receipt.admitted?.effects?.length,1);
  const effect=receipt.admitted.effects[0].output;
  if(replay){
    assert.equal(effect.state,'REPLAY_COMPLETE',JSON.stringify(effect));
    assert.equal(effect.replayFixture,true);
    assert.equal(effect.agentLaunch,false);
    assert.equal(effect.taskPass,false);
  }else{
    assert.ok(['REVIEW_DECISION_RECORDED','REVIEW_DECISION_UNKNOWN','NOT_ADMITTED']
      .includes(effect.state),JSON.stringify(effect));
    assert.equal(effect.agentLaunch,false);
    assert.equal(effect.releasePass,false);
    if(effect.state==='REVIEW_DECISION_RECORDED'){
      assert.equal(effect.decision,decision);
      assert.ok(effect.decisionRecord.decisionKey
        .startsWith('tally:v2:single-review-decision:'));
      assert.equal(effect.taskPass,
        effect.taskOutcome?.status==='QUALIFIED_TASK_OUTCOME_PUBLISHED');
    }
  }
  const before=(await facade.query({workflowId:'tally-readiness',
    instanceKey:`task-single-review-${issueNumber}`,
    businessPackageId:'business-tally-task-readiness@v2.0.0'})).snapshot;
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
  assert.deepEqual((await facade.query({workflowId:'tally-readiness',
    instanceKey:`task-single-review-${issueNumber}`,
    businessPackageId:'business-tally-task-readiness@v2.0.0'})).snapshot,before);
  const replayed=await facade.send(message);
  assert.equal(replayed.status,'admitted');
  assert.equal(replayed.idempotentReplay,true);
  assert.deepEqual(await sqlite.list(''),recordsBefore);
  const summary={provider:'local-codex-host',via:'real-sealed-k-b-c',lane:'t264-single-review',
    sourceSha,decisionId,decision,attemptId,candidateCommit,repairCommit,prNumber,
    result:effect.state,reason:effect.reason,taskPass:effect.taskPass??false,
    taskOutcomeStatus:effect.taskOutcome?.status??null,
    decisionKey:effect.decisionRecord?.decisionKey??null,
    replayFixture:replay,reopened:true,idempotentReplay:replayed.idempotentReplay,
    releasePass:false,docCount:recordsBefore.length,hostStorePath:db};
  if(process.env.TALLY_LOCAL_RESULT_PATH)
    writeFileSync(process.env.TALLY_LOCAL_RESULT_PATH,JSON.stringify(summary,null,2)+'\n',{mode:0o600});
  console.log(JSON.stringify(summary));
  console.log(replay?'T264_SINGLE_REVIEW_REPLAY_ONLY_NO_EFFECT=PASS':
    'T264_SINGLE_REVIEW_K_HOST_DECISION=PASS');
  facade.stop();facade=null;await sqlite.close();sqlite=null;
}finally{
  for(const registry of registries)await registry.close().catch(()=>{});
  if(replay||process.env.TALLY_LOCAL_CLEAN==='1')
    await rm(tmp,{recursive:true,force:true}).catch(()=>{});
}
