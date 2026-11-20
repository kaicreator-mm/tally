/** Bounded GitHub draft Issue readback, never a frozen Task/DAG read. */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const denied=reason=>({connected:false,reason});
export async function readGitHubTaskDrafts({
  repository,versionId,sourceSha,token,fetchImpl=fetch
}={}){
  if(!REPO.test(repository??'')||typeof versionId!=='string'||
     !SHA.test(sourceSha??'')||typeof token!=='string'||token.length<8)
    return denied('DRAFT_READER_SOURCE_INVALID');
  const [owner,name]=repository.split('/');
  const url='https://api.github.com/repos/'+encodeURIComponent(owner)+'/'+
    encodeURIComponent(name)+'/issues?state=all&per_page=100&sort=created&direction=desc';
  let response;
  try{response=await fetchImpl(url,{headers:{
    authorization:'Bearer '+token,accept:'application/vnd.github+json',
    'user-agent':'tally-v2-task-drafts'},signal:AbortSignal.timeout(15000)});}
  catch{return denied('GITHUB_DRAFT_READ_UNAVAILABLE');}
  if(response.status===401||response.status===403)
    return denied('GITHUB_ISSUES_READ_PERMISSION_REQUIRED');
  if(!response.ok)return denied('GITHUB_DRAFT_READ_UNAVAILABLE');
  let rows;
  try{rows=await response.json();}catch{return denied('GITHUB_DRAFT_READ_INVALID');}
  if(!Array.isArray(rows)||rows.length>100)
    return denied('GITHUB_DRAFT_READ_INVALID');
  const tasks=[];
  for(const row of rows){
    const n=row?.number;
    if(!Number.isSafeInteger(n)||n<1||row.pull_request)continue;
    if(row.html_url!==`https://github.com/${repository}/issues/${n}`)continue;
    if(!['open','closed'].includes(row.state)||
       typeof row.title!=='string'||typeof row.body!=='string')continue;
    const body=row.body;
    if(!body.startsWith('Tally development task draft — not yet admitted to Task DAG.\n')||
       !body.includes('\nRepository: '+repository+'\n')||
       !body.includes('\nSelected branch: '+versionId+'\n')||
       !/\nRequest: tally:draft:[a-zA-Z0-9-]{8,80}\n/.test(body)||
       !body.includes('\nStatus: UNQUALIFIED_DRAFT. No Agent, Claim or Release authority.'))
      continue;
    const observed=/\nObserved source SHA: ([a-f0-9]{40})\n/.exec(body)?.[1];
    if(!observed)continue;
    tasks.push({issueNumber:n,title:row.title,issueUrl:row.html_url,
      githubIssueState:row.state,sourceSha:observed,
      taskQualified:false,agentAuthorized:false});
  }
  return Object.freeze({connected:true,repository,versionId,sourceSha,
    coverage:rows.length===100?'LATEST_100_ISSUES_ONLY':'COMPLETE_SINGLE_PAGE',
    drafts:tasks});
}
