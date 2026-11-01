/** J07b: a narrow GitHub transport, never a release-authority source.
 * Only the selected privileged Host may call the mutation methods after an
 * independent, current RQ and Owner grant. All mutations read back the
 * canonical GitHub objects; ambiguous outcomes are UNKNOWN, never success.
 */
const SHA=/^[0-9a-f]{40}$/;
const TAG=/^v[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:-[a-z0-9.-]+)?$/;
const OWNER='kaicreator-mm',REPO='tally',REPOSITORY=`${OWNER}/${REPO}`;
const out=(state,reason,more={})=>Object.freeze({state,reason,repository:REPOSITORY,
  authorized:false,taskPass:false,releasePass:false,releasePublished:false,...more});
const deny=(reason)=>out('NOT_ADMITTED',reason);
const uncertain=(reason)=>out('GITHUB_EFFECT_UNKNOWN',reason);
const apiError=(error)=>{
  const status=Number(error?.status??error?.response?.status);
  return status===401?'GITHUB_UNAUTHENTICATED':status===403?'GITHUB_ACCESS_DENIED':
    status===404?'GITHUB_RESOURCE_NOT_FOUND':status===409||status===422?'GITHUB_CONFLICT':
    status===429?'GITHUB_RATE_LIMITED':'GITHUB_TRANSPORT_UNKNOWN';
};
const requestValid=({pullNumber,headSha,headTree,targetSha,targetTree}={})=>
  Number.isSafeInteger(pullNumber)&&pullNumber>0&&
  [headSha,headTree,targetSha,targetTree].every(x=>SHA.test(x??''));
const refSha=async(o,ref)=>(await o.rest.git.getRef({owner:OWNER,repo:REPO,ref:`heads/${ref}`}))?.data?.object?.sha;
const commit=async(o,sha)=>(await o.rest.git.getCommit({owner:OWNER,repo:REPO,commit_sha:sha}))?.data;
const same=(a,b)=>a===b&&SHA.test(a??'');
/** Exact-source preflight: no supplied PASS flags or approval bytes are used. */
export async function readReleaseCandidateFromClient({octokit,pullNumber,headSha,headTree,
  targetSha,targetTree}={}){
  if(!requestValid({pullNumber,headSha,headTree,targetSha,targetTree})||
    !octokit?.rest?.pulls?.get||!octokit?.rest?.git?.getRef||!octokit?.rest?.git?.getCommit)
    return deny('RELEASE_CANDIDATE_INPUT_INVALID');
  try{
    const [pr,headRef,baseRef,headCommit,baseCommit]=await Promise.all([
      octokit.rest.pulls.get({owner:OWNER,repo:REPO,pull_number:pullNumber}),
      refSha(octokit,'version/v2'),refSha(octokit,'main'),
      commit(octokit,headSha),commit(octokit,targetSha)]);
    const p=pr?.data;
    if(!p||p.state!=='open'||p.head?.ref!=='version/v2'||p.base?.ref!=='main'||
      p.head?.repo?.full_name!==REPOSITORY||p.base?.repo?.full_name!==REPOSITORY||
      p.head?.sha!==headSha||p.base?.sha!==targetSha||
      !same(headRef,headSha)||!same(baseRef,targetSha)||
      !same(headCommit?.tree?.sha,headTree)||!same(baseCommit?.tree?.sha,targetTree))
      return deny('EXACT_HEAD_TREE_OR_TARGET_MOVED');
    if(p.mergeable!==true)return deny('MERGEABILITY_NOT_PROVEN');
    return out('EXACT_PREFLIGHT_OBSERVED','REQUIRES_INDEPENDENT_HOST_RQ_AND_OWNER_GRANT',
      {pullNumber,headSha,headTree,targetSha,targetTree,hostGateRequired:true});
  }catch(error){return uncertain(apiError(error));}
}
/** One merge attempt. Host must admit it separately; this function has no RQ API. */
export async function mergeReleaseCandidateFromClient(input={}){
  const preflight=await readReleaseCandidateFromClient(input);
  if(preflight.state!=='EXACT_PREFLIGHT_OBSERVED')return preflight;
  const {octokit,pullNumber,headSha,targetSha}=input;
  if(!octokit?.rest?.pulls?.merge)return deny('GITHUB_MERGE_TRANSPORT_UNAVAILABLE');
  let receipt;
  try{receipt=(await octokit.rest.pulls.merge({owner:OWNER,repo:REPO,pull_number:pullNumber,
    sha:headSha,merge_method:'merge'}))?.data;}
  catch(error){return uncertain(apiError(error));}
  // An acknowledged merge is still not qualified as an actual release.
  if(receipt?.merged!==true||!SHA.test(receipt.sha??''))
    return uncertain('GITHUB_MERGE_RECEIPT_INCOMPLETE');
  try{
    const [newRef,merged]=await Promise.all([refSha(octokit,'main'),commit(octokit,receipt.sha)]);
    if(newRef!==receipt.sha||merged?.sha!==receipt.sha||
      !Array.isArray(merged.parents)||
      ![targetSha,headSha].every(sha=>merged.parents.some(p=>p.sha===sha))||
      !SHA.test(merged.tree?.sha??''))
      return uncertain('MERGE_READBACK_OR_PARENT_MISMATCH');
    return out('MERGED_REQUALIFY_REQUIRED','FINAL_MERGE_TREE_REQUIRES_NEW_INDEPENDENT_RQ',
      {mergedSha:receipt.sha,mergedTree:merged.tree.sha,targetBefore:targetSha,
        headSha,pullNumber,canTag:false,canRelease:false});
  }catch(error){return uncertain(apiError(error));}
}
/** Read-only tag reconciliation: do not infer TAG_ABSENT from a transport error. */
export async function readReleaseTagFromClient({octokit,tag,expectedSha}={}){
  if(!TAG.test(tag??'')||!SHA.test(expectedSha??'')||!octokit?.rest?.git?.getRef)
    return deny('TAG_READBACK_INPUT_INVALID');
  try{
    const current=(await octokit.rest.git.getRef({owner:OWNER,repo:REPO,
      ref:`tags/${tag}`}))?.data;
    return current?.object?.type==='commit'&&current.object.sha===expectedSha?
      out('TAG_READBACK_VERIFIED','TAG_POINTS_TO_QUALIFIED_EXACT_SHA',
        {tag,commitSha:expectedSha}):deny('TAG_CONFLICT_OR_ANNOTATED_TAG_REQUIRES_REVIEW');
  }catch(error){return uncertain(apiError(error));}
}
/** Single tag creation attempt after a *fresh* Host independent RQ (not from UI). */
export async function createQualifiedTagFromClient({octokit,tag,commitSha,commitTree}={}){
  if(!TAG.test(tag??'')||!SHA.test(commitSha??'')||!SHA.test(commitTree??'')||
    !octokit?.rest?.git?.createRef||!octokit?.rest?.git?.getRef||!octokit?.rest?.git?.getCommit)
    return deny('RELEASE_TAG_INPUT_INVALID');
  try{
    const [mainSha,mainCommit]=await Promise.all([refSha(octokit,'main'),commit(octokit,commitSha)]);
    if(mainSha!==commitSha||mainCommit?.tree?.sha!==commitTree)
      return deny('POST_MERGE_TARGET_REQUALIFICATION_REQUIRED');
  }catch(error){return uncertain(apiError(error));}
  try{await octokit.rest.git.createRef({owner:OWNER,repo:REPO,
    ref:`refs/tags/${tag}`,sha:commitSha});}
  catch(error){return uncertain(apiError(error));}
  const back=await readReleaseTagFromClient({octokit,tag,expectedSha:commitSha});
  if(back.state!=='TAG_READBACK_VERIFIED')return uncertain('TAG_CREATE_READBACK_UNKNOWN');
  return out('TAG_CREATED_RELEASE_NOT_PUBLISHED','RELEASE_OBJECT_AND_USER_ACCEPTANCE_SEPARATE',
    {tag,commitSha,commitTree});
}
