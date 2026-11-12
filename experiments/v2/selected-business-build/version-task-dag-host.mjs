/**
 * E0 #284 — Host composition entry for the two pre-freeze privileged
 * operations: addNativeTaskDependency and freezeVersionTaskDag.
 *
 * This module is transport/composition ONLY. It binds trusted Host
 * configuration (env-pinned version SHA, dedicated CAS registry path,
 * write-scoped token, exclusive writer authority, exact review/Owner-approval
 * comment refs and per-Task frozen write-set documents) to the selected C
 * port in packages/tally-github-native-c/src/version-task-dag-write-port.mjs,
 * which consumes the A-owned closeout-roster freeze writer read-only.
 *
 * The T06d root Claim publication stays Host-process-only: it is NOT exposed
 * here and must never become an HTTP route or browser action.
 */
import {createVersionTaskDagWritePort}
  from '../../../packages/tally-github-native-c/src/version-task-dag-write-port.mjs';
import {readUiNativeVersionFromClient}
  from '../../../packages/tally-github-native-c/src/ui-version-readonly.mjs';

const OWNER='kaicreator-mm';
const REPO='tally';
const REPOSITORY=OWNER+'/'+REPO;
const VERSION_ID='version/v2';
const SHA=/^[a-f0-9]{40}$/;
const DIGEST=/^[0-9a-f]{64}$/;
const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const denied=(reason)=>Object.freeze({status:'BLOCKED',reason,
  repository:REPOSITORY,versionId:VERSION_ID,githubEffect:'NOT_RUN',
  authorized:false,admitted:false,taskPass:false,releasePass:false});

/** Parse `https://github.com/<owner>/<repo>/blob/<sha>/<path>` (immutably). */
function parseBlobRef(taskPackRef){
  if(typeof taskPackRef!=='string')return null;
  const prefix='https://github.com/'+REPOSITORY+'/blob/';
  if(!taskPackRef.startsWith(prefix))return null;
  const rest=taskPackRef.slice(prefix.length),slash=rest.indexOf('/');
  if(slash!==40||!SHA.test(rest.slice(0,slash))||rest.length===slash+1)return null;
  return {commitSha:rest.slice(0,slash),path:rest.slice(slash+1)};
}

/** GitHub Contents blob reader: content-addressed by the exact blob SHA. */
function createTaskPackReaderFromClient({octokit}){
  return async function taskPackReader({taskPackRef,expectedBlobSha}){
    const parsed=parseBlobRef(taskPackRef);
    if(!parsed||!SHA.test(expectedBlobSha??''))
      return {status:'NOT_VERIFIED',reason:'TASK_PACK_IDENTITY_INVALID'};
    try{
      const response=await octokit.rest.repos.getContent({owner:OWNER,repo:REPO,
        path:parsed.path,ref:parsed.commitSha});
      const data=response?.data;
      if(data?.encoding!=='base64'||typeof data?.content!=='string'||
         data.sha!==expectedBlobSha)
        return {status:'NOT_VERIFIED',reason:'TASK_PACK_BLOB_MISMATCH'};
      return {status:'VERIFIED',blobSha:data.sha,
        content:Buffer.from(data.content,'base64').toString('utf8')};
    }catch{return {status:'NOT_VERIFIED',reason:'TASK_PACK_READ_FAILED'};}
  };
}

/** Exact-comment witness reader: the trusted Host pins the exact comment ids;
 * the reader re-reads the comment twice and only accepts a stable JSON body
 * that binds the SAME PackBlob SHA and source SHA being frozen. */
function createWitnessReaderFromClient({octokit,issueNumber,commentId,login,kind}){
  return async function witnessReader({taskPackBlobSha,sourceSha}){
    if(!Number.isSafeInteger(issueNumber)||issueNumber<1||
       !Number.isSafeInteger(commentId)||commentId<1)return {status:'NOT_VERIFIED'};
    try{
      const params={owner:OWNER,repo:REPO,comment_id:commentId};
      const first=(await octokit.rest.issues.getComment(params))?.data;
      const second=(await octokit.rest.issues.getComment(params))?.data;
      const valid=c=>plain(c)&&c.id===commentId&&
        c.html_url==='https://github.com/'+REPOSITORY+'/issues/'+issueNumber+
          '#issuecomment-'+commentId&&
        c.issue_url==='https://api.github.com/repos/'+OWNER+'/'+REPO+
          '/issues/'+issueNumber&&
        c.user?.login===login&&typeof c.body==='string'&&c.body.length<=65536;
      if(!valid(first)||!valid(second)||first.body!==second.body||
         first.updated_at!==second.updated_at)
        return {status:'NOT_VERIFIED'};
      let witness;
      try{witness=JSON.parse(first.body);}catch{return {status:'NOT_VERIFIED'};}
      if(!plain(witness)||witness.taskPackBlobSha!==taskPackBlobSha||
         witness.sourceSha!==sourceSha)return {status:'NOT_VERIFIED'};
      if(kind==='review'){
        return witness.status==='INDEPENDENTLY_VERIFIED'&&
          witness.reviewedHeadSha===sourceSha&&ID.test(witness.reviewer??'')&&
          ID.test(witness.author??'')&&witness.reviewer!==witness.author&&
          typeof witness.reviewRef==='string'&&
          witness.reviewRef.startsWith('https://github.com/'+REPOSITORY+'/')&&
          Number.isFinite(Date.parse(witness.verifiedAt??''))?witness:{status:'NOT_VERIFIED'};
      }
      return witness.status==='OWNER_APPROVED'&&witness.owner===OWNER&&
        typeof witness.approvalRef==='string'&&
        witness.approvalRef.startsWith('https://github.com/'+REPOSITORY+'/')&&
        Number.isFinite(Date.parse(witness.approvedAt??''))?witness:{status:'NOT_VERIFIED'};
    }catch{return {status:'NOT_VERIFIED'};}
  };
}

/**
 * Bind the Host environment. Required env pins (Host-side ONLY):
 *   TALLY_V2_VERSION_SHA            exact selected version source SHA
 *   TALLY_VERSION_DAG_REGISTRY_PATH dedicated CAS registry file
 *   TALLY_VERSION_DAG_WRITER_ID     exclusive Host writer identity
 *   TALLY_VERSION_DAG_WRITER_LOGIN  native transport actor login
 *   TALLY_VERSION_DAG_REVIEW_ISSUE/_COMMENT_ID
 *   TALLY_VERSION_DAG_APPROVAL_ISSUE/_COMMENT_ID
 *   TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON  per-issue {writeSetSha256,document}
 *   TALLY_VERSION_DAG_FREEZE_POLICY  ALL_TASKS (legacy) or VERSION_ONLY
 * VERSION_ONLY is a Host-only explicit Core MVP structural Freeze; no Task FSW
 * is minted and later Task admission independently requires its own Claim/FSW.
 * The octokit client and openStore factory are injected (never env strings).
 */
export function createVersionTaskDagHost({env=process.env,octokit,openStore}){
  if(!octokit?.rest?.issues?.get||!octokit?.rest?.repos?.getContent||
     !octokit?.rest?.issues?.getComment||!octokit?.paginate?.iterator||
     !octokit?.request||typeof openStore!=='function')
    throw new TypeError('TRUSTED_VERSION_TASK_DAG_HOST_TRANSPORTS_REQUIRED');
  const expectedVersionSha=env.TALLY_V2_VERSION_SHA;
  const registryPath=env.TALLY_VERSION_DAG_REGISTRY_PATH;
  const writerId=env.TALLY_VERSION_DAG_WRITER_ID;
  const writerLogin=env.TALLY_VERSION_DAG_WRITER_LOGIN;
  if(!SHA.test(expectedVersionSha??'')||typeof registryPath!=='string'||
     !registryPath||!ID.test(writerId??'')||!LOGIN.test(writerLogin??''))
    throw new TypeError('TRUSTED_VERSION_TASK_DAG_HOST_CONFIGURATION_REQUIRED');
  const reviewWitnessReader=createWitnessReaderFromClient({octokit,
    issueNumber:Number(env.TALLY_VERSION_DAG_REVIEW_ISSUE),
    commentId:Number(env.TALLY_VERSION_DAG_REVIEW_COMMENT_ID),
    login:String(env.TALLY_VERSION_DAG_REVIEW_LOGIN??writerLogin),kind:'review'});
  const ownerApprovalReader=createWitnessReaderFromClient({octokit,
    issueNumber:Number(env.TALLY_VERSION_DAG_APPROVAL_ISSUE),
    commentId:Number(env.TALLY_VERSION_DAG_APPROVAL_COMMENT_ID),
    login:String(env.TALLY_VERSION_DAG_APPROVAL_LOGIN??OWNER),kind:'approval'});
  const freezeWriteSetPolicy=env.TALLY_VERSION_DAG_FREEZE_POLICY??'ALL_TASKS';
  if(!['ALL_TASKS','VERSION_ONLY'].includes(freezeWriteSetPolicy)||
     (freezeWriteSetPolicy==='VERSION_ONLY'&&
      env.TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON!==undefined))
    throw new TypeError('INVALID_VERSION_TASK_DAG_FREEZE_POLICY');
  let writeSetDocumentByIssue=null;
  if(env.TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON!==undefined){
    try{writeSetDocumentByIssue=JSON.parse(env.TALLY_VERSION_DAG_WRITESET_DOCUMENTS_JSON);}
    catch{throw new TypeError('INVALID_VERSION_TASK_DAG_WRITESET_DOCUMENTS');}
    if(!plain(writeSetDocumentByIssue)||Object.values(writeSetDocumentByIssue)
        .some(x=>!plain(x)||!DIGEST.test(x.writeSetSha256??'')||
          typeof x.document!=='string'||x.document.length===0||
          x.document.length>65536))
      throw new TypeError('INVALID_VERSION_TASK_DAG_WRITESET_DOCUMENTS');
  }
  const taskPackReader=createTaskPackReaderFromClient({octokit});
  const nativeRosterReader=({taskIssueNumbers})=>readUiNativeVersionFromClient({
    octokit,owner:OWNER,repo:REPO,versionId:VERSION_ID,expectedSha:expectedVersionSha,
    taskIssueNumbers});
  const writerAuthority={mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
    repository:REPOSITORY,writerId,githubWriterLogin:writerLogin};
  async function withPort(run){
    let store=null;
    try{
      store=await openStore({path:registryPath,busyTimeoutMs:1000});
      const port=createVersionTaskDagWritePort({repository:REPOSITORY,
        versionId:VERSION_ID,expectedVersionSha,store,taskPackReader,
        reviewWitnessReader,ownerApprovalReader,nativeRosterReader,
        octokit,owner:OWNER,repo:REPO,writerAuthority,
        writeSetDocumentByIssue:writeSetDocumentByIssue??{},
        freezeWriteSetPolicy});
      return await run(port);
    }catch{
      return denied('VERSION_TASK_DAG_HOST_UNAVAILABLE');
    }finally{
      if(store)await store.close().catch(()=>{});
    }
  }
  return Object.freeze({
    repository:REPOSITORY,versionId:VERSION_ID,expectedVersionSha,
    addNativeTaskDependency(input){
      const parsed=plain(input)?input:{};
      return withPort(port=>port.addNativeTaskDependency({
        taskPackRef:parsed.taskPackRef,taskPackBlobSha:parsed.taskPackBlobSha,
        blockedIssueNumber:parsed.blockedIssueNumber,
        blockedByIssueNumber:parsed.blockedByIssueNumber}));
    },
    freezeVersionTaskDag(input){
      const parsed=plain(input)?input:{};
      return withPort(port=>port.freezeVersionTaskDag({
        taskPackRef:parsed.taskPackRef,taskPackBlobSha:parsed.taskPackBlobSha}));
    },
  });
}

export const VERSION_TASK_DAG_HOST_IDS=Object.freeze({
  repository:REPOSITORY,versionId:VERSION_ID});
