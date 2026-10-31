import test from 'node:test';
import assert from 'node:assert/strict';
import {projectMaintenanceIntake} from '../src/maintenance-intake.mjs';
const repository='kaicreator-mm/tally';
const observedAsOf='2026-10-31T08:18:45Z';
const subjectSha='5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02';
const issue=n=>`https://github.com/${repository}/issues/${n}`;
const priorRelease={version:'v1.5.5',commitSha:'5904e7efd68b63fa4a2a0f97f54d88e9f99f4b02',
  evidenceRef:`https://github.com/${repository}/releases/tag/v1.5.5`};
const defect={issueRef:issue(207),requirementRef:issue(92),subjectSha,
  observedAt:observedAsOf,summary:'Post-release behavior regression'};
const learning={taskIssueRef:issue(172),targetIssueRef:issue(203),subjectSha,
  observedAt:observedAsOf,qualification:'REPORTED_QUALIFIED'};
const input=()=>({repository,observedAsOf,priorRelease,defects:[defect],learning:[learning]});
const safe=result=>{
  assert.equal(result.historicalReleaseMutation,false);
  assert.equal(result.nativeWriteAuthorized,false);
  assert.equal(result.advisoryOnly,true);
  assert.equal(result.coverage.taskPass,false);
  assert.equal(result.coverage.releasePass,false);
};
test('historical release remains immutable and defect only becomes planning candidate',()=>{
  const actual=projectMaintenanceIntake(input());safe(actual);
  assert.equal(actual.status,'CANDIDATES_ONLY');
  assert.equal(actual.priorRelease,'REPORTED_RELEASE_UNVERIFIED');
  assert.equal(actual.previous.immutable,true);
  assert.equal(actual.candidates[0].status,'PROPOSED');
  assert.equal(actual.candidates[0].nativeTaskCreated,false);
  assert.equal(actual.candidates[0].claimAccepted,false);
  assert.equal(actual.candidates[0].requirementRef,issue(92));
  assert.equal(actual.learning[0].status,'CANDIDATE_UNVERIFIED');
  assert.equal(actual.learning[0].adopted,false);
  assert.equal(actual.learning[0].measuredEffect,'NOT_MEASURED');
});
test('missing prior release can never be labeled released',()=>{
  const value=input();delete value.priorRelease;
  const actual=projectMaintenanceIntake(value);safe(actual);
  assert.equal(actual.priorRelease,'UNKNOWN');
  assert.equal(actual.candidates[0].priorRelease,null);
});
test('broken repository, SHA and release refs fail closed',()=>{
  for(const changes of [{repository:'../../evil'},
    {priorRelease:{...priorRelease,commitSha:'not-a-sha'}},
    {priorRelease:{...priorRelease,evidenceRef:'https://evil.example/release'}},
    {observedAsOf:'unknown'},
    {defects:[{...defect,subjectSha:'bad'}]},
    {defects:[{...defect,issueRef:'https://github.com/other/repo/issues/1'}]},
    {defects:[{...defect,requirementRef:'https://github.com/other/repo/issues/2'}]}]){
    const actual=projectMaintenanceIntake({...input(),...changes});safe(actual);
    assert.equal(actual.status,'UNKNOWN');
  }
});
test('duplicate defect ref refuses misleading candidate counts',()=>{
  const actual=projectMaintenanceIntake({...input(),defects:[defect,defect]});safe(actual);
  assert.equal(actual.status,'UNKNOWN');
  assert.equal(actual.candidates.length,0);
});
test('future data cannot enter an as-of snapshot',()=>{
  const future='2026-11-01T00:00:00Z';
  for(const changes of [{defects:[{...defect,observedAt:future}]},
    {learning:[{...learning,observedAt:future}]}]){
    const actual=projectMaintenanceIntake({...input(),...changes});safe(actual);
    assert.equal(actual.status,'UNKNOWN');
  }
});
test('unknown task learning is not claimed as quality gain',()=>{
  for(const qualification of ['UNKNOWN','NOT_RUN']){
    const actual=projectMaintenanceIntake({...input(),learning:[{...learning,qualification}]});safe(actual);
    assert.equal(actual.learning[0].status,qualification);
    assert.equal(actual.learning[0].measuredEffect,'NOT_MEASURED');
    assert.equal(actual.coverage.metrics,'UNKNOWN');
  }
});
test('reported evidence cannot be upgraded to native Task or product freeze',()=>{
  const actual=projectMaintenanceIntake({...input(),nativeWriteAuthorized:true,
    releasePass:true,defects:[{...defect,claimAccepted:true}]});safe(actual);
  assert.equal(actual.candidates[0].claimAccepted,false);
  assert.equal(actual.candidates[0].authorized,false);
});
test('bad or absent inputs do not invent history',()=>{
  for(const bad of [null,undefined,{}, {...input(),defects:{}},
    {...input(),learning:[{...learning,qualification:'PASS'}]}]){
    const actual=projectMaintenanceIntake(bad);safe(actual);
    assert.equal(actual.status,'UNKNOWN');
  }
});
// J08a integration pins over the existing lineage/casebook/planning/feedback semantics.
test('absent task-meta stays UNKNOWN, never blocks defect candidates nor invents metrics',()=>{
  const value=input();delete value.learning;
  const actual=projectMaintenanceIntake(value);safe(actual);
  assert.equal(actual.status,'CANDIDATES_ONLY');
  assert.equal(actual.candidates.length,1);
  assert.equal(actual.learning.length,0);
  assert.equal(actual.coverage.learningSuggestions,0);
  assert.equal(actual.coverage.metrics,'UNKNOWN');
  assert.equal(actual.coverage.activeHours,'NOT_MEASURED');
  assert.equal(actual.coverage.comparison,'COUNTS_NOT_PERCENTAGES');
});
test('candidate stays a source-linked advisory aligned with sibling projections',()=>{
  const actual=projectMaintenanceIntake(input());safe(actual);
  assert.equal(actual.candidates[0].sourceIssueRef,issue(207));
  assert.equal(actual.candidates[0].subjectSha,subjectSha);
  assert.equal(actual.candidates[0].priorReleaseEvidence,'REPORTED_RELEASE_UNVERIFIED');
  assert.equal(actual.candidates[0].nextAction,'PRODUCT_REVIEW_OR_AUTHORIZED_INTAKE');
  assert.match(actual.candidates[0].mustNot.join(' '),/DO_NOT_REWRITE_HISTORICAL_RELEASE/);
  assert.match(actual.candidates[0].mustNot.join(' '),/DO_NOT_AUTO_CREATE_NATIVE_TASK/);
  assert.equal(actual.previous.commitSha,subjectSha);
  assert.equal(actual.previous.version,'v1.5.5');
  assert.equal(actual.learning[0].provenance,'GITHUB_REPORTED_NOT_INDEPENDENTLY_ATTESTED');
  assert.equal(actual.learning[0].noAgentRanking,true);
  assert.equal(actual.coverage.defectCandidates,1);
  assert.equal(actual.coverage.learningSuggestions,1);
});
test('observation exactly at the as-of boundary is inside the snapshot',()=>{
  const actual=projectMaintenanceIntake({...input(),learning:[{...learning,observedAt:observedAsOf}]});
  safe(actual);
  assert.equal(actual.status,'CANDIDATES_ONLY');
  assert.equal(actual.learning[0].status,'CANDIDATE_UNVERIFIED');
});
