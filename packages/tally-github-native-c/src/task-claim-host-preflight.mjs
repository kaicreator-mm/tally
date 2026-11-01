/**
 * T06h: real GitHub current Task/blocked_by/ADS-event preflight for a Host.
 * Native GitHub source qualification is a prerequisite, not a CAS lock,
 * the designated Host publisher's identity, or a selected K Task permit.
 */
import {Octokit} from '@octokit/rest';
import {readNativeTaskSourcePairFromClient} from './task-source-pair.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[0-9a-f]{40}$/;
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
  octokit,owner,repo,issueNumber,expectedSourceSha,limit=1000
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !SHA.test(expectedSourceSha??'')||
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
  if(facts.claimEvents?.reason!=='NO_DISPATCH_CLAIM_EVENTS'||
     facts.claimEvents.candidateCount!==0||
     !Array.isArray(facts.claimEvents.candidates)||
     facts.claimEvents.candidates.length!==0||
     facts.claimEvents.claimStatus!=='NOT_VERIFIED'||
     facts.claimEvents.admitted!==false)
    return blocked('NATIVE_CLAIM_ABSENCE_UNQUALIFIED');
  return Object.freeze({
    status:'NATIVE_SOURCE_PREFLIGHT_READY',
    reason:'HOST_BINDING_AND_ATOMIC_CAS_STILL_REQUIRED',
    sourceKind:'NATIVE_GITHUB_READ_ONLY_PREFLIGHT',
    sourceQualified:true,nativePreflightReady:true,
    issueRef:facts.issueRef,sourceSha:expectedSourceSha,
    issueState:'open',workflowState:'state:ready',workItemType:'type:task',
    dependencyState:'CLEAR',openBlockerCount:0,claimCandidates:0,
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
