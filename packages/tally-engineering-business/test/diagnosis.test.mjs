import test from 'node:test';
import assert from 'node:assert/strict';
import {diagnoseWorkfront} from '../src/diagnosis.mjs';

const repo='kaicreator-mm/domain-harness',observedAsOf='2026-10-31T05:00:00.000Z';
const ref=number=>`https://github.com/${repo}/issues/${number}`;
const item=(number,extra={})=>({issue:`${repo}#${number}`,issueRef:ref(number),completion:'INCOMPLETE',
  publicAcceptance:'UNKNOWN',sourceRefs:{issue:ref(number)},...extra});
const timeline=items=>({subjectStatus:'OBSERVED',items,coverage:{sourceMergedCount:items.filter(x=>x.sourceMerged).length,
  realCompletedCount:items.filter(x=>x.completion==='REAL_COMPLETED').length}});
const observations=extra=>({observedAsOf,...extra});
const one=(result,classification)=>result.diagnoses.find(row=>row.classification===classification);

test('false version dependency remains a lawful hypothesis with competing explanations',()=>{
  const result=diagnoseWorkfront(timeline([item(530)]),observations({dependencies:[{issue:`${repo}#530`,
    kind:'VERSION_SERIAL',proven:false,blockedBy:[`${repo}#528`],lawful:true,sourceRef:ref(530)}]}));
  const row=one(result,'POTENTIAL_FALSE_DEPENDENCY');
  assert.equal(row.status,'HYPOTHESIS');assert.equal(row.competingExplanations.length,2);
  assert.equal(row.lawfulParallel,false);assert.equal(row.authorizationNeeded,true);
  assert.match(row.mustNot.join(' '),/DO_NOT_REMOVE_NATIVE_DEPENDENCY/);
});

test('component progress with public acceptance gap diagnoses integration not percentage delivery',()=>{
  const result=diagnoseWorkfront(timeline([item(1013,{sourceMerged:true,completion:'SOURCE_MERGED',
    publicAcceptance:'PUBLIC_ACCEPTANCE_NOT_PROVEN',sourceRefs:{pullRequest:`https://github.com/${repo}/pull/1016`}})]),
  observations({selectedPaths:[{issue:`${repo}#1013`,status:'FAIL',sourceRef:ref(1013)}]}));
  const row=one(result,'INTEGRATION_COVERAGE_GAP');
  assert.equal(row.status,'OBSERVED');assert.equal(row.likelyRootOwner,'Public Host Integration');
  assert.equal(row.competingExplanations.includes('ADDITIONAL_COMPONENT_TESTS_ONLY'),true);
  assert.match(row.mustNot.join(' '),/DO_NOT_REPORT_100_PERCENT_DELIVERED/);
  assert.equal(result.coverage.integrationGapCount,1);assert.equal(result.coverage.realCompletedCount,0);
});

test('real P0 remains required and loaded-but-not-invoked stays a selected-path gap',()=>{
  const items=[item(1100),item(1264)];
  const result=diagnoseWorkfront(timeline(items),observations({
    repairs:[{issue:`${repo}#1100`,severity:'P0',status:'OPEN',sourceRef:ref(1100)}],
    selectedPaths:[{issue:`${repo}#1264`,status:'NOT_RUN',sourceRef:ref(1264)}]}));
  const p0=one(result,'REQUIRED_SECURITY_REPAIR');
  assert.equal(p0.status,'OBSERVED');assert.match(p0.mustNot.join(' '),/DO_NOT_WAIVE_P0_FOR_COST/);
  assert.match(p0.mustNot.join(' '),/DO_NOT_REWRITE_OLD_RELEASE_FACTS/);
  assert.equal(one(result,'SELECTED_PATH_NOT_INVOKED').status,'OBSERVED');
});

test('effective Review is positive output while rootless repetition is postmortem only',()=>{
  const result=diagnoseWorkfront(timeline([item(1354),item(999)]),observations({reviews:[
    {issue:`${repo}#1354`,verdict:'PASS',sourceSha:'5079210cf872a12e07d2ffd8bc2c339d0d954b46',sourceRef:`https://github.com/${repo}/pull/1355`,firstCounterexampleRef:ref(1354)},
    {issue:`${repo}#999`,verdict:'CHANGES_REQUESTED',sourceRef:ref(999)},
    {issue:`${repo}#999`,verdict:'CHANGES_REQUESTED',sourceRef:ref(999)}]}));
  assert.equal(one(result,'EFFECTIVE_REVIEW_CONVERGENCE').status,'OBSERVED');
  const repeated=one(result,'NO_ROOT_PROGRESS');
  assert.equal(repeated.status,'HYPOTHESIS');assert.equal(repeated.retrospective,'POSTMORTEM_ONLY');
  assert.equal(result.coverage.reviewEvidenceCount,3);assert.equal(result.coverage.capabilityCount,2);
  assert.equal(result.coverage.comparison,'COUNTS_NOT_PERCENTAGES');
});

test('calendar-only observability never becomes Agent active cost or ranking',()=>{
  const result=diagnoseWorkfront(timeline([item(64)]),observations({costs:[{issue:`${repo}#64`,
    calendarSpanMs:172800000,sourceRef:ref(64)}]}));
  const row=one(result,'CALENDAR_SPAN_ONLY');
  assert.equal(row.agentActive,'UNKNOWN');assert.equal(row.cost,'NOT_MEASURED');
  assert.equal(row.calendarSpanMs,172800000);assert.match(row.mustNot.join(' '),/DO_NOT_RANK_AGENT_EFFICIENCY/);
  assert.equal(result.coverage.agentActive,'UNKNOWN');
});

test('scoped CI fast path and structural parallel candidates remain non-authoritative',()=>{
  const items=[item(1354),item(530)];
  const result=diagnoseWorkfront(timeline(items),observations({
    capabilities:[{issue:`${repo}#1354`,lowRisk:true,scopedCi:'PASS',sourceRef:ref(1354)},
      {issue:`${repo}#530`,role:'builder',hostAvailable:true,sourceRef:ref(530)}],
    dependencies:[{issue:`${repo}#530`,blockedBy:[],lawful:true,sourceRef:ref(530)}]}));
  assert.equal(one(result,'SCOPED_FAST_PATH_ELIGIBLE').status,'OBSERVED');
  assert.equal(result.parallelCandidates.length,1);assert.equal(result.parallelCandidates[0].issue,`${repo}#530`);
  assert.equal(result.parallelCandidates[0].authorizationNeeded,true);
  assert.match(result.parallelCandidates[0].mustNot.join(' '),/DO_NOT_CREATE_CLAIM_OR_EFFECT/);
});

test('missing or malformed source stays UNKNOWN and never grants Release',()=>{
  const invalid=diagnoseWorkfront(null,null);
  assert.equal(invalid.status,'UNKNOWN');assert.equal(invalid.coverage.releasePass,false);
  const noTime=diagnoseWorkfront(timeline([item(530)]),{});
  assert.equal(noTime.status,'UNKNOWN');assert.equal(noTime.diagnoses.length,0);
  const result=diagnoseWorkfront(timeline([item(530)]),observations({dependencies:[null],repairs:[null]}));
  assert.equal(result.status,'UNKNOWN');assert.equal(result.diagnoses.length,0);
  assert.equal(result.coverage.taskPass,false);assert.equal(result.advisoryOnly,true);
  const malformedTimeline=diagnoseWorkfront({subjectStatus:'OBSERVED',items:[null]},observations({}));
  assert.equal(malformedTimeline.status,'UNKNOWN');
  const sourceLess=diagnoseWorkfront(timeline([item(1013,{issueRef:'invalid',publicAcceptance:'PUBLIC_ACCEPTANCE_NOT_PROVEN',
    sourceRefs:{pullRequest:'invalid'}})]),observations({}));
  assert.equal(one(sourceLess,'INTEGRATION_COVERAGE_GAP').status,'UNKNOWN');
});
