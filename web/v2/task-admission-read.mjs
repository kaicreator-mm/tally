/** New-project GitHub draft → canonical native Task preflight.
 * Pure read-only. Native ready is not CAS Claim acceptance or an Agent grant.
 */
import {readGitHubTaskDrafts} from './task-draft-reader.mjs';
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const deny=reason=>Object.freeze({connected:false,reason,admitted:false,
  authorized:false,agentAuthorized:false});
export async function readProjectTaskAdmission({
  repository,versionId,sourceSha,token,issueNumber,
  readDrafts=readGitHubTaskDrafts,readNativePreflight=null
}={}){
  if(!REPO.test(repository??'')||typeof versionId!=='string'||
     !SHA.test(sourceSha??'')||typeof token!=='string'||token.length<8||
     !Number.isSafeInteger(issueNumber)||issueNumber<1)
    return deny('TASK_ADMISSION_SOURCE_INVALID');
  let draftList;
  try{draftList=await readDrafts({repository,versionId,sourceSha,token});}
  catch{return deny('TASK_DRAFT_SOURCE_UNAVAILABLE');}
  if(draftList?.connected!==true)return deny(draftList?.reason??'TASK_DRAFT_SOURCE_UNAVAILABLE');
  const draft=draftList.drafts?.find(d=>d.issueNumber===issueNumber);
  if(!draft)return deny(draftList.coverage==='LATEST_100_ISSUES_ONLY'?
    'TASK_DRAFT_NOT_IN_LATEST_100':'TASK_DRAFT_NOT_FOUND');
  const base={connected:true,repository,versionId,issueNumber,sourceSha,
    issueUrl:draft.issueUrl,nativeWorkflow:draft.nativeWorkflow,
    admitted:false,authorized:false,agentAuthorized:false};
  if(draft.sourceSha!==sourceSha)
    return Object.freeze({...base,stage:'STALE_SOURCE',
      reason:'TASK_DRAFT_SOURCE_SHA_STALE'});
  if(draft.githubIssueState!=='open')
    return Object.freeze({...base,stage:'BLOCKED',reason:'TASK_DRAFT_NOT_OPEN'});
  if(draft.nativeWorkflow==='NONCANONICAL')
    return Object.freeze({...base,stage:'NEEDS_CANONICAL_LABELS',
      reason:'TASK_CANONICAL_LABELS_REQUIRED'});
  if(draft.nativeWorkflow==='PLANNED')
    return Object.freeze({...base,stage:'PLANNED',reason:
      'TRUSTED_HOST_READY_TRANSITION_REQUIRED'});
  if(draft.nativeWorkflow!=='READY')
    return Object.freeze({...base,stage:'BLOCKED',reason:'TASK_WORKFLOW_UNKNOWN'});
  try{
    const read=readNativePreflight??(await import(
      '../../packages/tally-github-native-c/src/task-claim-host-preflight.mjs'
    )).readGitHubNativeHostClaimPreflight;
    const [owner,repo]=repository.split('/');
    const result=await read({token,owner,repo,issueNumber,
      expectedSourceSha:sourceSha,actorRole:'builder'});
    if(result?.status==='NATIVE_SOURCE_PREFLIGHT_READY'&&
       result.sourceQualified===true&&result.nativePreflightReady===true&&
       result.sourceSha===sourceSha&&
       result.issueRef===draft.issueUrl&&result.admitted===false&&
       result.authorized===false)
      return Object.freeze({...base,stage:'NATIVE_PREFLIGHT_READY',
        reason:'HOST_BINDING_AND_ATOMIC_CAS_CLAIM_REQUIRED'});
    return Object.freeze({...base,stage:'BLOCKED',
      reason:/^[A-Z][A-Z0-9_]{2,96}$/.test(result?.reason??'')?
        result.reason:'NATIVE_PREFLIGHT_UNQUALIFIED'});
  }catch{return Object.freeze({...base,stage:'BLOCKED',
    reason:'NATIVE_PREFLIGHT_UNAVAILABLE'});}
}
