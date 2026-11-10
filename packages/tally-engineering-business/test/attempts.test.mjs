import test from 'node:test';
import assert from 'node:assert/strict';
import {projectTaskAttempts} from '../src/attempts.mjs';
const sha='a'.repeat(40);
const ref='https://github.com/kaicreator-mm/tally/issues/275#issuecomment-1234';
const identity={repository:'kaicreator-mm/tally',versionId:'version/v2',
  taskIssueNumber:275,sourceSha:sha};
const a={...identity,attemptId:'attempt-1',role:'builder',writerId:'host:1',
  workflowAddress:'workflow/v2/attempt-1',packageId:'tally-b',epoch:1,
  environment:'windows-host',worktree:'C:/secret/worktree',receipt:{
    status:'VERIFIED',attemptId:'attempt-1',sourceSha:sha,
    outcome:'COMPLETED',observedAt:'2026-11-10T00:00:00Z',ref}};
const obs={attemptId:'attempt-1',status:'COMPLETE',cursorComplete:true,
  highWatermark:2,observations:[
    {attemptId:'attempt-1',workflowAddress:a.workflowAddress,
      packageId:a.packageId,epoch:1,eventType:'INSTANCE_OPENED',
      sequence:1,observedAt:'2026-11-10T00:00:00Z'},
    {attemptId:'attempt-1',workflowAddress:a.workflowAddress,
      packageId:a.packageId,epoch:1,eventType:'INSTANCE_TERMINALIZED',
      sequence:2,observedAt:'2026-11-10T00:01:00Z'}]};
const make=(o={})=>({...identity,roster:{...identity,status:'VERIFIED',
  complete:true,attempts:[a]},observationPages:[obs],adsReports:[],...o});
test('selected Host observations are advisory, not Task PASS',()=>{
  const r=projectTaskAttempts(make());
  assert.equal(r.status,'CURRENT_OBSERVED');
  assert.equal(r.attempts[0].runtime.runtimeTerminal,true);
  assert.equal(r.attempts[0].hostOutcome.outcome,'COMPLETED');
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
  assert.equal(r.attempts[0].metrics,'UNKNOWN');
  assert.equal(JSON.stringify(r).includes('secret/worktree'),false);
});
test('missing Host roster does not mean zero attempts',()=>{
  assert.equal(projectTaskAttempts({...identity}).reason,
    'HOST_ATTEMPT_INDEX_UNAVAILABLE');
});
test('complete proven empty roster is distinct from missing index',()=>{
  const r=projectTaskAttempts(make({roster:{...identity,status:'VERIFIED',
    complete:true,attempts:[]},observationPages:[]}));
  assert.equal(r.status,'OBSERVED_EMPTY');assert.equal(r.coverage.attemptCount,0);
});
test('gapped/duplicated/out of order observations fail closed',()=>{
  let bad=structuredClone(obs);bad.observations[1].sequence=3;
  assert.equal(projectTaskAttempts(make({observationPages:[bad]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
  assert.equal(projectTaskAttempts(make({observationPages:[obs,obs]})).reason,
    'AMBIGUOUS_OBSERVATION_PAGE');
});
test('retention gap never looks like terminal PASS',()=>{
  const gap={...obs,status:'RETENTION_TRUNCATED',cursorComplete:false};
  const r=projectTaskAttempts(make({observationPages:[gap]}));
  assert.equal(r.status,'PARTIAL');assert.equal(r.attempts[0].runtime.runtimeTerminal,false);
});
test('runtime-only source never fabricates Agent exit',()=>{
  const noReceipt={...a,receipt:undefined};
  const r=projectTaskAttempts(make({roster:{...identity,status:'VERIFIED',
    complete:true,attempts:[noReceipt]}}));
  assert.equal(r.status,'PARTIAL');assert.equal(r.attempts[0].hostOutcome.status,'UNKNOWN');
});
test('wrong Task/repo/SHA and duplicate attempt are denied',()=>{
  assert.equal(projectTaskAttempts(make({sourceSha:'b'.repeat(40)})).reason,
    'HOST_ATTEMPT_INDEX_UNAVAILABLE');
  assert.equal(projectTaskAttempts(make({roster:{...identity,status:'VERIFIED',
    complete:true,attempts:[a,a]}})).reason,'INVALID_OR_DUPLICATE_ATTEMPT_IDENTITY');
});
test('report identity mismatch and self-report cannot become PASS',()=>{
  const report={...identity,attemptId:'attempt-1',status:'CURRENT_OBSERVED',
    reportId:'r-1',revision:1,nativeRef:ref};
  const ok=projectTaskAttempts(make({adsReports:[report]}));
  assert.equal(ok.attempts[0].ads.status,'OBSERVED_ADVISORY');
  assert.equal(ok.taskPass,false);
  const wrong={...report,sourceSha:'b'.repeat(40)};
  assert.equal(projectTaskAttempts(make({adsReports:[wrong]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
});
test('unexpected observation for a foreign attempt is refused',()=>{
  const extraneous={...obs,attemptId:'foreign-attempt'};
  assert.equal(projectTaskAttempts(make({observationPages:[extraneous]})).reason,
    'FOREIGN_ATTEMPT_OBSERVATION');
});

test('empty complete runtime stream is still an incomplete Attempt observation',()=>{
  const empty={...obs,observations:[],highWatermark:0};
  const r=projectTaskAttempts(make({observationPages:[empty]}));
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.attempts[0].runtime.runtimeTerminal,false);
  assert.equal(r.attempts[0].runtime.status,'INCOMPLETE');
});
test('foreign native receipt and report refs cannot qualify Task identity',()=>{
  const forged={...a,receipt:{...a.receipt,
    ref:'https://github.com/foreign/repo/issues/275#issuecomment-1234'}};
  assert.equal(projectTaskAttempts(make({roster:{...identity,status:'VERIFIED',
    complete:true,attempts:[forged]}})).reason,'CONFLICTING_ATTEMPT_EVIDENCE');
  const report={...identity,attemptId:'attempt-1',status:'CURRENT_OBSERVED',
    reportId:'r-1',revision:1,
    nativeRef:'https://github.com/foreign/repo/issues/275#issuecomment-2345'};
  assert.equal(projectTaskAttempts(make({adsReports:[report]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
});
