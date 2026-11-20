/** User-requested GitHub Issue draft. NOT a qualified Tally Task/Claim.
 * Write scope is explicit and one-shot; no Agent/CAS permission is derived.
 */
const REPOSITORY=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const marker=id=>'tally:draft:'+id;
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
  let posted;
  try{posted=await fetchImpl(base+'/issues',{method:'POST',headers,
    body:JSON.stringify({title:title.trim(),body}),
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
  return Object.freeze({status:'DRAFT_ISSUE_CREATED',issueNumber:issued.number,
    issueUrl:issued.html_url,repository,ref,sourceSha:expectedSha,
    taskQualified:false,agentAuthorized:false});
}
