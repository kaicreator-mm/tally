/** Host-only reviewed-candidate publication. No Builder credentials or caller authority. */
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
const execFileAsync=promisify(execFile);
const SHA=/^[0-9a-f]{40}$/,ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const JOIN_PREFIX='tally:v2:reviewer-join:',OWNER='kaicreator-mm',REPO='tally';
const denied=reason=>Object.freeze({state:'NOT_ADMITTED',reason,authorized:false,
  taskPass:false,releasePass:false});
const unknown=reason=>Object.freeze({state:'PUBLICATION_UNKNOWN',reason,authorized:false,
  taskPass:false,releasePass:false});
const branchFor=attemptId=>'tally/reviewed/'+attemptId.replace(/[^a-z0-9]+/gi,'-').replace(/^-|-$/g,'').toLowerCase();
function latestReviewed(rows,sourceSha,attemptId,issueRef){
  const entries=[];
  for(const row of rows){
    const value=row?.value;
    if(value?.phase!=='JOINED'||value.sourceSha!==sourceSha||!Array.isArray(value.candidates))continue;
    for(const candidate of value.candidates)entries.push({candidate,joinedAt:value.joinedAt,revision:row.revision??0});
  }
  const selected=entries.find(x=>x.candidate.attemptId===attemptId);
  if(!selected||selected.candidate.issueRef!==issueRef)return {reason:'CANDIDATE_REVIEW_NOT_FOUND'};
  const later=entries.some(x=>x.candidate.issueRef===issueRef&&x.candidate.attemptId!==attemptId&&
    (Date.parse(x.joinedAt)>Date.parse(selected.joinedAt)||
      (x.joinedAt===selected.joinedAt&&x.revision>selected.revision)));
  if(later)return {reason:'CANDIDATE_SUPERSEDED_BY_REPAIR'};
  return {candidate:selected.candidate};
}
export function createCandidatePublisher({openStore,publish,exec=execFileAsync}={}){
  return Object.freeze({async publishReviewedCandidate(config){
    const {issueRef,sourceSha,attemptId,hostStorePath,repositoryPath,githubToken}=config??{};
    if(issueRef!==`https://github.com/${OWNER}/${REPO}/issues/197`&&
      issueRef!==`https://github.com/${OWNER}/${REPO}/issues/198`)
      return denied('PUBLICATION_TASK_NOT_ALLOWED');
    if(!SHA.test(sourceSha??'')||!ATTEMPT.test(attemptId??'')||!hostStorePath||!repositoryPath)
      return denied('PUBLICATION_INPUT_INVALID');
    if(typeof openStore!=='function'||typeof publish!=='function'||typeof githubToken!=='string'||githubToken.length<8)
      return denied('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
    let store,reviewed;
    try{
      store=await openStore({path:hostStorePath,busyTimeoutMs:5000});
      const keys=await store.list(JOIN_PREFIX),rows=[];
      for(const key of keys)rows.push(await store.get(key));
      reviewed=latestReviewed(rows,sourceSha,attemptId,issueRef);
    }catch{return unknown('REVIEW_STORE_UNAVAILABLE');}
    finally{if(store)await store.close();}
    if(reviewed.reason)return denied(reviewed.reason);
    const candidate=reviewed.candidate,commit=candidate.candidateCommit;
    if(!SHA.test(commit??'')||!SHA.test(candidate.candidateTree??''))return denied('CANDIDATE_READBACK_MISMATCH');
    const git=config.gitPath??'C:\\Program Files\\Git\\cmd\\git.exe';
    try{
      const [parent,tree]=await Promise.all([
        exec(git,['-C',repositoryPath,'rev-parse',commit+'^'],{windowsHide:true}),
        exec(git,['-C',repositoryPath,'rev-parse',commit+'^{tree}'],{windowsHide:true})]);
      if(parent.stdout.trim()!==sourceSha||tree.stdout.trim()!==candidate.candidateTree)
        return denied('CANDIDATE_READBACK_MISMATCH');
    }catch{return denied('CANDIDATE_READBACK_MISMATCH');}
    const branch=branchFor(attemptId);
    const env={...process.env,TALLY_GITHUB_TOKEN:githubToken,GIT_TERMINAL_PROMPT:'0',
      GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:config.emptyGitConfig??'NUL',GIT_CONFIG_COUNT:'1',
      GIT_CONFIG_KEY_0:'credential.helper',
      GIT_CONFIG_VALUE_0:'!f() { echo username=x-access-token; echo password=$TALLY_GITHUB_TOKEN; }; f'};
    try{
      await exec(git,['-C',repositoryPath,'push',`https://github.com/${OWNER}/${REPO}.git`,
        `${commit}:refs/heads/${branch}`],{env,windowsHide:true,timeout:30000});
    }catch{return unknown('GITHUB_REF_PUBLICATION_UNKNOWN');}
    const result=await publish({owner:OWNER,repo:REPO,branch,baseRef:'version/v2',candidateCommit:commit,
      title:`Reviewed candidate ${attemptId}`,
      body:`Exact reviewed candidate ${commit} from ${issueRef}.\n\nPublication only; no CI, merge, Task or Release PASS is inferred.\n\nRefs: #200 #203`});
    if(result?.state!=='PUBLISHED_VERIFIED')return result??unknown('GITHUB_PUBLICATION_UNKNOWN');
    return Object.freeze({...result,issueRef,sourceSha,attemptId,candidateTree:candidate.candidateTree,
      claimGeneration:1,claimAccepted:true,admitted:true,agentLaunch:false,replayFixture:false,
      authorized:false,taskPass:false,releasePass:false});
  }});
}
