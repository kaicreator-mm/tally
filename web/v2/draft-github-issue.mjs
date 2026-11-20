/** User-requested GitHub Issue draft. NOT a qualified Tally Task/Claim.
 * Write scope is explicit and one-shot; no Agent/CAS permission is derived.
 */
const REPOSITORY=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const marker=id=>'tally:draft:'+id;
const NATIVE_TASK_LABELS=Object.freeze([
  {name:'type:task',color:'1f6feb',description:'GitHub-native Task item'},
  {name:'state:planned',color:'afb8c1',description:'Planned Task, not ready or claimed'}
]);
const labelNames=NATIVE_TASK_LABELS.map(item=>item.name);
const labelObserved=(labels)=>Array.isArray(labels)&&
  labels.map(l=>typeof l==='string'?l:l?.name).filter(x=>typeof x==='string')
    .filter(x=>x.startsWith('type:')||x.startsWith('state:'))
    .sort().join('|')==='state:planned|type:task';
async function ensureNativeTaskLabels(base,headers,fetchImpl){
  for(const label of NATIVE_TASK_LABELS){
    const path=base+'/labels/'+encodeURIComponent(label.name);
    let found;
    try{found=await fetchImpl(path,{headers,signal:AbortSignal.timeout(15000)});}
    catch{throw new Error('TASK_LABEL_READ_UNKNOWN');}
    if(found.status===401||found.status===403)
      throw new Error('GITHUB_ISSUES_WRITE_PERMISSION_REQUIRED');
    if(found.status===404){
      let created;
      try{created=await fetchImpl(base+'/labels',{method:'POST',headers,
        body:JSON.stringify(label),signal:AbortSignal.timeout(15000)});}
      catch{throw new Error('TASK_LABEL_CREATE_UNKNOWN');}
      if(created.status===401||created.status===403)
        throw new Error('GITHUB_ISSUES_WRITE_PERMISSION_REQUIRED');
      if(!created.ok&&created.status!==422)
        throw new Error('TASK_LABEL_CREATE_FAILED');
      try{found=await fetchImpl(path,{headers,signal:AbortSignal.timeout(15000)});}
      catch{throw new Error('TASK_LABEL_READ_UNKNOWN');}
    }
    if(!found.ok)throw new Error('TASK_LABEL_NOT_VERIFIED');
    let data;
    try{data=await found.json();}catch{throw new Error('TASK_LABEL_NOT_VERIFIED');}
    if(data?.name?.toLowerCase()!==label.name)
      throw new Error('TASK_LABEL_NOT_VERIFIED');
  }
}

export async function createGitHubTaskDraft({
  repository,ref,expectedSha,title,description='',writeToken,requestId,
  fetchImpl=fetch
}={}){
  if(!REPOSITORY.test(repository??'')||typeof ref!=='string'||
      !SHA.test(expectedSha??'')||!/^[-a-zA-Z0-9]{8,80}$/.test(requestId??'')||
      typeof title!=='string'||title.trim().length<3||
      title.trim().length>180||/[\r\n\x00-\x1f]/.test(title)||
      typeof description!=='string'||description.length>8000||
      typeof writeToken!=='string'||writeToken.length<8)
    throw new TypeError('INVALID_DRAFT_TASK_INPUT');
  const [owner,name]=repository.split('/');
  const base='https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+
    encodeURIComponent(name);
  const body=['Tally development task draft — not yet admitted to Task DAG.',
    'Repository: '+repository,'Selected branch: '+ref,
    'Observed source SHA: '+expectedSha,
    'Request: '+marker(requestId),'',
    description.trim()||'(No description supplied)','',
    'Status: UNQUALIFIED_DRAFT. No Agent, Claim or Release authority.'].join('\n');
  const headers={authorization:'Bearer '+writeToken,
    accept:'application/vnd.github+json',
    'content-type':'application/json','user-agent':'tally-v2-task-draft'};
  await ensureNativeTaskLabels(base,headers,fetchImpl);
  let posted;
  try{posted=await fetchImpl(base+'/issues',{method:'POST',headers,
    body:JSON.stringify({title:title.trim(),body,labels:labelNames}),
    signal:AbortSignal.timeout(15000)});}
  catch{throw new Error('DRAFT_ISSUE_CREATION_UNKNOWN');}
  if(posted.status===401||posted.status===403)
    throw new Error('GITHUB_ISSUES_WRITE_PERMISSION_REQUIRED');
  if(!posted.ok)throw new Error('DRAFT_ISSUE_CREATION_FAILED');
  let issued;
  try{issued=await posted.json();}catch{throw new Error('DRAFT_ISSUE_READBACK_UNKNOWN');}
  if(!Number.isSafeInteger(issued?.number)||issued.number<1||
      issued.html_url!=='https://github.com/'+repository+'/issues/'+issued.number)
    throw new Error('DRAFT_ISSUE_READBACK_UNKNOWN');
  let observed;
  try{observed=await fetchImpl(base+'/issues/'+issued.number,{
    headers:{authorization:headers.authorization,accept:headers.accept,
      'user-agent':headers['user-agent']},
    signal:AbortSignal.timeout(15000)});}
  catch{throw new Error('DRAFT_ISSUE_READBACK_UNKNOWN');}
  if(!observed.ok)throw new Error('DRAFT_ISSUE_READBACK_UNKNOWN');
  let data;
  try{data=await observed.json();}catch{throw new Error('DRAFT_ISSUE_READBACK_UNKNOWN');}
  if(data?.number!==issued.number||data.html_url!==issued.html_url||
      data.title!==title.trim()||!data.body?.includes(marker(requestId))||
      data.pull_request)
    throw new Error('DRAFT_ISSUE_READBACK_MISMATCH');
  // GitHub may silently drop labels despite a successful Issue POST.
  // A created but unlabelled Issue remains a visible repair candidate,
  // not an invisible error and never a claimable Task.
  const canonical=labelObserved(data.labels);
  return Object.freeze({status:canonical?'DRAFT_ISSUE_CREATED':
    'DRAFT_ISSUE_REQUIRES_LABEL_REPAIR',issueNumber:issued.number,
    issueUrl:issued.html_url,repository,ref,sourceSha:expectedSha,
    nativeWorkflow:canonical?'PLANNED':'NONCANONICAL',
    taskQualified:false,agentAuthorized:false});
}
