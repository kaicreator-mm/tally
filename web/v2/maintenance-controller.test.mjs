import test from 'node:test';
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {readFileSync} from 'node:fs';
import {defaultAlmConfig,createJourneyController} from './journey-controller.mjs';
import {createMaintenanceController,renderMaintenanceWorkspace,__test}
  from './maintenance-controller.mjs';
import {createAlmServer} from './journey-server.mjs';

const sourceSha='e'.repeat(40);
const ISSUE_REF='https://github.com/kaicreator-mm/tally/issues/219';
const make=({host,sealedDispatch}={})=>createMaintenanceController(
  {currentSourceSha:sourceSha},{host,sealedDispatch});
const reportArgs=()=>({summary:'Post-release regression in the v1.5.5 importer',
  requirementIssue:'92',releaseVersion:'v1.5.5',
  releaseCommitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02'});
const act=(controller,kind,args)=>controller.dispatch(sourceSha,kind,args);
const seedReviewed=async controller=>{
  await act(controller,'report-defect',reportArgs());
  return act(controller,'review-candidate',{reviewer:'reviewer:ada'});
};

test('initial snapshot: frozen write set, immutable release UNKNOWN, nothing run',()=>{
  const snapshot=make().snapshot();
  assert.equal(snapshot.status,'OBSERVED');assert.equal(snapshot.authorized,false);
  assert.equal(snapshot.priorRelease,'UNKNOWN');
  assert.equal(snapshot.candidate,null);
  assert.equal(snapshot.publication.state,'NOT_RUN');
  assert.equal(snapshot.publication.githubEffect,'NOT_RUN');
  assert.equal(snapshot.learning.length,0,
    'missing task-meta stays UNKNOWN and nonblocking');
  assert.equal(__test.WRITE_SET.length,14);
  assert.equal(snapshot.constraints.writeSetCount,14);
  assert.equal(snapshot.constraints.pinnedDispatch,'dispatch:219:maintenance:20261031:1');
  assert.equal(snapshot.historicalReleaseMutation,false);
  assert.equal(snapshot.nativeWriteAuthorized,false);
  assert.equal(snapshot.taskPass,false);assert.equal(snapshot.releasePass,false);
  assert.equal(JSON.stringify(snapshot).includes('secret'),false);
});

test('ordinary user journey: report defect, independent review, digest stays stable',async()=>{
  const controller=make();
  const noRelease=await act(controller,'report-defect',
    {...reportArgs(),releaseVersion:undefined,releaseCommitSha:undefined});
  assert.equal(noRelease.reason,'PRIOR_RELEASE_IDENTITY_REQUIRED');
  const reported=await act(controller,'report-defect',reportArgs());
  assert.equal(reported.status,'OK');
  assert.match(reported.digest,/^[0-9a-f]{64}$/);
  const snapshot=controller.snapshot();
  assert.equal(snapshot.priorRelease.immutable,true);
  assert.equal(snapshot.priorRelease.version,'v1.5.5');
  assert.equal(snapshot.candidate.state,'DRAFT');
  assert.equal(snapshot.candidate.historicalReleaseMutation,false);
  assert.equal(snapshot.candidate.nextAction,'PRODUCT_REVIEW_OR_AUTHORIZED_INTAKE');
  const selfReview=await act(controller,'review-candidate',{reviewer:'workbench:operator'});
  assert.equal(selfReview.reason,'REVIEW_NOT_INDEPENDENT');
  const reviewed=await seedReviewed(controller);
  assert.equal(reviewed.status,'OK');
  assert.equal(controller.snapshot().review.current,true);
  // A changed draft invalidates the review.
  await act(controller,'report-defect',{...reportArgs(),summary:'Changed summary'});
  assert.equal(controller.snapshot().review,null);
  assert.equal(controller.snapshot().publication.state,'NOT_RUN');
});

test('publish goes only through the sealedDispatch seam; default stays NOT_RUN',async()=>{
  const withoutSeam=make();
  await seedReviewed(withoutSeam);
  const notRun=await act(withoutSeam,'publish-maintenance-issue');
  assert.equal(notRun.state,'NOT_RUN');
  assert.equal(notRun.reason,'SEALED_K_DISPATCH_UNAVAILABLE');
  assert.equal(notRun.githubEffect,'NOT_RUN');assert.equal(notRun.sealedDispatch,'NOT_RUN');
  const calls=[];
  const seam=async request=>{calls.push(request);
    assert.equal(request.intentType,'publishMaintenanceIssue');
    assert.equal(request.input.issueNumber,219);
    assert.equal(request.input.sourceSha,sourceSha);
    return {phase:'terminal',outcome:{ok:true,output:{
      state:'MAINTENANCE_ISSUE_PUBLISHED',
      reason:'MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED',
      issueRef:ISSUE_REF,sourceSha:request.input.sourceSha,
      candidateDigest:request.input.candidateDigest,claimGeneration:1,
      claimAccepted:true,admitted:true,authorized:false,replayFixture:false,
      githubEffect:'PERFORMED_VERIFIED',repository:'kaicreator-mm/tally',
      duplicate:false,alreadyPresent:false,historicalReleaseMutation:false,
      issue:{number:319,title:'[maintenance] regression',
        url:'https://github.com/kaicreator-mm/tally/issues/319'},
      taskPass:false,releasePass:false}}}};
  const withSeam=make({sealedDispatch:seam});
  await seedReviewed(withSeam);
  const published=await act(withSeam,'publish-maintenance-issue');
  assert.equal(published.state,'MAINTENANCE_ISSUE_PUBLISHED');
  assert.equal(published.githubEffect,'PERFORMED_VERIFIED');
  assert.equal(published.sealedDispatch,'VIA_SEAM');
  assert.equal(published.issue.number,319);
  assert.equal(published.historicalReleaseMutation,false);
  assert.deepEqual(calls.map(x=>x.input.issueNumber),[219]);
  const snapshot=withSeam.snapshot();
  assert.equal(snapshot.publication.state,'MAINTENANCE_ISSUE_PUBLISHED');
  assert.equal(snapshot.taskPass,false);assert.equal(snapshot.releasePass,false);
  assert.equal(snapshot.authorized,false);
  // A seam that throws must degrade to NOT_RUN instead of faking an effect.
  const throwing=make({sealedDispatch:async()=>{throw new Error('k down');}});
  await seedReviewed(throwing);
  const failed=await act(throwing,'publish-maintenance-issue');
  assert.equal(failed.state,'NOT_RUN');assert.equal(failed.reason,'SEALED_K_DISPATCH_FAILED');
  assert.equal(failed.sealedDispatch,'SEAM_ERROR');
  // Lost-ACK UNKNOWN passes through as UNKNOWN with a null issue.
  const unknownSeam=make({sealedDispatch:async request=>({phase:'terminal',outcome:{ok:true,
    output:{state:'PUBLICATION_UNKNOWN',reason:'GITHUB_PUBLISH_UNKNOWN',
      issueRef:ISSUE_REF,sourceSha,candidateDigest:request.input.candidateDigest,
      claimGeneration:1,claimAccepted:true,admitted:true,
      authorized:false,replayFixture:false,githubEffect:'UNKNOWN',issue:null,
      historicalReleaseMutation:false,duplicate:false,taskPass:false,releasePass:false}}})});
  await seedReviewed(unknownSeam);
  const unknown=await act(unknownSeam,'publish-maintenance-issue');
  assert.equal(unknown.state,'PUBLICATION_UNKNOWN');
  assert.equal(unknown.githubEffect,'UNKNOWN');assert.equal(unknown.issue,null);
});

test('publish requires the current independent review and exact source',async()=>{
  const controller=make();
  assert.equal((await act(controller,'publish-maintenance-issue')).reason,
    'NO_CANDIDATE_TO_PUBLISH');
  await act(controller,'report-defect',reportArgs());
  assert.equal((await act(controller,'publish-maintenance-issue')).reason,'REVIEW_REQUIRED');
  await act(controller,'review-candidate',{reviewer:'reviewer:ada'});
  const stale=await controller.dispatch('0'.repeat(40),'publish-maintenance-issue',{});
  assert.equal(stale.status,'STALE_SOURCE');
  // A changed draft invalidates the recorded review entirely.
  await act(controller,'report-defect',{...reportArgs(),summary:'Second look'});
  assert.equal(controller.snapshot().review,null);
  assert.equal((await act(controller,'publish-maintenance-issue')).reason,'REVIEW_REQUIRED');
});

test('Task Learning loads only through the Host seam with provenance intact',async()=>{
  const plain=make();
  const none=await act(plain,'load-learning');
  assert.equal(none.status,'NOT_RUN');assert.equal(none.reason,'HOST_GITHUB_READ_REQUIRED');
  assert.equal(plain.snapshot().learning.length,0);
  const suggestion={learningId:'f'.repeat(64),
    taskIssueRef:'https://github.com/kaicreator-mm/tally/issues/172',
    targetIssueRef:'https://github.com/kaicreator-mm/tally/issues/203',
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'rep-1',revision:1,kind:'EXECUTION_SUMMARY',
      commentId:6540000001,issueNumber:203,namespace:'tally',
      publicationRef:'https://github.com/kaicreator-mm/tally/issues/203#issuecomment-6540000001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const withHost=make({host:{readTaskLearning:async()=>({suggestions:[suggestion]})}});
  const loaded=await act(withHost,'load-learning');
  assert.equal(loaded.status,'OK');assert.equal(loaded.suggestions,1);
  const row=withHost.snapshot().learning[0];
  assert.equal(row.qualification,'REPORTED_QUALIFIED');
  assert.equal(row.provenance.reportId,'rep-1');
  assert.equal(row.decision,null);assert.equal(row.measuredEffect,'NOT_MEASURED');
  assert.equal(row.noAgentRanking,true);assert.equal(row.qualityGainClaimed,false);
});

test('learning use/reject and outcome record only through the sealed seam',async()=>{
  const withoutSeam=make({host:{readTaskLearning:async()=>({suggestions:[]})}});
  await act(withoutSeam,'load-learning');
  const notRun=await act(withoutSeam,'record-learning',
    {learningId:'f'.repeat(64),decision:'ADOPTED'});
  assert.equal(notRun.state,'NOT_RUN');
  assert.equal(notRun.reason,'SEALED_K_DISPATCH_UNAVAILABLE');
  const envelopes=[];
  const seam=make({sealedDispatch:async request=>{
    envelopes.push(request);
    const isOutcome=request.input.outcomeAttested!==null;
    return {phase:'terminal',outcome:{ok:true,output:{
      state:'LEARNING_RECORDED',
      reason:isOutcome?'LEARNING_OUTCOME_RECORDED_IN_HOST_CAS':
        'LEARNING_DECISION_RECORDED_IN_HOST_CAS',
      issueRef:ISSUE_REF,sourceSha:request.input.sourceSha,
      learningId:request.input.learningId,decision:request.input.decision,
      claimGeneration:1,claimAccepted:true,admitted:true,authorized:false,
      replayFixture:false,githubEffect:'NOT_RUN',
      measuredEffect:isOutcome?'ATTESTED_OUTCOME_ONLY':'NOT_MEASURED',
      ...(isOutcome?{outcome:{attested:true,measuredEffect:'ATTESTED_OUTCOME_ONLY'}}:{}),
      noAgentRanking:true,qualityGainClaimed:false,taskPass:false,releasePass:false}}};
  }});
  const decision=await act(seam,'record-learning',
    {learningId:'f'.repeat(64),decision:'ADOPTED'});
  assert.equal(decision.state,'LEARNING_RECORDED');
  assert.equal(decision.decision,'ADOPTED');
  assert.equal(decision.measuredEffect,'NOT_MEASURED');
  const outcome=await act(seam,'record-learning',{learningId:'f'.repeat(64),
    decision:'ADOPTED',outcome:{attested:true,observedAt:'2026-11-02T10:00:00.000Z',
      note:'independent follow-up readback'}});
  assert.equal(outcome.outcome.measuredEffect,'ATTESTED_OUTCOME_ONLY');
  assert.equal(outcome.githubEffect,'NOT_RUN');
  assert.deepEqual(envelopes.map(x=>x.intentType),
    ['recordTaskLearningOutcome','recordTaskLearningOutcome']);
  const badDecision=await act(seam,'record-learning',
    {learningId:'f'.repeat(64),decision:'RANK_FIRST'});
  assert.equal(badDecision.status,'BLOCKED');
  const badOutcome=await act(seam,'record-learning',{learningId:'f'.repeat(64),
    decision:'ADOPTED',outcome:{attested:'yes',observedAt:'x',note:'n'}});
  assert.equal(badOutcome.status,'BLOCKED');
});

test('invalid args and unknown kinds refuse without touching state',async()=>{
  const controller=make();
  for(const [kind,args] of [
    ['report-defect',{summary:''}],
    ['report-defect',{summary:'x'.repeat(513),requirementIssue:'1',
      releaseVersion:'v1.0.0',releaseCommitSha:'a'.repeat(40)}],
    ['report-defect',{...reportArgs(),requirementIssue:'0'}],
    ['report-defect',{...reportArgs(),releaseCommitSha:'nope'}],
    ['review-candidate',{}],
    ['record-learning',{learningId:'zz',decision:'ADOPTED'}],
  ]){
    const result=await controller.dispatch(sourceSha,kind,args);
    assert.equal(result.status,'BLOCKED',kind);
  }
  assert.equal(controller.snapshot().candidate,null);
  const unknown=await controller.dispatch(sourceSha,'delete-history',{});
  assert.equal(unknown.status,'DENIED');assert.equal(unknown.reason,'UNKNOWN_ACTION');
  const stale=await controller.dispatch('0'.repeat(40),'report-defect',reportArgs());
  assert.equal(stale.status,'STALE_SOURCE');
});

test('constructor rejects an invalid current source or author',()=>{
  assert.throws(()=>createMaintenanceController({currentSourceSha:'bad'}),
    /CURRENT_SOURCE_SHA_REQUIRED/);
  assert.throws(()=>createMaintenanceController({currentSourceSha:sourceSha,
    author:'bad author!'}),
  /AUTHOR_IDENTITY_INVALID/);
});

test('rendered workspace keeps the accessibility and no-JSON contract',()=>{
  const controller=make();
  const empty=renderMaintenanceWorkspace(controller.snapshot(),{mode:'write'});
  assert.match(empty,/aria-label="Tally v2 maintenance workbench"/);
  assert.doesNotMatch(empty,/onclick|onerror|onsubmit/i);
  assert.match(empty,/UNKNOWN/);
  assert.match(empty,/Missing task-meta stays UNKNOWN and nonblocking/);
});

test('same-origin HTTP maintenance actions enforce CSRF, exact SHA and sanitized args',async()=>{
  const config=defaultAlmConfig({dhPath:'/dh',tallyPath:'/tally',tallyBPath:'/tally',
    githubToken:'secret-token',currentSourceSha:sourceSha,workRoot:'C:/work'});
  const journey=createJourneyController(config,{run:async()=>({status:'FAIL',
    exitCode:1,result:null,evidenceDir:'C:/evidence'})});
  const app=createAlmServer(config,{run:journey.start,
    intakeJourneySnapshot:journey.snapshot});
  const {origin,actionKey}=await app.listen();
  const post=(path,payload,headers={})=>fetch(origin+path,{method:'POST',
    headers:{'content-type':'application/json',origin,'x-tally-action-key':actionKey,...headers},
    body:JSON.stringify(payload)});
  try{
    const page=await fetch(origin+'/journey-view.html').then(r=>r.text());
    assert.match(page,/id="maintenance-app"/);
    assert.match(page,/aria-label="Governed maintenance workbench \(J08b–J08c\)"/);
    assert.match(page,/<script defer src="\.\/maintenance-bootstrap\.mjs"><\/script>/);
    for(const name of ['/maintenance-client.mjs','/maintenance-bootstrap.mjs']){
      const source=await fetch(origin+name).then(r=>r.text());
      const syntax=spawnSync(process.execPath,['--input-type=module','--check'],
        {input:source,encoding:'utf8'});
      assert.equal(syntax.status,0,`${name}: ${syntax.stderr}`);
    }
    const denied=await post('/api/maintenance/actions',{kind:'report-defect',sourceSha},
      {'x-tally-action-key':'wrong'});
    assert.equal(denied.status,403);
    const stale=await post('/api/maintenance/actions',{kind:'report-defect',
      sourceSha:'0'.repeat(40),args:reportArgs()});
    assert.equal(stale.status,409);
    const unknown=await post('/api/maintenance/actions',{kind:'nuke',sourceSha});
    assert.equal(unknown.status,400);
    const badArgs=await post('/api/maintenance/actions',{kind:'report-defect',sourceSha,
      args:{...reportArgs(),evil:{deep:true}}});
    assert.equal(badArgs.status,400);
    const badOutcome=await post('/api/maintenance/actions',{kind:'record-learning',
      sourceSha,args:{learningId:'f'.repeat(64),decision:'ADOPTED',
        outcome:{attested:'yes'}}});
    assert.equal(badOutcome.status,400);
    const reported=await post('/api/maintenance/actions',{kind:'report-defect',
      sourceSha,args:reportArgs()});
    assert.equal(reported.status,200);
    const reviewed=await post('/api/maintenance/actions',{kind:'review-candidate',
      sourceSha,args:{reviewer:'reviewer:ada'}});
    assert.equal(reviewed.status,200);
    const snap=await fetch(origin+'/api/maintenance/snapshot').then(r=>r.json());
    assert.equal(snap.snapshot.review.reviewer,'reviewer:ada');
    assert.equal(snap.lastResult.kind,'review-candidate');
    assert.match(snap.html,/reviewer:ada/);
    assert.match(snap.html,/J08b/);
    assert.doesNotMatch(snap.html,/onclick|onerror/i);
    assert.equal(JSON.stringify(snap).includes('secret-token'),false);
  }finally{await app.close();}
});

test('static page and rendered view keep the no-JSON and accessibility contract',()=>{
  const html=readFileSync(new URL('./journey-view.html',import.meta.url),'utf8');
  assert.doesNotMatch(html,/<button/i);assert.doesNotMatch(html,/onclick|onerror|onsubmit/i);
});
