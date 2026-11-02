/** UI-C #230 — Task-centric read-model view logic. No hard-coded issue
 * numbers anywhere; statuses/authority come from the projection (single
 * source of truth) — this module only filters/groups/sorts what the Host
 * issued; it never mutates task state.
 */
export type TaskStatus='open'|'in_progress'|'review'|'blocked'|'done'|'failed'|'unknown';
export type TaskExecutionState='QUEUED'|'ADMISSION_BLOCKED'|'ADMITTED'|'STARTING'|'RUNNING'|'CANDIDATE_READY'|'REVIEWING'|'REPAIRING'|'CI_RUNNING'|'COMPLETED'|'FAILED'|'BLOCKED'|'CANCELLED'|'UNKNOWN'|'RECONCILING';
export const TASK_EXECUTION_STATES:readonly TaskExecutionState[]=Object.freeze([
  'QUEUED','ADMISSION_BLOCKED','ADMITTED','STARTING','RUNNING','CANDIDATE_READY',
  'REVIEWING','REPAIRING','CI_RUNNING','COMPLETED','FAILED','BLOCKED','CANCELLED',
  'UNKNOWN','RECONCILING']);
export type TaskExecutionRole='BUILDER'|'REVIEWER'|'REPAIR'|'VALIDATOR'|'ORCHESTRATOR'|'UNKNOWN';
/** Host-owned task-level execution snapshot. It contains summaries, never raw logs. */
export interface TaskExecutionSnapshot{
  state:TaskExecutionState;provider:string|null;role:TaskExecutionRole|null;
  attemptId:string|null;progressSummary:string|null;
  progressSummaryKind:'SANITIZED'|'NOT_AVAILABLE';
  heartbeatAt:string|null;elapsedMs:number|null;
  tokensIn:number|null;tokensOut:number|null;costUsd:number|null;
  candidateRef:string|null;
}
export const TASK_STATUSES:readonly TaskStatus[]=Object.freeze(
  ['open','in_progress','review','blocked','done','failed','unknown']);
export interface TaskRecord{
  /** Native work-unit id from the projection (e.g. issue number as string). */
  id:string;
  versionId:string;
  /** Selected repository identity used to bind candidate links. */
  projectId?:string;
  title:string;
  status:TaskStatus;
  severity:'P0'|'P1'|'P2'|'UNKNOWN';
  /** Native blocked_by task ids (source-qualified only). */
  blockedBy:readonly string[];
  agentId:string|null;
  claimState:'ACCEPTED'|'NOT_VERIFIED'|'UNKNOWN';
  attemptId:string|null;
  attemptState:'RUNNING'|'EXITED'|'UNKNOWN'|null;
  prNumber:number|null;
  reviewState:'JOINED'|'REPAIR'|'NOT_RUN'|'UNKNOWN';
  ciState:'SUCCESS'|'FAILURE'|'RUNNING'|'NOT_RUN'|'UNKNOWN';
  writeSet:readonly string[];
  requirementRef:string|null;
  updatedAt:string;
  /** Current task-level execution state; absent data is rendered UNKNOWN. */
  execution?:TaskExecutionSnapshot|null;
}
export interface TaskFilters{
  text:string;
  status:TaskStatus|null;
  severity:'P0'|'P1'|'P2'|'UNKNOWN'|null;
  agentId:string|null;
  onlyBlocked:boolean;
}
export const EMPTY_FILTERS:TaskFilters=Object.freeze({text:'',status:null,
  severity:null,agentId:null,onlyBlocked:false});

export function filterTasks(tasks:readonly TaskRecord[],filters:TaskFilters,
  activeVersionId:string|null):TaskRecord[]{
  return tasks.filter(task=>{
    // Strict version isolation: never surface another version's tasks.
    if(activeVersionId&&task.versionId!==activeVersionId)return false;
    if(filters.status&&task.status!==filters.status)return false;
    if(filters.severity&&task.severity!==filters.severity)return false;
    if(filters.agentId&&task.agentId!==filters.agentId)return false;
    if(filters.onlyBlocked&&task.status!=='blocked'
      &&task.blockedBy.length===0)return false;
    const needle=filters.text.trim().toLowerCase();
    if(needle){
      const haystack=[task.id,task.title,...task.writeSet]
        .join(' ').toLowerCase();
      if(!haystack.includes(needle))return false;
    }
    return true;
  });
}
export type SortKey='severity'|'status'|'updated'|'id';
const severityRank={'P0':0,'P1':1,'P2':2,'UNKNOWN':3} as const;
const statusRank:Record<TaskStatus,number>={blocked:0,in_progress:1,review:2,
  open:3,failed:4,done:5,unknown:6};
export function sortTasks(tasks:readonly TaskRecord[],key:SortKey,
  direction:'asc'|'desc'='asc'):TaskRecord[]{
  const factor=direction==='asc'?1:-1;
  return [...tasks].sort((a,b)=>{
    let order=0;
    if(key==='severity')order=severityRank[a.severity]-severityRank[b.severity];
    else if(key==='status')order=statusRank[a.status]-statusRank[b.status];
    else if(key==='updated')order=a.updatedAt.localeCompare(b.updatedAt);
    else order=a.id.localeCompare(b.id,undefined,{numeric:true});
    if(order===0)order=a.id.localeCompare(b.id,undefined,{numeric:true});
    return order*factor;
  });
}
/** Board columns in canonical workflow order; identity (task id) is shared
 * across list/board/DAG so switching views never loses selection. */
export function boardColumns(tasks:readonly TaskRecord[]):
  {status:TaskStatus;tasks:TaskRecord[]}[]{
  return TASK_STATUSES.map(status=>({status,
    tasks:tasks.filter(task=>task.status===status)}));
}
/** 1–3 actionable blockers per task for the drawer summary; deduped and
 * bounded, full detail stays in Technical evidence. */
export interface BlockerSummary{code:string;detail:string;}
export function blockerSummary(task:TaskRecord,
  unresolved:readonly {taskId:string;reason:string}[]=[]):BlockerSummary[]{
  const blockers:BlockerSummary[]=[];
  if(task.status==='blocked')
    blockers.push({code:'BLOCKED',detail:`被 ${task.blockedBy.length} 个前置依赖阻塞`});
  if(task.claimState!=='ACCEPTED')
    blockers.push({code:'CLAIM_NOT_ACCEPTED',detail:'任务 Claim 未接受,不能派发执行'});
  if(task.attemptState==='UNKNOWN')
    blockers.push({code:'ATTEMPT_UNKNOWN',detail:'Agent 执行状态未知,需 Host 对账'});
  if(task.ciState==='FAILURE')
    blockers.push({code:'CI_FAILED',detail:'CI 失败,需修复后重新提交'});
  for(const item of unresolved)
    if(item.taskId===task.id)blockers.push({code:'DEPENDENCY',detail:item.reason});
  // Stable dedupe, bounded to 3 actionable reasons (owner UX requirement).
  const seen=new Set<string>();
  return blockers.filter(b=>seen.has(b.code)?false:(seen.add(b.code),true)).slice(0,3);
}
/** Detect write-set conflicts among visible tasks (same path owned twice). */
export function writeSetConflicts(tasks:readonly TaskRecord[]):
  {path:string;taskIds:string[]}[]{
  const owners=new Map<string,string[]>();
  for(const task of tasks)
    for(const path of task.writeSet)
      owners.set(path,[...(owners.get(path)??[]),task.id]);
  return [...owners.entries()]
    .filter(([,ids])=>new Set(ids).size>1)
    .map(([path,ids])=>({path,taskIds:[...new Set(ids)]}));
}
/** Keyboard-friendly flat index used by list navigation and search. */
export function taskIndex(tasks:readonly TaskRecord[]):Map<string,TaskRecord>{
  return new Map(tasks.map(task=>[task.id,task]));
}
/** Human status labels (zh/en) shared by list/board/drawer. */
export const STATUS_LABEL:Record<TaskStatus,{zh:string;en:string}>={
  open:{zh:'待处理',en:'Open'},in_progress:{zh:'进行中',en:'In progress'},
  review:{zh:'评审中',en:'Review'},blocked:{zh:'已阻塞',en:'Blocked'},
  done:{zh:'已完成',en:'Done'},failed:{zh:'失败',en:'Failed'},
  unknown:{zh:'状态未知',en:'Unknown'},
};

export interface TaskExecutionSummary{
  state:TaskExecutionState;provider:string;role:TaskExecutionRole;attemptId:string;
  progressSummary:string;heartbeatAt:string;elapsed:string;
  tokensIn:string;tokensOut:string;cost:string;candidateRef:string|null;
}
const EXECUTION_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,79}$/;
const PROVIDER_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$/;
const PROGRESS_SECRET=/\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|sk-[A-Za-z0-9_-]{16,}|Bearer\s+\S+)\b/i;
const CANDIDATE_PR=/^https:\/\/github\.com\/([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)\/pull\/[1-9][0-9]*$/;
const EXECUTION_ROLES=new Set<TaskExecutionRole>(['BUILDER','REVIEWER','REPAIR','VALIDATOR','ORCHESTRATOR','UNKNOWN']);
const formatElapsed=(ms:number)=>{
  const seconds=Math.floor(ms/1000);
  return seconds>=60?`${Math.floor(seconds/60)}m${seconds%60}s`:`${seconds}s`;
};
/** Project only bounded Host summaries; never expose raw logs or secret-bearing text. */
export function taskExecutionSummary(task:Pick<TaskRecord,'attemptId'|'execution'|'projectId'>):TaskExecutionSummary{
  const e=task.execution??null;
  const state=TASK_EXECUTION_STATES.includes(e?.state as TaskExecutionState)?e!.state:'UNKNOWN';
  const provider=typeof e?.provider==='string'&&PROVIDER_ID.test(e.provider)&&
    !PROGRESS_SECRET.test(e.provider)?e.provider:'UNKNOWN';
  const role=EXECUTION_ROLES.has(e?.role as TaskExecutionRole)?e!.role!:'UNKNOWN';
  const executionAttemptId=e?.attemptId;
  // A supplied but unsafe execution id must not fall back to a legacy task id.
  // The legacy value is used only when the execution snapshot omits the field.
  const attemptId=executionAttemptId===null||executionAttemptId===undefined
    ?typeof task.attemptId==='string'&&EXECUTION_ID.test(task.attemptId)&&
      !PROGRESS_SECRET.test(task.attemptId)?task.attemptId:'UNKNOWN'
    :typeof executionAttemptId==='string'&&EXECUTION_ID.test(executionAttemptId)&&
      !PROGRESS_SECRET.test(executionAttemptId)?executionAttemptId:'UNKNOWN';
  const rawProgress=e?.progressSummary;
  const progressSummary=e?.progressSummaryKind==='SANITIZED'&&
    typeof rawProgress==='string'&&rawProgress.length>0&&rawProgress.length<=240&&
    !/[\u0000-\u001f\u007f]/.test(rawProgress)&&!PROGRESS_SECRET.test(rawProgress)
      ?rawProgress:'UNKNOWN';
  const heartbeatAt=typeof e?.heartbeatAt==='string'&&e.heartbeatAt.length<=40&&
    Number.isFinite(Date.parse(e.heartbeatAt))?e.heartbeatAt:'UNKNOWN';
  const elapsed=Number.isSafeInteger(e?.elapsedMs)&&e!.elapsedMs!>=0
    ?formatElapsed(e!.elapsedMs!):'UNKNOWN';
  const tokensIn=Number.isSafeInteger(e?.tokensIn)&&e!.tokensIn!>=0?String(e!.tokensIn):'UNKNOWN';
  const tokensOut=Number.isSafeInteger(e?.tokensOut)&&e!.tokensOut!>=0?String(e!.tokensOut):'UNKNOWN';
  const cost=typeof e?.costUsd==='number'&&Number.isFinite(e.costUsd)&&e.costUsd>=0
    ?'$'+e.costUsd.toFixed(4):'UNKNOWN';
  const candidateMatch=typeof e?.candidateRef==='string'?
    CANDIDATE_PR.exec(e.candidateRef):null;
  const candidateRef=candidateMatch&&task.projectId===candidateMatch[1]+'/'+candidateMatch[2]
    ?e!.candidateRef:null;
  return {state,provider,role,attemptId,progressSummary,heartbeatAt,elapsed,
    tokensIn,tokensOut,cost,candidateRef};
}
export const TASK_EXECUTION_LABEL:Record<TaskExecutionState,{zh:string;en:string}>={
  QUEUED:{zh:'排队中',en:'Queued'},
  ADMISSION_BLOCKED:{zh:'准入阻塞',en:'Admission blocked'},
  ADMITTED:{zh:'已准入',en:'Admitted'},
  STARTING:{zh:'启动中',en:'Starting'},
  RUNNING:{zh:'执行中',en:'Running'},
  CANDIDATE_READY:{zh:'候选已就绪',en:'Candidate ready'},
  REVIEWING:{zh:'评审中',en:'Reviewing'},
  REPAIRING:{zh:'修复中',en:'Repairing'},
  CI_RUNNING:{zh:'CI 运行中',en:'CI running'},
  COMPLETED:{zh:'已完成',en:'Completed'},
  FAILED:{zh:'失败',en:'Failed'},
  BLOCKED:{zh:'已阻塞',en:'Blocked'},
  CANCELLED:{zh:'已取消',en:'Cancelled'},
  UNKNOWN:{zh:'状态未知',en:'Unknown'},
  RECONCILING:{zh:'对账中',en:'Reconciling'},
};
export function taskExecutionTone(state:TaskExecutionState){
  if(state==='COMPLETED')return 'ok';
  if(state==='FAILED')return 'danger';
  if(state==='BLOCKED'||state==='ADMISSION_BLOCKED'||state==='CANCELLED'||
     state==='UNKNOWN'||state==='RECONCILING')return 'warn';
  return 'info';
};