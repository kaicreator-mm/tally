/** Canonical human-action queue projection; displays Host-qualified decisions only.
 * No UI owned task scheduler, approval authority or inferred Task PASS.
 */
export type WorkActionKind='APPROVAL'|'DECISION'|'BLOCKER'|'REVIEW'|'RECOVERY';
export type WorkActionState='PENDING'|'BLOCKED'|'RESOLVED';
export interface WorkAction{
  id:string;taskId:string|null;projectId:string;versionId:string|null;
  title:string;kind:WorkActionKind;state:WorkActionState;
  ownerRole:string;sourceRef:string;reason:string|null;updatedAt:string|null;
}
export interface WorkbenchSnapshot{
  connected:boolean;source:{repository:string;sourceSha:string;currentness:string;
    readback:string};actions:readonly WorkAction[];
}
export type WorkbenchPage='inbox'|'approvals'|'blockers'|'overview';
export function activeWorkActions(snapshot:WorkbenchSnapshot|null,
  page:WorkbenchPage):WorkAction[]|null{
  if(!snapshot?.connected)return null;
  const actions=snapshot.actions.filter(a=>a.state!=='RESOLVED');
  if(page==='approvals')
    return actions.filter(a=>a.kind==='APPROVAL'||a.kind==='DECISION');
  if(page==='blockers')
    return actions.filter(a=>a.state==='BLOCKED'||a.kind==='BLOCKER'||a.kind==='RECOVERY');
  return actions;
}
export function inboxCounts(snapshot:WorkbenchSnapshot|null){
  const actions=activeWorkActions(snapshot,'inbox');
  if(!actions)return null;
  return {pending:actions.length,
    approvals:activeWorkActions(snapshot,'approvals')!.length,
    blockers:activeWorkActions(snapshot,'blockers')!.length};
}
export function actionTaskLink(action:WorkAction):string|null{
  if(!action.taskId||!action.versionId)return null;
  return '#/development/tasks?projectId='+encodeURIComponent(action.projectId)
    +'&versionId='+encodeURIComponent(action.versionId)
    +'&taskId='+encodeURIComponent(action.taskId);
}
