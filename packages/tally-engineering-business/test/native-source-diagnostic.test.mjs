import test from 'node:test';
import assert from 'node:assert/strict';
import {diagnoseNativeTaskReadiness as diagnose} from '../src/index.mjs';
const sha='a'.repeat(40),ref='https://github.com/kaicreator-mm/tally/issues/125';
const item={issueRef:ref,sourceSha:sha,dependencyState:'CLEAR',
  workflowState:'state:ready',reason:'CLAIM_NOT_VERIFIED',
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,taskPass:false};
const events={claimStatus:'NOT_VERIFIED',candidateCount:0,candidates:[],
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  reason:'NO_DISPATCH_CLAIM_EVENTS'};
const input={expectedIssueRef:ref,expectedSourceSha:sha,workItem:item,claimEvents:events};
const full={localAgentStatus:'AVAILABLE',builderConfinementStatus:'VERIFIED',
  hiddenVerifierStatus:'VERIFIED'};
test('source dependencies CLEAR + all OS booleans STILL no accepted Claim',()=>{
  for(const info of [
    {...input,...full},
    {...input,...full,claimAccepted:true,dependenciesComplete:true},
    {...input,...full,claimEvents:{...events,claimAccepted:true}}
  ]){
    const result=diagnose(info);
    assert.equal(result.reason,'ACCEPTED_CAS_CLAIM_NOT_VERIFIED');
    assert.equal(result.admitted,false);
    assert.equal(result.authorized,false);
    assert.equal(result.taskPass,false);
    assert.equal(result.releasePass,false);
    assert.equal(result.dependencyState,'CLEAR');
  }
});
test('native open dependencies and unknown source never become Task Ready',()=>{
  assert.equal(diagnose({...input,workItem:{...item,dependencyState:'BLOCKED'}}).reason,
    'NATIVE_DEPENDENCIES_BLOCKED');
  assert.equal(diagnose({...input,workItem:{...item,dependencyState:'UNKNOWN'}}).reason,
    'NATIVE_DEPENDENCIES_UNKNOWN');
  assert.equal(diagnose({...input,workItem:{...item,sourceSha:'b'.repeat(40)}}).reason,
    'NATIVE_WORK_ITEM_SOURCE_UNVERIFIED');
  assert.equal(diagnose({...input,workItem:{...item,issueRef:ref+'/wrong'}}).reason,
    'NATIVE_WORK_ITEM_SOURCE_UNVERIFIED');
  assert.equal(diagnose({...input,expectedSourceSha:'bad'}).reason,
    'SOURCE_IDENTITY_UNAVAILABLE');
});
test('wrong state or label-only claimed never proves CAS',()=>{
  for(const reason of ['CANONICAL_WORK_ITEM_STATE_UNVERIFIED','WORK_ITEM_NOT_READY']){
    assert.equal(diagnose({...input,workItem:{...item,reason}}).reason,
      'WORK_ITEM_NOT_CLAIMABLE_SOURCE_ONLY');
  }
  assert.equal(diagnose({...input,workItem:{...item,workflowState:'state:claimed'}}).reason,
    'WORK_ITEM_NOT_CLAIMABLE_SOURCE_ONLY');
});
test('unbound or ambiguous Claim event cannot self-attest',()=>{
  assert.equal(diagnose({...input,claimEvents:null}).reason,'CLAIM_EVENT_SOURCE_UNVERIFIED');
  assert.equal(diagnose({...input,claimEvents:{...events,claimStatus:'ACCEPTED'}}).reason,
    'CLAIM_EVENT_SOURCE_UNVERIFIED');
  assert.equal(diagnose({...input,claimEvents:{...events,candidateCount:2,candidates:[{},{}]}}).reason,
    'AMBIGUOUS_CLAIM_EVENT_CANDIDATES');
  const c={dispatchId:'claim1',sourceSha:sha,
    sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',claimAccepted:false};
  assert.equal(diagnose({...input,...full,claimEvents:{...events,
    candidateCount:1,candidates:[c]}}).reason,'ACCEPTED_CAS_CLAIM_NOT_VERIFIED');
  assert.equal(diagnose({...input,...full,claimEvents:{...events,
    candidateCount:1,candidates:[{...c,sourceSha:'b'.repeat(40)}]}}).reason,
    'CLAIM_EVENT_SOURCE_UNVERIFIED');
});
test('distinct Local Agent, Builder confinement, Held-out verifier denial typed',()=>{
  assert.equal(diagnose({...input}).reason,'LOCAL_AGENT_UNAVAILABLE');
  assert.equal(diagnose({...input,localAgentStatus:'AVAILABLE'}).reason,
    'BUILDER_CONFINEMENT_UNAVAILABLE');
  assert.equal(diagnose({...input,localAgentStatus:'AVAILABLE',builderConfinementStatus:'VERIFIED'}).reason,
    'HIDDEN_VERIFIER_UNAVAILABLE');
});
test('hostile property getter never leaks exception or secret',()=>{
  const secret='TOKEN_DO_NOT_EXPOSE_7bb';
  const result=diagnose({...input,workItem:{
    ...item,get sourceSha(){throw new Error(secret);}
  }});
  assert.equal(result.reason,'UNTRUSTED_SOURCE_INPUT');
  assert.equal(JSON.stringify(result).includes(secret),false);
  assert.equal(result.admitted,false);
  assert.equal(diagnose(null).admitted,false);
});

import {projectNativeTaskEffectiveSnapshot as project} from '../src/index.mjs';
const snapshotRef='https://github.com/kaicreator-mm/tally/issues/125';
const snapshotSha='a'.repeat(40);
const snapshotWorkItem={...item,issueState:'open',workItemType:'type:task',
  openBlockerCount:0};
const snapshotPair={status:'SOURCE_PAIR_OBSERVED',sourceQualified:true,
  issueRef:snapshotRef,sourceSha:snapshotSha,claimStatus:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  workItem:snapshotWorkItem,claimEvents:events};
const projectPair=(s=snapshotPair,override={})=>project({sourcePair:s,
  expectedIssueRef:snapshotRef,expectedSourceSha:snapshotSha,...override});
const noTaskPermit=r=>{
  for(const k of ['authorized','admitted','taskPass','releasePass'])
    assert.equal(r[k],false);
  for(const k of ['claimState','kernelTaskState','agentAttemptState',
      'reviewState','ciState','hostState','releaseState'])
    assert.equal(r[k],'NOT_VERIFIED');
};
test('source current Ready only requests a selected Host: observation time UNKNOWN',()=>{
  const r=projectPair();assert.equal(r.status,'AWAITING_HOST_ADMISSION');
  assert.equal(r.nextActions[0].authority,'HOST_AND_ADS_REQUIRED');
  assert.equal(r.observedAsOf,null);assert.equal(r.currentness,'AGE_NOT_VERIFIED');
  assert.equal(r.sourceRef,snapshotRef);noTaskPermit(r);
});
test('native blockers and incomplete source remain separate',()=>{
  const a=projectPair({...snapshotPair,workItem:{...snapshotWorkItem,
    dependencyState:'BLOCKED',openBlockerCount:2}});
  assert.equal(a.status,'WAIT_EXTERNAL');assert.equal(a.openBlockerCount,2);
  noTaskPermit(a);
  const b=projectPair({...snapshotPair,workItem:{...snapshotWorkItem,
    dependencyState:'UNKNOWN'}});
  assert.equal(b.status,'UNKNOWN');noTaskPermit(b);
});
test('observed self-reported claim is not accepted, multiple is CONFLICT',()=>{
  const a=projectPair({...snapshotPair,claimEvents:{...events,
    reason:'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED',candidateCount:1,
    candidates:[{claimAccepted:false}]}});
  assert.equal(a.nextActions[0].kind,'RECONCILE_NATIVE_CLAIM');noTaskPermit(a);
  const b=projectPair({...snapshotPair,claimEvents:{...events,
    reason:'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED',candidateCount:2,
    candidates:[{},{}]}});
  assert.equal(b.status,'CONFLICT');
  assert.equal(b.reason,'AMBIGUOUS_CLAIM_EVENT_CANDIDATES');
  noTaskPermit(b);
});
test('closed, planning, wrongly claimed labels never become executable',()=>{
  for(const change of [{issueState:'closed'}, {workItemType:'type:planning'},
    {workflowState:'state:claimed'}]){
    const r=projectPair({...snapshotPair,workItem:{...snapshotWorkItem,...change}});
    assert.equal(r.reason,'TASK_NOT_NATIVE_READY');noTaskPermit(r);
  }
});
test('forged CAS/K/Agent grants have no effect on read-model',()=>{
  const x=projectPair({...snapshotPair,claimAccepted:true,
    kAuthorized:true,releasePass:true,
    workItem:{...snapshotWorkItem,claimAccepted:true}});
  assert.equal(x.status,'AWAITING_HOST_ADMISSION');noTaskPermit(x);
});
test('stale SHA, malformed pair, conflicts in cardinality and 403 are UNKNOWN',()=>{
  for(const p of [null,{...snapshotPair,sourceSha:'b'.repeat(40)},
    {...snapshotPair,authorized:true},
    {...snapshotPair,claimEvents:{...events,candidateCount:1}},
    {...snapshotPair,claimEvents:{...events,reason:'GITHUB_ACCESS_DENIED'}},
    {...snapshotPair,workItem:{...snapshotWorkItem,sourceSha:'b'.repeat(40)}}]){
    const r=projectPair(p);assert.equal(r.status,'UNKNOWN');noTaskPermit(r);
  }
});
test('untrusted property getters cannot leak raw exception/credential',()=>{
  const secret='GITHUB_TOKEN_NOT_FOR_OUTPUT';
  const x=projectPair({...snapshotPair,
    get claimEvents(){throw new Error(secret);}});
  assert.equal(x.status,'UNKNOWN');assert.equal(x.reason,'SOURCE_OBSERVATION_UNREADABLE');
  assert.equal(JSON.stringify(x).includes(secret),false);noTaskPermit(x);
  const y=projectPair(snapshotPair,{expectedSourceSha:'invalid'});
  assert.equal(y.reason,'EXPECTED_SOURCE_IDENTITY_UNAVAILABLE');noTaskPermit(y);
});

