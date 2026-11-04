import test from 'node:test';
import assert from 'node:assert/strict';
import {createUiActionPort,businessIntentForUiAction} from './ui-action-port.mjs';
const repository='org/tally',versionId='version/v2',sha='a'.repeat(40);
const source={repository,sourceSha:sha,currentness:'CURRENT',readback:'VERIFIED'};
const eligible={id:'start-260',kind:'START_AGENT',revision:1,label:'启动已准入 Agent',
  taskIssue:260,state:'ELIGIBLE',hostAdmission:'ADMITTED',
  permitSource:'SELECTED_K_HOST',permitSourceSha:sha};
const snapshot=()=>({connected:true,source,versionId,actions:[eligible]});
const request={repository,versionId,sourceSha:sha,actionId:'start-260',revision:1};
const receipt={status:'OBSERVED',sourceSha:sha,actionId:'start-260',
  receiptRef:'https://github.com/org/tally/issues/260#issuecomment-100'};
const make=(overrides={})=>createUiActionPort({repository,versionId,expectedSha:sha,
  host:{readCurrentActions:async()=>snapshot(),dispatchAuthorizedAction:async()=>receipt,...overrides}});
test('port publishes immutable selected scope for the HTTP boundary',()=>{
  const port=make();
  assert.deepEqual(port.scope,{repository,versionId,sourceSha:sha});
  assert.equal(Object.isFrozen(port.scope),true);
  assert.equal(Reflect.set(port.scope,'repository','evil/repo'),false);
  assert.equal(port.scope.repository,repository);
});
test('missing Host and foreign scope fail closed',async()=>{
  assert.equal((await createUiActionPort({repository,versionId,expectedSha:sha}).actions()).connected,false);
  assert.equal((await make().actions({repository:'evil/repo'})).reason,'ACTION_SCOPE_MISMATCH');
});
test('read model exposes a bounded action, never a raw grant or permit',async()=>{
  const data=await make().actions({repository,versionId});
  assert.equal(data.connected,true);
  assert.deepEqual(Object.keys(data.actions[0]).sort(),
    ['id','kind','revision','label','taskIssue','state','reason'].sort());
  assert.equal(data.actions[0].hostAdmission,undefined);
});
test('only selected Host can dispatch an exact current admitted action',async()=>{
  let calls=0;
  const host=make({dispatchAuthorizedAction:async()=>{calls++;return receipt;}});
  const result=await host.submit(request);
  assert.equal(calls,1);
  assert.equal(result.ok,true);
  assert.equal(result.receiptRef,receipt.receiptRef);
  assert.equal(result.taskQualification,'NOT_VERIFIED');
  assert.equal(result.releaseQualification,'NOT_VERIFIED');
});
test('browser cannot supply a grant, payload or stale revision',async()=>{
  let dispatch=0;
  const host=make({dispatchAuthorizedAction:async()=>{dispatch++;return receipt;}});
  for(const bad of [{...request,token:'ghp_secret'},
    {...request,revision:2},{...request,sourceSha:'b'.repeat(40)},
    {...request,actionId:'MERGE'}, {...request,repository:'evil/repo'}]){
    assert.equal((await host.submit(bad)).status,'DENIED');
  }
  assert.equal(dispatch,0);
});
test('candidate Claim or unsigned permit cannot grant an effect',async()=>{
  for(const bad of [{...eligible,hostAdmission:'CLAIM_CANDIDATE'},
    {...eligible,permitSource:'BROWSER'},
    {...eligible,permitSourceSha:'b'.repeat(40)}]){
    let calls=0;
    const host=make({readCurrentActions:async()=>({...snapshot(),actions:[bad]}),
      dispatchAuthorizedAction:async()=>{calls++;return receipt;}});
    assert.equal((await host.submit(request)).status,'DENIED');
    assert.equal(calls,0);
  }
});
test('stale second read refuses dispatch',async()=>{
  let reads=0,dispatch=0;
  const host=make({readCurrentActions:async()=>({...snapshot(),actions:[
    {...eligible,revision:++reads}]}),
    dispatchAuthorizedAction:async()=>{dispatch++;return receipt;}});
  assert.equal((await host.submit(request)).reason,'ACTION_SOURCE_CHANGED_BEFORE_DISPATCH');
  assert.equal(dispatch,0);
});
test('cross-project, malformed Host results and attempted PASS elevation rejected',async()=>{
  const bad=[{...snapshot(),source:{...source,repository:'evil/repo'}},
    {...snapshot(),source:{...source,currentness:'STALE'}},
    {...snapshot(),actions:[eligible,eligible]},
    ...['ghp_abcdefghijklmnopqrstuvwxyz','github_pat_abcdefghijklmnopqrstuvwxyz',
      'Bearer abcdefghijklmnopqrstuvwxyz'].map(label=>({...snapshot(),
        actions:[{...eligible,label}]}))];
  for(const x of bad){
    assert.equal((await make({readCurrentActions:async()=>x}).actions()).connected,false);
  }
  for(const res of [{...receipt,taskPass:true},{...receipt,releasePass:true},
    {...receipt,receiptRef:'https://evil/repo'},{...receipt,authorized:true}]){
    assert.equal((await make({dispatchAuthorizedAction:async()=>res}).submit(request)).status,'DENIED');
  }
});
test('lost dispatch acknowledgement does not claim success or retry',async()=>{
  let count=0;
  const r=await make({dispatchAuthorizedAction:async()=>{count++;throw Error('private key');}})
    .submit(request);
  assert.equal(count,1);
  assert.equal(r.status,'UNKNOWN');
  assert.equal(JSON.stringify(r).includes('private key'),false);
});

test('UI cannot invent Task creation or CAS Claim publication as Business intents',async()=>{
  for(const kind of ['CREATE_TASK','CLAIM_TASK']){
    const action={...eligible,kind};
    const result=await make({readCurrentActions:async()=>({...snapshot(),actions:[action]})})
      .actions();
    assert.equal(result.connected,false);
  }
});

test('Host action kinds map only to existing selected Business intents',()=>{
  assert.equal(businessIntentForUiAction('FREEZE_PRODUCT'),'freezeProductIntake');
  assert.equal(businessIntentForUiAction('ADMIT_PROJECT'),'admitProjectIntake');
  assert.equal(businessIntentForUiAction('ADMIT_CLAIMED_TASK'),'admitClaimedTask');
  assert.equal(businessIntentForUiAction('START_AGENT'),'runLocalCodexAttempt');
  assert.equal(businessIntentForUiAction('REVIEW_REPAIR'),'joinReviewedCandidates');
  assert.equal(businessIntentForUiAction('PUBLISH_CANDIDATE'),'publishReviewedCandidate');
  assert.equal(businessIntentForUiAction('PUBLISH_MAINTENANCE_ISSUE'),'publishMaintenanceIssue');
  assert.equal(businessIntentForUiAction('RECORD_TASK_LEARNING'),'recordTaskLearningOutcome');
  assert.equal(businessIntentForUiAction('DELIVER_QUALIFIED_RELEASE'),'deliverQualifiedRelease');
  assert.equal(businessIntentForUiAction('CREATE_TASK'),null);
  assert.equal(businessIntentForUiAction('CLAIM_TASK'),null);
  assert.equal(businessIntentForUiAction('__proto__'),null);
  assert.equal(businessIntentForUiAction('toString'),null);
});

const executionSnapshot=()=>({connected:true,source,versionId,taskIssue:260,
  execution:{state:'RUNNING',provider:'local-codex',role:'BUILDER',
    attemptId:'attempt-260',progressSummary:'正在检查依赖并执行任务',
    progressSummaryKind:'SANITIZED',heartbeatAt:'2026-11-02T12:30:00.000Z',
    elapsedMs:65000,tokensIn:1200,tokensOut:300,costUsd:null,
    candidateRef:'https://github.com/org/tally/pull/300'}});
test('task execution read is selected-source scoped and exposes only bounded lifecycle evidence',async()=>{
  let received;
  const port=make({readTaskExecution:async input=>{
    received=input;return executionSnapshot();
  }});
  const result=await port.taskExecution(260);
  assert.deepEqual(received,{repository,versionId,sourceSha:sha,taskIssue:260});
  assert.equal(result.connected,true);
  assert.equal(result.execution.state,'RUNNING');
  assert.equal(result.execution.provider,'local-codex');
  assert.equal(result.execution.role,'BUILDER');
  assert.equal(result.execution.progressSummary,'正在检查依赖并执行任务');
  assert.equal(result.execution.candidateRef,'https://github.com/org/tally/pull/300');
  assert.equal(result.execution.rawLog,undefined);
});
test('task execution read keeps missing source UNKNOWN and refuses foreign or unsafe evidence',async()=>{
  assert.equal((await make().taskExecution(260)).reason,
    'SELECTED_HOST_TASK_EXECUTION_NOT_CONNECTED');
  assert.equal((await make().taskExecution(0)).reason,'TASK_EXECUTION_ISSUE_INVALID');
  const foreign=await make({readTaskExecution:async()=>({...executionSnapshot(),
    source:{...source,repository:'evil/repo'}})}).taskExecution(260);
  assert.equal(foreign.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
  const wrongTask=await make({readTaskExecution:async()=>({...executionSnapshot(),
    taskIssue:261})}).taskExecution(260);
  assert.equal(wrongTask.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
  for(const progressSummary of ['Bearer ghp_abcdefghijklmnopqrstuvwxyz',
    'github_pat_abcdefghijklmnopqrstuvwxyz']){
    const secret=await make({readTaskExecution:async()=>({...executionSnapshot(),
      execution:{...executionSnapshot().execution,progressSummary}})}).taskExecution(260);
    assert.equal(secret.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
  }
  const foreignPr=await make({readTaskExecution:async()=>({...executionSnapshot(),
    execution:{...executionSnapshot().execution,
      candidateRef:'https://github.com/evil/repo/pull/300'}})}).taskExecution(260);
  assert.equal(foreignPr.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
});
test('task execution permits absent telemetry only as explicit nulls',async()=>{
  const result=await make({readTaskExecution:async()=>({...executionSnapshot(),
    execution:{state:'UNKNOWN',provider:null,role:null,attemptId:null,
      progressSummary:null,progressSummaryKind:'NOT_AVAILABLE',heartbeatAt:null,
      elapsedMs:null,tokensIn:null,tokensOut:null,costUsd:null,candidateRef:null}})})
    .taskExecution(260);
  assert.equal(result.connected,true);
  assert.equal(result.execution.state,'UNKNOWN');
  assert.equal(result.execution.tokensIn,null);
  assert.equal(result.execution.costUsd,null);
});

test('credential-shaped action ids, providers, attempts and summaries fail closed',async()=>{
  const secrets=['ghp_12345678901234567890',
    'github_pat_abcdefghijklmnopqrstuvwxyz012345',
    'safe_ghp_12345678901234567890',
    'safe_github_pat_abcdefghijklmnopqrstuvwxyz012345',
    'Bearer abcdefghijklmnopqrstuvwxyz'];
  for(const secret of secrets){
    const badAction=await make({readCurrentActions:async()=>({...snapshot(),
      actions:[{...eligible,id:secret}]})}).actions();
    assert.equal(badAction.connected,false,
      `secret-shaped action id must deny: ${secret}`);
    const badProvider=await make({readTaskExecution:async()=>({...executionSnapshot(),
      execution:{...executionSnapshot().execution,provider:secret}})}).taskExecution(260);
    assert.equal(badProvider.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED',
      `secret-shaped provider must deny: ${secret}`);
    const badAttempt=await make({readTaskExecution:async()=>({...executionSnapshot(),
      execution:{...executionSnapshot().execution,attemptId:secret}})}).taskExecution(260);
    assert.equal(badAttempt.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED',
      `secret-shaped attemptId must deny: ${secret}`);
    const badSummary=await make({readTaskExecution:async()=>({...executionSnapshot(),
      execution:{...executionSnapshot().execution,progressSummary:secret}})}).taskExecution(260);
    assert.equal(badSummary.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED',
      `secret-shaped progress summary must deny: ${secret}`);
  }
});
test('unexpected sensitive source keys never pass the read projections',async()=>{
  const poisoned={...source,telemetryToken:'ghp_12345678901234567890'};
  const catalog=await make({readCurrentActions:async()=>({...snapshot(),
    source:poisoned})}).actions();
  assert.equal(catalog.connected,false);
  const execution=await make({readTaskExecution:async()=>({...executionSnapshot(),
    source:poisoned})}).taskExecution(260);
  assert.equal(execution.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
});
test('published source/execution projections are reconstructed allowlists',async()=>{
  const catalog=await make().actions();
  assert.deepEqual(Object.keys(catalog.source).sort(),
    ['currentness','readback','repository','sourceSha']);
  assert.deepEqual(catalog.source,{repository,sourceSha:sha,
    currentness:'CURRENT',readback:'VERIFIED'});
  const extra=await make({readTaskExecution:async()=>{
    const ok=executionSnapshot();
    return {...ok,execution:Object.defineProperties({...ok.execution},{
      rawLog:{value:'secret log',enumerable:true},
      envToken:{value:'ghp_12345678901234567890',enumerable:true}})};
  }}).taskExecution(260);
  assert.equal(extra.reason,'HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
  const result=await make({readTaskExecution:async()=>executionSnapshot()})
    .taskExecution(260);
  assert.equal(result.connected,true);
  assert.equal(Object.isFrozen(result.execution),true);
  assert.deepEqual(Object.keys(result.execution).sort(),
    ['attemptId','candidateRef','costUsd','elapsedMs','heartbeatAt',
      'progressSummary','progressSummaryKind','provider','role','state',
      'tokensIn','tokensOut']);
  assert.equal(JSON.stringify(result).includes('rawLog'),false);
});
