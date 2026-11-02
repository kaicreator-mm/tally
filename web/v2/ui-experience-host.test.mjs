import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiExperienceHost} from './ui-experience-host.mjs';
const repository='org/tally',versionId='version/v2',sha='a'.repeat(40);
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const candidate={id:'lesson-1',revision:1,title:'Validated rule',state:'REVIEWED',
  reportQualification:'VERIFIED',sourceRef:'https://github.com/org/tally/issues/52',
  eligibleForNextTask:false};
const snap=()=>({connected:true,source,versionId,candidates:[candidate]});
const request={repository,versionId,sourceSha:sha,candidateId:'lesson-1',
  revision:1,decision:'ADOPT'};
const receipt={status:'OBSERVED',candidateId:'lesson-1',sourceSha:sha,
  decisionReceiptRef:'https://github.com/org/tally/issues/52#issuecomment-1'};
const make=(overrides={})=>createUiExperienceHost({repository,versionId,
  expectedSha:sha,selectedHost:{readExperienceCandidates:async()=>snap(),
    dispatchExperienceDecision:async()=>receipt,...overrides}});
test('actual selected evidence is projected without private reviewer grant',async()=>{
  const r=await make().candidates();
  assert.equal(r.connected,true);
  assert.equal(r.qualification,'SOURCE_OBSERVED_NOT_MEASURED_OUTCOME');
  assert.equal(r.candidates[0].sourceRef,candidate.sourceRef);
});
test('no Host, stale or stale ADS report cannot be adopted',async()=>{
  assert.equal((await createUiExperienceHost({repository,versionId,expectedSha:sha})
    .candidates()).connected,false);
  let called=0;
  const h=make({readExperienceCandidates:async()=>({...snap(),candidates:[
    {...candidate,reportQualification:'STALE'}]}),
    dispatchExperienceDecision:async()=>{called++;return receipt;}});
  assert.equal((await h.decision(request)).status,'DENIED');
  assert.equal(called,0);
});
test('user cannot inject grant, outside repo or stale revision',async()=>{
  let calls=0;
  const h=make({dispatchExperienceDecision:async()=>{calls++;return receipt;}});
  for(const r of [{...request,grant:'GRANTED'},
    {...request,revision:2},{...request,repository:'evil/repo'},
    {...request,decision:'AUTO_ADOPT'}])assert.equal((await h.decision(r)).status,'DENIED');
  assert.equal(calls,0);
});
test('reviewed candidate has a Host-gated decision and no auto context reuse',async()=>{
  let calls=0;
  const r=await make({dispatchExperienceDecision:async()=>{calls++;return receipt;}})
    .decision(request);
  assert.equal(calls,1);assert.equal(r.ok,true);
  assert.equal(r.nextTaskUsage,'NOT_VERIFIED');
  assert.equal(r.measuredBenefit,'UNKNOWN');
});
test('candidate changed during recheck blocks dispatch',async()=>{
  let reads=0,calls=0;
  const h=make({readExperienceCandidates:async()=>({...snap(),candidates:[
    {...candidate,revision:++reads}]}),
    dispatchExperienceDecision:async()=>{calls++;return receipt;}});
  assert.equal((await h.decision(request)).reason,'EXPERIENCE_DECISION_STALE');
  assert.equal(calls,0);
});
test('Host cannot smuggle unqualified benefit or foreign receipt',async()=>{
  for(const bad of [{...receipt,qualityImproved:true},
    {...receipt,taskPass:true},{...receipt,decisionReceiptRef:'https://evil/repo'}]){
    assert.equal((await make({dispatchExperienceDecision:async()=>bad})
      .decision(request)).status,'DENIED');
  }
});
test('next Task Context usage requires independent selected Host witness',async()=>{
  assert.equal((await make().nextTaskUsage(20)).connected,false);
  const h=make({readNextTaskContextUsage:async()=>({connected:true,source,
    taskIssue:20,usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})});
  const r=await h.nextTaskUsage(20);
  assert.equal(r.connected,true);
  assert.deepEqual(r.usedCandidates,['lesson-1']);
  assert.equal(r.measuredBenefit,'UNKNOWN');
});
test('dispatch transport failure stays UNKNOWN without automatic repeat',async()=>{
  let calls=0;
  const r=await make({dispatchExperienceDecision:async()=>{calls++;throw Error('token secret');}})
    .decision(request);
  assert.equal(calls,1);assert.equal(r.status,'UNKNOWN');
  assert.equal(JSON.stringify(r).includes('token secret'),false);
});

import {createMaintenanceControllerExperienceAdapter} from './ui-experience-host.mjs';
test('existing maintenance controller adapter preserves ADS provenance and sealed K decision',async()=>{
  const suggestion={learningId:'f'.repeat(64),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',
      publicationRef:'https://github.com/org/tally/issues/60#issuecomment-1001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  let decision=null;
  const controller={
    async dispatch(_sha,kind,args){
      if(kind==='load-learning')return {status:'OK',
        reason:'PUBLIC_ADS_REPORTS_OBSERVED',suggestions:1};
      if(kind==='record-learning'){
        decision=args.decision;
        return {state:'LEARNING_RECORDED',decision,claimAccepted:true,
          admitted:true,authorized:false,taskPass:false,releasePass:false,
          githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
          noAgentRanking:true,qualityGainClaimed:false};
      }
      return {status:'DENIED'};
    },
    snapshot(){return {status:'OBSERVED',sourceSha:sha,learning:[{
      ...suggestion,decision,recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED',
      qualityGainClaimed:false,noAgentRanking:true,measuredEffect:'NOT_MEASURED'}]};}
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha});
  const source=await adapter.readExperienceCandidates({repository,versionId,sourceSha:sha});
  assert.equal(source.connected,true);
  assert.equal(source.candidates[0].state,'PROPOSED');
  assert.equal(source.candidates[0].sourceRef,suggestion.provenance.publicationRef);
  const result=await adapter.dispatchExperienceDecision({repository,versionId,
    sourceSha:sha,candidateId:suggestion.learningId,revision:2,decision:'ADOPT'});
  assert.equal(result.status,'OBSERVED');
  assert.equal(result.decisionReceiptAuthority,
    'SELECTED_HOST_K_RESPONSE_AND_HOST_SNAPSHOT');
  assert.equal(result.decisionReceiptRef,null);
  assert.equal(decision,'ADOPTED');
});
test('maintenance experience adapter rejects malformed report provenance and foreign scope',async()=>{
  const suggestion={learningId:'f'.repeat(64),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',
      publicationRef:'https://evil.example/issues/60#issuecomment-1001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const controller={dispatch:async()=>({status:'OK',
    reason:'PUBLIC_ADS_REPORTS_OBSERVED'}),
    snapshot:()=>({status:'OBSERVED',sourceSha:sha,learning:[suggestion]})};
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha});
  assert.equal((await adapter.readExperienceCandidates()).reason,
    'ADS_LEARNING_PROVENANCE_UNQUALIFIED');
  assert.equal((await adapter.dispatchExperienceDecision({repository:'evil/repo',
    versionId,sourceSha:sha,candidateId:'f'.repeat(64),revision:2,
    decision:'ADOPT'})).status,'DENIED');
});
