import test from 'node:test';
import assert from 'node:assert/strict';
import {projectNativeTaskEffectiveSnapshot as project} from '../src/task-effective-snapshot.mjs';
const ref='https://github.com/kaicreator-mm/runx/issues/578',sha='a'.repeat(40);
const w={issueRef:ref,sourceSha:sha,dependencyState:'CLEAR',openBlockerCount:0,
  issueState:'open',workflowState:'state:ready',workItemType:'type:task',
  reason:'CLAIM_NOT_VERIFIED',claimStatus:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false};
const c={reason:'NO_DISPATCH_CLAIM_EVENTS',candidateCount:0,candidates:[],
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
  taskPass:false,releasePass:false};
const pair={status:'SOURCE_PAIR_OBSERVED',sourceQualified:true,
  issueRef:ref,sourceSha:sha,claimStatus:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  workItem:w,claimEvents:c};
const read=(sourcePair=pair,more={})=>project({sourcePair,expectedIssueRef:ref,
  expectedSourceSha:sha,...more});
const denied=r=>{for(const k of ['authorized','admitted','taskPass','releasePass'])
  assert.equal(r[k],false);assert.equal(r.claimState,'NOT_VERIFIED');
  assert.equal(r.kernelTaskState,'NOT_VERIFIED');};
test('canonical Task ready => selected Host workfront, not Agent authority',()=>{
  const r=read();assert.equal(r.status,'AWAITING_HOST_ADMISSION');
  assert.equal(r.nextActions[0].kind,'REQUEST_SELECTED_HOST_ADMISSION');
  assert.equal(r.nextActions[0].authority,'HOST_AND_ADS_REQUIRED');denied(r);
});
test('real blocked_by => wait, UNKNOWN evidence => no action authorization',()=>{
  const a=read({...pair,workItem:{...w,dependencyState:'BLOCKED',openBlockerCount:2}});
  assert.equal(a.status,'WAIT_EXTERNAL');assert.equal(a.openBlockerCount,2);denied(a);
  const b=read({...pair,workItem:{...w,dependencyState:'UNKNOWN'}});
  assert.equal(b.status,'UNKNOWN');denied(b);
});
test('claimed label, planning issue, closed issue never means native readiness',()=>{
  for(const change of [{workflowState:'state:claimed'},
    {workItemType:'type:planning'},{issueState:'closed'}]){
    const x=read({...pair,workItem:{...w,...change}});
    assert.equal(x.reason,'TASK_NOT_NATIVE_READY');denied(x);
  }
});
test('observed ADS comment => reconciliation, never accepted CAS',()=>{
  const x=read({...pair,claimEvents:{...c,
    reason:'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED',
    candidateCount:1,candidates:[{sourceSha:sha,claimAccepted:false}]}});
  assert.equal(x.nextActions[0].kind,'RECONCILE_NATIVE_CLAIM');
  assert.equal(x.nextActions[0].authority,'HOST_REQUIRED');denied(x);
});
test('forged K, Host, Agent and Review fields do not change the view',()=>{
  const r=read({...pair,claimAccepted:true,kernelTaskState:'ADMITTED',
    agentAttemptState:'PASS',workItem:{...w,claimAccepted:true},
    claimEvents:{...c,accepted:true}});
  assert.equal(r.status,'AWAITING_HOST_ADMISSION');
  assert.equal(r.releaseState,'NOT_VERIFIED');denied(r);
});
test('mismatched SHA, empty, conflicting candidate, rate limited source deny',()=>{
  for(const v of [null,{...pair,sourceSha:'b'.repeat(40)},
    {...pair,authorized:true},{...pair,claimEvents:{...c,candidateCount:1}},
    {...pair,claimEvents:{...c,reason:'GITHUB_RATE_LIMITED'}}]){
    const r=read(v);assert.equal(r.status,'UNKNOWN');denied(r);
  }
});
test('invalid selected exact source and hostile getter cannot leak secrets',()=>{
  const x=read(pair,{expectedSourceSha:'invalid'});
  assert.equal(x.reason,'EXPECTED_SOURCE_IDENTITY_UNAVAILABLE');denied(x);
  const secret='PRIVATE_TOKEN_DO_NOT_PRINT';
  const y=read({...pair,get claimEvents(){throw Error(secret);}});
  assert.equal(y.reason,'SOURCE_OBSERVATION_UNREADABLE');
  assert.equal(JSON.stringify(y).includes(secret),false);denied(y);
});
