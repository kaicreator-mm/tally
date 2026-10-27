/**
 * Tally v2 T06c Engineering Business advisory readiness projection.
 *
 * Native C observations can be supplied by selected K/Host only in a future
 * admitted operation. This helper never attests its own sources, creates a
 * durable CAS Claim, takes a K role or upgrades Task/Release authority.
 *
 * An observed v3 DISPATCH_CLAIMED comment is not an accepted CAS Start Record.
 */
const SOURCE_SHA=/^[0-9a-f]{40}$/;
const ISSUE=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/;
const output=(reason,more={})=>Object.freeze({
  state:'NOT_ADMITTED',reason,dependencyState:'UNKNOWN',
  claimState:'NOT_VERIFIED',agentState:'UNKNOWN',
  admitted:false,authorized:false,taskPass:false,releasePass:false,
  ...more
});
/**
 * Only present a bounded explainable reason, not raw GitHub events/comments.
 * Even a caller-spoofed "accepted" flag is never proof of serialized CAS.
 */
export function diagnoseNativeTaskReadiness(options={}){
  try{
    const {workItem,claimEvents,expectedIssueRef,expectedSourceSha,
      localAgentStatus='UNKNOWN',builderConfinementStatus='UNKNOWN',
      hiddenVerifierStatus='UNKNOWN'}=options??{};
    if(!ISSUE.test(expectedIssueRef??'')||
      !SOURCE_SHA.test(expectedSourceSha??''))
      return output('SOURCE_IDENTITY_UNAVAILABLE');
    if(!workItem||typeof workItem!=='object'||Array.isArray(workItem)||
      workItem.issueRef!==expectedIssueRef||
      workItem.sourceSha!==expectedSourceSha||
      workItem.authorized!==false||
      workItem.admitted!==false||
      workItem.taskPass!==false||
      workItem.claimStatus!=='NOT_VERIFIED')
      return output('NATIVE_WORK_ITEM_SOURCE_UNVERIFIED');
    const deps=workItem.dependencyState;
    if(deps==='BLOCKED')return output('NATIVE_DEPENDENCIES_BLOCKED',{
      dependencyState:'BLOCKED'});
    if(deps!=='CLEAR')return output('NATIVE_DEPENDENCIES_UNKNOWN');
    if(workItem.reason!=='CLAIM_NOT_VERIFIED'||workItem.workflowState!=='state:ready')
      return output('WORK_ITEM_NOT_CLAIMABLE_SOURCE_ONLY',{dependencyState:'CLEAR'});
    if(!claimEvents||typeof claimEvents!=='object'||Array.isArray(claimEvents)||
      claimEvents.claimStatus!=='NOT_VERIFIED'||
      claimEvents.authorized!==false||claimEvents.admitted!==false||
      claimEvents.taskPass!==false||claimEvents.releasePass!==false||
      !Number.isSafeInteger(claimEvents.candidateCount)||
      claimEvents.candidateCount<0||claimEvents.candidateCount>1000||
      !Array.isArray(claimEvents.candidates)||
      claimEvents.candidates.length!==claimEvents.candidateCount)
      return output('CLAIM_EVENT_SOURCE_UNVERIFIED',{dependencyState:'CLEAR'});
    if(claimEvents.candidateCount>1)
      return output('AMBIGUOUS_CLAIM_EVENT_CANDIDATES',{
        dependencyState:'CLEAR',claimState:'AMBIGUOUS'});
    if(claimEvents.candidateCount===1){
      const c=claimEvents.candidates[0];
      if(!c||c.sourceSha!==expectedSourceSha||
        c.sourceClass!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'||
        c.claimAccepted!==false||
        typeof c.dispatchId!=='string'||!c.dispatchId)
        return output('CLAIM_EVENT_SOURCE_UNVERIFIED',{dependencyState:'CLEAR'});
    }
    // OS gates are independently necessary and cannot be asserted by an
    // unprivileged event. An omitted witness is UNKNOWN, never AVAILABLE.
    if(localAgentStatus!=='AVAILABLE')
      return output('LOCAL_AGENT_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'UNAVAILABLE_OR_UNKNOWN'});
    if(builderConfinementStatus!=='VERIFIED')
      return output('BUILDER_CONFINEMENT_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'CONFINEMENT_UNVERIFIED'});
    if(hiddenVerifierStatus!=='VERIFIED')
      return output('HIDDEN_VERIFIER_UNAVAILABLE',{dependencyState:'CLEAR',
        agentState:'HIDDEN_UNVERIFIED'});
    // Even all three positive status literals are not independent OS proof,
    // and a correct comment is not an accepted CAS writer's Start Record.
    return output('ACCEPTED_CAS_CLAIM_NOT_VERIFIED',{
      dependencyState:'CLEAR',
      claimState:claimEvents.candidateCount?'SELF_REPORTED_EVENT_ONLY':'NONE',
      agentState:'DECLARED_READY_ONLY'
    });
  }catch{return output('UNTRUSTED_SOURCE_INPUT');}
}
