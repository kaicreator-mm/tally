/** Exact reviewed-candidate PR transport. Review/Join qualification is Host-owned. */
const SHA=/^[0-9a-f]{40}$/,BRANCH=/^tally\/reviewed\/[a-z0-9-]{1,160}$/;
const denied=(reason,more={})=>Object.freeze({state:'NOT_ADMITTED',reason,authorized:false,
  taskPass:false,releasePass:false,...more});
const unknown=reason=>Object.freeze({state:'PUBLICATION_UNKNOWN',reason,authorized:false,
  taskPass:false,releasePass:false});
function errorReason(error){
  const status=Number(error?.status??error?.response?.status);
  if(status===429)return 'GITHUB_RATE_LIMITED';
  if(status===401||status===403||status===404)return 'GITHUB_ACCESS_DENIED';
  if(status===409||status===422)return 'GITHUB_PUBLICATION_CONFLICT';
  return 'GITHUB_PUBLICATION_UNKNOWN';
}
export async function publishCandidatePullRequestFromClient({octokit,owner,repo,branch,baseRef,
  candidateCommit,title,body}={}){
  if(!octokit||owner!=='kaicreator-mm'||repo!=='tally'||!BRANCH.test(branch??'')||
    !['version/v2','main'].includes(baseRef)||!SHA.test(candidateCommit??'')||
    typeof title!=='string'||!title.trim()||title.length>200||typeof body!=='string'||body.length>10000)
    return denied('PUBLICATION_INPUT_INVALID');
  const repository=`${owner}/${repo}`;
  try{
    const ref=await octokit.git.getRef({owner,repo,ref:`heads/${branch}`});
    if(ref.data?.object?.sha!==candidateCommit)return denied('CANDIDATE_REF_MISMATCH');
    const existing=await octokit.pulls.list({owner,repo,state:'all',head:`${owner}:${branch}`,per_page:100});
    const matches=existing.data.filter(pr=>pr.head?.ref===branch);
    if(matches.length>1)return denied('PUBLICATION_IDENTITY_CONFLICT');
    let pr=matches[0]??null;
    if(pr&&pr.state!=='open')return denied('PUBLICATION_CLOSED_OR_MERGED');
    if(pr&&(pr.base?.ref!==baseRef||pr.head?.sha!==candidateCommit))
      return denied('PUBLICATION_IDENTITY_CONFLICT');
    if(!pr){
      const created=await octokit.pulls.create({owner,repo,head:branch,base:baseRef,title,body});
      pr=created.data;
    }
    const readback=await octokit.pulls.get({owner,repo,pull_number:pr.number});
    pr=readback.data;
    const valid=pr?.head?.sha===candidateCommit&&pr.head?.ref===branch&&pr.base?.ref===baseRef&&
      pr.head?.repo?.full_name===repository&&pr.base?.repo?.full_name===repository&&pr.state==='open'&&
      Number.isSafeInteger(pr.number)&&pr.number>0&&typeof pr.html_url==='string';
    if(!valid)return unknown('PUBLICATION_READBACK_MISMATCH');
    return Object.freeze({state:'PUBLISHED_VERIFIED',reason:'GITHUB_BRANCH_AND_PR_READBACK_VERIFIED',
      repository,branch,baseRef,candidateCommit,pullNumber:pr.number,pullRef:pr.html_url,
      externalRef:pr.html_url,authorized:false,taskPass:false,releasePass:false});
  }catch(error){return unknown(errorReason(error));}
}
