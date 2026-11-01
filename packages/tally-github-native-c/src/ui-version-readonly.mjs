/**
 * Tally v2 UI native source C — pinned branch and explicit Task roster.
 *
 * Uses existing native C WorkItem/dependency reader. No GitHub writes, no
 * CAS Claim acceptance, no fake issue→version assignment or implied PASS.
 * A Host-approved version roster is REQUIRED; "all repo issues" is NOT a
 * safe substitute. Incomplete or drifting reads fail closed.
 */

const SHA=/^[0-9a-f]{40}$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const denied=(reason)=>Object.freeze({connected:false,reason});
const taskStatus=state=>({
  'state:planned':'open','state:ready':'open','state:claimed':'open',
  'state:implementing':'in_progress','state:review-ready':'review',
  'state:reviewing':'review','state:validation-needed':'review',
  'state:validating':'review','state:changes-requested':'blocked',
  'state:merge-ready':'review','state:blocked':'blocked',
  'state:done':'done','state:superseded':'done','state:cancelled':'done',
})[state]??'unknown';
const labelNames=raw=>Array.isArray(raw)?
  raw.map(label=>typeof label==='string'?label:label?.name):null;
const allowedRef=(ref,owner,repo,n)=>
  ref===`https://github.com/${owner}/${repo}/issues/${n}`;
const correctIssue=(issue,owner,repo,n)=>{
  const labels=labelNames(issue?.labels);
  return issue?.number===n&&Number.isSafeInteger(issue.id)&&issue.id>0&&
    allowedRef(issue.html_url,owner,repo,n)&&
    ['open','closed'].includes(issue.state)&&
    typeof issue.title==='string'&&issue.title.length>0&&
    issue.title.length<=512&&typeof issue.updated_at==='string'&&
    !issue.pull_request&&labels&&!labels.some(l=>typeof l!=='string');
};
const source=(owner,repo,expectedSha)=>({
  repository:`${owner}/${repo}`,sourceSha:expectedSha,
  currentness:'CURRENT',readback:'VERIFIED',
});
async function verifyBranch(octokit,owner,repo,versionId,expectedSha){
  const [b,c]=await Promise.all([
    octokit.rest.repos.getBranch({owner,repo,branch:versionId}),
    octokit.rest.repos.getCommit({owner,repo,ref:expectedSha}),
  ]);
  return b?.data?.commit?.sha===expectedSha&&c?.data?.sha===expectedSha;
}
const validInput=(octokit,owner,repo,versionId,expectedSha)=>
  NAME.test(owner??'')&&NAME.test(repo??'')&&VERSION.test(versionId??'')&&
  SHA.test(expectedSha??'')&&!!octokit?.rest?.repos?.getBranch&&
  !!octokit?.rest?.repos?.getCommit;
export async function readUiNativeCatalogFromClient({
  octokit,owner,repo,versionId,expectedSha
}={}){
  if(!validInput(octokit,owner,repo,versionId,expectedSha))
    return denied('INVALID_SELECTED_GITHUB_SOURCE');
  try{
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_NOT_PINNED');
    const projectId=`${owner}/${repo}`;
    return Object.freeze({connected:true,source:source(owner,repo,expectedSha),
      projects:[{id:projectId,title:repo,repository:projectId}],
      versionsByProject:{[projectId]:[{id:versionId,projectId,label:versionId,
        ref:versionId}]},defaultVersionId:versionId});
  }catch{return denied('GITHUB_NATIVE_SOURCE_UNAVAILABLE');}
}
export async function readUiNativeVersionFromClient({
  octokit,owner,repo,versionId,expectedSha,taskIssueNumbers,
  workItemReader=null
}={}){
  if(!validInput(octokit,owner,repo,versionId,expectedSha)||
     !octokit?.rest?.issues?.get||!octokit?.paginate?.iterator||
     !Array.isArray(taskIssueNumbers)||taskIssueNumbers.length<1||
     taskIssueNumbers.length>200||taskIssueNumbers.some(n=>
       !Number.isSafeInteger(n)||n<1)||
     new Set(taskIssueNumbers).size!==taskIssueNumbers.length)
    return denied('VERSION_TASK_ROSTER_NOT_CONFIGURED');
  try{
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_NOT_PINNED');
    const roster=new Set(taskIssueNumbers);
    const tasks=[],nodes=[],edges=[],seenEdges=new Set();
    for(const number of taskIssueNumbers){
      const args={owner,repo,issue_number:number};
      const issue=(await octokit.rest.issues.get(args))?.data;
      if(!correctIssue(issue,owner,repo,number))
        return denied('NATIVE_ISSUE_IDENTITY_UNVERIFIED');
      // The existing C adapter rechecks Issue drift and all native
      // dependency pages. It intentionally NEVER accepts a Claim.
      const selectedReader=typeof workItemReader==='function'?workItemReader:
        (await import('./task-workitem-source.mjs')).readWorkItemSourceFromClient;
      const observed=await selectedReader({
        octokit,owner,repo,issueNumber:number,expectedSourceSha:expectedSha
      });
      if(observed?.sourceSha!==expectedSha||
         observed?.workItemType!=='type:task'||
         observed?.claimStatus!=='NOT_VERIFIED'||
         observed?.authorized!==false||observed?.taskPass!==false||
         !['CLEAR','BLOCKED'].includes(observed?.dependencyState)||
         !Number.isSafeInteger(observed.blockerCount)||
         observed.workflowState===null||
         observed.issueState!==issue.state)
        return denied('WORKITEM_NATIVE_SOURCE_UNQUALIFIED');
      const labels=labelNames(issue.labels);
      const stateLabels=labels.filter(l=>l.startsWith('state:'));
      const typeLabels=labels.filter(l=>l.startsWith('type:'));
      if(typeLabels.length!==1||typeLabels[0]!=='type:task'||
         stateLabels.length!==1||stateLabels[0]!==observed.workflowState)
        return denied('NATIVE_ISSUE_STATE_DRIFT');
      const priority=labels.filter(l=>/^(?:priority:)?P[012]$/.test(l))
        .map(l=>l.slice(-2));
      const severity=priority.length===1?priority[0]:'UNKNOWN';
      const dependencies=[],uniqueDeps=new Set();
      const route='GET /repos/{owner}/{repo}/issues/{issue_number}/dependencies/blocked_by';
      for await(const page of octokit.paginate.iterator(route,{...args,per_page:100})){
        if(!Array.isArray(page?.data))return denied('NATIVE_DEPENDENCY_PAGE_INVALID');
        for(const dependency of page.data){
          if(!Number.isSafeInteger(dependency?.number)||dependency.number<1||
             !allowedRef(dependency.html_url,owner,repo,dependency.number)||
             !roster.has(dependency.number)||uniqueDeps.has(dependency.number)||
             !['open','closed'].includes(dependency.state))
            return denied('NATIVE_DEPENDENCY_OUTSIDE_PINNED_ROSTER');
          uniqueDeps.add(dependency.number);
          dependencies.push(String(dependency.number));
          const edgeId=`${dependency.number}->${number}`;
          if(!seenEdges.has(edgeId)){
            seenEdges.add(edgeId);
            edges.push({from:String(dependency.number),to:String(number),
              kind:'NATIVE',evidenceRef:dependency.html_url,amendedAt:null});
          }
        }
      }
      if(dependencies.length!==observed.blockerCount)
        return denied('NATIVE_DEPENDENCY_COUNT_DRIFT');
      const status=taskStatus(observed.workflowState);
      const entry={id:String(number),versionId,title:issue.title,
        status,severity,blockedBy:dependencies,
        agentId:null,claimState:'NOT_VERIFIED',attemptId:null,
        attemptState:null,prNumber:null,reviewState:'UNKNOWN',
        ciState:'UNKNOWN',writeSet:[],requirementRef:null,
        updatedAt:issue.updated_at};
      tasks.push(entry);
      nodes.push({taskId:entry.id,versionId,title:entry.title,status,severity,
        requirementGroup:null,phase:null,lane:null,agentId:null,
        executable:false});
    }
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_DRIFT');
    const prov=source(owner,repo,expectedSha);
    return Object.freeze({connected:true,source:prov,versionId,
      tasks,dag:{connected:true,source:prov,versionId,
        asOf:{sourceSha:expectedSha,frozenAt:null},nodes,edges}});
  }catch{return denied('NATIVE_TASK_READ_FAILED');}
}
/** Production creates the Octokit transport INSIDE C, not the browser. */
export function createUiNativeSource({token,owner,repo,versionId,
  expectedSha,taskIssueNumbers}={}){
  if(typeof token!=='string'||token.length<8)
    throw new TypeError('TRUSTED_GITHUB_C_CREDENTIAL_REQUIRED');
  // Native production adapter imports Octokit only on actual Host use.
  // Source tests need no provider/credential packages or external effects.
  const getClient=async()=>{
    const [{Octokit},{createNativeReadRetryFetch}]=await Promise.all([
      import('@octokit/rest'),import('./read-retry.mjs')]);
    return new Octokit({auth:token,
      request:{timeout:15000,fetch:createNativeReadRetryFetch()}});
  };
  let clientPromise;
  const client=()=>clientPromise??=getClient();
  return Object.freeze({
    catalog:async()=>readUiNativeCatalogFromClient({
      octokit:await client(),owner,repo,versionId,expectedSha}),
    version:async()=>readUiNativeVersionFromClient({
      octokit:await client(),owner,repo,versionId,expectedSha,taskIssueNumbers}),
  });
}
