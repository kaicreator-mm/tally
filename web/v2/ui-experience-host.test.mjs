import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {createUiExperienceHost} from './ui-experience-host.mjs';
const repository='org/tally',versionId='version/v2',sha='a'.repeat(40);
const learningIdFor=(reportId,revision,commentId,kind='EXECUTION_SUMMARY')=>
  createHash('sha256').update(JSON.stringify([reportId,revision,commentId,kind])).digest('hex');
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const candidate={id:'lesson-1',revision:1,title:'Validated rule',state:'REVIEWED',
  reportQualification:'VERIFIED',sourceRef:'https://github.com/org/tally/issues/52',
  eligibleForNextTask:false};
const snap=()=>({connected:true,source,versionId,candidates:[candidate]});
const request={repository,versionId,sourceSha:sha,candidateId:'lesson-1',
  revision:1,decision:'ADOPT'};
const receipt={status:'OBSERVED',candidateId:'lesson-1',sourceSha:sha,
  decision:'ADOPTED',decisionReceiptRef:'https://github.com/org/tally/issues/52#issuecomment-1'};
const ownerApprovalFor=learningId=>({
  state:'OWNER_APPROVAL_VERIFIED',reason:'GITHUB_OWNER_APPROVAL_VERIFIED',
  repository,issueNumber:219,sourceSha:sha,learningId,decision:'ADOPTED',
  commentId:21901,commentUrl:'https://github.com/'+repository+
    '/issues/219#issuecomment-21901',author:'owner-admin',permission:'admin'
});
const make=(overrides={})=>createUiExperienceHost({repository,versionId,
  expectedSha:sha,selectedHost:{readExperienceCandidates:async()=>snap(),
    dispatchExperienceDecision:async()=>receipt,...overrides}});
test('actual selected evidence is projected without private reviewer grant',async()=>{
  const r=await make().candidates();
  assert.equal(r.connected,true);
  assert.equal(r.qualification,'SOURCE_OBSERVED_NOT_MEASURED_OUTCOME');
  assert.equal(r.candidates[0].sourceRef,candidate.sourceRef);
});

test('experience source projection rejects unrecognized Host source fields',async()=>{
  const normal=await make().candidates();
  assert.deepEqual(Object.keys(normal.source).sort(),
    ['currentness','readback','repository','sourceSha']);
  const polluted=await make({readExperienceCandidates:async()=>({...snap(),
    source:{...source,internalField:'internal-only-value'}})}).candidates();
  assert.equal(polluted.connected,false);
  assert.equal(polluted.reason,'EXPERIENCE_PROVENANCE_UNQUALIFIED');
});
test('overlong candidate source references fail closed',async()=>{
  const sourceRef='https://github.com/org/tally/issues/'+'1'.repeat(300)+'#issuecomment-1';
  const result=await make({readExperienceCandidates:async()=>({...snap(),
    candidates:[{...candidate,sourceRef}]})}).candidates();
  assert.equal(result.connected,false);
  assert.equal(result.reason,'EXPERIENCE_PROVENANCE_UNQUALIFIED');
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
test('Host cannot smuggle unqualified benefit or foreign or noncanonical receipt',async()=>{
  for(const bad of [{...receipt,qualityImproved:true},
    {...receipt,taskPass:true},{...receipt,decisionReceiptRef:'https://evil/repo'},
    {...receipt,decisionReceiptRef:'https://github.com/org/tally/other'},
    {...receipt,decisionReceiptRef:'https://github.com/org/tally/issues/52?token=private'}]){
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
      readNextTaskContextUsage:async()=>({connected:true,source,versionId,
        taskIssue:20,usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})
    }});
  const r=await h.nextTaskUsage(20);
  assert.equal(r.connected,true);
  assert.deepEqual(r.usedCandidates,['lesson-1']);
  assert.equal(r.measuredBenefit,'UNKNOWN');

  // A REVIEWED candidate is not eligible for a real applied-usage receipt.
  const ineligible=await make({readNextTaskContextUsage:async()=>({connected:true,source,versionId,
    taskIssue:20,usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})})
    .nextTaskUsage(20);
  assert.equal(ineligible.connected,false);
  assert.equal(ineligible.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
});
test('next Task usage witness from another version is refused',async()=>{
  const eligibleCandidate={...candidate,state:'ADOPTED',eligibleForNextTask:true};
  const h=createUiExperienceHost({repository,versionId,expectedSha:sha,
    selectedHost:{
      readExperienceCandidates:async()=>({...snap(),candidates:[eligibleCandidate]}),
      readNextTaskContextUsage:async()=>({connected:true,source,
        versionId:'version/v3',taskIssue:20,
        usageState:'EXPLICITLY_APPLIED',usedCandidates:['lesson-1']})
    }});
  const result=await h.nextTaskUsage(20);
  assert.equal(result.connected,false);
  assert.equal(result.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
});
test('dispatch transport failure stays UNKNOWN without automatic repeat',async()=>{
  let calls=0;
  const r=await make({dispatchExperienceDecision:async()=>{calls++;throw Error('token secret');}})
    .decision(request);
  assert.equal(calls,1);assert.equal(r.status,'UNKNOWN');
  assert.equal(JSON.stringify(r).includes('token secret'),false);
});

import {createMaintenanceControllerExperienceAdapter} from './ui-experience-host.mjs';
import {createMaintenanceController} from './maintenance-controller.mjs';

test('real maintenance controller keeps recorded and unconfirmed decisions distinct',async()=>{
  const reportId='real-controller',commentId=1001,revision=1;
  const suggestion={learningId:learningIdFor(reportId,revision,commentId),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',subjectSha:sha,
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId,revision,kind:'EXECUTION_SUMMARY',commentId,issueNumber:60,
      namespace:'tally',repository,sourceSha:sha,dataDigest:'d'.repeat(64),
      publicationRef:'https://github.com/org/tally/issues/60#issuecomment-1001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const ownerApproval=ownerApprovalFor(suggestion.learningId);
  for(const fault of [null,'claim','replay','authorized']){
    const controller=createMaintenanceController({currentSourceSha:sha},{
      host:{readTaskLearning:async()=>({suggestions:[suggestion]})},
      sealedDispatch:async({input})=>({phase:'terminal',outcome:{ok:true,output:{
        state:fault==='replay'?'REPLAY_COMPLETE':'LEARNING_RECORDED',
        reason:fault==='replay'?'REPLAY_FIXTURE_ONLY':'LEARNING_DECISION_RECORDED_IN_HOST_CAS',
        issueRef:'https://github.com/kaicreator-mm/tally/issues/219',
        sourceSha:input.sourceSha,learningId:input.learningId,decision:input.decision,
        claimGeneration:1,claimAccepted:fault!=='claim',admitted:true,
        authorized:fault==='authorized',
        replayFixture:fault==='replay',githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
        noAgentRanking:true,qualityGainClaimed:false,taskPass:false,releasePass:false,
        ...(fault===null?{ownerApproval}:{})}}})});
    const adapter=createMaintenanceControllerExperienceAdapter({
      maintenanceController:controller,repository,versionId,expectedSha:sha});
    const request={repository,versionId,sourceSha:sha,candidateId:suggestion.learningId,
      revision,decision:'ADOPT',evidenceRef:suggestion.provenance.publicationRef,
      publisherRepository:repository,reportQualification:'VERIFIED'};
    const result=await adapter.dispatchExperienceDecision(request);
    assert.equal(result.status,fault===null?'OBSERVED':'UNKNOWN');
    const read=await adapter.readExperienceCandidates();
    if(fault===null){
      assert.equal(controller.snapshot().learning[0].recordState,'LEARNING_RECORDED');
      assert.equal(read.candidates[0].state,'ADOPTED');
    }else{
      assert.notEqual(controller.snapshot().learning[0].recordState,'LEARNING_RECORDED');
      assert.equal(read.connected,false);
      assert.equal(read.reason,'ADS_LEARNING_DECISION_UNRECONCILED');
    }
  }
});
test('existing maintenance controller adapter preserves ADS provenance and sealed K decision',async()=>{
  const suggestion={learningId:learningIdFor('ads-report-1',2,1001),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',subjectSha:sha,
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',repository,sourceSha:sha,dataDigest:'d'.repeat(64),
      publicationRef:'https://github.com/org/tally/issues/60#issuecomment-1001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const ownerApproval=ownerApprovalFor(suggestion.learningId);
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
          noAgentRanking:true,qualityGainClaimed:false,ownerApproval};
      }
      return {status:'DENIED'};
    },
    snapshot(){return {status:'OBSERVED',sourceSha:sha,learning:[{
      ...suggestion,decision,recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED',
      ...(decision==='ADOPTED'?{ownerApproval}:{}),
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
  const adopted=await adapter.readExperienceCandidates();
  assert.equal(adopted.candidates[0].eligibleForNextTask,true);
});
test('maintenance adapter permits next-Task use only with the exact Host Owner approval',async()=>{
  const suggestion={learningId:learningIdFor('ads-report-1',2,1001),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',subjectSha:sha,
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',repository,sourceSha:sha,
      dataDigest:'d'.repeat(64),
      publicationRef:'https://github.com/org/tally/issues/60#issuecomment-1001'},
    decision:'ADOPTED',recordState:'LEARNING_RECORDED',
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const approval=ownerApprovalFor(suggestion.learningId);
  const makeHost=ownerApproval=>{
    const row={...suggestion,ownerApproval};
    const controller={dispatch:async()=>({status:'OK',
      reason:'PUBLIC_ADS_REPORTS_OBSERVED'}),
      snapshot:()=>({status:'OBSERVED',sourceSha:sha,learning:[row]})};
    const adapter=createMaintenanceControllerExperienceAdapter({
      maintenanceController:controller,repository,versionId,expectedSha:sha});
    return createUiExperienceHost({repository,versionId,expectedSha:sha,
      selectedHost:{...adapter,
        readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,
          versionId,taskIssue,usageState:'EXPLICITLY_APPLIED',
          usedCandidates:[suggestion.learningId]})}});
  };
  const valid=makeHost(approval);
  const listed=await valid.candidates();
  assert.equal(listed.candidates[0].eligibleForNextTask,true);
  const applied=await valid.nextTaskUsage(21);
  assert.equal(applied.connected,true);
  assert.equal(applied.usageState,'EXPLICITLY_APPLIED');
  assert.deepEqual(applied.usedCandidates,[suggestion.learningId]);

  for(const invalid of [null,{...approval,sourceSha:'b'.repeat(40)},
    {...approval,learningId:'c'.repeat(64)},{...approval,permission:'write'},
    {...approval,commentUrl:'https://github.com/org/tally/issues/219'},
    {...approval,untrustedExtra:true}]){
    const blocked=await makeHost(invalid).nextTaskUsage(21);
    assert.equal(blocked.connected,false);
    assert.equal(blocked.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
  }
});
test('maintenance experience decision refuses post-CAS readback from a moved source',async()=>{
  const suggestion={learningId:learningIdFor('ads-report-1',2,1001),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',subjectSha:sha,
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',repository,sourceSha:sha,dataDigest:'d'.repeat(64),
      publicationRef:'https://github.com/org/tally/issues/60#issuecomment-1001'},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const ownerApproval=ownerApprovalFor(suggestion.learningId);
  let decision=null,snapshotReads=0,recordCalls=0;
  const controller={
    async dispatch(_sha,kind,args){
      if(kind==='load-learning')return {status:'OK',
        reason:'PUBLIC_ADS_REPORTS_OBSERVED',suggestions:1};
      if(kind==='record-learning'){
        recordCalls++;decision=args.decision;
        return {state:'LEARNING_RECORDED',decision,claimAccepted:true,
          admitted:true,authorized:false,taskPass:false,releasePass:false,
          githubEffect:'NOT_RUN',measuredEffect:'NOT_MEASURED',
          noAgentRanking:true,qualityGainClaimed:false,ownerApproval};
      }
      return {status:'DENIED'};
    },
    snapshot(){
      snapshotReads++;
      return {status:'OBSERVED',sourceSha:snapshotReads===1?sha:'b'.repeat(40),
        learning:[{...suggestion,decision,
          recordState:decision?'LEARNING_RECORDED':'NOT_RECORDED',
          ...(decision==='ADOPTED'?{ownerApproval}:{}),
          qualityGainClaimed:false,noAgentRanking:true,measuredEffect:'NOT_MEASURED'}]};
    }
  };
  const adapter=createMaintenanceControllerExperienceAdapter({
    maintenanceController:controller,repository,versionId,expectedSha:sha});
  const result=await adapter.dispatchExperienceDecision({repository,versionId,
    sourceSha:sha,candidateId:suggestion.learningId,revision:2,decision:'ADOPT',
    evidenceRef:suggestion.provenance.publicationRef,
    publisherRepository:repository,reportQualification:'VERIFIED'});
  assert.equal(recordCalls,1);
  assert.equal(snapshotReads,2);
  assert.equal(result.status,'UNKNOWN');
  assert.equal(result.reason,'LEARNING_CAS_RESULT_NOT_RECONCILED');
});

test('maintenance experience adapter rejects malformed report provenance and foreign scope',async()=>{
  const suggestion={learningId:learningIdFor('ads-report-1',2,1001),
    taskIssueRef:'https://github.com/org/tally/issues/52',
    targetIssueRef:'https://github.com/org/tally/issues/60',subjectSha:sha,
    qualification:'REPORTED_QUALIFIED',
    provenance:{reportId:'ads-report-1',revision:2,kind:'EXECUTION_SUMMARY',
      commentId:1001,issueNumber:60,namespace:'tally',repository,sourceSha:sha,dataDigest:'d'.repeat(64),
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
  const staleSuggestion={learningId:learningIdFor('task-meta:sess_aebeaf3e',1,6381704604),
    taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/9',
    subjectSha:'5b7072d78016052b1ea816b2b36322b946ef5c78',
    qualification:'UNKNOWN',
    provenance:{reportId:'task-meta:sess_aebeaf3e',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704604,issueNumber:9,
      namespace:'task-meta:'+publisherRepository,repository:publisherRepository,
      sourceSha:'5b7072d78016052b1ea816b2b36322b946ef5c78',dataDigest:'e'.repeat(64),
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
      readNextTaskContextUsage:async()=>({connected:true,source,versionId,taskIssue:20,
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
  const staleSuggestion={learningId:learningIdFor('task-meta:unknown',1,6381704604),
    taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/9',
    subjectSha:sha,qualification:'UNKNOWN',
    provenance:{reportId:'task-meta:unknown',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704604,issueNumber:9,
      namespace:'task-meta:'+publisherRepository,repository:publisherRepository,
      sourceSha:sha,dataDigest:'f'.repeat(64),
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
  assert.equal(result.reason,'UNQUALIFIED_OR_TEST_ONLY_REPORT_CANNOT_BE_ADOPTED');
  assert.equal(recordCalls,0);
});


test('STALE ADS report remains visibly STALE and permits REJECT only',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const publicationRef='https://github.com/'+publisherRepository+
    '/issues/10#issuecomment-6381704605';
  const staleSuggestion={learningId:learningIdFor('task-meta:stale',2,6381704605),taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/10',
    subjectSha:sha,qualification:'STALE',
    provenance:{reportId:'task-meta:stale',revision:2,
      kind:'EXECUTION_SUMMARY',commentId:6381704605,issueNumber:10,
      namespace:'task-meta:'+publisherRepository,repository:publisherRepository,
      sourceSha:sha,dataDigest:'c'.repeat(64),
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
        connected:true,versionId,source:{repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'},
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
  const suggestion={learningId:learningIdFor('task-meta:unverified',1,6381704606),taskIssueRef:null,
    targetIssueRef:'https://github.com/'+publisherRepository+'/issues/11',
    subjectSha:sha,qualification:'UNVERIFIED',
    provenance:{reportId:'task-meta:unverified',revision:1,
      kind:'EXECUTION_SUMMARY',commentId:6381704606,issueNumber:11,
      namespace:'task-meta:'+publisherRepository,repository:publisherRepository,
      sourceSha:sha,dataDigest:'d'.repeat(64),
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
  const publisherRepository='kaicreator-mm/task-meta';
  const id='e'.repeat(64);
  const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
  const candidate={id,revision:1,title:'Rejected stale report',state:'REJECTED',
    reportQualification:'STALE',sourceRef:'https://github.com/'+publisherRepository+
      '/issues/12#issuecomment-6381704607',publisherRepository,
    eligibleForNextTask:false,sourceKind:'PUBLIC_ADS_REPORT'};
  const host=createUiExperienceHost({repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],selectedHost:{
      readExperienceCandidates:async()=>({connected:true,source,versionId,candidates:[candidate]}),
      readNextTaskContextUsage:async()=>({connected:true,source,versionId,taskIssue:21,
        usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]})
    }});
  const result=await host.nextTaskUsage(21);
  assert.equal(result.connected,false);
  assert.equal(result.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
});

test('next-Task usage accepts only unique adopted verified eligible candidates',async()=>{
  const publisherRepository='kaicreator-mm/task-meta';
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
      readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,versionId,
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
  const publisherRepository='kaicreator-mm/task-meta';
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
      readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,versionId,
        taskIssue,usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]})
    }});
  const result=await host.nextTaskUsage(22);
  assert.equal(reads,2);
  assert.equal(result.connected,false);
  assert.equal(result.reason,'EXPERIENCE_SOURCE_CHANGED_DURING_USAGE_READ');
});


test('experience projection rejects forged learning IDs and mismatched report source bindings',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const reportId='task-meta:bound',revision=1,commentId=6381704608;
  const suggestion={learningId:learningIdFor(reportId,revision,commentId),
    taskIssueRef:null,targetIssueRef:'https://github.com/'+publisherRepository+'/issues/13',
    subjectSha:sha,qualification:'REPORTED_QUALIFIED',
    provenance:{reportId,revision,kind:'EXECUTION_SUMMARY',commentId,issueNumber:13,
      namespace:'task-meta:'+publisherRepository,repository:publisherRepository,
      sourceSha:sha,dataDigest:'a'.repeat(64),
      publicationRef:'https://github.com/'+publisherRepository+
        '/issues/13#issuecomment-'+commentId},
    measuredEffect:'NOT_MEASURED',noAgentRanking:true,qualityGainClaimed:false};
  const read=row=>createMaintenanceControllerExperienceAdapter({
    maintenanceController:{dispatch:async()=>({status:'OK',
      reason:'PUBLIC_ADS_REPORTS_OBSERVED'}),
      snapshot:()=>({status:'OBSERVED',sourceSha:sha,learning:[row]})},
    repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository]
  }).readExperienceCandidates();
  assert.equal((await read(suggestion)).connected,true);
  for(const bad of [
    {...suggestion,learningId:'0'.repeat(64)},
    {...suggestion,provenance:{...suggestion.provenance,sourceSha:'b'.repeat(40)}},
    {...suggestion,provenance:{...suggestion.provenance,repository:'evil/repo'}},
    {...suggestion,subjectSha:'c'.repeat(40)}
  ]){
    const result=await read(bad);
    assert.equal(result.connected,false);
    assert.equal(result.reason,'ADS_LEARNING_PROVENANCE_UNQUALIFIED');
  }
});

test('isolated E2E publisher remains test-only and can never be adopted or reused',async()=>{
  const publisherRepository='kaicreator-mm/task-meta-e2e';
  const id='d'.repeat(64);
  const isolated={
    id,revision:1,title:'Isolated E2E report',state:'PROPOSED',
    reportQualification:'VERIFIED',
    sourceRef:'https://github.com/'+publisherRepository+
      '/issues/30#issuecomment-6381704609',
    publisherRepository,sourceKind:'ISOLATED_E2E_TEST_ONLY',
    eligibleForNextTask:false
  };
  let dispatches=0;
  const hostFor=(item,usage)=>createUiExperienceHost({
    repository,versionId,expectedSha:sha,
    trustedPublisherRepositories:[publisherRepository],
    selectedHost:{
      readExperienceCandidates:async()=>({connected:true,source,versionId,
        candidates:[item]}),
      dispatchExperienceDecision:async()=>{dispatches++;return {
        status:'OBSERVED',candidateId:id,sourceSha:sha,decision:'ADOPTED',
        decisionReceiptRef:'https://github.com/'+repository+
          '/issues/30#issuecomment-6381704610'
      };},
      readNextTaskContextUsage:async({taskIssue})=>({connected:true,source,versionId,
        taskIssue,...usage})
    }
  });
  const adoption=await hostFor(isolated,{}).decision({...request,
    candidateId:id,revision:1,decision:'ADOPT'});
  assert.equal(adoption.status,'DENIED');
  assert.equal(dispatches,0);

  const applied=await hostFor({...isolated,state:'ADOPTED'},{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(31);
  assert.equal(applied.connected,false);
  assert.equal(applied.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  const derivedPublisher=await hostFor({...isolated,publisherRepository:undefined},{}).candidates();
  assert.equal(derivedPublisher.connected,true);
  assert.equal(derivedPublisher.candidates[0].publisherRepository,publisherRepository);
  assert.equal(derivedPublisher.candidates[0].sourceKind,'ISOLATED_E2E_TEST_ONLY');
  const derivedUsage=await hostFor({...isolated,publisherRepository:undefined,state:'ADOPTED'},{
    usageState:'EXPLICITLY_APPLIED',usedCandidates:[id]
  }).nextTaskUsage(32);
  assert.equal(derivedUsage.connected,false);
  assert.equal(derivedUsage.reason,'NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');

  for(const mislabeled of [
    {...isolated,sourceKind:'PUBLIC_ADS_REPORT'},
    {...isolated,sourceKind:undefined},
    {...isolated,state:'ADOPTED',eligibleForNextTask:true}
  ]){
    const result=await hostFor(mislabeled,{}).candidates();
    assert.equal(result.connected,false);
    assert.equal(result.reason,'EXPERIENCE_PROVENANCE_UNQUALIFIED');
  }
});
