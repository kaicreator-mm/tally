/**
 * Tally v2 T06a: an actual GitHub-native Work Item/dependency observation.
 *
 * ADS v4.18 §12.1 accepted CAS DISPATCH_CLAIMED is NOT represented by a
 * state:claimed label, Issue body, transport login or a comment. This reader
 * intentionally cannot accept a Claim or admit an execution. Only a later
 * independent accepted-Claim verifier + Engineering B + selected K can do so.
 *
 * Trusted Octokit is a transport C. GitHub controls Issue/dependency truth.
 * No GitHub writes, no DB, no process effect, no task-meta or independent K.
 */
import { Octokit } from '@octokit/rest';
import {createNativeReadRetryFetch} from './read-retry.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
// ADS v4.18 GITHUB_WORK_ITEM_CONTRACT_STANDARD §5 exact canonical vocabulary.
// Recognition is descriptive only; neither READY nor CLAIMED establishes CAS.
const STATES=new Set(['state:planned','state:ready','state:claimed',
  'state:implementing','state:review-ready','state:reviewing',
  'state:validation-needed','state:validating',
  'state:changes-requested','state:merge-ready',
  'state:blocked','state:done','state:superseded','state:cancelled']);
const TYPES=new Set(['type:task','type:planning']);
const CAP=1000;
const fail=(reason,options={})=>Object.freeze({
  status:'UNKNOWN',reason,dependencyState:'UNKNOWN',blockerCount:null,
  issueState:null,workflowState:null,workItemType:null,sourceSha:null,
  claimStatus:'NOT_VERIFIED',admitted:false,authorized:false,
  taskPass:false,releasePass:false,...options
});
const providerReason=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_RESOURCE_NOT_FOUND':
  e?.status===429?'GITHUB_RATE_LIMITED':
  'NATIVE_GITHUB_SOURCE_READ_FAILED';
const extract=(data,repository,issueNumber)=>{
  if(!data||data.number!==issueNumber||data.html_url!==
    `https://github.com/${repository}/issues/${issueNumber}`||
    (data.state!=='open'&&data.state!=='closed')||
    typeof data.updated_at!=='string'||!Array.isArray(data.labels)||
    data.pull_request!==undefined)return null;
  const labels=data.labels.map(l=>typeof l==='string'?l:l?.name);
  if(labels.some(l=>typeof l!=='string'))return null;
  return {id:data.id,updatedAt:data.updated_at,issueState:data.state,
    stateLabels:labels.filter(l=>l.startsWith('state:')),
    typeLabels:labels.filter(l=>l.startsWith('type:'))};
};
/**
 * Octokit seam is injected ONLY for targeted tests. The public entrypoint
 * creates its authenticated Octokit internally.
 *
 * Reads are not an atomic cross-endpoint snapshot. Rechecking issue.updated_at
 * is a bounded drift sentinel; it cannot certify linearizable CAS admission.
 */
export async function readWorkItemSourceFromClient({
  octokit,owner,repo,issueNumber,expectedSourceSha,limit=CAP
}={}) {
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
    !Number.isSafeInteger(issueNumber)||issueNumber<1||
    !SHA.test(expectedSourceSha??'')||
    !Number.isSafeInteger(limit)||limit<1||limit>CAP||
    !octokit?.rest?.issues?.get||!octokit?.rest?.repos?.getCommit||
    !octokit?.paginate?.iterator)return fail('INVALID_NATIVE_QUERY');
  const repository=`${owner}/${repo}`;
  const params={owner,repo,issue_number:issueNumber};
  let first;
  try{
    const [initial,commit]=await Promise.all([
      octokit.rest.issues.get(params),
      octokit.rest.repos.getCommit({owner,repo,ref:expectedSourceSha})
    ]);
    first=extract(initial?.data,repository,issueNumber);
    if(!first||commit?.data?.sha!==expectedSourceSha)
      return fail('ISSUE_OR_SOURCE_SHA_UNVERIFIED');
    // Real GitHub Issue Dependencies; NEVER substitute a Markdown checklist.
    const seen=new Set();
    let blockers=0,open=0,observed=0;
    const route='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by';
    for await(const page of octokit.paginate.iterator(route,{...params,per_page:100})){
      if(!Array.isArray(page?.data))return fail('DEPENDENCY_PAGE_INVALID');
      for(const dependency of page.data){
        observed++;
        if(observed>limit)return fail('DEPENDENCY_PAGINATION_LIMIT');
        if(!Number.isSafeInteger(dependency?.id)||dependency.id<1||
          !Number.isSafeInteger(dependency?.number)||dependency.number<1||
          typeof dependency.html_url!=='string'||
          !/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/.test(dependency.html_url)||
          !dependency.html_url.endsWith(`/issues/${dependency.number}`)||
          (dependency.state!=='closed'&&dependency.state!=='open')||
          seen.has(dependency.id))return fail('DEPENDENCY_IDENTITY_UNVERIFIED');
        seen.add(dependency.id);
        blockers++;
        if(dependency.state!=='closed')open++;
      }
    }
    const final=(await octokit.rest.issues.get(params))?.data;
    const second=extract(final,repository,issueNumber);
    if(!second||first.id!==second.id||
      first.updatedAt!==second.updatedAt||
      first.issueState!==second.issueState||
      JSON.stringify(first.stateLabels)!==JSON.stringify(second.stateLabels)||
      JSON.stringify(first.typeLabels)!==JSON.stringify(second.typeLabels))
      return fail('ISSUE_DRIFT_DURING_OBSERVATION');
    const dependencyState=open>0?'BLOCKED':'CLEAR';
    const base={
      status:'UNKNOWN',reason:'CLAIM_NOT_VERIFIED',
      dependencyState,blockerCount:blockers,openBlockerCount:open,
      issueState:first.issueState,
      workflowState:null,workItemType:null,
      sourceSha:expectedSourceSha,
      issueRef:`https://github.com/${repository}/issues/${issueNumber}`,
      claimStatus:'NOT_VERIFIED',
      authorized:false,admitted:false,taskPass:false,releasePass:false
    };
    if(first.stateLabels.length!==1||
      !STATES.has(first.stateLabels[0])||
      first.typeLabels.length!==1||!TYPES.has(first.typeLabels[0]))
      return Object.freeze({...base,reason:'CANONICAL_WORK_ITEM_STATE_UNVERIFIED'});
    base.workflowState=first.stateLabels[0];
    base.workItemType=first.typeLabels[0];
    if(open>0)return Object.freeze({...base,reason:'NATIVE_ISSUE_DEPENDENCIES_BLOCKED'});
    if(first.issueState!=='open'||first.stateLabels[0]!=='state:ready')
      return Object.freeze({...base,reason:'WORK_ITEM_NOT_READY'});
    // Native ready is ONLY a prerequisite; even the strongest Source facts
    // here do not prove an accepted CAS Claim or Host/K dispatch permission.
    return Object.freeze(base);
  }catch(e){return fail(providerReason(e));}
}

export async function readGitHubWorkItemSource({
  token,owner,repo,issueNumber,expectedSourceSha,limit
}={}) {
  if(typeof token!=='string'||token.length<8)
    return fail('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000,fetch:createNativeReadRetryFetch()}});
  return readWorkItemSourceFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
}
