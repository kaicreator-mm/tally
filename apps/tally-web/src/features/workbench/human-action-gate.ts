/** D/#263: UI display gate for action identity issued by the selected Host.
 * This is NOT authorization; selected B/K/Host must recheck and dispatch.
 * workbenchActionControl adds the view-level display decision: a fresh Host
 * action without a connected dispatch callback renders an explicit
 * dispatch-not-connected state, and Host denial reasons are localized so raw
 * internal codes never reach the user-facing view.
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
const REPOSITORY_REF=/^[A-Za-z0-9][A-Za-z0-9_.-]{0,38}[/][A-Za-z0-9][A-Za-z0-9_.-]{0,99}$/;
const VERSION_REF=/^version[/][A-Za-z0-9][A-Za-z0-9._-]{0,79}$/;
// Project/version identifiers are authority scope labels, not optional
// display strings. Equality between two empty or malformed values is not
// evidence that the work item belongs to a canonical current scope.
const hostKindFor:Partial<Record<WorkAction['kind'],string>>={
  REVIEW:'REVIEW_REPAIR',DECISION:'FREEZE_PRODUCT',RECOVERY:'START_AGENT'};
export function workbenchActionGate(snapshot:WorkbenchSnapshot|null,
  catalog:SelectedActionCatalog|null,workItem:WorkAction):HumanActionGate{
  if(snapshot?.connected!==true||catalog?.connected!==true||
     !catalog.source||! /^[a-f0-9]{40}$/.test(snapshot.source?.sourceSha??'')||
     !/^[a-f0-9]{40}$/.test(catalog.source.sourceSha??'')||
     !REPOSITORY_REF.test(snapshot.source?.repository??'')||
     !REPOSITORY_REF.test(catalog.source.repository??'')||
     snapshot.source?.currentness!=='CURRENT'||
     snapshot.source.readback!=='VERIFIED'||
     catalog.source.currentness!=='CURRENT'||catalog.source.readback!=='VERIFIED'||
     catalog.source.repository!==snapshot.source.repository||
     catalog.source.sourceSha!==snapshot.source.sourceSha||
     !Array.isArray(catalog.actions))
    return deny('HOST_ACTION_SOURCE_NOT_CONNECTED');
  if(!REPOSITORY_REF.test(workItem.projectId??'')||
     workItem.projectId!==snapshot.source.repository||
     !VERSION_REF.test(workItem.versionId??'')||
     !VERSION_REF.test(catalog.versionId??'')||
     catalog.versionId!==workItem.versionId||
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

/** Stable localized user-facing messages for Host display-gate outcomes; raw
 * internal reason codes never reach the view. Unknown future codes fail closed
 * to a generic localized denial instead of leaking the code.
 */
export const ACTION_STATUS_MESSAGES:Readonly<Record<string,string>>={
  FRESH_HOST_ACTION:'已从受信 Host 获取新鲜且可执行的人工操作',
  HOST_ACTION_SOURCE_NOT_CONNECTED:
    'Host 操作源未连接：需要已连接且来源核验一致的受信 Host 操作目录',
  WORKITEM_NOT_CURRENT:
    '工作项不在当前可操作范围：仅当前项目与版本内的待办事项可执行操作',
  NO_HOST_ACTION_FOR_ITEM:'该事项没有匹配的 Host 人工操作',
  HOST_ACTION_IDENTITY_NOT_BOUND:
    '该事项未绑定匹配类别的 Host 签发操作标识，不能视为可执行操作',
  HOST_ACTION_NOT_ELIGIBLE_OR_AMBIGUOUS:'Host 操作不可执行或存在歧义，已拒绝启用操作',
  HOST_ACTION_IDENTITY_INVALID:'Host 操作标识无效，已拒绝启用操作'};
const ACTION_UNAVAILABLE_FALLBACK='操作当前不可用（来源校验未通过）';
const ACTION_LABEL_UNAVAILABLE='执行操作';
/** Explicit state for a fresh eligible Host action whose dispatch callback has
 * not been supplied by the integrating app; the control stays disabled. */
export const DISPATCH_NOT_CONNECTED_MESSAGE=
  '操作派发尚未接入：Host 已确认该操作新鲜可执行，但应用尚未提供派发通道，操作保持禁用';
export function localizedActionStatus(reason:string):string{
  return ACTION_STATUS_MESSAGES[reason]??ACTION_UNAVAILABLE_FALLBACK;
}
export interface WorkbenchActionRequestPayload{
  repository:string;versionId:string;sourceSha:string;
  actionId:string;revision:number;}
export interface WorkbenchActionControl{
  disabled:boolean;label:string;status:string|null;
  dispatchReady:boolean;request:WorkbenchActionRequestPayload|null;}
/** View-level display decision combining the Host display gate with whether
 * the integrating app supplied a dispatch callback. Display-only: a positive
 * label is shown only when dispatch is actually connected, and no request is
 * ever built without an exact fresh Host-issued identity.
 */
export function workbenchActionControl(snapshot:WorkbenchSnapshot|null,
  catalog:SelectedActionCatalog|null,workItem:WorkAction,
  dispatchConnected:boolean):WorkbenchActionControl{
  const gate=workbenchActionGate(snapshot,catalog,workItem);
  if(gate.enabled){
    if(!dispatchConnected)return {disabled:true,label:ACTION_LABEL_UNAVAILABLE,
      status:DISPATCH_NOT_CONNECTED_MESSAGE,dispatchReady:false,request:null};
    const source=catalog?.source,versionId=catalog?.versionId;
    if(gate.actionId===null||gate.revision===null||
       !source||versionId===undefined)
      return {disabled:true,label:ACTION_LABEL_UNAVAILABLE,
        status:localizedActionStatus('HOST_ACTION_IDENTITY_INVALID'),
        dispatchReady:false,request:null};
    return {disabled:false,label:gate.label,status:null,dispatchReady:true,
      request:{repository:source.repository,versionId,
        sourceSha:source.sourceSha,actionId:gate.actionId,
        revision:gate.revision}};
  }
  return {disabled:true,label:ACTION_LABEL_UNAVAILABLE,
    status:localizedActionStatus(gate.reason),dispatchReady:false,request:null};
}
