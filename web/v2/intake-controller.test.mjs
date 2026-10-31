import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {defaultAlmConfig,createJourneyController} from './journey-controller.mjs';
import {createIntakeController,__test} from './intake-controller.mjs';
import {createAlmServer} from './journey-server.mjs';
import {renderIntakeWorkspace} from './intake-view.mjs';

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

test('ordinary user journey J01→J03: guided forms, draft, independent review, freeze, DONE',async()=>{
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
  assert.equal(reviewed.status,'OK');assert.equal(reviewed.independence,'REVIEWER_DIFFERS_FROM_AUTHOR');
  const frozen=await goalSha(controller,{},'freeze-draft');
  assert.equal(frozen.status,'OK');assert.equal(frozen.githubEffect,'NOT_RUN');
  const frozenSnapshot=controller.snapshot();
  assert.equal(frozenSnapshot.frozen.state,'FROZEN');
  assert.match(frozenSnapshot.frozen.receiptRef,/^receipt:sha256:[0-9a-f]{64}$/);
  assert.equal(frozenSnapshot.guided.decision.status,'DONE');
  assert.equal(frozenSnapshot.frozen.authority.includes('NOT_RUN'),true);
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
  assert.equal((await goalSha(controller,{},'freeze-draft')).status,'OK');
  const stale=await controller.dispatch('0'.repeat(40),'freeze-draft',{});
  assert.equal(stale.status,'STALE_SOURCE');
});

test('admit goes only through the sealedDispatch seam; default stays NOT_RUN',async()=>{
  const withoutSeam=make();
  await seedFrozen(withoutSeam);
  const notRun=await withoutSeam.dispatch(sourceSha,'admit-project-intake',
    {projectTitle:'Tally v2 Intake'});
  assert.equal(notRun.state,'NOT_RUN');assert.equal(notRun.reason,'SEALED_K_DISPATCH_UNAVAILABLE');
  assert.equal(notRun.githubEffect,'NOT_RUN');assert.equal(notRun.sealedDispatch,'NOT_RUN');
  const calls=[];
  // The seam receives exactly the K consumer envelope contract
  // ({phase:'terminal',outcome:{ok,output}}); the controller has no other
  // privileged path — the host fixture below deliberately has no
  // hostProjectIntake member at all.
  const seam=async request=>{calls.push(request);
    const {intentType,input}=request;
    assert.equal(intentType,'admitProjectIntake');
    return {phase:'terminal',outcome:{ok:true,output:{
      state:'REPLAY_COMPLETE',reason:'REPLAY_FIXTURE_ONLY',
      issueRef:ISSUE_REF,sourceSha:input.sourceSha,productDigest:input.productDigest,
      claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      replayFixture:true,githubEffect:'NOT_RUN',project:null,
      taskPass:false,releasePass:false}}}};
  const withSeam=make({sealedDispatch:seam});
  await seedFrozen(withSeam);
  const replayed=await withSeam.dispatch(sourceSha,'admit-project-intake',
    {projectTitle:'Tally v2 Intake'});
  assert.equal(replayed.state,'REPLAY_COMPLETE');
  assert.equal(replayed.githubEffect,'NOT_RUN');
  assert.equal(replayed.sealedDispatch,'VIA_SEAM');
  assert.deepEqual(calls.map(x=>x.input.issueNumber),[210]);
  assert.equal(calls[0].input.sourceSha,sourceSha);
  const snapshot=withSeam.snapshot();
  assert.equal(snapshot.intake.state,'REPLAY_COMPLETE');
  assert.equal(snapshot.taskPass,false);assert.equal(snapshot.releasePass,false);
  assert.equal(snapshot.authorized,false);
  // A seam that throws must degrade to NOT_RUN instead of faking an effect.
  const throwing=make({sealedDispatch:async()=>{throw new Error('k down');}});
  await seedFrozen(throwing);
  const failed=await throwing.dispatch(sourceSha,'admit-project-intake',
    {projectTitle:'Tally v2 Intake'});
  assert.equal(failed.state,'NOT_RUN');assert.equal(failed.reason,'SEALED_K_DISPATCH_FAILED');
  assert.equal(failed.sealedDispatch,'SEAM_ERROR');
});
async function seedFrozen(controller){
  await goalSha(controller,{goal:'Goal'});
  await goalSha(controller,{refs:['157','158']},'select-evidence');
  await goalSha(controller,{},'confirm-inspect');
  await goalSha(controller,{},'draft-product');
  await goalSha(controller,{reviewer:'reviewer:ada'},'review-draft');
  await goalSha(controller,{},'freeze-draft');
}

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
  }finally{await app.close();}
});

test('static page and rendered intake view keep the no-JSON and accessibility contract',()=>{
  const html=readFileSync(new URL('./journey-view.html',import.meta.url),'utf8');
  assert.doesNotMatch(html,/<button/i);assert.doesNotMatch(html,/onclick|onerror|onsubmit/i);
  assert.match(html,/aria-label="Guided intake workbench \(J01–J03\)"/);
});
