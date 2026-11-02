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
// Only intents already present in the selected Tally Engineering Business
// package are dispatchable. Task creation and CAS Claim publication are not
// invented here: admitClaimedTask consumes an already accepted Host Claim.
const KINDS=new Set(['FREEZE_PRODUCT','ADMIT_PROJECT','ADMIT_CLAIMED_TASK',
  'START_AGENT','REVIEW_REPAIR','PUBLISH_CANDIDATE',
  'PUBLISH_MAINTENANCE_ISSUE','RECORD_TASK_LEARNING',
  'DELIVER_QUALIFIED_RELEASE']);
const BUSINESS_INTENT_BY_KIND=Object.freeze({
  FREEZE_PRODUCT:'freezeProductIntake',
  ADMIT_PROJECT:'admitProjectIntake',
  ADMIT_CLAIMED_TASK:'admitClaimedTask',
  START_AGENT:'runLocalCodexAttempt',
  REVIEW_REPAIR:'joinReviewedCandidates',
  PUBLISH_CANDIDATE:'publishReviewedCandidate',
  PUBLISH_MAINTENANCE_ISSUE:'publishMaintenanceIssue',
  RECORD_TASK_LEARNING:'recordTaskLearningOutcome',
  DELIVER_QUALIFIED_RELEASE:'deliverQualifiedRelease',
});
/** Exact public Tally Engineering Business intent mapping for E's one-K adapter. */
export function businessIntentForUiAction(kind){
  if(typeof kind!=='string'||
     !Object.prototype.hasOwnProperty.call(BUSINESS_INTENT_BY_KIND,kind))
    return null;
  return BUSINESS_INTENT_BY_KIND[kind];
}

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
const EXECUTION_STATES=new Set([
  'QUEUED','ADMISSION_BLOCKED','ADMITTED','STARTING','RUNNING','CANDIDATE_READY',
  'REVIEWING','REPAIRING','CI_RUNNING','COMPLETED','FAILED','BLOCKED','CANCELLED',
  'UNKNOWN','RECONCILING'
]);
const EXECUTION_ROLES=new Set(['BUILDER','REVIEWER','REPAIR','VALIDATOR','ORCHESTRATOR','UNKNOWN']);
const EXECUTION_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,79}$/;
const PROVIDER_ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$/;
const PROGRESS_SECRET=/\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|sk-[A-Za-z0-9_-]{16,}|Bearer\s+\S+)\b/i;
const CANDIDATE_PR=/^https:\/\/github\.com\/([A-Za-z0-9_.-]+)\/([A-Za-z0-9_.-]+)\/pull\/[1-9][0-9]*$/;
function validTaskExecution(execution,repository){
  if(!plain(execution)||Object.keys(execution).sort().join(',')!==
    'attemptId,candidateRef,costUsd,elapsedMs,heartbeatAt,progressSummary,progressSummaryKind,provider,role,state,tokensIn,tokensOut'||
    !EXECUTION_STATES.has(execution.state)||
    !(execution.provider===null||typeof execution.provider==='string'&&PROVIDER_ID.test(execution.provider))||
    !(execution.role===null||EXECUTION_ROLES.has(execution.role))||
    !(execution.attemptId===null||typeof execution.attemptId==='string'&&EXECUTION_ID.test(execution.attemptId))||
    !['SANITIZED','NOT_AVAILABLE'].includes(execution.progressSummaryKind)||
    !(execution.progressSummaryKind==='NOT_AVAILABLE'&&execution.progressSummary===null||
      execution.progressSummaryKind==='SANITIZED'&&typeof execution.progressSummary==='string'&&
      execution.progressSummary.length>0&&execution.progressSummary.length<=240&&
      !/[\u0000-\u001f\u007f]/.test(execution.progressSummary)&&
      !PROGRESS_SECRET.test(execution.progressSummary))||
    !(execution.heartbeatAt===null||typeof execution.heartbeatAt==='string'&&
      execution.heartbeatAt.length<=40&&Number.isFinite(Date.parse(execution.heartbeatAt)))||
    !(execution.elapsedMs===null||Number.isSafeInteger(execution.elapsedMs)&&execution.elapsedMs>=0)||
    !(execution.tokensIn===null||Number.isSafeInteger(execution.tokensIn)&&execution.tokensIn>=0)||
    !(execution.tokensOut===null||Number.isSafeInteger(execution.tokensOut)&&execution.tokensOut>=0)||
    !(execution.costUsd===null||typeof execution.costUsd==='number'&&
      Number.isFinite(execution.costUsd)&&execution.costUsd>=0))
    return false;
  if(execution.candidateRef===null)return true;
  const match=typeof execution.candidateRef==='string'&&CANDIDATE_PR.exec(execution.candidateRef);
  return Boolean(match&&match[1]+'/'+match[2]===repository);
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
  async function taskExecution(taskIssue){
    if(!Number.isSafeInteger(taskIssue)||taskIssue<1)
      return denied('TASK_EXECUTION_ISSUE_INVALID');
    if(typeof host?.readTaskExecution!=='function')
      return denied('SELECTED_HOST_TASK_EXECUTION_NOT_CONNECTED');
    try{
      const snapshot=await host.readTaskExecution({
        repository,versionId,sourceSha:expectedSha,taskIssue
      });
      if(!plain(snapshot)||snapshot.connected!==true||
         snapshot.source?.repository!==repository||
         snapshot.source?.sourceSha!==expectedSha||
         snapshot.source?.currentness!=='CURRENT'||
         snapshot.source?.readback!=='VERIFIED'||
         snapshot.versionId!==versionId||snapshot.taskIssue!==taskIssue||
         !validTaskExecution(snapshot.execution,repository))
        return denied('HOST_TASK_EXECUTION_SOURCE_UNQUALIFIED');
      return {connected:true,source:snapshot.source,versionId,taskIssue,
        execution:Object.freeze({...snapshot.execution})};
    }catch{return denied('HOST_TASK_EXECUTION_READ_FAILED');}
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
  return Object.freeze({
    scope:Object.freeze({repository,versionId,sourceSha:expectedSha}),
    actions,taskExecution,submit,
  });
}

/** Pure HTTP boundary for E's same-origin route. E owns socket/body parsing and
 * calls this adapter; it cannot be mounted by a browser or expose credentials.
 * Its selected project/version/SHA are derived only from the immutable scope
 * issued by createUiActionPort; caller-supplied scope cannot override the port.
 */
export function createUiActionHttpAdapter({actionPort,origin,csrfKey}={}){
  const scope=actionPort?.scope;
  if(typeof actionPort?.actions!=='function'||typeof actionPort?.submit!=='function'||
     !plain(scope)||!Object.isFrozen(scope)||
     Object.keys(scope).sort().join(',')!=='repository,sourceSha,versionId'||
     !REPO.test(scope.repository??'')||!VERSION.test(scope.versionId??'')||
     !SHA.test(scope.sourceSha??'')||
     typeof origin!=='string'||!/^http:\/\/127\.0\.0\.1:[0-9]+$/.test(origin)||
     typeof csrfKey!=='string'||csrfKey.length<32)
    throw new TypeError('UI_ACTION_HTTP_HOST_CONFIGURATION_REQUIRED');
  const {repository,versionId,sourceSha}=scope;
  const response=(status,body)=>Object.freeze({status,body});
  const sameOrigin=request=>request?.origin===origin;
  return Object.freeze({async handle(request={}){
    if(!sameOrigin(request))return response(403,denied('CSRF_OR_ORIGIN'));
    if(request.method==='GET'&&request.path==='/api/v2/actions'){
      const q=request.query;
      if(!plain(q)||Object.keys(q).sort().join(',')!=='projectId,versionId'||
         q.projectId!==undefined&&q.projectId!==repository||
         q.versionId!==undefined&&q.versionId!==versionId)
        return response(400,denied('ACTION_SCOPE_MISMATCH'));
      const result=await actionPort.actions({repository,versionId});
      return response(result.connected?200:503,result);
    }
    if(request.method==='GET'&&request.path==='/api/v2/task-execution'){
      const q=request.query;
      if(!plain(q)||Object.keys(q).sort().join(',')!=='projectId,taskIssue,versionId'||
         q.projectId!==repository||q.versionId!==versionId||
         !Number.isSafeInteger(q.taskIssue)||q.taskIssue<1)
        return response(400,denied('TASK_EXECUTION_SCOPE_OR_QUERY_INVALID'));
      if(typeof actionPort.taskExecution!=='function')
        return response(503,denied('SELECTED_HOST_TASK_EXECUTION_NOT_CONNECTED'));
      const result=await actionPort.taskExecution(q.taskIssue);
      return response(result.connected?200:503,result);
    }
    if(request.method!=='POST'||request.path!=='/api/v2/actions/dispatch')
      return response(404,denied('ACTION_ROUTE_NOT_FOUND'));
    if(request.csrfKey!==csrfKey)return response(403,denied('CSRF_ORIGIN_KEY'));
    if(!plain(request.body)||Object.keys(request.body).sort().join(',')!==
       'actionId,projectId,revision,sourceSha,versionId'||
       request.body.projectId!==repository||
       request.body.versionId!==versionId)
      return response(400,denied('ACTION_REQUEST_SHAPE_INVALID'));
    const result=await actionPort.submit({
      repository,versionId,sourceSha:request.body.sourceSha,
      actionId:request.body.actionId,revision:request.body.revision});
    return response(result.ok===true?200:409,result);
  }});
}
