/** One-Task currentness projection, composed ONLY from a selected native GitHub C pair.
 * Read-only B/UI explanation, never a CAS Claim, K permit, Agent start or Release.
 */
const SHA=/^[a-f0-9]{40}$/;
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/;
const state=(reason)=>Object.freeze({status:'UNKNOWN',reason,sourceQualified:false,
  sourceRef:null,sourceSha:null,dependencyState:'UNKNOWN',openBlockerCount:null,
  workflowState:null,claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',
  agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
  ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  nextActions:Object.freeze([{kind:'READ_GITHUB_SOURCE',actorRole:'source-reader',
    authority:'READ_ONLY',reason}])});
const action=(s,status,reason,kind,actorRole,authority)=>Object.freeze({
  ...s,status,reason,nextActions:Object.freeze([Object.freeze({kind,actorRole,
    authority,reason})])});
export function projectNativeTaskEffectiveSnapshot({
  sourcePair,expectedIssueRef,expectedSourceSha
}={}){
  try{
    if(!REF.test(expectedIssueRef??'')||!SHA.test(expectedSourceSha??''))
      return state('EXPECTED_SOURCE_IDENTITY_UNAVAILABLE');
    const s=sourcePair;
    if(!s||typeof s!=='object'||Array.isArray(s)||
      s.status!=='SOURCE_PAIR_OBSERVED'||s.sourceQualified!==true||
      s.issueRef!==expectedIssueRef||s.sourceSha!==expectedSourceSha||
      s.authorized!==false||s.admitted!==false||s.taskPass!==false||
      s.releasePass!==false||s.claimStatus!=='NOT_VERIFIED')
      return state('SOURCE_PAIR_NOT_QUALIFIED');
    const w=s.workItem,c=s.claimEvents;
    if(!w||typeof w!=='object'||Array.isArray(w)||
      w.issueRef!==expectedIssueRef||w.sourceSha!==expectedSourceSha||
      w.authorized!==false||w.admitted!==false||w.taskPass!==false||
      w.claimStatus!=='NOT_VERIFIED'||!c||typeof c!=='object'||
      Array.isArray(c)||c.claimStatus!=='NOT_VERIFIED'||c.authorized!==false||
      c.admitted!==false||c.taskPass!==false||c.releasePass!==false||
      !Number.isSafeInteger(c.candidateCount)||c.candidateCount<0||
      c.candidateCount>1000||!Array.isArray(c.candidates)||
      c.candidates.length!==c.candidateCount)
      return state('CURRENT_NATIVE_FACTS_UNVERIFIED');
    const p={
      sourceQualified:true,sourceRef:expectedIssueRef,sourceSha:expectedSourceSha,
      dependencyState:w.dependencyState,
      openBlockerCount:Number.isSafeInteger(w.openBlockerCount)?w.openBlockerCount:null,
      workflowState:typeof w.workflowState==='string'?w.workflowState:null,
      claimState:'NOT_VERIFIED',kernelTaskState:'NOT_VERIFIED',
      agentAttemptState:'NOT_VERIFIED',reviewState:'NOT_VERIFIED',
      ciState:'NOT_VERIFIED',hostState:'NOT_VERIFIED',releaseState:'NOT_VERIFIED',
      authorized:false,admitted:false,taskPass:false,releasePass:false
    };
    if(w.dependencyState==='BLOCKED')
      return action(p,'WAIT_EXTERNAL','NATIVE_DEPENDENCIES_BLOCKED',
        'WAIT_NATIVE_DEPENDENCIES','source-reader','READ_ONLY');
    if(w.dependencyState!=='CLEAR')
      return state('NATIVE_DEPENDENCIES_UNKNOWN');
    if(w.issueState!=='open'||w.workflowState!=='state:ready'||
      w.workItemType!=='type:task'||w.reason!=='CLAIM_NOT_VERIFIED')
      return action(p,'WAIT_EXTERNAL','TASK_NOT_NATIVE_READY',
        'WAIT_NATIVE_TASK_READY','source-reader','READ_ONLY');
    if(c.reason==='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'&&
      c.candidateCount>0)
      return action(p,'WAIT_EXTERNAL','SELF_REPORTED_CLAIM_NEEDS_HOST_RECONCILIATION',
        'RECONCILE_NATIVE_CLAIM','host-controller','HOST_REQUIRED');
    if(c.reason!=='NO_DISPATCH_CLAIM_EVENTS'||c.candidateCount!==0)
      return state('NATIVE_CLAIM_ABSENCE_UNVERIFIED');
    return action(p,'AWAITING_HOST_ADMISSION','NATIVE_READY_NO_ACCEPTED_CLAIM',
      'REQUEST_SELECTED_HOST_ADMISSION','host-controller','HOST_AND_ADS_REQUIRED');
  }catch{return state('SOURCE_OBSERVATION_UNREADABLE');}
}
