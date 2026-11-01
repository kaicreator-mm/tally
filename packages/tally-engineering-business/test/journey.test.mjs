import test from 'node:test';
import assert from 'node:assert/strict';
import {projectJourneySnapshot,nextActions} from '../src/journey.mjs';
const sourceSha='4467bfd166f881cca88317eb34381217c9fc0d6f';
const observedAt='2026-10-30T12:00:00.000Z';
const task=(sourceRef,changes={})=>({sourceRef,sourceSha,observedAsOf:observedAt,
  currentness:'OBSERVED_AT_QUERY_COMPLETION',status:'AWAITING_HOST_ADMISSION',
  reason:'NATIVE_READY_NO_ACCEPTED_CLAIM',dependencyState:'CLEAR',claimState:'NOT_VERIFIED',
  kernelTaskState:'NOT_VERIFIED',agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
  ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,...changes});
const one=task('https://github.com/kaicreator-mm/tally/issues/172');
const two=task('https://github.com/kaicreator-mm/tally/issues/177',{status:'AWAITING_REVIEW',
  reason:'LOCAL_CANDIDATE_READY',agentAttemptState:'LOCAL_CANDIDATE_READY'});
const three=task('https://github.com/kaicreator-mm/tally/issues/179',{status:'REVIEW_CANDIDATE_EVIDENCE',
  reason:'REVIEW_JOINED',reviewState:'REVIEW_JOINED',hostState:'REVIEW_JOINED'});
const snapshot=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,
  tasks:[one,two,three],observedAt});
test('Project Journey exposes independent legal Agent assignments without human cards',()=>{
  assert.equal(snapshot.status,'OBSERVED');assert.equal(snapshot.progressSignals.candidates,1);
  assert.equal(snapshot.progressSignals.reviewed,1);assert.equal(snapshot.authorized,false);
  const {actions,humanActionInbox}=nextActions(snapshot);
  assert.deepEqual(actions.map(x=>x.authorizationType),
    ['START_ELIGIBLE_CLAIM','START_ELIGIBLE_INDEPENDENT_REVIEW','START_ELIGIBLE_VALIDATOR']);
  assert.deepEqual(actions.map(x=>x.actorRole),['builder','reviewer','validator']);
  assert.equal(humanActionInbox.length,0);
  for(const action of actions){assert.equal(action.authorization,'HOST_AND_ADS_REQUIRED');
    assert.equal(action.evidence.sourceSha,sourceSha);assert.equal(action.approvalNeeded,true);}
});
test('native blockers and no currentness never unlock an Agent',()=>{
  const blocked=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,
    tasks:[task(one.sourceRef,{status:'WAIT_EXTERNAL',reason:'NATIVE_DEPENDENCIES_BLOCKED',
      dependencyState:'BLOCKED'})],observedAt});
  assert.deepEqual(nextActions(blocked).actions.map(x=>x.authorizationType),['WAIT_EXTERNAL']);
  const stale=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,
    tasks:[{...one,currentness:'AGE_NOT_VERIFIED',observedAsOf:null}],observedAt});
  const action=nextActions(stale).actions[0];
  assert.equal(action.authorizationType,'WAIT_EXTERNAL');assert.equal(action.reason,'CURRENTNESS_UNVERIFIED');
  assert.equal(action.approvalNeeded,false);
});
test('only genuine Host/scope duties enter Human-action inbox with exact readback',()=>{
  const hostless=task(one.sourceRef,{requiredHostEvidence:{kind:'WINDOWS_CODEX_HOST',available:false,
    reason:'CODEX_HOST_UNREACHABLE'}});
  const conflict=task(two.sourceRef,{scopeConflict:true});
  const s=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,tasks:[hostless,conflict],observedAt});
  const {actions,humanActionInbox}=nextActions(s);
  assert.deepEqual(actions.map(x=>x.authorizationType),['NEEDS_HUMAN_HOST','NEEDS_USER_DECISION']);
  assert.equal(humanActionInbox.length,2);
  assert.equal(humanActionInbox[0].expectedEvidence,'WINDOWS_CODEX_HOST');
  assert.equal(humanActionInbox[0].readback,one.sourceRef);
  assert.equal(humanActionInbox[1].reason,'PRODUCT_SCOPE_CONFLICT');
});
test('candidate failure permits bounded Repair without rewriting old evidence',()=>{
  const failed=task(two.sourceRef,{status:'CONFLICT',reason:'AGENT_ATTEMPT_NOT_COMPLETE',
    agentAttemptState:'LOCAL_ATTEMPT_UNKNOWN'});
  const s=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,tasks:[failed],observedAt});
  const action=nextActions(s).actions[0];
  assert.equal(action.authorizationType,'REPAIR_ELIGIBLE');assert.equal(action.actorRole,'builder');
  assert.equal(action.approvalNeeded,true);assert.equal(action.authorization,'HOST_REQUIRED');
  assert.equal(action.costRisk,'MEDIUM');
});
test('source changes recompute only affected lanes, not the whole project',()=>{
  const other=task('https://github.com/kaicreator-mm/tally/issues/999',{sourceSha:'f'.repeat(40)});
  const s=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,tasks:[one,two,other],observedAt});
  const {actions,changedSourceSha:actualSha}=nextActions(s,{changedSourceSha:sourceSha});
  assert.equal(actions.length,2);
  assert.deepEqual(actions.map(x=>x.authorizationType),
    ['START_ELIGIBLE_CLAIM','START_ELIGIBLE_INDEPENDENT_REVIEW']);
  assert.equal(actualSha,sourceSha);
});
test('invalid snapshots, timestamps and conflict states cannot authorize action',()=>{
  assert.equal(projectJourneySnapshot({project:'tally',version:'v2',sourceSha,tasks:[],observedAt:'bad'}).status,
    'UNKNOWN');
  assert.equal(projectJourneySnapshot({project:'tally',version:'v2',sourceSha,tasks:[null],observedAt}).status,
    'UNKNOWN');
  const conflict=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,
    tasks:[task(one.sourceRef,{status:'CONFLICT',reason:'AMBIGUOUS_CLAIM_EVENT_CANDIDATES'})],observedAt});
  assert.deepEqual(nextActions(conflict).actions.map(x=>x.authorizationType),['NO_VALID_ACTION']);
  const unknown=projectJourneySnapshot({project:'tally',version:'v2',sourceSha,
    tasks:[task(one.sourceRef,{status:'UNKNOWN',reason:'NATIVE_SOURCE_PAIR_NOT_VERIFIED'})],observedAt});
  assert.equal(nextActions(unknown).actions[0].reason,'NATIVE_SOURCE_PAIR_NOT_VERIFIED');
  const invalidResult=nextActions(null);
  assert.equal(invalidResult.actions.length,1);
  assert.equal(invalidResult.actions[0].authorizationType,'NO_VALID_ACTION');
  assert.deepEqual(invalidResult.humanActionInbox,[]);
});
