/** UI-C #230 — Task-centric read-model view logic. No hard-coded issue
 * numbers anywhere; statuses/authority come from the projection (single
 * source of truth) — this module only filters/groups/sorts what the Host
 * issued; it never mutates task state.
 */
export type TaskStatus='open'|'in_progress'|'review'|'blocked'|'done'|'failed'|'unknown';
export const TASK_STATUSES:readonly TaskStatus[]=Object.freeze(
  ['open','in_progress','review','blocked','done','failed','unknown']);
export interface TaskRecord{
  /** Native work-unit id from the projection (e.g. issue number as string). */
  id:string;
  versionId:string;
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
