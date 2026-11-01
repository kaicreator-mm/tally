/** Read-only Project Journey projection. No second Task state or execution engine. */
const SHA=/^[0-9a-f]{40}$/;
const ISSUE=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/;
const isoTime=value=>typeof value==='string'&&Number.isFinite(Date.parse(value))&&
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(value);
const freeze=value=>Object.freeze(value);
const invalid=reason=>freeze({authorizationType:'NO_VALID_ACTION',reason,
  actorRole:'validator',requiredCapability:'NATIVE_SOURCE_READ',approvalNeeded:false,costRisk:'HIGH'});
const capabilityCost=new Set(['LOW','MEDIUM','HIGH']);
const VALID_STATUSES=new Set(['AWAITING_HOST_ADMISSION','AWAITING_REVIEW',
  'REVIEW_CANDIDATE_EVIDENCE','WAIT_EXTERNAL','CONFLICT','BLOCKED','UNKNOWN']);
const REASON=/^[A-Z][A-Z0-9_]{1,64}$/;
const HOST_KINDS=new Set(['WINDOWS_CODEX_HOST','ANDROID_HERMES_HOST',
  'LOCAL_BUILD_HOST','RELEASE_AUTHORITY']);
function action(type,task,{actorRole,capability,approvalNeeded=false,reason,risk='LOW',authorization='ADVISORY_ONLY'}){
  const evidence=freeze({sourceRef:task.sourceRef,sourceSha:task.sourceSha,observedAsOf:task.observedAsOf});
  return freeze({authorizationType:type,reason,evidence,currentness:task.currentness,
    actorRole,requiredCapability:capability,approvalNeeded,
    costRisk:capabilityCost.has(risk)?risk:'HIGH',authorization});
}
function validTask(task,sourceSha){
  return task&&typeof task==='object'&&!Array.isArray(task)&&ISSUE.test(task.sourceRef??'')&&
    SHA.test(task.sourceSha??'')&&
    (task.observedAsOf===null||isoTime(task.observedAsOf))&&
    (task.currentness==='AGE_NOT_VERIFIED'||task.currentness==='OBSERVED_AT_QUERY_COMPLETION')&&
    VALID_STATUSES.has(task.status)&&REASON.test(task.reason??'')&&task.authorized===false&&
    task.admitted===false&&task.taskPass===false&&task.releasePass===false;
}
export function projectJourneySnapshot({project,version,sourceSha,tasks,observedAt}={}){
  if(typeof project!=='string'||!project||typeof version!=='string'||!version||
    !SHA.test(sourceSha??'')||!isoTime(observedAt)||!Array.isArray(tasks))
    return Object.freeze({status:'UNKNOWN',reason:'JOURNEY_INPUT_UNVERIFIED',tasks:[]});
  if(!tasks.every(task=>validTask(task,sourceSha)))
    return Object.freeze({status:'UNKNOWN',reason:'JOURNEY_TASK_UNVERIFIED',tasks:[]});
  // Counts describe this snapshot's lanes only, never a global progress percentage.
  const progress=Object.freeze({
    candidates:tasks.filter(x=>x.agentAttemptState==='LOCAL_CANDIDATE_READY').length,
    reviewed:tasks.filter(x=>x.reviewState==='REVIEW_JOINED').length,
    released:tasks.filter(x=>x.releaseState==='RELEASED').length,
    comparison:'COUNTS_NOT_PERCENTAGES'
  });
  return Object.freeze({status:'OBSERVED',project,version,sourceSha,observedAsOf:observedAt,
    currentness:'OBSERVED_AT_QUERY_COMPLETION',tasks:Object.freeze(tasks.map(x=>Object.freeze(x))),
    progressSignals:progress,authorized:false,admitted:false,taskPass:false,releasePass:false});
}
function oneAction(task){
  if(task.scopeConflict===true)
    return action('NEEDS_USER_DECISION',task,{actorRole:'human',capability:'PRODUCT_SCOPE_DECISION',
      approvalNeeded:true,reason:'PRODUCT_SCOPE_CONFLICT',risk:'HIGH'});
  if(task.requiredHostEvidence?.available===false&&HOST_KINDS.has(task.requiredHostEvidence.kind)&&
    REASON.test(task.requiredHostEvidence.reason??''))
    return action('NEEDS_HUMAN_HOST',task,{actorRole:'human',
      capability:task.requiredHostEvidence.kind,approvalNeeded:true,
      reason:task.requiredHostEvidence.reason,risk:'HIGH'});
  if(task.requiredHostEvidence?.available===false)
    return invalid('REQUIRED_HOST_EVIDENCE_UNVERIFIED');
  if(task.currentness!=='OBSERVED_AT_QUERY_COMPLETION')
    return action('WAIT_EXTERNAL',task,{actorRole:'validator',capability:'NATIVE_SOURCE_READ',
      approvalNeeded:false,reason:'CURRENTNESS_UNVERIFIED'});
  if(task.status==='AWAITING_HOST_ADMISSION')
    return action('START_ELIGIBLE_CLAIM',task,{actorRole:'builder',capability:'HOST_CLAIM_WRITER',
      approvalNeeded:true,reason:'NATIVE_READY_NO_ACCEPTED_CLAIM',authorization:'HOST_AND_ADS_REQUIRED'});
  if(task.status==='AWAITING_REVIEW')
    return action('START_ELIGIBLE_INDEPENDENT_REVIEW',task,{actorRole:'reviewer',
      capability:'INDEPENDENT_REVIEWER',approvalNeeded:true,reason:'LOCAL_CANDIDATE_READY',
      authorization:'HOST_AND_ADS_REQUIRED'});
  if(task.status==='REVIEW_CANDIDATE_EVIDENCE')
    return action('START_ELIGIBLE_VALIDATOR',task,{actorRole:'validator',capability:'VALIDATOR',
      approvalNeeded:true,reason:'REVIEW_JOINED',authorization:'HOST_AND_ADS_REQUIRED'});
  if(task.status==='WAIT_EXTERNAL')
    return action('WAIT_EXTERNAL',task,{actorRole:'validator',capability:'NATIVE_SOURCE_READ',
      approvalNeeded:false,reason:task.reason});
  if(task.status==='CONFLICT'&&task.reason==='AGENT_ATTEMPT_NOT_COMPLETE')
    return action('REPAIR_ELIGIBLE',task,{actorRole:'builder',capability:'REPAIR_BUILDER',
      approvalNeeded:true,reason:task.reason,risk:'MEDIUM',authorization:'HOST_REQUIRED'});
  if(task.status==='CONFLICT')return invalid('CONCURRENT_AUTHORITY_CONFLICT');
  if(task.status==='BLOCKED')return invalid('NATIVE_DEPENDENCIES_BLOCKED');
  if(task.status==='UNKNOWN')return invalid(task.reason);
  return invalid('JOURNEY_TASK_STATE_UNVERIFIED');
}
export function nextActions(snapshot,{changedSourceSha}={}){
  if(!snapshot||snapshot.status!=='OBSERVED'||!Array.isArray(snapshot.tasks))
    return Object.freeze({actions:Object.freeze([invalid('JOURNEY_SNAPSHOT_UNVERIFIED')]),
      humanActionInbox:Object.freeze([]),changedSourceSha:null,authorized:false,
      admitted:false,taskPass:false,releasePass:false});
  if(changedSourceSha!==undefined&&!SHA.test(changedSourceSha))
    return Object.freeze({actions:Object.freeze([invalid('CHANGED_SOURCE_IDENTITY_UNVERIFIED')]),
      humanActionInbox:Object.freeze([]),changedSourceSha:null,authorized:false,
      admitted:false,taskPass:false,releasePass:false});
  const tasks=snapshot.tasks.filter(task=>changedSourceSha===undefined||task.sourceSha===changedSourceSha);
  const actions=tasks.map(oneAction);
  const inbox=actions.filter(x=>x.actorRole==='human'&&x.approvalNeeded).map(x=>Object.freeze({
    kind:x.authorizationType,reason:x.reason,expectedEvidence:x.requiredCapability,
    readback:x.evidence.sourceRef,sourceSha:x.evidence.sourceSha}));
  return Object.freeze({actions:Object.freeze(actions),humanActionInbox:Object.freeze(inbox),
    changedSourceSha:changedSourceSha??null,authorized:false,admitted:false,
    taskPass:false,releasePass:false});
}
