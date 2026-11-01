import test from 'node:test';
import assert from 'node:assert/strict';
import {filterTasks,sortTasks,boardColumns,blockerSummary,
  writeSetConflicts,taskIndex,TASK_STATUSES} from './model.ts';
import type {TaskRecord} from './model.ts';

const make=(over:Partial<TaskRecord>):TaskRecord=>({
  id:'100',versionId:'v1',title:'任务',status:'open',severity:'P2',
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
