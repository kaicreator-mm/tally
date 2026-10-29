/**
 * T06j — privileged Host admission source for the existing T06d controller.
 *
 * Native C proves current GitHub facts; Host configuration proves dispatch and
 * writer identity; the public DH CAS registry proves incompatible reservations.
 * None of these facts alone is a CAS-accepted Claim, K permit or Agent effect.
 * This adapter is read-only: callers must never supply authority booleans or
 * writer identity through a request, comment, B port, Agent or UI endpoint.
 */
import {Octokit} from '@octokit/rest';
import {readNativeHostClaimPreflightFromClient} from '../../../packages/tally-github-native-c/src/task-claim-host-preflight.mjs';
import {createHostNativeGitHubClaimReadbackPortFromClient} from '../../../packages/tally-github-native-c/src/task-claim-host-readback-port.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human',
  'github-actions','woodpecker','other']);
const PREFIX='tally:v2:claim-admission:';
const PHASES=new Set(['PENDING_GITHUB_PUBLISH','PUBLISHED_VERIFIED']);
const CLAIM_KEY=new RegExp('^([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#([1-9][0-9]*):'+
  '(builder|reviewer|validator):__default__$');
const readbackFailure=()=>new Error('NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED');

function denied(reason,request){
  return Object.freeze({
    status:'BLOCKED',reason,sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',
    sourceQualified:false,complete:false,
    issueRef:request?.issueRef??null,sourceSha:request?.sourceSha??null,
    activeIncompatibleClaimCount:null,dispatchSourceBound:false,
    publisherSourceBound:false,claimStatus:'NOT_VERIFIED',
    authorized:false,admitted:false,taskPass:false,releasePass:false
  });
}
function validTask(task,repository){
  return task&&Number.isSafeInteger(task.issueNumber)&&task.issueNumber>0&&
    SHA.test(task.sourceSha??'')&&ROLES.has(task.actorRole)&&
    ID.test(task.dispatchId??'')&&KINDS.has(task.operatorKind)&&
    ID.test(task.operatorId??'')&&
    task.protectedClaimKey===undefined;
}
function boundRequest(request,task,repository){
  return request&&typeof request==='object'&&!Array.isArray(request)&&
    request.repository===repository&&request.issueNumber===task.issueNumber&&
    request.issueRef===`https://github.com/${repository}/issues/${task.issueNumber}`&&
    request.sourceSha===task.sourceSha&&request.actorRole===task.actorRole&&
    request.dispatchId===task.dispatchId&&request.operatorKind===task.operatorKind&&
    request.operatorId===task.operatorId&&request.generation===1&&
    request.protectedClaimKey===`${repository}#${task.issueNumber}:${task.actorRole}:__default__`;
}
function validWriter(authority,repository){
  return authority?.mode==='SINGLE_WRITER_ADMISSION'&&
    authority?.exclusiveHostControl===true&&authority?.repository===repository&&
    ID.test(authority?.writerId??'')&&LOGIN.test(authority?.githubWriterLogin??'');
}
function validRegistryRow(row){
  const value=row?.value;
  if(!((Number.isSafeInteger(row?.revision)&&row.revision>=0)||
    (typeof row?.revision==='string'&&row.revision.length>0))||
    !value||typeof value!=='object'||!PHASES.has(value.phase)||
    value.generation!==1||!SHA.test(value.sourceSha??'')||
    !ID.test(value.dispatchId??'')||!ID.test(value.operatorId??'')||
    !KINDS.has(value.operatorKind)||!ID.test(value.writerId??''))return false;
  const match=CLAIM_KEY.exec(value.protectedClaimKey??'');
  if(!match||value.repository!==match[1]||value.actorRole!==match[3]||
    value.issueRef!==`https://github.com/${match[1]}/issues/${match[2]}`)return false;
  if(value.phase==='PUBLISHED_VERIFIED'&&
    (!Number.isSafeInteger(value.commentId)||value.commentId<1||
     value.commentRef!==`${value.issueRef}#issuecomment-${value.commentId}`))return false;
  return true;
}

/** Test/selected-Host seam. All authority comes from Host-owned configuration. */
export function createHostAdmissionSourceFromClient({
  octokit,store,owner,repo,writerAuthority,taskAuthority
}={}){
  const repository=NAME.test(owner??'')&&NAME.test(repo??'')?`${owner}/${repo}`:'';
  if(!repository||!validWriter(writerAuthority,repository)||
    !validTask(taskAuthority,repository)||
    typeof octokit?.rest?.issues?.get!=='function'||
    typeof octokit?.rest?.issues?.listComments!=='function'||
    typeof octokit?.rest?.repos?.getCommit!=='function'||
    typeof octokit?.paginate?.iterator!=='function'||
    typeof store?.get!=='function'||typeof store?.list!=='function')
    throw new TypeError('TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED');
  const readback=createHostNativeGitHubClaimReadbackPortFromClient({
    octokit,owner,repo,writerLogin:writerAuthority.githubWriterLogin
  });
  async function registrySnapshot(){
    let rows;
    try{
      const read=async()=>{
        const keys=await store.list(PREFIX);
        if(!Array.isArray(keys)||keys.some(key=>typeof key!=='string'||!key.startsWith(PREFIX)))
          throw new Error('INCOMPLETE');
        const result=[];
        for(const key of keys)result.push({key,...await store.get(key)});
        return result;
      };
      const first=await read(),second=await read();
      if(first.length!==second.length||first.some((row,index)=>
        row.key!==second[index].key||row.revision!==second[index].revision))
        return {error:'HOST_RESERVATION_REGISTRY_DRIFT'};
      rows=first;
    }catch{return {error:'HOST_RESERVATION_REGISTRY_UNAVAILABLE'};}
    const seen=new Set();
    for(const row of rows){
      if(!validRegistryRow(row))return {error:'HOST_RESERVATION_REGISTRY_INCOMPLETE'};
      const key=row.value.protectedClaimKey;
      if(seen.has(key))return {error:'HOST_RESERVATION_REGISTRY_CONFLICT'};
      seen.add(key);
    }
    return {rows:rows.map(row=>row.value)};
  }
  async function readAdmissionFacts(request){
    if(!boundRequest(request,taskAuthority,repository))
      return denied('HOST_TASK_OR_WRITER_REQUEST_NOT_BOUND',request);
    const observed=await readNativeHostClaimPreflightFromClient({
      octokit,owner,repo,issueNumber:taskAuthority.issueNumber,
      expectedSourceSha:taskAuthority.sourceSha,limit:1000
    });
    if(observed.status!=='NATIVE_SOURCE_PREFLIGHT_READY')
      return denied(observed.reason??'NATIVE_GITHUB_SOURCE_INCOMPLETE_OR_STALE',request);
    const registry=await registrySnapshot();
    if(registry.error)return denied(registry.error,request);
    const activeIncompatibleClaimCount=registry.rows.filter(row=>
      row.protectedClaimKey!==request.protectedClaimKey).length;
    if(activeIncompatibleClaimCount>0)return Object.freeze({
      ...denied('INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS',request),
      activeIncompatibleClaimCount
    });
    return Object.freeze({
      sourceKind:'NATIVE_GITHUB_CURRENT_ADMISSION',sourceQualified:true,
      complete:true,issueRef:request.issueRef,sourceSha:request.sourceSha,
      issueState:observed.issueState,workflowState:observed.workflowState,
      workItemType:observed.workItemType,dependencyState:observed.dependencyState,
      activeIncompatibleClaimCount,dispatchSourceBound:true,
      publisherSourceBound:true,claimStatus:'NOT_VERIFIED',
      authorized:false,admitted:false,taskPass:false,releasePass:false
    });
  }
  async function readPublishedClaim(request,commentId){
    if(!boundRequest(request,taskAuthority,repository))throw readbackFailure();
    return readback.readPublishedClaim(request,commentId);
  }
  return Object.freeze({readAdmissionFacts,readPublishedClaim});
}

/** Production Host wiring; token is never returned, compared or journaled. */
export function createGitHubHostAdmissionSource({token,...input}={}){
  if(typeof token!=='string'||token.length<8)
    throw new TypeError('TRUSTED_HOST_ADMISSION_SOURCE_REQUIRED');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return createHostAdmissionSourceFromClient({octokit,...input});
}
