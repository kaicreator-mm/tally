/**
 * Closeout B/#261: selected Host action projection/dispatch gate for React UI.
 *
 * This module does NOT grant CAS Claims, select a Kernel, execute Agents or
 * write GitHub. It only passes a bounded user intention to an already selected
 * B->K->Host dispatcher after independently re-reading the Host-issued action.
 * The HTTP server (lane E) must enforce same-origin CSRF/session authorization.
 */
const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const ACTION_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/;
const KINDS=new Set(['FREEZE_PRODUCT','ADMIT_PROJECT','CREATE_TASK',
  'CLAIM_TASK','START_AGENT','REVIEW_REPAIR','PUBLISH_CANDIDATE']);
const SECRET=/\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|sk-[A-Za-z0-9_-]{16,})\b/;
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const denied=(reason)=>({connected:false,ok:false,status:'DENIED',reason});
const qualified=(snapshot,repository,versionId,sha)=>
  plain(snapshot)&&snapshot.connected===true&&
  snapshot.source?.repository===repository&&
  snapshot.source?.sourceSha===sha&&
  snapshot.source?.currentness==='CURRENT'&&
  snapshot.source?.readback==='VERIFIED'&&
  snapshot.versionId===versionId&&
  Array.isArray(snapshot.actions)&&snapshot.actions.length<=100;
function checkActions(actions,sha){
  const seen=new Set();
  for(const action of actions){
    if(!plain(action)||!ACTION_ID.test(action.id??'')||seen.has(action.id)||
       !KINDS.has(action.kind)||
       !Number.isSafeInteger(action.revision)||action.revision<1||
       !['ELIGIBLE','BLOCKED'].includes(action.state)||
       typeof action.label!=='string'||!action.label.trim()||
       action.label.length>160||SECRET.test(action.label)||
       (action.taskIssue!==null&&
         (!Number.isSafeInteger(action.taskIssue)||action.taskIssue<1))||
       (action.state==='ELIGIBLE'&&(
         action.hostAdmission!=='ADMITTED'||
         action.permitSource!=='SELECTED_K_HOST'||
         action.permitSourceSha!==sha))||
       (action.state==='BLOCKED'&&action.hostAdmission==='ADMITTED'))return false;
    seen.add(action.id);
  }
  return true;
}
function sanitized(actions){
  return actions.map(action=>({id:action.id,kind:action.kind,
    revision:action.revision,label:action.label,taskIssue:action.taskIssue,
    state:action.state,reason:action.state==='ELIGIBLE'?null:
      'HOST_ADMISSION_REQUIRED'}));
}
export function createUiActionPort({repository,versionId,expectedSha,host}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedSha??''))throw new TypeError('SELECTED_UI_ACTION_SCOPE_REQUIRED');
  const read=async()=>{
    if(typeof host?.readCurrentActions!=='function')
      return denied('SELECTED_HOST_ACTION_SOURCE_NOT_CONNECTED');
    try{
      const snapshot=await host.readCurrentActions({repository,versionId,sourceSha:expectedSha});
      if(!qualified(snapshot,repository,versionId,expectedSha)||
         !checkActions(snapshot.actions,expectedSha))
        return denied('HOST_ACTION_SOURCE_UNQUALIFIED');
      return snapshot;
    }catch{return denied('HOST_ACTION_READ_FAILED');}
  };
  async function actions(scope={}){
    if(scope.repository!==undefined&&scope.repository!==repository||
       scope.versionId!==undefined&&scope.versionId!==versionId)
      return denied('ACTION_SCOPE_MISMATCH');
    const result=await read();
    if(!qualified(result,repository,versionId,expectedSha))return result;
    return {connected:true,source:result.source,versionId,
      coverage:'SELECTED_HOST_ONLY',actions:sanitized(result.actions)};
  }
  async function submit(request){
    if(!plain(request)||Object.keys(request).sort().join(',')!==
      'actionId,repository,revision,sourceSha,versionId'||
      request.repository!==repository||request.versionId!==versionId||
      request.sourceSha!==expectedSha||!ACTION_ID.test(request.actionId??'')||
      !Number.isSafeInteger(request.revision)||request.revision<1)
      return denied('ACTION_REQUEST_STALE_OR_INVALID');
    // No caller-supplied permit, Claim, payload, GitHub token or effect plan.
    const first=await read();
    if(!qualified(first,repository,versionId,expectedSha))return first;
    const a=first.actions.find(x=>x.id===request.actionId);
    if(!a||a.revision!==request.revision||a.state!=='ELIGIBLE')
      return denied('ACTION_NOT_CURRENTLY_ADMITTED');
    if(typeof host?.dispatchAuthorizedAction!=='function')
      return denied('SELECTED_HOST_ACTION_DISPATCH_NOT_CONNECTED');
    // Host permits are ephemeral: a second source observation must match
    // exactly before dispatch. The Host/K also rechecks under its CAS writer.
    const second=await read();
    if(!qualified(second,repository,versionId,expectedSha))return second;
    const b=second.actions.find(x=>x.id===request.actionId);
    if(!b||b.state!=='ELIGIBLE'||b.revision!==a.revision||
       b.kind!==a.kind||b.taskIssue!==a.taskIssue)
      return denied('ACTION_SOURCE_CHANGED_BEFORE_DISPATCH');
    try{
      const result=await host.dispatchAuthorizedAction({
        repository,versionId,sourceSha:expectedSha,
        actionId:a.id,revision:a.revision,kind:a.kind,taskIssue:a.taskIssue});
      if(!plain(result)||result.sourceSha!==expectedSha||
         result.actionId!==a.id||
         !['OBSERVED','DENIED','UNKNOWN','RECONCILIATION_REQUIRED']
           .includes(result.status)||
         result.taskPass===true||result.releasePass===true||
         result.authorized===true||result.releasePublished===true||
         (result.status==='OBSERVED'&&
           (typeof result.receiptRef!=='string'||
            !result.receiptRef.startsWith('https://github.com/'+repository+'/'))))
        return denied('SELECTED_HOST_ACTION_RECEIPT_UNQUALIFIED');
      // Never promote Host receipt to independent Task/Release qualification.
      return {connected:true,ok:result.status==='OBSERVED',status:result.status,
        actionId:a.id,sourceSha:expectedSha,
        receiptRef:result.status==='OBSERVED'?result.receiptRef:null,
        taskQualification:'NOT_VERIFIED',releaseQualification:'NOT_VERIFIED'};
    }catch{return {connected:false,ok:false,status:'UNKNOWN',
      reason:'HOST_DISPATCH_ACK_UNKNOWN_RECONCILE_BEFORE_RETRY'};}
  }
  return Object.freeze({actions,submit});
}
