/**
 * T06e — source-qualified, READ-ONLY GitHub Work Item + ADS Claim event pair.
 *
 * This is NOT an accepted native CAS Claim, DomainHarness K admission,
 * agent scheduler, release decision, or authoritative status writer.
 * It reuses T06a and T06b Octokit readers, with one outer as-of sentinel.
 * A caller can pass this bounded C projection to selected Engineering B;
 * only K may authorize later physical effects after separate Host authority.
 */
import {Octokit} from '@octokit/rest';
import {readWorkItemSourceFromClient} from './task-workitem-source.mjs';
import {readClaimEventCandidatesFromClient} from './task-claim-events.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[0-9a-f]{40}$/;
const denied=reason=>Object.freeze({
  status:'UNKNOWN',reason,sourceQualified:false,
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
  taskPass:false,releasePass:false
});
const errorClass=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===429?'GITHUB_RATE_LIMITED':'NATIVE_SOURCE_PAIR_READ_FAILED';

function currentIssue(data,url,n){
  if(!data||data.html_url!==url||data.number!==n||
     !Number.isSafeInteger(data.id)||data.id<1||
     typeof data.updated_at!=='string'||!Array.isArray(data.labels)||
     !['open','closed'].includes(data.state)||data.pull_request!==undefined)
    return null;
  const labels=data.labels.map(x=>typeof x==='string'?x:x?.name);
  if(labels.some(x=>typeof x!=='string'))return null;
  const canonical=labels.filter(x=>x.startsWith('state:')||x.startsWith('type:')).sort();
  return Object.freeze({id:data.id,updatedAt:data.updated_at,
    issueState:data.state,canonical:canonical.join('|')});
}
const same=(a,b)=>a&&b&&a.id===b.id&&
  a.updatedAt===b.updatedAt&&a.issueState===b.issueState&&
  a.canonical===b.canonical;

/** Source-only test seam; the production entrypoint creates Octokit itself. */
export async function readNativeTaskSourcePairFromClient({
  octokit,owner,repo,issueNumber,expectedSourceSha,limit=1000
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !SHA.test(expectedSourceSha??'')||
     !Number.isSafeInteger(limit)||limit<1||limit>1000||
     !octokit?.rest?.issues?.get||!octokit?.rest?.issues?.listComments||
     !octokit?.rest?.repos?.getCommit||!octokit?.paginate?.iterator)
    return denied('INVALID_NATIVE_SOURCE_PAIR_REQUEST');
  const url=`https://github.com/${owner}/${repo}/issues/${issueNumber}`;
  try{
    const params={owner,repo,issue_number:issueNumber};
    const before=currentIssue((await octokit.rest.issues.get(params))?.data,url,issueNumber);
    if(!before)return denied('NATIVE_ISSUE_IDENTITY_UNVERIFIED');
    const input={octokit,owner,repo,issueNumber,expectedSourceSha,limit};
    const workItem=await readWorkItemSourceFromClient(input);
    // Errors/partial dependency pages do not get a second chance to authorize.
    if(workItem?.sourceSha!==expectedSourceSha||
       workItem?.issueRef!==url||workItem?.admitted!==false||
       workItem?.authorized!==false||workItem?.taskPass!==false||
       workItem?.claimStatus!=='NOT_VERIFIED')
      return denied('NATIVE_WORK_ITEM_NOT_SOURCE_QUALIFIED');
    const claim=await readClaimEventCandidatesFromClient(input);
    if(!['NO_DISPATCH_CLAIM_EVENTS',
      'CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'].includes(claim?.reason)||
       claim?.claimStatus!=='NOT_VERIFIED'||claim?.authorized!==false||
       claim?.admitted!==false||claim?.taskPass!==false||
       claim?.releasePass!==false||!Array.isArray(claim?.candidates)||
       claim.candidates.length!==claim.candidateCount)
      return denied('NATIVE_CLAIM_EVENTS_NOT_SOURCE_QUALIFIED');
    const after=currentIssue((await octokit.rest.issues.get(params))?.data,url,issueNumber);
    if(!same(before,after))return denied('NATIVE_ISSUE_DRIFT_ACROSS_PAIR');
    // Source projections only. Never pass raw GitHub comment bodies, provider
    // errors, credentials, actor claims or original unbounded objects to K.
    const w=Object.freeze({
      issueRef:url,sourceSha:expectedSourceSha,
      reason:workItem.reason,issueState:workItem.issueState,
      workflowState:workItem.workflowState,workItemType:workItem.workItemType,
      dependencyState:workItem.dependencyState,
      blockerCount:workItem.blockerCount,
      openBlockerCount:workItem.openBlockerCount??null,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
    const candidates=Object.freeze(claim.candidates.map(c=>Object.freeze({
      sourceSha:c.sourceSha,dispatchId:c.dispatchId,
      sourceClass:c.sourceClass,claimAccepted:false
    })));
    const events=Object.freeze({
      reason:claim.reason,candidateCount:candidates.length,candidates,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
    return Object.freeze({
      status:'SOURCE_PAIR_OBSERVED',
      reason:'ACCEPTED_CAS_CLAIM_NOT_VERIFIED',
      issueRef:url,sourceSha:expectedSourceSha,
      sourceQualified:true,workItem:w,claimEvents:events,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }catch(e){return denied(errorClass(e));}
}
export async function readGitHubNativeTaskSourcePair({
  token,owner,repo,issueNumber,expectedSourceSha,limit
}={}){
  if(typeof token!=='string'||token.length<8)
    return denied('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return readNativeTaskSourcePairFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
}
