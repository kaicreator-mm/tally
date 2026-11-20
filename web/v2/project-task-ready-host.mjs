/** Project-scoped trusted Host adapter for existing native planned→ready.
 * It does NOT publish CAS Claim, freeze a DAG, or authorize Agent execution.
 */
import {createTaskReadyTransitionPort}
  from '../../packages/tally-github-native-c/src/task-ready-transition-port.mjs';
import {readProjectTaskAdmission} from './task-admission-read.mjs';
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const deny=(reason)=>Object.freeze({status:'BLOCKED',reason,
  githubEffect:'NOT_RUN',authorized:false,admitted:false,
  taskPass:false,releasePass:false});
export function createProjectTaskReadyHost({
  settings,env,octokit,openStore,readAdmission=readProjectTaskAdmission,
  createPort=createTaskReadyTransitionPort
}={}){
  if(!settings||env?.TALLY_V2_TASK_READY!=='host-mounted')
    return null;
  const registryPath=env.TALLY_TASK_READY_REGISTRY_PATH;
  const writerId=env.TALLY_TASK_READY_WRITER_ID;
  const writerLogin=env.TALLY_TASK_READY_WRITER_LOGIN;
  const targetRepo=env.TALLY_TASK_READY_REPOSITORY;
  if(!SHA.test(settings.expectedSha??'')||
     !ID.test(writerId??'')||!LOGIN.test(writerLogin??'')||
     typeof registryPath!=='string'||registryPath.length<3||
     targetRepo!==settings.repository||
     typeof openStore!=='function'||!octokit?.rest?.issues?.get||
     !octokit?.rest?.repos?.getBranch||!octokit?.paginate?.iterator)
    throw new TypeError('TRUSTED_TASK_READY_HOST_CONFIGURATION_REQUIRED');
  const writerAuthority=Object.freeze({mode:'SINGLE_WRITER_ADMISSION',
    exclusiveHostControl:true,repository:settings.repository,writerId,
    githubWriterLogin:writerLogin});
  return Object.freeze({
    repository:settings.repository,versionId:settings.versionId,
    sourceSha:settings.expectedSha,
    async transition({issueNumber,sourceSha}={}){
      if(!Number.isSafeInteger(issueNumber)||issueNumber<1||
         sourceSha!==settings.expectedSha)
        return deny('TASK_READY_SCOPE_MISMATCH');
      const admission=await readAdmission({
        repository:settings.repository,versionId:settings.versionId,
        sourceSha:settings.expectedSha,token:settings.token,issueNumber});
      if(admission?.connected!==true||admission.stage!=='PLANNED')
        return deny(admission?.reason??'TASK_DRAFT_NOT_PLANNED');
      let store=null,enteredPort=false;
      try{
        store=await openStore({path:registryPath,busyTimeoutMs:1000});
        const port=createPort({
          repository:settings.repository,owner:settings.owner,
          repo:settings.repo,targetIssue:issueNumber,
          expectedSourceSha:settings.expectedSha,
          versionBranch:settings.versionId,octokit,store,writerAuthority,
          draftSourceIdentity:{repository:settings.repository,
            versionId:settings.versionId,sourceSha:settings.expectedSha}});
        // Existing port re-reads the official Task, blocked_by, CAS rows,
        // current branch and verifies the single GitHub PATCH after write.
        enteredPort=true;
        return await port.transitionToReady();
      }catch{
        // Once the native port has started, a thrown transport error may
        // follow an unacknowledged GitHub PATCH: do not claim NOT_RUN.
        return enteredPort?Object.freeze({status:'UNKNOWN',
          reason:'TASK_READY_EFFECT_RECONCILIATION_REQUIRED',
          githubEffect:'UNKNOWN',authorized:false,admitted:false,
          taskPass:false,releasePass:false}):
          deny('TASK_READY_HOST_UNAVAILABLE');
      }finally{if(store)await store.close().catch(()=>{});}
    }
  });
}
