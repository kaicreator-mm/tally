import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {defaultAlmConfig,createJourneyController} from './journey-controller.mjs';
import {createIntakeController,__test} from './intake-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {renderIntakeWorkspace} from './intake-view.mjs';
import {createProductReviewHost312,PRODUCT312_DRAFT_KEY,
  PRODUCT312_FREEZE_KEY} from '../../experiments/v2/project-intake/host-project.mjs';
import {readProductReviewWitnessFromClient}
  from '../../packages/tally-github-c/src/project-intake.mjs';

const sourceSha='d'.repeat(40);
const ISSUE_REF='https://github.com/kaicreator-mm/tally/issues/210';
const tasks=[
  {sourceRef:'https://github.com/kaicreator-mm/tally/issues/196',sourceSha,
    observedAsOf:'2026-10-31T08:00:00.000Z',currentness:'OBSERVED_AT_QUERY_COMPLETION',
    status:'AWAITING_HOST_ADMISSION',reason:'NATIVE_READY_NO_ACCEPTED_CLAIM',
    dependencyState:'CLEAR',claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',
    agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',ciState:'NOT_VERIFIED',
    hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
    authorized:false,admitted:false,taskPass:false,releasePass:false},
  {sourceRef:'https://github.com/kaicreator-mm/tally/issues/177',sourceSha,
    observedAsOf:'2026-10-31T08:00:00.000Z',currentness:'OBSERVED_AT_QUERY_COMPLETION',
    status:'BLOCKED',reason:'NATIVE_DEPENDENCIES_BLOCKED',dependencyState:'BLOCKED',
    claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',
    reviewState:'NOT_VERIFIED',ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',
    releaseState:'NOT_VERIFIED',authorized:false,admitted:false,taskPass:false,releasePass:false}];
const journeySnapshot=()=>({status:'OBSERVED',project:'tally',version:'v2',sourceSha,
  observedAsOf:'2026-10-31T08:00:00.000Z',currentness:'OBSERVED_AT_QUERY_COMPLETION',
  tasks,authorized:false,admitted:false,taskPass:false,releasePass:false});
const make=({host,sealedDispatch}={})=>createIntakeController({currentSourceSha:sourceSha},
  {host,sealedDispatch,journeySnapshot});
const goalSha=(controller,args,kind='submit-goal')=>controller.dispatch(sourceSha,kind,args);

test('initial snapshot: real sources pinned, live facts NOT_RUN, DAG lanes with native claim facts',()=>{
  const snapshot=make().snapshot();
  assert.equal(snapshot.status,'OBSERVED');assert.equal(snapshot.authorized,false);
  assert.equal(snapshot.sources.length,3);
  assert.deepEqual(snapshot.sources.map(s=>s.issueNumber),[157,158,210]);
  for(const source of snapshot.sources)
    assert.equal(source.fact.status,'NOT_RUN');
  assert.equal(snapshot.guided.decision.status,'NEED_EVIDENCE');
  assert.equal(snapshot.guided.decision.reasonCode,'GOAL_REQUIRED');
  const intake=snapshot.dag.lanes.at(-1);
  assert.equal(intake.lane,'j01-intake-workbench');
  assert.equal(intake.writeSetCount,__test.WRITE_SET.length);
  assert.equal(__test.WRITE_SET.length,16);
  assert.equal(snapshot.dag.blockedCount,1,'real blocked native task is presented');
  assert.equal(snapshot.dag.cycleCount,0);
  assert.equal(snapshot.intake.state,'NOT_RUN');
  assert.equal(snapshot.intake.githubEffect,'NOT_RUN');
  assert.equal(snapshot.constraints.pinnedDispatch,'dispatch:210:v2:20261031:1');
  assert.equal(JSON.stringify(snapshot).includes('secret'),false);
});

test('guided Product draft remains unqualified until trusted Host ReviewWitness',async()=>{
  const controller=make();
  assert.equal((await goalSha(controller,{goal:'Ship the guided intake workbench'})).status,'OK');
  assert.equal(controller.snapshot().guided.decision.reasonCode,'EVIDENCE_REQUIRED');
  assert.equal((await goalSha(controller,{refs:['157','158']},'select-evidence')).status,'OK');
  const inspect=controller.snapshot().guided.decision;
  assert.equal(inspect.status,'STEP');assert.equal(inspect.step.kind,'INSPECT');
  assert.equal((await goalSha(controller,{},'confirm-inspect')).status,'OK');
  assert.equal(controller.snapshot().guided.decision.step.kind,'SUMMARY');
  const drafted=await goalSha(controller,{nonGoals:['No raw JSON anywhere'],l2Decision:'KEEP_MERGED_L2_ARCHITECTURE'},'draft-product');
  assert.equal(drafted.status,'OK');assert.equal(drafted.conflictCount,0);
  const draft=controller.snapshot().draft;
  assert.equal(draft.state,'CANDIDATE');assert.equal(draft.product.goal,'Ship the guided intake workbench');
  assert.match(draft.digest,/^[0-9a-f]{64}$/);
  const selfReview=await goalSha(controller,{reviewer:'workbench:operator'},'review-draft');
  assert.equal(selfReview.status,'BLOCKED');assert.equal(selfReview.reason,'REVIEW_NOT_INDEPENDENT');
  const reviewed=await goalSha(controller,{reviewer:'reviewer:ada'},'review-draft');
  assert.equal(reviewed.status,'OK');
  assert.equal(reviewed.independence,
    'NOT_VERIFIED_REQUIRES_ACCEPTED_REVIEWER_CLAIM_AND_GITHUB_WITNESS');
  const frozen=await goalSha(controller,{},'freeze-draft');
  assert.equal(frozen.status,'BLOCKED');
  assert.equal(frozen.reason,'HOST_PRODUCT_REVIEW_WITNESS_REQUIRED');
  assert.equal(frozen.githubEffect,'NOT_RUN');
  const snapshot=controller.snapshot();
  assert.equal(snapshot.frozen,null,'no UI-minted Host CAS freeze receipt');
  assert.equal(snapshot.review.state,'REVIEW_PROPOSED_UNVERIFIED');
  assert.equal(snapshot.reviewBlock.state,'NOT_VERIFIED');
  assert.notEqual(snapshot.guided.decision.status,'DONE',
    'no guided EFFECT_RECEIPT/DONE without trusted Host');
});

test('model or author cannot self-freeze: draft change invalidates a recorded review',async()=>{
  const controller=make();
  await goalSha(controller,{goal:'Goal A'});
  await goalSha(controller,{refs:['210']},'select-evidence');
  await goalSha(controller,{},'confirm-inspect');
  await goalSha(controller,{},'draft-product');
  const freezeWithoutReview=await goalSha(controller,{},'freeze-draft');
  assert.equal(freezeWithoutReview.reason,'REVIEW_REQUIRED');
  await goalSha(controller,{reviewer:'reviewer:ada'},'review-draft');
  await goalSha(controller,{goal:'Goal B changed by an advisor'},'draft-product');
  const staleFreeze=await goalSha(controller,{},'freeze-draft');
  assert.equal(staleFreeze.reason,'REVIEW_REQUIRED');
  const snapshot=controller.snapshot();
  assert.equal(snapshot.review,null);assert.equal(snapshot.frozen,null);
  const admitBlocked=await controller.dispatch(sourceSha,'admit-project-intake',
    {projectTitle:'Tally v2 Intake'});
  assert.equal(admitBlocked.reason,'PRODUCT_FREEZE_REQUIRED');
});

test('freeze requires the guided summary step and current source',async()=>{
  const controller=make();
  await goalSha(controller,{goal:'G'});
  await goalSha(controller,{refs:['157']},'select-evidence');
  await goalSha(controller,{},'draft-product');
  await goalSha(controller,{reviewer:'r1'},'review-draft');
  const early=await goalSha(controller,{},'freeze-draft');
  assert.equal(early.reason,'GUIDED_SUMMARY_NOT_READY');
  await goalSha(controller,{},'confirm-inspect');
  const withheld=await goalSha(controller,{},'freeze-draft');
  assert.equal(withheld.status,'BLOCKED');
  assert.equal(withheld.reason,'HOST_PRODUCT_REVIEW_WITNESS_REQUIRED');
  const stale=await controller.dispatch('0'.repeat(40),'freeze-draft',{});
  assert.equal(stale.status,'STALE_SOURCE');
});

test('unverified UI reviewer cannot dispatch Project intake even through sealed K',async()=>{
  const dispatched=[];
  const controller=make({sealedDispatch:async request=>{
    dispatched.push(request);throw new Error('unexpected privileged effect');
  }});
  await goalSha(controller,{goal:'Goal'});
  await goalSha(controller,{refs:['157','158']},'select-evidence');
  await goalSha(controller,{},'confirm-inspect');
  await goalSha(controller,{},'draft-product');
  await goalSha(controller,{reviewer:'reviewer:ada'},'review-draft');
  assert.equal((await goalSha(controller,{},'freeze-draft')).reason,
    'HOST_PRODUCT_REVIEW_WITNESS_REQUIRED');
  const denied=await controller.dispatch(sourceSha,'admit-project-intake',
    {projectTitle:'Tally v2 Intake'});
  assert.equal(denied.status,'BLOCKED');
  assert.equal(denied.reason,'PRODUCT_FREEZE_REQUIRED');
  assert.deepEqual(dispatched,[],'no K intent without authenticated Host Freeze');
  assert.equal(controller.snapshot().frozen,null);
  assert.equal(controller.snapshot().intake.state,'NOT_RUN');
  assert.equal(controller.snapshot().taskPass,false);
});

test('refresh-source-facts reads only through an injected Host reader',async()=>{
  const plain=make();
  const none=await plain.dispatch(sourceSha,'refresh-source-facts',{});
  assert.equal(none.status,'NOT_RUN');assert.equal(none.reason,'HOST_GITHUB_READ_REQUIRED');
  const host={readSourceFacts:async()=>({'157':{status:'PROJECT_INTAKE_OBSERVED',
    reason:'CANONICAL_GITHUB_FACTS_OBSERVED',observedAsOf:'2026-10-31T10:00:00.000Z',
    currentness:'OBSERVED_AT_QUERY_COMPLETION'}})};
  const withHost=make({host});
  await withHost.dispatch(sourceSha,'refresh-source-facts',{});
  const snapshot=withHost.snapshot();
  assert.equal(snapshot.sources[0].fact.status,'PROJECT_INTAKE_OBSERVED');
  assert.equal(snapshot.sources[1].fact.status,'NOT_RUN','unobserved facts stay NOT_RUN');
  assert.equal(snapshot.sources[2].fact.status,'NOT_RUN');
});

test('invalid args and unknown kinds refuse without touching state',async()=>{
  const controller=make();
  for(const [kind,args] of [
    ['submit-goal',{goal:''}],['submit-goal',{goal:'x'.repeat(257)}],
    ['select-evidence',{refs:[]}],['select-evidence',{refs:['999']}],
    ['draft-product',{nonGoals:'nope'}],['draft-product',{l2Decision:'FREE_FOR_ALL'}],
    ['review-draft',{}],['admit-project-intake',{projectTitle:''}],
  ]){
    const result=await controller.dispatch(sourceSha,kind,args);
    assert.equal(result.status,'BLOCKED',kind);
  }
  const unknown=await controller.dispatch(sourceSha,'delete-everything',{});
  assert.equal(unknown.status,'DENIED');assert.equal(unknown.reason,'UNKNOWN_ACTION');
  const stale=await controller.dispatch('0'.repeat(40),'submit-goal',{goal:'x'});
  assert.equal(stale.status,'STALE_SOURCE');
  assert.equal(controller.snapshot().draft,null);
});

test('constructor rejects an invalid current source',()=>{
  assert.throws(()=>createIntakeController({currentSourceSha:'bad'}),/CURRENT_SOURCE_SHA_REQUIRED/);
});

test('same-origin HTTP intake actions enforce CSRF, exact SHA and sanitized args',async()=>{
  const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',
    githubToken:'secret-token',currentSourceSha:sourceSha,workRoot:'C:/work'});
  const journey=createJourneyController(config,{run:async()=>({status:'FAIL',exitCode:1,
    result:null,evidenceDir:'C:/evidence'})});
  const app=createAlmServer(config,{run:journey.start,intakeJourneySnapshot:journeySnapshot});
  const {origin,actionKey}=await app.listen();
  const post=(path,payload,headers={})=>fetch(origin+path,{method:'POST',
    headers:{'content-type':'application/json',origin,'x-tally-action-key':actionKey,...headers},
    body:JSON.stringify(payload)});
  try{
    const page=await fetch(origin+'/journey-view.html').then(r=>r.text());
    assert.match(page,/id="intake-app"/);
    assert.match(page,/<script defer src="\.\/journey-bootstrap\.mjs"><\/script>/);
    assert.match(page,/<script defer src="\.\/intake-bootstrap\.mjs"><\/script>/);
    for(const name of ['/intake-client.mjs','/intake-bootstrap.mjs','/intake-view.mjs']){
      const source=await fetch(origin+name).then(r=>r.text());
      const syntax=spawnSync(process.execPath,['--input-type=module','--check'],
        {input:source,encoding:'utf8'});
      assert.equal(syntax.status,0,`${name}: ${syntax.stderr}`);
    }
    const denied=await post('/api/intake/actions',{kind:'submit-goal',sourceSha},
      {'x-tally-action-key':'wrong'});
    assert.equal(denied.status,403);
    const stale=await post('/api/intake/actions',{kind:'submit-goal',
      sourceSha:'0'.repeat(40),args:{goal:'x'}});
    assert.equal(stale.status,409);
    const unknown=await post('/api/intake/actions',{kind:'nuke',sourceSha});
    assert.equal(unknown.status,400);
    const badArgs=await post('/api/intake/actions',{kind:'submit-goal',sourceSha,
      args:{goal:'ok',evil:{deep:true}}});
    assert.equal(badArgs.status,400);
    const goal=await post('/api/intake/actions',{kind:'submit-goal',sourceSha,
      args:{goal:'HTTP goal'}});
    assert.equal(goal.status,200);
    const snap=await fetch(origin+'/api/intake/snapshot').then(r=>r.json());
    assert.equal(snap.snapshot.guided.goals[0],'HTTP goal');
    assert.equal(snap.lastResult.kind,'submit-goal');
    const html=renderIntakeWorkspace(snap.snapshot,{mode:'write',
      result:snap.lastResult});
    assert.match(html,/HTTP goal/);assert.match(html,/NOT_RUN/);
    assert.doesNotMatch(html,/onclick|onerror/i);
    assert.equal(JSON.stringify(snap).includes('secret-token'),false);
    // #312: malformed ReviewWitness coordinates are refused at the transport
    // allowlist before any controller or seam work.
    for(const args of [{reviewCommentId:'12345'},{reviewCommentId:0},
      {taskPackBlobSha:'Z'.repeat(40)},{writeSetSha256:'a'.repeat(63)},
      {reviewedHead:null,reviewedTree:'d'.repeat(40),reviewedBase:'f'.repeat(40)}])
      assert.equal((await post('/api/intake/actions',{kind:'freeze-draft',sourceSha,
        args})).status,400,JSON.stringify(args));
    // Well-formed coordinates pass the allowlist; with the served app's no-seam
    // construction the freeze still fail-closes after the guided guards pass.
    const act=(kind,args)=>post('/api/intake/actions',{kind,sourceSha,args});
    await act('select-evidence',{refs:['157','158']});
    await act('confirm-inspect',{});
    await act('draft-product',{nonGoals:['No raw JSON anywhere'],
      l2Decision:'KEEP_MERGED_L2_ARCHITECTURE'});
    await act('review-draft',{reviewer:'reviewer:ada'});
    const witnessAccepted=await act('freeze-draft',{reviewCommentId:12345,
      taskPackBlobSha:'b'.repeat(40),writeSetSha256:'c'.repeat(64),
      reviewedHead:'d'.repeat(40),reviewedTree:'e'.repeat(40),
      reviewedBase:'f'.repeat(40)});
    assert.equal(witnessAccepted.status,200);
    assert.equal((await witnessAccepted.json()).reason,
      'HOST_PRODUCT_REVIEW_WITNESS_REQUIRED');
  }finally{await app.close();}
});

test('static page and rendered intake view keep the no-JSON and accessibility contract',()=>{
  const html=readFileSync(new URL('./journey-view.html',import.meta.url),'utf8');
  assert.doesNotMatch(html,/<button/i);assert.doesNotMatch(html,/onclick|onerror|onsubmit/i);
  assert.match(html,/aria-label="Guided intake workbench \(J01–J03\)"/);
});


// ---- #312 positive Product ReviewWitness wiring: Business → single K seam →
// GitHub C witness reader → Host CAS (persist draft, fenced freeze). ----
const pack312='b'.repeat(40),set312='c'.repeat(64),head312='d'.repeat(40),
  tree312='e'.repeat(40),base312=sourceSha,commentId312=12345;
const reviewRef312='https://github.com/kaicreator-mm/tally/issues/312#issuecomment-12345';
const claim312=role=>({accepted:true,issueNumber:312,role,sourceSha,
  taskPackBlobSha:pack312,writeSetSha256:set312,claimRef:'claim:312:'+role,
  operatorId:'operator:'+role,writerId:'writer:'+role,transportActor:'kaicreator-mm'});
const memoryStore=()=>{const map=new Map();return {
  get:async key=>map.get(key)??null,
  async put(key,value,condition){
    if(condition?.mustBeAbsent&&map.has(key))throw Error('EEXIST');
    map.set(key,{value});},
  async close(){},keys:async()=>[...map.keys()]};};
/** Real GitHub C witness reader over a stub transport whose comment lazily
 * binds whatever digest the Host persisted for the reviewed draft. */
const witnessReader312=comment=>request=>readProductReviewWitnessFromClient({
  octokit:{rest:{
    issues:{getComment:async()=>({data:comment()})},
    pulls:{get:async()=>({data:{number:401,state:'open',merged:false,
      head:{sha:head312,repo:{full_name:'kaicreator-mm/tally'}},
      base:{sha:base312,ref:'version/v2',
        repo:{full_name:'kaicreator-mm/tally'}}}})},
    repos:{
      getCommit:async({ref})=>({data:{sha:ref,commit:{tree:{
        sha:ref===head312?tree312:'b'.repeat(40)}}}}),
      getBranch:async()=>({data:{commit:{sha:sourceSha}}})}}},...request});
const host312=(comment,{store,readCurrentVersion}={})=>createProductReviewHost312({
  openStore:async()=>store??memoryStore(),
  verifyAcceptedClaim:async(issue,role)=>({ok:true,claim:claim312(role)}),
  verifyFrozenWriteSet:async()=>({ok:true,taskPackBlobSha:pack312,
    writeSetSha256:set312}),
  readCurrentVersion:readCurrentVersion??(async()=>({ok:true,sha:sourceSha})),
  readReviewWitness:witnessReader312(comment),
  withVersionFence:async(meta,fn)=>fn(),
  env:{TALLY_CLAIM_SOURCE_SHA:sourceSha,
    TALLY_TASK_CLAIM_312_REGISTRY_PATH:'fixture-312.sqlite'}});
const canonicalWitness312=()=>({id:commentId312,html_url:reviewRef312,
  issue_url:'https://api.github.com/repos/kaicreator-mm/tally/issues/312',
  user:{login:'kaicreator-mm'},created_at:'2026-11-12T12:00:00Z',
  updated_at:'2026-11-12T12:00:00Z',body:JSON.stringify({
    schema:'tally.v2.product-review-witness/2',status:'INDEPENDENTLY_VERIFIED',
    repository:'kaicreator-mm/tally',issueNumber:312,reviewedPullNumber:401,
    draftRef:'draft:sha256:'+bound312Digest,productDigest:bound312Digest,
    sourceSha,taskPackBlobSha:pack312,writeSetSha256:set312,
    reviewedHead:head312,reviewedTree:tree312,reviewedBase:base312,
    builderClaimRef:'claim:312:builder',reviewerClaimRef:'claim:312:reviewer',
    reviewerOperatorId:'operator:reviewer',reviewerWriterId:'writer:reviewer',
    p0:0,p1:0,p2:1})});
let bound312Digest=null;
const witness312Args={reviewCommentId:commentId312,taskPackBlobSha:pack312,
  writeSetSha256:set312,reviewedHead:head312,reviewedTree:tree312,
  reviewedBase:base312};
async function reviewed312Controller(seam){
  const controller=make(seam?{sealedDispatch:seam.seam}:{});
  await goalSha(controller,{goal:'Ship the #312 Product ReviewWitness'});
  await goalSha(controller,{refs:['157','158']},'select-evidence');
  await goalSha(controller,{},'confirm-inspect');
  await goalSha(controller,{nonGoals:['No forged Task PASS'],
    l2Decision:'KEEP_MERGED_L2_ARCHITECTURE'},'draft-product');
  await goalSha(controller,{reviewer:'reviewer:ada'},'review-draft');
  return controller;
}
let dispatched312=[];
function resetSeam312(provider,mutatePersist){
  dispatched312=[];
  return {get dispatched(){return dispatched312;},
    seam:async intent=>{
      dispatched312.push(intent);
      if(intent.intentType==='persistProductDraft312'){
        const out=await provider.persistDraft(intent.input);
        // The independent reviewer binds the Host-persisted digest, not any
        // caller-supplied digest.
        bound312Digest=out.state==='DRAFT_PENDING_REVIEW'?out.productDigest:null;
        return mutatePersist?mutatePersist(out):out;
      }
      if(intent.intentType==='freezeProductReview312')
        return provider.freezeReviewedDraft(intent.input);
      throw Error('unexpected intent '+intent.intentType);
    }};
}
test('#312 positive: seal-persisted draft then witnessed Host freeze returns the only FROZEN receipt',async()=>{
  const store=memoryStore();
  const provider=host312(canonicalWitness312,{store});
  const seam=resetSeam312(provider);
  const controller=await reviewed312Controller(seam);
  const result=await controller.dispatch(sourceSha,'freeze-draft',witness312Args);
  assert.equal(result.status,'OK');
  assert.equal(result.frozen,true);
  assert.equal(result.githubEffect,'NOT_RUN');
  assert.equal(result.receiptRef,'receipt:sha256:'+bound312Digest);
  assert.match(result.receiptRef,/^receipt:sha256:[0-9a-f]{64}$/);
  assert.equal(result.reviewRef,reviewRef312);
  assert.equal(seam.dispatched.length,2);
  assert.equal(seam.dispatched[0].intentType,'persistProductDraft312');
  assert.equal(seam.dispatched[0].input.sourceSha,sourceSha);
  assert.equal(seam.dispatched[0].input.draft.product.goal,'Ship the #312 Product ReviewWitness');
  assert.equal(seam.dispatched[0].input.draft.provenance.join(','),'157,158');
  assert.equal(seam.dispatched[1].intentType,'freezeProductReview312');
  assert.equal(seam.dispatched[1].input.reviewCommentId,commentId312);
  const snapshot=controller.snapshot();
  assert.equal(snapshot.frozen.state,'FROZEN');
  assert.equal(snapshot.frozen.digest,bound312Digest);
  assert.equal(snapshot.frozen.sourceSha,sourceSha);
  assert.equal(snapshot.frozen.receiptRef,result.receiptRef);
  assert.equal(snapshot.frozen.reviewRef,reviewRef312);
  assert.equal(snapshot.reviewBlock,undefined);
  assert.equal(snapshot.guided.decision.status,'DONE');
  assert.equal(snapshot.guided.decision.reasonCode,'OUTCOME_OBSERVED');
  assert.equal(snapshot.authorized,false);assert.equal(snapshot.taskPass,false);
  assert.equal(snapshot.releasePass,false);
  // Host CAS really holds the persisted draft and the frozen record.
  const draft=(await store.get(PRODUCT312_DRAFT_KEY+sourceSha)).value;
  assert.equal(draft.phase,'DRAFT_PENDING_REVIEW');
  assert.equal(draft.productDigest,bound312Digest);
  const frozen=(await store.get(PRODUCT312_FREEZE_KEY+sourceSha)).value;
  assert.equal(frozen.phase,'FROZEN_PRODUCT_INTAKE');
  assert.equal(frozen.reviewRef,reviewRef312);
});
test('#312 freeze without complete witness coordinates dispatches nothing',async()=>{
  const provider=host312(canonicalWitness312);
  const seam=resetSeam312(provider);
  const controller=await reviewed312Controller(seam);
  for(const args of [{},{...witness312Args,reviewCommentId:0},
    {...witness312Args,taskPackBlobSha:'not-a-sha'},
    {...witness312Args,writeSetSha256:'short'},
    {...witness312Args,reviewedHead:null}]){
    const refused=await controller.dispatch(sourceSha,'freeze-draft',args);
    assert.equal(refused.status,'BLOCKED',JSON.stringify(args));
    assert.equal(refused.reason,'HOST_PRODUCT_REVIEW_WITNESS_REQUIRED');
    assert.equal(refused.frozen,false);
  }
  assert.equal(seam.dispatched.length,0);
  assert.equal(controller.snapshot().frozen,null);
});
test('#312 edited witness comment during freeze stays BLOCKED with zero CAS freeze',async()=>{
  const store=memoryStore();
  const provider=host312(()=>({...canonicalWitness312(),
    updated_at:'2026-11-12T12:00:01Z'}),{store});
  const seam=resetSeam312(provider);
  const controller=await reviewed312Controller(seam);
  const refused=await controller.dispatch(sourceSha,'freeze-draft',witness312Args);
  assert.equal(refused.status,'BLOCKED');
  assert.equal(refused.reason,'HOST_PRODUCT_REVIEW_WITNESS_UNVERIFIED');
  assert.equal(refused.frozen,false);
  assert.equal(controller.snapshot().frozen,null);
  assert.notEqual(controller.snapshot().guided.decision.status,'DONE');
  assert.equal(await store.get(PRODUCT312_FREEZE_KEY+sourceSha),null);
  assert.equal(controller.snapshot().taskPass,false);
});
test('#312 persisted-digest mismatch or seam failure never reaches the freeze intent',async()=>{
  const provider=host312(canonicalWitness312);
  const seam=resetSeam312(provider,
    out=>({...out,productDigest:'0'.repeat(64)}));
  const controller=await reviewed312Controller(seam);
  const mismatch=await controller.dispatch(sourceSha,'freeze-draft',witness312Args);
  assert.equal(mismatch.status,'BLOCKED');
  assert.equal(mismatch.reason,'HOST_DRAFT_PERSISTENCE_UNVERIFIED');
  assert.equal(seam.dispatched.length,1,'no freeze intent after bad persistence');
  assert.equal(controller.snapshot().frozen,null);
  const throwing={dispatched:[],seam:async intent=>{
    throwing.dispatched.push(intent);
    if(intent.intentType==='persistProductDraft312'){
      const out=await provider.persistDraft(intent.input);
      bound312Digest=out.productDigest;
      return out;
    }
    throw Error('K seam died mid-freeze');
  }};
  const second=await reviewed312Controller(throwing);
  const seamDeath=await second.dispatch(sourceSha,'freeze-draft',witness312Args);
  assert.equal(seamDeath.status,'BLOCKED');
  assert.equal(seamDeath.reason,'SEALED_K_FREEZE_DISPATCH_FAILED');
  assert.equal(second.snapshot().frozen,null);
  assert.equal(second.snapshot().taskPass,false);
});
