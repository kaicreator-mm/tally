import test from 'node:test';
import assert from 'node:assert/strict';
import {filterTasks,sortTasks,boardColumns,blockerSummary,
  writeSetConflicts,taskIndex,TASK_STATUSES,taskExecutionSummary,
  TASK_EXECUTION_STATES,taskExecutionTone,TASK_EXECUTION_LABEL} from './model.ts';
import type {TaskRecord} from './model.ts';

const make=(over:Partial<TaskRecord>):TaskRecord=>({
  id:'100',versionId:'v1',projectId:'kaicreator-mm/tally',title:'任务',status:'open',severity:'P2',
  blockedBy:[],agentId:null,claimState:'NOT_VERIFIED',attemptId:null,
  attemptState:null,prNumber:null,reviewState:'NOT_RUN',ciState:'NOT_RUN',
  writeSet:[],requirementRef:null,updatedAt:'2026-11-01T00:00:00Z',...over});
const tasks:TaskRecord[]=[
  make({id:'101',title:'表单引擎',status:'in_progress',severity:'P1',
    agentId:'zcode',claimState:'ACCEPTED',ciState:'RUNNING',
    writeSet:['apps/tally-web/src/features/forms/**']}),
  make({id:'102',title:'任务中心',status:'blocked',severity:'P0',
    blockedBy:['101'],claimState:'NOT_VERIFIED',ciState:'FAILURE',
    writeSet:['apps/tally-web/src/features/tasks/**']}),
  make({id:'103',title:'旧版任务',versionId:'v0',status:'done'}),
  make({id:'104',title:'看板视图',status:'review',severity:'P1',
    writeSet:['apps/tally-web/src/features/tasks/**','apps/x']},
    ),
  make({id:'105',title:'失败尝试',status:'failed',attemptState:'UNKNOWN'}),
];
test('version isolation: other versions never leak into the active view',()=>{
  assert.equal(filterTasks(tasks,{...filterTasksEmpty()},'v1').length,4);
  assert.equal(filterTasks(tasks,{...filterTasksEmpty()},'v0').length,1);
  assert.equal(filterTasks(tasks,{...filterTasksEmpty()},'v0')[0].id,'103');
  assert.equal(filterTasks(tasks,{...filterTasksEmpty()},null).length,5);
});
test('filters combine: text + status + severity + agent + blocked-only',()=>{
  const base=()=>({...filterTasksEmpty()});
  assert.equal(filterTasks(tasks,{...base(),text:'表单'},'v1').length,1);
  assert.equal(filterTasks(tasks,{...base(),text:'写集路径'.replace(/.*/,'tasks/**')},'v1').length,2);
  assert.equal(filterTasks(tasks,{...base(),status:'blocked'},'v1').length,1);
  assert.equal(filterTasks(tasks,{...base(),severity:'P0'},'v1').length,1);
  assert.equal(filterTasks(tasks,{...base(),agentId:'zcode'},'v1').length,1);
  assert.equal(filterTasks(tasks,{...base(),onlyBlocked:true},'v1').length,1);
  assert.equal(filterTasks(tasks,{...base(),text:'不存在'},'v1').length,0);
});
test('sort orders and board grouping preserve identity',()=>{
  const bySeverity=sortTasks(tasks,'severity');
  assert.equal(bySeverity[0].id,'102'); // P0 first
  const ids=boardColumns(filterTasks(tasks,{...filterTasksEmpty()},'v1'))
    .flatMap(column=>column.tasks.map(task=>task.id)).sort();
  assert.deepEqual(ids,['101','102','104','105']);
  assert.deepEqual(boardColumns([]).map(column=>column.status),[...TASK_STATUSES]);
  // Same record object identity across views (shared selection state).
  const index=taskIndex(tasks);
  assert.equal(index.get('102'),tasks[1]);
});
test('blocker summary bounded to 3 actionable reasons, deduped',()=>{
  const summary=blockerSummary(tasks[1]);
  assert.equal(summary.length,3);
  assert.deepEqual(summary.map(item=>item.code),
    ['BLOCKED','CLAIM_NOT_ACCEPTED','CI_FAILED']);
  const clean=blockerSummary(make({id:'x',claimState:'ACCEPTED',status:'in_progress'}));
  assert.equal(clean.length,0);
  const withDeps=blockerSummary(make({id:'y',claimState:'ACCEPTED'}),
    [{taskId:'y',reason:'前置未完成'}]);
  assert.deepEqual(withDeps,[{code:'DEPENDENCY',detail:'前置未完成'}]);
});
test('write-set conflict detection across visible tasks',()=>{
  const conflicts=writeSetConflicts(filterTasks(tasks,{...filterTasksEmpty()},'v1'));
  assert.equal(conflicts.length,1);
  assert.equal(conflicts[0].path,'apps/tally-web/src/features/tasks/**');
  assert.deepEqual(conflicts[0].taskIds.sort(),['102','104']);
  assert.deepEqual(writeSetConflicts([]),[]);
});
function filterTasksEmpty(){return{text:'',status:null,severity:null,
  agentId:null,onlyBlocked:false};}

test('task execution lifecycle covers the full Tally-managed orchestration path',()=>{
  assert.deepEqual(TASK_EXECUTION_STATES,[
    'QUEUED','ADMISSION_BLOCKED','ADMITTED','STARTING','RUNNING','CANDIDATE_READY',
    'REVIEWING','REPAIRING','CI_RUNNING','COMPLETED','FAILED','BLOCKED','CANCELLED',
    'UNKNOWN','RECONCILING']);
  assert.equal(TASK_EXECUTION_LABEL.QUEUED.zh,'排队中');
  assert.equal(TASK_EXECUTION_LABEL.RECONCILING.en,'Reconciling');
  assert.equal(taskExecutionTone('COMPLETED'),'ok');
  assert.equal(taskExecutionTone('FAILED'),'danger');
  assert.equal(taskExecutionTone('UNKNOWN'),'warn');
});
test('task execution summary exposes only source-provided sanitized progress and measured usage',()=>{
  const task=make({attemptId:'attempt-legacy',execution:{
    state:'RUNNING',provider:'local-codex',role:'BUILDER',attemptId:'attempt-300',
    progressSummary:'正在检查任务依赖并生成候选变更',progressSummaryKind:'SANITIZED',
    heartbeatAt:'2026-11-02T12:30:00.000Z',elapsedMs:65000,
    tokensIn:1200,tokensOut:300,costUsd:0.0123,
    candidateRef:'https://github.com/kaicreator-mm/tally/pull/300'
  }});
  assert.deepEqual(taskExecutionSummary(task),{
    state:'RUNNING',provider:'local-codex',role:'BUILDER',attemptId:'attempt-300',
    progressSummary:'正在检查任务依赖并生成候选变更',
    heartbeatAt:'2026-11-02T12:30:00.000Z',elapsed:'1m5s',
    tokensIn:'1200',tokensOut:'300',cost:'$0.0123',
    candidateRef:'https://github.com/kaicreator-mm/tally/pull/300'
  });
  assert.equal(taskExecutionSummary({...task,projectId:'other/repo'}).candidateRef,null);
  assert.equal(taskExecutionSummary({...task,execution:{
    ...task.execution!,provider:'ghp_12345678901234567890'
  }}).provider,'UNKNOWN');
  assert.equal(taskExecutionSummary({...task,execution:{
    ...task.execution!,attemptId:'ghp_12345678901234567890'
  }}).attemptId,'UNKNOWN');
  for(const progressSummary of ['github_pat_abcdefghijklmnopqrstuvwxyz',
    'Bearer abcdefghijklmnopqrstuvwxyz']){
    assert.equal(taskExecutionSummary({...task,execution:{
      ...task.execution!,progressSummary
    }}).progressSummary,'UNKNOWN');
  }
});

test('execution identifiers and candidate refs fail closed without unsafe fallback',()=>{
  const secret='safe_ghp_12345678901234567890';
  const baseExecution:NonNullable<TaskRecord['execution']> = {
    state:'RUNNING',provider:'local-codex',role:'BUILDER',attemptId:'attempt-300',
    progressSummary:null,progressSummaryKind:'NOT_AVAILABLE',heartbeatAt:null,
    elapsedMs:null,tokensIn:null,tokensOut:null,costUsd:null,candidateRef:null
  };
  const cases:{
    name:string;
    field:'provider'|'attemptId'|'candidateRef';
    patch:Partial<NonNullable<TaskRecord['execution']>>;
    legacyAttemptId:string|null;
    expected:string|null;
  }[]=[
    {name:'secret provider is UNKNOWN',field:'provider',
      patch:{provider:secret},legacyAttemptId:'attempt-legacy',expected:'UNKNOWN'},
    {name:'missing provider stays UNKNOWN',field:'provider',
      patch:{provider:null},legacyAttemptId:'attempt-legacy',expected:'UNKNOWN'},
    {name:'secret execution attempt does not fall back',field:'attemptId',
      patch:{attemptId:secret},legacyAttemptId:'attempt-legacy',expected:'UNKNOWN'},
    {name:'malformed execution attempt does not fall back',field:'attemptId',
      patch:{attemptId:'bad attempt id'},legacyAttemptId:'attempt-legacy',expected:'UNKNOWN'},
    {name:'absent execution attempt may use a valid legacy id',field:'attemptId',
      patch:{attemptId:null},legacyAttemptId:'attempt-legacy',expected:'attempt-legacy'},
    {name:'absent execution attempt rejects a secret legacy id',field:'attemptId',
      patch:{attemptId:null},legacyAttemptId:secret,expected:'UNKNOWN'},
    {name:'missing candidate ref remains absent',field:'candidateRef',
      patch:{candidateRef:null},legacyAttemptId:'attempt-legacy',expected:null},
    {name:'foreign candidate ref is rejected',field:'candidateRef',
      patch:{candidateRef:'https://github.com/other/repo/pull/300'},
      legacyAttemptId:'attempt-legacy',expected:null},
    {name:'same-project candidate ref is retained',field:'candidateRef',
      patch:{candidateRef:'https://github.com/kaicreator-mm/tally/pull/300'},
      legacyAttemptId:'attempt-legacy',
      expected:'https://github.com/kaicreator-mm/tally/pull/300'}
  ];
  for(const item of cases){
    const summary=taskExecutionSummary(make({projectId:'kaicreator-mm/tally',
      attemptId:item.legacyAttemptId,
      execution:{...baseExecution,...item.patch}}));
    assert.equal(summary[item.field],item.expected,item.name);
  }
});
test('missing, raw, secret-bearing or malformed execution evidence remains UNKNOWN',()=>{
  const empty=taskExecutionSummary(make({}));
  assert.equal(empty.state,'UNKNOWN');
  assert.equal(empty.provider,'UNKNOWN');
  assert.equal(empty.role,'UNKNOWN');
  assert.equal(empty.progressSummary,'UNKNOWN');
  assert.equal(empty.elapsed,'UNKNOWN');
  const unsafe=taskExecutionSummary(make({execution:{
    state:'RUNNING',provider:'local-codex',role:'BUILDER',attemptId:'attempt-1',
    progressSummary:'token ghp_12345678901234567890',progressSummaryKind:'SANITIZED',
    heartbeatAt:'not-a-date',elapsedMs:-1,tokensIn:-1,tokensOut:null,
    costUsd:null,candidateRef:'https://evil.example/pull/1'
  }}));
  assert.equal(unsafe.progressSummary,'UNKNOWN');
  assert.equal(unsafe.heartbeatAt,'UNKNOWN');
  assert.equal(unsafe.elapsed,'UNKNOWN');
  assert.equal(unsafe.tokensIn,'UNKNOWN');
  assert.equal(unsafe.tokensOut,'UNKNOWN');
  assert.equal(unsafe.cost,'UNKNOWN');
  assert.equal(unsafe.candidateRef,null);
  const raw=taskExecutionSummary(make({execution:{
    state:'RUNNING',provider:'local-codex',role:'BUILDER',attemptId:'attempt-1',
    progressSummary:'raw log line',progressSummaryKind:'NOT_AVAILABLE',
    heartbeatAt:null,elapsedMs:null,tokensIn:null,tokensOut:null,costUsd:null,candidateRef:null
  }}));
  assert.equal(raw.progressSummary,'UNKNOWN');
});


test('NOT_CLAIMED remains distinct from UNKNOWN and blocks dispatch',()=>{
  const task=make({claimState:'NOT_CLAIMED'});
  assert.equal(task.claimState,'NOT_CLAIMED');
  assert.deepEqual(blockerSummary(task).find(item=>item.code==='CLAIM_NOT_ACCEPTED'),
    {code:'CLAIM_NOT_ACCEPTED',detail:'任务 Claim 未接受,不能派发执行'});
});
