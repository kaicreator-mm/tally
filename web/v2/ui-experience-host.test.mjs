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
  decision:'ADOPTED',decisionReceiptRef:'https://github.com/org/tally/issues/52#issuecomment-1'};
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

  // A positive usage witness may name only a previously adopted, verified,
  // explicitly eligible candidate. Keep the shared REVIEWED fixture ineligible.
  const eligibleCandidate={...candidate,state:'ADOPTED',eligibleForNextTask:true};
  const h=createUiExperienceHost({repository,versionId,expectedSha:sha,
    selectedHost:{
      readExperienceCandidates:async()=>({...snap(),candidates:[eligibleCandidate]}),
      readNextTaskContextUsage:async()=>({connected:true,source,
        taskIssue:20,usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})
    }});
  const r=await h.nextTaskUsage(20);
  assert.equal(r.connected,true);
  assert.deepEqual(r.usedCandidates,['lesson-1']);
  assert.equal(r.measuredBenefit,'UNKNOWN');

  // A REVIEWED candidate is not eligible for a real applied-usage receipt.
  const ineligible=await make({readNextTaskContextUsage:async()=>({connected:true,source,
    taskIssue:20,usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})})
    .nextTaskUsage(20);
  assert.equal(ineligible.connected,false);
  assert.equal(ineligible.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
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
      commentId:1001,issueNumber:60,namespace:'tally',dataDigest:'d'.repeat(64),
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
    sourceSha:sha,candidateId:suggestion.learningId,revision:2,decision:'ADOPT',
    evidenceRef:suggestion.provenance.publicationRef,
    publisherRepository:repository,reportQualification:'VERIFIED'});
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


test('real isolated external ADS report stays UNKNOWN and permits REJECT only',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const publicationRef='https://github.com/'+publisherRepository+
    '/issues/9#issuecomment-6381704604';
  const staleSuggestion={learningId:'a'.repeat(64),
    taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/9',
    subjectSha:'5b7072d78016052b1ea816b2b36322b946ef5c78',
    qualification:'UNKNOWN',
    provenance:{reportId:'task-meta:sess_aebeaf3e',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704604,issueNumber:9,
      namespace:'task-meta:'+publisherRepository,dataDigest:'e'.repeat(64),
      publicationRef},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  let decision=null,recordCalls=0;
  const controller={
    async dispatch(_sha,kind,args){
      if(kind==='load-learning')return {status:'OK',
        reason:'PUBLIC_ADS_REPORTS_OBSERVED',suggestions:1};
      if(kind==='record-learning'){
        recordCalls++;decision=args.decision;
        return {state:'LEARNING_RECORDED',decision,claimAccepted:true,
          admitted:true,authorized:false,taskPass:false,releasePass:false,
          githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
          noAgentRanking:true,qualityGainClaimed:false};
      }
      return {status:'DENIED'};
    },
    snapshot(){return {status:'OBSERVED',sourceSha:sha,learning:[{
      ...staleSuggestion,decision,recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED',
      qualityGainClaimed:false,noAgentRanking:true,measuredEffect:'NOT_MEASURED'}]};}
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository]});
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:{
      ...adapter,
      readNextTaskContextUsage:async()=>({connected:true,source,taskIssue:20,
        usageState:'EXPLICITLY_NOT_APPLIED',usedCandidates:[]})
    }});
  const listed=await host.candidates();
  assert.equal(listed.connected,true);
  assert.equal(listed.candidates[0].reportQualification,'UNKNOWN');
  assert.equal(listed.candidates[0].sourceKind,'ISOLATED_E2E_TEST_ONLY');
  assert.equal(listed.candidates[0].eligibleForNextTask,false);
  assert.equal(listed.candidates[0].sourceRef,publicationRef);

  const adopted=await host.decision({...request,candidateId:staleSuggestion.learningId,
    revision:1,decision:'ADOPT'});
  assert.equal(adopted.status,'DENIED');
  assert.equal(recordCalls,0);

  const rejected=await host.decision({...request,candidateId:staleSuggestion.learningId,
    revision:1,decision:'REJECT'});
  assert.equal(rejected.status,'OBSERVED');
  assert.equal(rejected.ok,true);
  assert.equal(decision,'REJECTED');
  assert.equal(recordCalls,1);

  const usage=await host.nextTaskUsage(20);
  assert.equal(usage.connected,true);
  assert.equal(usage.usageState,'EXPLICITLY_NOT_APPLIED');
  assert.deepEqual(usage.usedCandidates,[]);
  assert.equal(usage.measuredBenefit,'UNKNOWN');
});

test('selected maintenance Host independently refuses ADOPT for UNKNOWN report',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const staleSuggestion={learningId:'b'.repeat(64),
    taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/9',
    subjectSha:null,qualification:'UNKNOWN',
    provenance:{reportId:'task-meta:unknown',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704604,issueNumber:9,
      namespace:'task-meta:'+publisherRepository,dataDigest:'f'.repeat(64),
      publicationRef:'https://github.com/'+publisherRepository+
        '/issues/9#issuecomment-6381704604'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  let recordCalls=0;
  const controller={
    dispatch:async(_sha,kind)=>kind==='load-learning'?{status:'OK',
      reason:'PUBLIC_ADS_REPORTS_OBSERVED'}:
      (recordCalls++,{state:'LEARNING_RECORDED'}),
    snapshot:()=>({status:'OBSERVED',sourceSha:sha,learning:[staleSuggestion]})
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository]});
  const result=await adapter.dispatchExperienceDecision({repository,versionId,
    sourceSha:sha,candidateId:staleSuggestion.learningId,revision:1,decision:'ADOPT',
    evidenceRef:staleSuggestion.provenance.publicationRef,
    publisherRepository,reportQualification:'UNKNOWN'});
  assert.equal(result.status,'DENIED');
  assert.equal(result.reason,'UNQUALIFIED_REPORT_CANNOT_BE_ADOPTED');
  assert.equal(recordCalls,0);
});


test('STALE ADS report remains visibly STALE and permits REJECT only',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const publicationRef='https://github.com/'+publisherRepository+
    '/issues/10#issuecomment-6381704605';
  const staleSuggestion={learningId:'c'.repeat(64),taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/10',
    subjectSha:null,qualification:'STALE',
    provenance:{reportId:'task-meta:stale',revision:2,
      kind:'EXECUTION_SUMMARY',commentId:6381704605,issueNumber:10,
      namespace:'task-meta:'+publisherRepository,dataDigest:'c'.repeat(64),
      publicationRef},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  let decision=null,recordCalls=0;
  const controller={
    async dispatch(_sha,kind,args){
      if(kind==='load-learning')return {status:'OK',
        reason:'PUBLIC_ADS_REPORTS_OBSERVED'};
      if(kind==='record-learning'){
        recordCalls++;decision=args.decision;
        return {state:'LEARNING_RECORDED',decision,claimAccepted:true,
          admitted:true,authorized:false,taskPass:false,releasePass:false,
          githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
          noAgentRanking:true,qualityGainClaimed:false};
      }
      return {status:'DENIED'};
    },
    snapshot(){return {status:'OBSERVED',sourceSha:sha,learning:[{
      ...staleSuggestion,decision,recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED',
      qualityGainClaimed:false,noAgentRanking:true,measuredEffect:'NOT_MEASURED'}]};}
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository]});
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:{
      ...adapter,readNextTaskContextUsage:async({taskIssue})=>({
        connected:true,source:{repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'},
        taskIssue,usageState:'EXPLICITLY_NOT_APPLIED',usedCandidates:[]})
    }});
  const listed=await host.candidates();
  assert.equal(listed.connected,true);
  assert.equal(listed.candidates[0].reportQualification,'STALE');
  const adopted=await host.decision({...request,candidateId:staleSuggestion.learningId,
    revision:2,decision:'ADOPT'});
  assert.equal(adopted.status,'DENIED');
  assert.equal(recordCalls,0);
  const rejected=await host.decision({...request,candidateId:staleSuggestion.learningId,
    revision:2,decision:'REJECT'});
  assert.equal(rejected.status,'OBSERVED');
  assert.equal(rejected.ok,true);
  assert.equal(decision,'REJECTED');
  assert.equal(recordCalls,1);
  const usage=await host.nextTaskUsage(20);
  assert.equal(usage.connected,true);
  assert.equal(usage.usageState,'EXPLICITLY_NOT_APPLIED');
  assert.deepEqual(usage.usedCandidates,[]);
  assert.equal(usage.measuredBenefit,'UNKNOWN');
});


test('UNVERIFIED ADS evidence is preserved and permits REJECT but never ADOPT',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const suggestion={learningId:'d'.repeat(64),taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/11',
    subjectSha:null,qualification:'UNVERIFIED',
    provenance:{reportId:'task-meta:unverified',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704606,issueNumber:11,
      namespace:'task-meta:'+publisherRepository,dataDigest:'d'.repeat(64),
      publicationRef:'https://github.com/'+publisherRepository+
        '/issues/11#issuecomment-6381704606'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  let decision=null,recordCalls=0;
  const controller={
    async dispatch(_sha,kind,args){
      if(kind==='load-learning')return {status:'OK',reason:'PUBLIC_ADS_REPORTS_OBSERVED'};
      if(kind==='record-learning'){
        recordCalls++;decision=args.decision;
        return {state:'LEARNING_RECORDED',decision,claimAccepted:true,admitted:true,
          authorized:false,taskPass:false,releasePass:false,githubEffect:'NOT_RUN',
          measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
      }
      return {status:'DENIED'};
    },
    snapshot(){return {status:'OBSERVED',sourceSha:sha,learning:[{
      ...suggestion,decision,recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED'}]};}
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository]});
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:adapter});
  const listed=await host.candidates();
  assert.equal(listed.connected,true);
  assert.equal(listed.candidates[0].reportQualification,'UNVERIFIED');
  const adopted=await host.decision({...request,candidateId:suggestion.learningId,
    revision:1,decision:'ADOPT'});
  assert.equal(adopted.status,'DENIED');
  assert.equal(recordCalls,0);
  const rejected=await host.decision({...request,candidateId:suggestion.learningId,
    revision:1,decision:'REJECT'});
  assert.equal(rejected.status,'OBSERVED');
  assert.equal(decision,'REJECTED');
  assert.equal(recordCalls,1);
});

test('rejected learning cannot appear in a later Task Context Pack usage readback',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const id='e'.repeat(64);
  const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
  const candidate={id,revision:1,title:'Rejected stale report',state:'REJECTED',
    reportQualification:'STALE',sourceRef:'https://github.com/'+publisherRepository+
      '/issues/12#issuecomment-6381704607',publisherRepository,
    eligibleForNextTask:false,sourceKind:'PUBLIC_ADS_REPORT'};
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:{
      readExperienceCandidates:async()=>({connected:true,source,versionId,candidates:[candidate]}),
      readNextTaskContextUsage:async()=>({connected:true,source,taskIssue:21,
        usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]})
    }});
  const result=await host.nextTaskUsage(21);
  assert.equal(result.connected,false);
  assert.equal(result.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
});

test('next-Task usage accepts only unique adopted verified eligible candidates',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
  const id='e'.repeat(64);
  const candidate=(overrides={})=>({
    id,revision:1,title:'Verified adopted report',state:'ADOPTED',
    reportQualification:'VERIFIED',
    sourceRef:'https://github.com/'+publisherRepository+
      '/issues/12#issuecomment-6381704607',
    publisherRepository,eligibleForNextTask:true,sourceKind:'PUBLIC_ADS_REPORT',
    ...overrides
  });
  const hostFor=(item,usage)=>createUiExperienceHost({
    repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],
    selectedHost:{
      readExperienceCandidates:async()=>({connected:true,source,versionId,
        candidates:[item]}),
      readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,
        taskIssue,...usage})
    }
  });
  const applied=await hostFor(candidate(),{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(21);
  assert.equal(applied.connected,true);
  assert.equal(applied.usageState,'EXPLICITLY_APPLIED');
  assert.deepEqual(applied.usedCandidates,[id]);

  const proposed=await hostFor(candidate({state:'PROPOSED',
    eligibleForNextTask:false}),{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(21);
  assert.equal(proposed.connected,false);
  assert.equal(proposed.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const stale=await hostFor(candidate({state:'PROPOSED',
    reportQualification:'STALE',eligibleForNextTask:false}),{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(21);
  assert.equal(stale.connected,false);
  assert.equal(stale.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const notApplied=await hostFor(candidate(),{
    usageState:'EXPLICITLY_NOT_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(21);
  assert.equal(notApplied.connected,false);
  assert.equal(notApplied.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const duplicate=await hostFor(candidate(),{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id,id]
  }).nextTaskUsage(21);
  assert.equal(duplicate.connected,false);
  assert.equal(duplicate.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const unknown=await hostFor(candidate(),{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:['f'.repeat(64)]
  }).nextTaskUsage(21);
  assert.equal(unknown.connected,false);
  assert.equal(unknown.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const emptyNotApplied=await hostFor(candidate(),{
    usageState:'EXPLICITLY_NOT_APPLIED',usedCandidates:[]
  }).nextTaskUsage(21);
  assert.equal(emptyNotApplied.connected,true);
  assert.equal(emptyNotApplied.usageState,'EXPLICITLY_NOT_APPLIED');
});

test('next-Task usage refuses a candidate set that changes during readback',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
  const id='a'.repeat(64);
  const candidate=(state,eligible)=>({
    id,revision:state==='ADOPTED'?1:2,title:'Changing candidate',
    state,reportQualification:'VERIFIED',
    sourceRef:'https://github.com/'+publisherRepository+
      '/issues/13#issuecomment-6381704608',
    publisherRepository,eligibleForNextTask:eligible,sourceKind:'PUBLIC_ADS_REPORT'
  });
  let reads=0;
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:{
      readExperienceCandidates:async()=>{
        reads++;
        return {connected:true,source,versionId,
          candidates:[reads===1?candidate('ADOPTED',true):candidate('REJECTED',false)]};
      },
      readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,
        taskIssue,usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]})
    }});
  const result=await host.nextTaskUsage(22);
  assert.equal(reads,2);
  assert.equal(result.connected,false);
  assert.equal(result.reason,'EXPERIENCE_SOURCE_CHANGED_DURING_USAGE_READ');
});
