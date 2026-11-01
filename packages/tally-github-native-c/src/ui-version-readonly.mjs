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
// ADS workflow labels are descriptive, not Closure evidence. A closed Issue
// with state:ready (or a state:done label alone) must not render Task DONE.
const taskStatus=(state,issueState)=>{
  if(issueState!=='open'||['state:done','state:superseded','state:cancelled']
    .includes(state))return 'unknown';
  return ({
    'state:planned':'open','state:ready':'open','state:claimed':'open',
    'state:implementing':'in_progress','state:review-ready':'review',
    'state:reviewing':'review','state:validation-needed':'review',
    'state:validating':'review','state:changes-requested':'blocked',
    'state:merge-ready':'review','state:blocked':'blocked',
  })[state]??'unknown';
};
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
      let openDeps=0;
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
          if(dependency.state==='open')openDeps++;
          const edgeId=`${dependency.number}->${number}`;
          if(!seenEdges.has(edgeId)){
            seenEdges.add(edgeId);
            edges.push({from:String(dependency.number),to:String(number),
              kind:'NATIVE',evidenceRef:dependency.html_url,amendedAt:null});
          }
        }
      }
      if(dependencies.length!==observed.blockerCount||
         openDeps!==observed.openBlockerCount||
         (openDeps>0)!==(observed.dependencyState==='BLOCKED'))
        return denied('NATIVE_DEPENDENCY_COUNT_DRIFT');
      const status=taskStatus(observed.workflowState,observed.issueState);
      const entry={id:String(number),versionId,title:issue.title,
        status,severity,blockedBy:dependencies,
        dependencyState:observed.dependencyState,
        openBlockerCount:observed.openBlockerCount,
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
    // A Host-selected list is a scope declaration, not an independently
    // verified COMPLETE frozen DAG. This must be visible in the UI.
    const rosterCoverage='UNVERIFIED';
    return Object.freeze({connected:true,source:prov,versionId,
      rosterCoverage,tasks,dag:{connected:true,source:prov,versionId,
        rosterCoverage,
        asOf:{sourceSha:expectedSha,frozenAt:null},nodes,edges}});
  }catch{return denied('NATIVE_TASK_READ_FAILED');}
}
/** Read selected, immutable-at-query GitHub Requirement titles using the
 * existing GitHub C project-intake transport. Kind and hierarchy are
 * explicitly Host-declared, NOT an independently frozen Product structure.
 */
export async function readUiNativeRequirementsFromClient({
  octokit,owner,repo,versionId,expectedSha,requirementIssueSpecs,
  intakeReader=null
}={}){
  if(!validInput(octokit,owner,repo,versionId,expectedSha)||
     !Array.isArray(requirementIssueSpecs)||
     requirementIssueSpecs.length<1||requirementIssueSpecs.length>16||
     !octokit?.rest?.issues?.get)return denied('REQUIREMENT_ROSTER_NOT_CONFIGURED');
  const ids=new Set();
  for(const item of requirementIssueSpecs){
    if(!item||!Number.isSafeInteger(item.number)||item.number<1||
       ids.has(item.number)||
       !['requirement','epic','story'].includes(item.kind)||
       (item.parent!==null&&(!Number.isSafeInteger(item.parent)||item.parent<1)))
      return denied('INVALID_HOST_REQUIREMENT_ROSTER');
    ids.add(item.number);
  }
  for(const item of requirementIssueSpecs){
    if(item.parent!==null&&(!ids.has(item.parent)||item.parent===item.number))
      return denied('REQUIREMENT_PARENT_OUTSIDE_ROSTER');
    const ancestors=new Set([item.number]);
    let parent=item.parent;
    while(parent!==null){
      if(ancestors.has(parent))return denied('REQUIREMENT_CYCLE');
      ancestors.add(parent);
      parent=requirementIssueSpecs.find(x=>x.number===parent)?.parent??null;
    }
  }
  try{
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_NOT_PINNED');
    const read=intakeReader??(await import(
      '../../tally-github-c/src/project-intake.mjs')).readProjectIntakeFromClient;
    const observed=await read({octokit,owner,repo,expectedSourceSha:expectedSha,
      sources:requirementIssueSpecs.map(x=>({number:x.number}))});
    if(observed?.status!=='PROJECT_INTAKE_OBSERVED'||
       observed.sourceQualified!==true||
       observed.authorized!==false||observed.taskPass!==false||
       observed.releasePass!==false||
       observed.repository?.fullName!==owner+'/'+repo||
       typeof observed.observedAt!=='string'||
       !Array.isArray(observed.sources)||
       observed.sources.length!==requirementIssueSpecs.length)
      return denied('NATIVE_REQUIREMENT_SOURCE_UNQUALIFIED');
    const nodes=[];
    for(let i=0;i<requirementIssueSpecs.length;i++){
      const spec=requirementIssueSpecs[i],actual=observed.sources[i];
      if(actual?.number!==spec.number||actual.isPullRequest!==false||
         actual.url!==`https://github.com/${owner}/${repo}/issues/${spec.number}`||
         !['open','closed'].includes(actual.state)||
         typeof actual.title!=='string'||!actual.title.trim())
        return denied('REQUIREMENT_ISSUE_IDENTITY_UNVERIFIED');
      // Double read all selected Issue facts to reject mutation during
      // source observation. This does not make a Product freeze attestation.
      const last=(await octokit.rest.issues.get({owner,repo,
        issue_number:spec.number}))?.data;
      if(last?.number!==spec.number||last?.html_url!==actual.url||
         last?.pull_request!==undefined||last?.state!==actual.state||
         last?.title!==actual.title)
        return denied('REQUIREMENT_ISSUE_CHANGED_DURING_READ');
      nodes.push({id:String(spec.number),kind:spec.kind,title:actual.title,
        parent:spec.parent===null?null:String(spec.parent),
        productRevision:null,acceptanceRefs:[],l2Decision:null,
        freezeState:'UNKNOWN',updatedAt:observed.observedAt});
    }
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_DRIFT');
    return Object.freeze({connected:true,source:source(owner,repo,expectedSha),
      versionId,structureCoverage:'HOST_DECLARED_UNVERIFIED',
      freezeCoverage:'UNKNOWN',nodes});
  }catch{return denied('NATIVE_REQUIREMENT_READ_FAILED');}
}
/** Selected GitHub commit-status observation for one Host-qualified
 * Full regression CI context. Status writers are NOT independent QA witnesses.
 */
export async function readUiNativeQaFromClient({
  octokit,owner,repo,versionId,expectedSha,qaStatusContext
}={}){
  if(!validInput(octokit,owner,repo,versionId,expectedSha)||
     typeof qaStatusContext!=='string'||
     !/^ci\/woodpecker\/(?:pr|push)\/[A-Za-z0-9_.-]{1,80}$/.test(qaStatusContext)||
     typeof octokit?.rest?.repos?.getCombinedStatusForRef!=='function')
    return denied('QA_STATUS_SOURCE_NOT_CONFIGURED');
  try{
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_NOT_PINNED');
    const result=(await octokit.rest.repos.getCombinedStatusForRef({
      owner,repo,ref:expectedSha,per_page:100}))?.data;
    if(result?.sha!==expectedSha||!Number.isSafeInteger(result.total_count)||
       result.total_count<0||result.total_count>100||
       !Array.isArray(result.statuses)||
       result.statuses.length!==result.total_count)
      return denied('GITHUB_CI_STATUS_PAGE_UNQUALIFIED');
    const matches=result.statuses.filter(item=>item?.context===qaStatusContext);
    const records=[];
    if(matches.length){
      const selected=matches[0];
      if(!['success','failure','error','pending'].includes(selected.state)||
         typeof selected.target_url!=='string'||
         !/^https:\/\/ci\.kaicreator\.com\/repos\/[0-9]+\/pipeline\/[0-9]+\/[0-9]+$/
           .test(selected.target_url)||
         typeof selected.updated_at!=='string'||
         !Number.isFinite(Date.parse(selected.updated_at)))
        return denied('GITHUB_CI_STATUS_IDENTITY_UNQUALIFIED');
      records.push({layer:'fullRegression',provenance:'REPORTED',
        outcome:selected.state==='success'?'PASS':
          selected.state==='failure'||selected.state==='error'?'FAIL':'UNKNOWN',
        witness:'github-native-c-status-reported',
        evidenceRef:selected.target_url,subjectSha:expectedSha,
        observedAt:selected.updated_at});
    }
    if(!await verifyBranch(octokit,owner,repo,versionId,expectedSha))
      return denied('VERSION_BRANCH_DRIFT');
    return Object.freeze({connected:true,source:source(owner,repo,expectedSha),
      versionId,ciCoverage:matches.length?'REPORTED_ONLY':'NO_MATCHING_STATUS',
      independentCoverage:'UNKNOWN',records});
  }catch{return denied('GITHUB_CI_STATUS_READ_FAILED');}
}
/** Production creates the Octokit transport INSIDE C, not the browser. */
export function createUiNativeSource({token,owner,repo,versionId,
  expectedSha,taskIssueNumbers,requirementIssueSpecs,qaStatusContext}={}){
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
    requirements:async()=>readUiNativeRequirementsFromClient({
      octokit:await client(),owner,repo,versionId,expectedSha,requirementIssueSpecs}),
    qa:async()=>readUiNativeQaFromClient({
      octokit:await client(),owner,repo,versionId,expectedSha,qaStatusContext}),
  });
}
