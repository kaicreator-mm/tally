import test from 'node:test';
import assert from 'node:assert/strict';
import {eligibleExperienceDecision,experienceOutcomeLabel} from './model.ts';
import type {ExperienceCandidate,ExperienceSnapshot} from './model.ts';
const snapshot={connected:true,versionId:'version/v2',
  source:{repository:'org/tally',sourceSha:'a'.repeat(40),
    currentness:'CURRENT',readback:'VERIFIED'},candidates:[]} as const satisfies ExperienceSnapshot;
const candidate={id:'b'.repeat(64),revision:1,title:'Rule',state:'PROPOSED',
  reportQualification:'VERIFIED',sourceRef:'https://github.com/org/tally/issues/2#issuecomment-1',
  eligibleForNextTask:false} as const satisfies ExperienceCandidate;
test('verified current candidate enables both explicit decisions',()=>{
  const s={...snapshot,candidates:[candidate]};
  assert.deepEqual(eligibleExperienceDecision(s,candidate,'org/tally','version/v2'),
    {eligible:true,canAdopt:true,canReject:true,reason:'HUMAN_DECISION_REQUIRED'});
});
test('STALE, UNVERIFIED and UNKNOWN evidence is rejectable but never adoptable',()=>{
  for(const qualification of ['STALE','UNVERIFIED','UNKNOWN'] as const){
    const item={...candidate,reportQualification:qualification};
    const gate=eligibleExperienceDecision({...snapshot,candidates:[item]},item,
      'org/tally','version/v2');
    assert.deepEqual(gate,{eligible:true,canAdopt:false,canReject:true,
      reason:'UNQUALIFIED_REPORT_REJECT_ONLY'});
  }
});
test('stale source, foreign project and old version disable both decisions',()=>{
  for(const s of [{...snapshot,source:{...snapshot.source,currentness:'STALE'}},
    {...snapshot,source:{...snapshot.source,repository:'evil/repo'}},
    {...snapshot,versionId:'version/v1'}]){
    const gate=eligibleExperienceDecision({...s,candidates:[candidate]},
      candidate,'org/tally','version/v2');
    assert.equal(gate.canAdopt,false);assert.equal(gate.canReject,false);
  }
});
test('terminal, already-applied and stale candidate revisions cannot be decided again',()=>{
  for(const state of ['ADOPTED','REJECTED'] as const){
    const item={...candidate,state};
    const gate=eligibleExperienceDecision({...snapshot,candidates:[item]},item,
      'org/tally','version/v2');
    assert.equal(gate.canAdopt,false);assert.equal(gate.canReject,false);
  }
  const current={...candidate,revision:2};
  const stale={...candidate,revision:1};
  assert.equal(eligibleExperienceDecision({...snapshot,candidates:[current]},stale,
    'org/tally','version/v2').canReject,false);
  assert.match(experienceOutcomeLabel({...candidate,state:'ADOPTED'}),
    /等待下一 Task 实际使用证据/);
});

test('trusted external publisher is source-bound and isolated E2E remains reject-only',()=>{
  const external={...candidate,
    publisherRepository:'org/task-meta',
    sourceKind:'PUBLIC_ADS_REPORT',
    sourceRef:'https://github.com/org/task-meta/issues/9#issuecomment-2'};
  const externalGate=eligibleExperienceDecision({...snapshot,candidates:[external]},
    external,'org/tally','version/v2');
  assert.equal(externalGate.canAdopt,true);
  assert.equal(externalGate.canReject,true);

  const isolated={...candidate,
    publisherRepository:'kaicreator-mm/task-meta-e2e',
    sourceKind:'ISOLATED_E2E_TEST_ONLY',
    sourceRef:'https://github.com/kaicreator-mm/task-meta-e2e/issues/9#issuecomment-3'};
  const isolatedGate=eligibleExperienceDecision({...snapshot,candidates:[isolated]},
    isolated,'org/tally','version/v2');
  assert.deepEqual(isolatedGate,{eligible:true,canAdopt:false,canReject:true,
    reason:'ISOLATED_E2E_TEST_ONLY'});

  const forged={...isolated,publisherRepository:'org/task-meta'};
  const forgedGate=eligibleExperienceDecision({...snapshot,candidates:[isolated]},
    forged,'org/tally','version/v2');
  assert.equal(forgedGate.canAdopt,false);
  assert.equal(forgedGate.canReject,false);
});
