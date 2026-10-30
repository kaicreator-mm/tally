import test from 'node:test';
import assert from 'node:assert/strict';
import {planNextVersion} from '../src/planning.mjs';

const repo='kaicreator-mm/domain-harness',planningAsOf='2026-10-31T06:00:00.000Z';
const ref=number=>`https://github.com/${repo}/issues/${number}`;
const timeline={subjectStatus:'OBSERVED',subject:{sourceRef:ref(530)}};
const diagnoses=(rows=[],extra={})=>({status:'OBSERVED',diagnoses:rows,...extra});
const diag=(number,classification)=>({issue:`${repo}#${number}`,classification,sourceRefs:[ref(number)]});
const capability=(lane,extra={})=>({lane,userOutcome:`Deliver ${lane}`,owner:'Engineering B',blockedBy:[],writeSet:[`${lane}/src`],
  validation:[],hostRequired:'LocalWindows',environment:'LocalWindows',role:'builder',roleEvidence:true,
  sourceRefs:[ref(530)],evidenceIssues:[530],evidence:[{sourceRef:ref(530),observedAt:'2026-10-01T00:00:00.000Z'}],...extra});
const input=(capabilities,rows=[],extra={})=>({goal:'Deliver the next public Domain App package',nonGoals:['No automatic dispatch'],
  timeline,diagnoses:diagnoses(rows,extra),capabilities,planningAsOf});

test('public consumer false-serial P0 and campaign lessons produce bounded candidates',()=>{
  const result=planNextVersion(input([
    capability('public-consumer',{evidenceIssues:[530,1013],blockedBy:['v0.7-final-compatibility']}),
    capability('cas-repair',{evidenceIssues:[1100],sourceRefs:[ref(1100)]}),
    capability('verification-campaign',{evidenceIssues:[586],sourceRefs:[ref(586)],campaign:true})
  ],[diag(530,'POTENTIAL_FALSE_DEPENDENCY'),diag(1013,'INTEGRATION_COVERAGE_GAP'),
    diag(1100,'REQUIRED_SECURITY_REPAIR'),diag(586,'EFFECTIVE_REVIEW_CONVERGENCE')]));
  assert.equal(result.status,'CANDIDATE');assert.equal(result.dag.status,'CANDIDATE');
  const consumer=result.dag.lanes[0];
  assert.deepEqual(consumer.blockedBy,['v0.7-final-compatibility']);assert.equal(consumer.nativeDependenciesChanged,false);
  assert.equal(consumer.potentialParallelScope,'REVIEW_NATIVE_DEPENDENCY_ONLY');
  assert.deepEqual(consumer.requiredValidation,['PUBLIC_CONSUMER']);assert.equal(consumer.earlyIntegrationOwner,true);
  const p0=result.dag.lanes[1];assert.equal(p0.mustRetainInvariant,true);assert.equal(p0.fastPath,false);
  assert.match(p0.mustNot.join(' '),/DO_NOT_LOWER_SECURITY_INVARIANT/);
  assert.equal(result.dag.lanes[2].reuseExistingCampaign,true);
  assert.equal(result.l2Options[0].humanDecisionRequired,true);assert.equal(result.coverage.claimed,false);
});

test('generated manifest/dist conflicts require an alternative single-writer decision',()=>{
  const result=planNextVersion(input([
    capability('package-a',{evidenceIssues:[1354],writeSet:['dist','manifest'],sourceRefs:[ref(1354)]}),
    capability('package-b',{evidenceIssues:[1354],writeSet:['dist'],sourceRefs:[ref(1354)]})
  ],[diag(1354,'EFFECTIVE_REVIEW_CONVERGENCE')]));
  assert.equal(result.dag.conflicts.length,1);assert.equal(result.dag.conflicts[0].parallel,false);
  assert.deepEqual(result.dag.conflicts[0].shared,['dist']);
  assert.equal(result.dag.conflicts[0].alternative.humanDecisionRequired,true);
  assert.match(result.dag.conflicts[0].alternative.decision,/SINGLE_GENERATED_WRITER_OR_SERIALIZE/);
});

test('later findings are retrospective and incomparable cloud evidence gives no recommendation',()=>{
  const future=capability('cloud-agent',{environment:'Cloud',hostRequired:'LocalWindows',roleEvidence:true,
    evidence:[{sourceRef:ref(1264),observedAt:'2026-11-01T00:00:00.000Z'}],evidenceIssues:[1264],sourceRefs:[ref(1264)]});
  const result=planNextVersion(input([future],[diag(1264,'SELECTED_PATH_NOT_INVOKED')]));
  const lane=result.dag.lanes[0];
  assert.equal(lane.retrospective,'RETROSPECTIVE_ONLY');assert.equal(lane.agentRole,'NO_RECOMMENDATION/MORE_EVIDENCE');
  assert.equal(lane.agentRoleEvidence.comparable,false);assert.equal(lane.unresolvedDecision,'HUMAN_DECISION_REQUIRED');
  assert.equal(result.coverage.noRecommendationCount,1);assert.equal(result.coverage.retrospectiveCount,1);
  const unknownEnvironment=planNextVersion(input([capability('unknown-env',{environment:'cloud'})]));
  assert.equal(unknownEnvironment.dag.lanes[0].agentRole,'NO_RECOMMENDATION/MORE_EVIDENCE');
  assert.equal(unknownEnvironment.dag.lanes[0].agentRoleEvidence.comparable,false);
});

test('rejected options stay feedback and scoped Fast Path avoids forced full review',()=>{
  const result=planNextVersion(input([capability('small-fix',{lowRisk:true,scopedCi:'PASS',requiredReview:'SCOPED'})],[],
    {rejectedOptions:[{option:'cloud-agent',reason:'LOCAL_HOST_REQUIRED',sourceRef:ref(64)}]}));
  const lane=result.dag.lanes[0];assert.equal(lane.fastPath,true);assert.equal(lane.requiredReview,'SCOPED_REVIEW');
  assert.equal(result.feedback.rejectedOptions[0].historicalFeedback,true);
  assert.equal(result.feedback.rejectedOptions[0].ruleActive,false);
  assert.equal(result.feedback.ruleActive,false);assert.equal(result.feedback.packageActive,false);
  const noEvidence=planNextVersion(input([capability('unproved-fast',{lowRisk:true,scopedCi:'PASS',sourceRefs:['invalid'],evidence:[]})]));
  assert.equal(noEvidence.dag.lanes[0].fastPath,false);assert.equal(noEvidence.dag.lanes[0].requiredReview,'INDEPENDENT_REVIEW');
  assert.equal(noEvidence.dag.lanes[0].agentRole,'NO_RECOMMENDATION/MORE_EVIDENCE');
});

test('invalid planning facts cannot create accepted Task or Release state',()=>{
  for(const value of [null,{},input(null),input([null]),input([],[null]),input([],[{issue:1,classification:'X',sourceRefs:[]}]),
    input([],[{issue:'a/b#xyz',classification:'REQUIRED_SECURITY_REPAIR',sourceRefs:[ref(1100)]}]),
    input([capability('bad-link',{evidenceIssues:['1100']})]),input([capability('bad-role',{role:5})])]){
    const result=planNextVersion(value);
    assert.equal(result.status,'UNKNOWN');assert.equal(result.dag.status,'UNKNOWN');
    assert.equal(result.coverage.taskPass,false);assert.equal(result.coverage.releasePass,false);
    assert.equal(result.coverage.claimed,false);assert.equal(result.feedback.packageActive,false);
    assert.equal(result.advisoryOnly,true);
  }
});
