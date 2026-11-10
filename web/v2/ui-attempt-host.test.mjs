import test from 'node:test';
import assert from 'node:assert/strict';
import {readTaskAttemptHistoryFromHost} from './ui-attempt-host.mjs';
const id={repository:'kaicreator-mm/tally',versionId:'version/v2',
  taskIssueNumber:275,sourceSha:'a'.repeat(40)};
const stream={target:{workflowId:'tally-attempt',instanceKey:'attempt-1'},
  package:{domainId:'tally',version:'2.0.0',packageId:'tally-b',
    contentDigest:'d'.repeat(64),formatVersion:'1',
    runtimeContractMajor:1,executionEngineMajor:1,requiredCapabilities:[]},
  epochId:'1'};
const observationId=(stream,sequence)=>JSON.stringify({
  contentDigest:stream.package.contentDigest,epochId:stream.epochId,
  instanceKey:stream.target.instanceKey,packageId:stream.package.packageId,
  workflowId:stream.target.workflowId})+'#observation:'+sequence;
const a={...id,attemptId:'attempt-1',role:'builder',writerId:'host:1',
  runtimeStream:stream,environment:'windows-host',worktree:'C:/private/worktree'};
const event={stream,kind:'INSTANCE_OPENED',observationId:observationId(stream,1),sequence:1,
  observedAt:'2026-11-10T00:00:00Z'};
const port={readTaskAttemptRoster:async()=>({...id,status:'VERIFIED',
  complete:true,attempts:[a]}),
  readObservations:async({stream:requested})=>{
    assert.deepEqual(requested,stream);
    return {records:[event],highWatermark:1,nextCursor:'opaque-cursor'};
  }};
test('trusted selected Host roster and public read generate partial, never PASS',async()=>{
  const r=await readTaskAttemptHistoryFromHost({selectedHost:port,...id});
  assert.equal(r.status,'PARTIAL');assert.equal(r.coverage.attemptCount,1);
  assert.equal(r.attempts[0].runtime.status,'OBSERVED');
  assert.equal(r.attempts[0].hostOutcome.status,'UNKNOWN');
  assert.equal(r.taskPass,false);assert.equal(r.releasePass,false);
  assert.equal(JSON.stringify(r).includes('private/worktree'),false);
});
test('absence of roster API or a failed read cannot become zero attempts',async()=>{
  assert.equal((await readTaskAttemptHistoryFromHost({selectedHost:{},...id})).reason,
    'HOST_ATTEMPT_ROSTER_PORT_MISSING');
  assert.equal((await readTaskAttemptHistoryFromHost({selectedHost:{
    readTaskAttemptRoster:async()=>{throw Error('offline');}
  },...id})).reason,'HOST_ATTEMPT_ROSTER_READ_FAILED');
});
test('unsupported selected Host index blocks without SQLite fallback',async()=>{
  const r=await readTaskAttemptHistoryFromHost({selectedHost:{
    readTaskAttemptRoster:async()=>({status:'UNKNOWN',attempts:[]})},...id});
  assert.equal(r.reason,'HOST_ATTEMPT_ROSTER_UNQUALIFIED');
});
test('public observation storage gap does not grant complete history',async()=>{
  const p={...port,readObservations:async()=>({records:[],highWatermark:1,
    gap:{kind:'STORAGE_GAP',requestedAfter:0,highWatermark:1}})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.attempts[0].runtime.reason,'PUBLIC_RUNTIME_STORAGE_GAP');
});
test('broken cursor / repetition and network failure remain incomplete',async()=>{
  const p={...port,readObservations:async()=>({records:[],
    highWatermark:1,nextCursor:'again'})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
});
test('optional ADS failure cannot block essential attempt source',async()=>{
  const p={...port,readAdsTaskReports:async()=>{throw Error('task-meta offline');}};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.coverage.adsReports,0);assert.equal(r.status,'PARTIAL');
});

test('DH public pull cursor paginates with afterCursor and exact stream, not inferred status',async()=>{
  const got=[];
  const p={...port,readObservations:async req=>{
    got.push(req);
    if(req.afterCursor===undefined)
      return {records:[event],nextCursor:'opaque-1',highWatermark:2};
    if(req.afterCursor==='opaque-1')
      return {records:[{...event,sequence:2,observationId:observationId(stream,2),
        kind:'INSTANCE_TERMINALIZED'}],highWatermark:2,nextCursor:'opaque-2'};
    throw new Error('unexpected cursor');
  }};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.attempts[0].runtime.eventCount,2);
  assert.equal(r.attempts[0].runtime.runtimeTerminal,true);
  assert.equal(got.length,2);
  assert.equal(got[1].afterCursor,'opaque-1');
  assert.deepEqual(got[0].stream,stream);
});
test('DH explicit CURSOR_INVALID gap stays incomplete, never silently empty',async()=>{
  const p={...port,readObservations:async()=>({records:[],highWatermark:1,
    gap:{kind:'CURSOR_INVALID',requestedAfter:0,highWatermark:1}})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.attempts[0].runtime.reason,'PUBLIC_RUNTIME_CURSOR_INVALID');
});
test('native-looking but legacy status/observations page is rejected',async()=>{
  const p={...port,readObservations:async()=>({status:'COMPLETE',
    observations:[event],highWatermark:1})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.attempts[0].runtime.status,'INCOMPLETE');
});
