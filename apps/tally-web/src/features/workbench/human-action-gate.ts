/** D/#263: UI display gate for action identity issued by the selected Host.
 * This is NOT authorization; selected B/K/Host must recheck and dispatch.
 */
import type {WorkbenchSnapshot,WorkAction} from './model.ts';
export interface SelectedActionOption{
  id:string;kind:string;revision:number;taskIssue:number|null;
  state:'ELIGIBLE'|'BLOCKED';label:string;
}
export interface SelectedActionCatalog{
  connected:boolean;source?:{repository:string;sourceSha:string;
    currentness:string;readback:string};versionId?:string;
  actions?:readonly SelectedActionOption[];
}
export interface HumanActionGate{
  enabled:boolean;reason:string;
  actionId:string|null;revision:number|null;label:string;
}
const deny=(reason:string):HumanActionGate=>({enabled:false,reason,
  actionId:null,revision:null,label:'操作不可用'});
// A matching Task/category is not a Host-issued action identity.
const hostKindFor:Partial<Record<WorkAction['kind'],string>>={
  REVIEW:'REVIEW_REPAIR',DECISION:'FREEZE_PRODUCT',RECOVERY:'START_AGENT'};
export function workbenchActionGate(snapshot:WorkbenchSnapshot|null,
  catalog:SelectedActionCatalog|null,workItem:WorkAction):HumanActionGate{
  if(snapshot?.connected!==true||catalog?.connected!==true||
     !catalog.source||! /^[a-f0-9]{40}$/.test(snapshot.source?.sourceSha??'')||
     !/^[a-f0-9]{40}$/.test(catalog.source.sourceSha??'')||
     snapshot.source?.currentness!=='CURRENT'||
     snapshot.source.readback!=='VERIFIED'||
     catalog.source.currentness!=='CURRENT'||catalog.source.readback!=='VERIFIED'||
     catalog.source.repository!==snapshot.source.repository||
     catalog.source.sourceSha!==snapshot.source.sourceSha||
     !Array.isArray(catalog.actions))
    return deny('HOST_ACTION_SOURCE_NOT_CONNECTED');
  if(workItem.projectId!==snapshot.source.repository||
     workItem.versionId===null||catalog.versionId!==workItem.versionId||
     workItem.state!=='PENDING')return deny('WORKITEM_NOT_CURRENT');
  if(!Array.isArray(catalog.actions)||
     workItem.taskId===null||!/^\d+$/.test(workItem.taskId))
    return deny('NO_HOST_ACTION_FOR_ITEM');
  const taskIssue=Number(workItem.taskId);
  if(!Number.isSafeInteger(taskIssue)||taskIssue<1)
    return deny('NO_HOST_ACTION_FOR_ITEM');
  const expectedKind=hostKindFor[workItem.kind];
  if(!expectedKind||!workItem.hostActionId||
     !Number.isSafeInteger(workItem.hostActionRevision)||
     (workItem.hostActionRevision??0)<1||
     workItem.hostActionKind!==expectedKind)
    return deny('HOST_ACTION_IDENTITY_NOT_BOUND');
  const matching=catalog.actions.filter(a=>
    a.taskIssue===taskIssue&&a.state==='ELIGIBLE'&&
    a.id===workItem.hostActionId&&
    a.revision===workItem.hostActionRevision&&
    a.kind===expectedKind);
  if(matching.length!==1)return deny('HOST_ACTION_NOT_ELIGIBLE_OR_AMBIGUOUS');
  const a=matching[0];
  if(!/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/.test(a.id)||
     !Number.isSafeInteger(a.revision)||a.revision<1||
     typeof a.label!=='string'||!a.label.trim())
    return deny('HOST_ACTION_IDENTITY_INVALID');
  return {enabled:true,reason:'FRESH_HOST_ACTION',
    actionId:a.id,revision:a.revision,label:a.label};
}
