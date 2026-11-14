/**
 * T06h: real GitHub current Task/blocked_by/ADS-event preflight for a Host.
 * Native GitHub source qualification is a prerequisite, not a CAS lock,
 * the designated Host publisher's identity, or a selected K Task permit.
 *
 * #412 dual-role admission: a Task may hold one accepted Claim per role.
 * With an optional declared `actorRole` the preflight admits existing
 * DISPATCH_CLAIMED candidates only when every candidate is a DIFFERENT,
 * canonical-keyed, generation-1 role; the requested role, a repeated
 * protectedClaimKey or any unqualified candidate stays fail-closed.
 * Without a declared role the historical zero-candidate requirement is
 * preserved unchanged. Candidates are re-observed through the existing
 * canonical comment reader (imported, not modified); no permit or writer
 * grant is produced here.
 */
import {Octokit} from '@octokit/rest';
import {readNativeTaskSourcePairFromClient} from './task-source-pair.mjs';
import {readClaimEventCandidatesFromClient} from './task-claim-events.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[0-9a-f]{40}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const blocked=(reason)=>Object.freeze({
  status:'BLOCKED',reason,sourceQualified:false,
  nativePreflightReady:false,sourceKind:'NATIVE_GITHUB_READ_ONLY_PREFLIGHT',
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
  taskPass:false,releasePass:false
});

/**
 * Test seam for a Host-supplied authenticated Octokit. Callers cannot
 * inject dependenciesComplete, a claimed role, a writer grant or CAS result.
 * The complete existing native pair reader rechecks Issue currentness.
 */
export async function readNativeHostClaimPreflightFromClient({
  octokit,owner,repo,issueNumber,expectedSourceSha,actorRole,limit=1000
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !SHA.test(expectedSourceSha??'')||
     (actorRole!==undefined&&!ROLES.has(actorRole))||
     !Number.isSafeInteger(limit)||limit<1||limit>1000)
    return blocked('INVALID_NATIVE_PREFLIGHT_REQUEST');
  const facts=await readNativeTaskSourcePairFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
  if(facts.status!=='SOURCE_PAIR_OBSERVED'||facts.sourceQualified!==true||
     facts.sourceSha!==expectedSourceSha||facts.claimStatus!=='NOT_VERIFIED'||
     facts.admitted!==false||facts.authorized!==false||
     facts.workItem?.issueRef!==
       'https://github.com/'+owner+'/'+repo+'/issues/'+issueNumber||
     facts.workItem?.sourceSha!==expectedSourceSha)
    return blocked('NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE');
  if(facts.workItem.dependencyState==='BLOCKED')
    return blocked('NATIVE_ISSUE_DEPENDENCIES_BLOCKED');
  if(facts.workItem.issueState!=='open'||
     facts.workItem.workflowState!=='state:ready'||
     facts.workItem.workItemType!=='type:task'||
     facts.workItem.dependencyState!=='CLEAR'||
     facts.workItem.reason!=='CLAIM_NOT_VERIFIED')
    return blocked('NATIVE_ISSUE_NOT_CLAIMABLE');
  const claim=facts.claimEvents;
  const pairQualified=claim?.reason==='NO_DISPATCH_CLAIM_EVENTS'||
    claim?.reason==='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED';
  if(!pairQualified||!Array.isArray(claim.candidates)||
     claim.candidates.length!==claim.candidateCount||
     claim.claimStatus!=='NOT_VERIFIED'||claim.admitted!==false)
    return blocked('NATIVE_CLAIM_ABSENCE_UNQUALIFIED');
  // Role-bearing candidates come from the same canonical comment reader the
  // pair projection derives from; its own fail-closed behavior (duplicate
  // dispatch, cross-task/stale, edited identity, malformed comment, drift)
  // is preserved and propagates as BLOCKED.
  const observed=await readClaimEventCandidatesFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
  const observedQualified=observed.reason==='NO_DISPATCH_CLAIM_EVENTS'||
    observed.reason==='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED';
  if(!observedQualified||observed.claimStatus!=='NOT_VERIFIED'||
     observed.admitted!==false||observed.authorized!==false||
     !Array.isArray(observed.candidates)||
     observed.candidates.length!==observed.candidateCount)
    return blocked(observed.reason??'NATIVE_CLAIM_ABSENCE_UNQUALIFIED');
  if(observed.candidateCount!==claim.candidateCount)
    return blocked('NATIVE_CLAIM_ABSENCE_UNQUALIFIED');
  const roles=new Set();
  const projections=[];
  if(observed.candidates.length>0){
    // A declared requested role is required to classify existing candidates.
    if(actorRole===undefined)
      return blocked('NATIVE_CLAIM_ABSENCE_UNQUALIFIED');
    const seenKeys=new Set();
    for(const c of observed.candidates){
      if(!ROLES.has(c.actorRole)||c.claimGeneration!==1||
         c.protectedClaimKey!==
           owner+'/'+repo+'#'+issueNumber+':'+c.actorRole+':__default__'||
         c.sourceSha!==expectedSourceSha)
        return blocked('NATIVE_CLAIM_ROLE_CANDIDATE_UNQUALIFIED');
      if(c.actorRole===actorRole||seenKeys.has(c.protectedClaimKey))
        return blocked('NATIVE_CLAIM_ROLE_DUPLICATE_CANDIDATE');
      seenKeys.add(c.protectedClaimKey);
      roles.add(c.actorRole);
      projections.push(Object.freeze({
        commentId:c.commentId,commentRef:c.commentRef,
        dispatchId:c.dispatchId,actorRole:c.actorRole,
        operatorId:c.operatorId,protectedClaimKey:c.protectedClaimKey,
        claimGeneration:c.claimGeneration,sourceSha:c.sourceSha,
        transportActor:c.transportActor}));
    }
  }
  return Object.freeze({
    status:'NATIVE_SOURCE_PREFLIGHT_READY',
    reason:'HOST_BINDING_AND_ATOMIC_CAS_STILL_REQUIRED',
    sourceKind:'NATIVE_GITHUB_READ_ONLY_PREFLIGHT',
    sourceQualified:true,nativePreflightReady:true,
    issueRef:facts.issueRef,sourceSha:expectedSourceSha,
    issueState:'open',workflowState:'state:ready',workItemType:'type:task',
    dependencyState:'CLEAR',openBlockerCount:0,
    claimCandidates:observed.candidates.length,
    claimCandidateRoles:Object.freeze([...roles].sort()),
    claimRoleCandidates:Object.freeze(projections),
    // Explicitly NOT the Host controller's privileged
    // NATIVE_GITHUB_CURRENT_ADMISSION: writer/dispatch may be spoofed.
    dispatchSourceBound:false,publisherSourceBound:false,
    claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
    taskPass:false,releasePass:false
  });
}
export async function readGitHubNativeHostClaimPreflight({token,...input}={}){
  if(typeof token!=='string'||token.length<8)
    return blocked('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return readNativeHostClaimPreflightFromClient({octokit,...input});
}
