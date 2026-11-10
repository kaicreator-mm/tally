import test from 'node:test';
import assert from 'node:assert/strict';
import {readTaskAttemptHistoryFromHost} from './ui-attempt-host.mjs';
const id={repository:'kaicreator-mm/tally',versionId:'version/v2',
  taskIssueNumber:275,sourceSha:'a'.repeat(40)};
const a={...id,attemptId:'attempt-1',role:'builder',writerId:'host:1',
  workflowAddress:'workflow/v2/attempt-1',packageId:'tally-b',epoch:1,
  environment:'windows-host',worktree:'C:/private/worktree'};
const event={attemptId:a.attemptId,workflowAddress:a.workflowAddress,
  packageId:a.packageId,epoch:1,eventType:'INSTANCE_OPENED',sequence:1,
  observedAt:'2026-11-10T00:00:00Z'};
const port={readTaskAttemptRoster:async()=>({...id,status:'VERIFIED',complete:true,
  attempts:[a]}),readRuntimeObservations:async()=>({status:'COMPLETE',
  observations:[event],highWatermark:1,nextCursor:null})};
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
  const p={...port,readRuntimeObservations:async()=>({status:'STORAGE_GAP',
    observations:[],nextCursor:null})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
  assert.equal(r.attempts[0].runtime.reason,'PUBLIC_RUNTIME_STORAGE_GAP');
});
test('broken cursor / repetition and network failure remain incomplete',async()=>{
  const p={...port,readRuntimeObservations:async()=>({status:'COMPLETE',
    observations:[],highWatermark:0,nextCursor:'again'})};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.status,'PARTIAL');
});
test('optional ADS failure cannot block essential attempt source',async()=>{
  const p={...port,readAdsTaskReports:async()=>{throw Error('task-meta offline');}};
  const r=await readTaskAttemptHistoryFromHost({selectedHost:p,...id});
  assert.equal(r.coverage.adsReports,0);assert.equal(r.status,'PARTIAL');
});
