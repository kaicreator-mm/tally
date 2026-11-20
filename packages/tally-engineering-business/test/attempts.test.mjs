import test from 'node:test';
import assert from 'node:assert/strict';
import {projectTaskAttempts} from '../src/attempts.mjs';
const sha='a'.repeat(40);
const ref='https://github.com/kaicreator-mm/tally/issues/275#issuecomment-1234';
const identity={repository:'kaicreator-mm/tally',versionId:'version/v2',
  taskIssueNumber:275,sourceSha:sha};
const stream={
  target:{workflowId:'tally-attempt',instanceKey:'attempt-1'},
  package:{domainId:'tally',version:'2.0.0',
    packageId:'tally-b',contentDigest:'d'.repeat(64),
    formatVersion:'1',runtimeContractMajor:1,executionEngineMajor:1,
    requiredCapabilities:[]},
  epochId:'1'
};
const observationId=(stream,sequence)=>JSON.stringify({
  contentDigest:stream.package.contentDigest,epochId:stream.epochId,
  instanceKey:stream.target.instanceKey,packageId:stream.package.packageId,
  workflowId:stream.target.workflowId})+'#observation:'+sequence;
const a={...identity,attemptId:'attempt-1',role:'builder',writerId:'host:1',
  runtimeStream:stream,environment:'windows-host',worktree:'C:/secret/worktree',
  receipt:{status:'VERIFIED',attemptId:'attempt-1',sourceSha:sha,
    outcome:'COMPLETED',observedAt:'2026-11-10T00:00:00Z',ref}};
const obs={attemptId:'attempt-1',status:'COMPLETE',cursorComplete:true,
  highWatermark:2,observations:[
    {stream,kind:'INSTANCE_OPENED',observationId:observationId(stream,1),
      sequence:1,observedAt:'2026-11-10T00:00:00Z'},
    {stream,kind:'INSTANCE_TERMINALIZED',observationId:observationId(stream,2),
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

test('foreign stream epoch, content digest and workflow target are rejected',()=>{
  for(const mismatched of [
    {...stream,epochId:'2'},
    {...stream,package:{...stream.package,contentDigest:'f'.repeat(64)}},
    {...stream,target:{...stream.target,instanceKey:'foreign'}}
  ]){
    const bad=structuredClone(obs);
    bad.observations[1].stream=mismatched;
    assert.equal(projectTaskAttempts(make({observationPages:[bad]})).reason,
      'CONFLICTING_ATTEMPT_EVIDENCE');
  }
});
test('public RuntimeObservation v1 kind and deterministic record identity are required',()=>{
  const wrong=structuredClone(obs);
  wrong.observations[1].kind='EFFECT_SETTLED';
  assert.equal(projectTaskAttempts(make({observationPages:[wrong]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
  const duplicate=structuredClone(obs);
  duplicate.observations[1].observationId=observationId(stream,1);
  assert.equal(projectTaskAttempts(make({observationPages:[duplicate]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
});

test('public DH record must carry exact deterministic observationId, not merely nonempty',()=>{
  const forged=structuredClone(obs);
  forged.observations[1].observationId='forged-nonempty';
  assert.equal(projectTaskAttempts(make({observationPages:[forged]})).reason,
    'CONFLICTING_ATTEMPT_EVIDENCE');
});

test('invalid public instance lifecycle order cannot qualify terminal evidence',()=>{
  const normal=structuredClone(obs);
  const invalidSequences=[
    // Terminal without an opening event.
    [{...normal.observations[0],kind:'INSTANCE_TERMINALIZED'},
      {...normal.observations[1],kind:'TURN_COMMITTED'}],
    // No INSTANCE_OPENED, although the terminal record is present.
    [{...normal.observations[0],kind:'MESSAGE_ACCEPTED'},
      normal.observations[1]],
    // Reopening the same instance in the very same durable stream.
    [normal.observations[0],
      {...normal.observations[1],kind:'INSTANCE_OPENED'}],
    // A terminal record followed by another ordinary event.
    [normal.observations[0],normal.observations[1],
      {...normal.observations[1],sequence:3,kind:'TURN_COMMITTED',
        observationId:observationId(stream,3)}]
  ];
  for(const observations of invalidSequences){
    const bad={...normal,observations,highWatermark:observations.length};
    assert.equal(projectTaskAttempts(make({observationPages:[bad]})).reason,
      'CONFLICTING_ATTEMPT_EVIDENCE');
  }
});
