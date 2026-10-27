import test from 'node:test';
import assert from 'node:assert/strict';
import {diagnoseNativeTaskReadiness as diagnose} from '../src/native-source-diagnostic.mjs';
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
