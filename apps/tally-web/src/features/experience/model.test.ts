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
test('only current source-backed proposed/reviewed candidate is actionable',()=>{
  const s={...snapshot,candidates:[candidate]};
  assert.deepEqual(eligibleExperienceDecision(s,candidate,'org/tally','version/v2'),
    {eligible:true,reason:'HUMAN_DECISION_REQUIRED'});
});
test('stale source, foreign project, old version and unknown report refuse',()=>{
  for(const s of [{...snapshot,source:{...snapshot.source,currentness:'STALE'}},
    {...snapshot,source:{...snapshot.source,repository:'evil/repo'}},
    {...snapshot,versionId:'version/v1'}]){
    assert.equal(eligibleExperienceDecision({...s,candidates:[candidate]},
      candidate,'org/tally','version/v2').eligible,false);
  }
  const unknown={...candidate,reportQualification:'UNKNOWN' as const};
  assert.equal(eligibleExperienceDecision({...snapshot,candidates:[unknown]},
    unknown,'org/tally','version/v2').eligible,false);
});
test('terminal decisions cannot be repeated or auto-used by next Task',()=>{
  for(const state of ['ADOPTED','REJECTED'] as const){
    const c={...candidate,state};
    assert.equal(eligibleExperienceDecision({...snapshot,candidates:[c]},c,
      'org/tally','version/v2').eligible,false);
  }
  assert.match(experienceOutcomeLabel({...candidate,state:'ADOPTED'}),
    /等待下一 Task 实际使用证据/);
});
