/**
 * T06i - selected Host-bound native GitHub transport for a serialized claim.
 *
 * This is a *transport implementation*, not a CAS writer or K permit.
 * Host must supply sealed Octokit credentials, a genuinely exclusive writer
 * and a previously accepted physical CAS reservation. Never expose the
 * factory to an Agent, model output, UI endpoint or untrusted B port.
 * POST is non-idempotent: on lost ACK the Host reservation remains PENDING,
 * and no caller is authorized to retry blindly.
 */
import {stringify} from 'yaml';
import {createHash} from 'node:crypto';
import {TARGET} from './task-dispatch-state-events.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const GH_LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const SHA=/^[a-f0-9]{40}$/;
const HEX64=/^[a-f0-9]{64}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions',
  'woodpecker','other']);
const marker='<!-- ads:github -->';
const hash=x=>createHash('sha256').update(x,'utf8').digest('hex');
const forbidden=()=>new Error('HOST_EXCLUSIVE_GITHUB_PUBLISHER_REQUIRED');
const ambiguous=()=>new Error('NATIVE_GITHUB_CLAIM_PUBLICATION_AMBIGUOUS');
// The Host resource fence this transport is bound to. The publisher is a
// package and cannot import the experiments-side library, so the port shape
// and the returned readback proof are validated inline here.
const FENCE_PORT_SOURCE_KIND='TRUSTED_HOST_RESOURCE_FENCE_PORT';
const FENCE_READBACK_SOURCE_KIND='TRUSTED_HOST_RESOURCE_FENCE_READBACK';
const FENCE_PROOF_MAX_AGE_MS=15000;
// readHeldFence recomputes the foreign-claim digest from its projections;
// for the explicit empty projection it is exactly sha256(JSON.stringify([])).
const EMPTY_FOREIGN_CLAIMS_SHA256=hash(JSON.stringify([]));

/** Values here are selected by the Host, never by an Agent or HTTP caller. */
export function terminalEventForReservation(reservation,writerAuthority){
  if(writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||writerAuthority?.exclusiveHostControl!==true||
    writerAuthority?.repository!==TARGET.repository||reservation?.phase!=='PENDING_GITHUB_TERMINALIZATION'||
    reservation.repository!==TARGET.repository||reservation.issueRef!==TARGET.issueRef||
    reservation.sourceSha!==TARGET.sourceSha||reservation.dispatchId!==TARGET.dispatchId||
    reservation.protectedClaimKey!==TARGET.protectedClaimKey||reservation.generation!==1||
    reservation.commentId!==TARGET.claimCommentId||reservation.commentRef!==TARGET.claimCommentRef||
    reservation.terminalReservationRevision!==2||
    reservation.terminalWriterId!==writerAuthority?.writerId||
    !ID.test(writerAuthority?.writerId??'')||!GH_LOGIN.test(writerAuthority?.githubWriterLogin??'')||
    typeof reservation.terminalEvidence!=='string'||reservation.terminalEvidence.length<1||reservation.terminalEvidence.length>16384||
    typeof reservation.terminalOccurredAt!=='string'||!Number.isFinite(Date.parse(reservation.terminalOccurredAt)))
    throw new Error('HOST_TERMINAL_RESERVATION_NOT_BOUND');
  return Object.freeze({schema:'ai-dev/event-v2',event:'DISPATCH_STATE_CHANGED',
    actor_role:'scheduler',operator_kind:'other',operator_id:writerAuthority.writerId,
    transport_actor:writerAuthority.githubWriterLogin,task:'#203',issue:'#203',sha:TARGET.sourceSha,
    dispatch_id:TARGET.dispatchId,dispatch_state:'DONE',protected_claim_key:TARGET.protectedClaimKey,
    admission_generation:1,predecessor_claim_comment_id:TARGET.claimCommentId,
    predecessor_claim_comment_ref:TARGET.claimCommentRef,terminal_reservation_revision:reservation.terminalReservationRevision,
    reason:'Candidate publication dispatch completed',
    occurred_at:reservation.terminalOccurredAt,evidence:reservation.terminalEvidence});
}

function requireReservation(i,owner,repo,writerId){
  if(!i||typeof i!=='object'||Array.isArray(i)||
     i.repository!==owner+'/'+repo||!Number.isSafeInteger(i.issueNumber)||
     i.issueNumber<1||!SHA.test(i.sourceSha??'')||
     i.issueRef!=='https://github.com/'+owner+'/'+repo+'/issues/'+i.issueNumber||
     !ROLES.has(i.actorRole)||!KINDS.has(i.operatorKind)||
     !ID.test(i.operatorId??'')||!ID.test(i.dispatchId??'')||
     i.writerId!==writerId||i.generation!==1||
     i.protectedClaimKey!==owner+'/'+repo+'#'+i.issueNumber+
       ':'+i.actorRole+':__default__')
    throw new Error('HOST_CLAIM_RESERVATION_NOT_BOUND');
  return i;
}
function renderEvent(i){
  // Only schema-allowed, host-reservation-sourced fields. No extra free-form
  // user text, token, Agent prompt, private transcript or secret in the post.
  const data={
    schema:'ai-dev/event-v2',event:'DISPATCH_CLAIMED',
    actor_role:i.actorRole,operator_kind:i.operatorKind,operator_id:i.operatorId,
    task:'#'+i.issueNumber,issue:'#'+i.issueNumber,sha:i.sourceSha,
    dispatch_id:i.dispatchId,dispatch_state:'CLAIMED',
    protected_claim_key:i.protectedClaimKey,admission_generation:1
  };
  return marker+'\n'+String.fromCharCode(96).repeat(3)+'yaml\n'+
    stringify({schema:'ads/github/v3',kind:'event',data}).trimEnd()+
    '\n'+String.fromCharCode(96).repeat(3)+'\n';
}
function readbackValid(c,id,issueUrl,apiIssueUrl,login,body){
  return c?.id===id&&c?.html_url===issueUrl+'#issuecomment-'+id&&
    c?.issue_url===apiIssueUrl&&c?.user?.login===login&&
    typeof c.body==='string'&&hash(c.body)===hash(body)&&
    typeof c.created_at==='string'&&c.created_at===c.updated_at;
}

/** The writerIdentity object is trusted only as Host-controlled configuration;
 * data from an API request cannot constitute this authority. The fencePort
 * is likewise Host-injected: it must be the trusted shared-Host resource
 * fence port for exactly this repository, writer and frozen resource set.
 */
export function createHostBoundNativeGitHubClaimPublisher({
  octokit,owner,repo,writerAuthority,fencePort
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
     writerAuthority?.exclusiveHostControl!==true||
     writerAuthority?.repository!==owner+'/'+repo||
     !ID.test(writerAuthority?.writerId??'')||
     !GH_LOGIN.test(writerAuthority?.githubWriterLogin??'')||
     typeof octokit?.rest?.issues?.createComment!=='function'||
     typeof octokit?.rest?.issues?.getComment!=='function'||
     fencePort?.sourceKind!==FENCE_PORT_SOURCE_KIND||
     fencePort?.repository!==owner+'/'+repo||
     fencePort?.writerId!==writerAuthority?.writerId||
     !HEX64.test(fencePort?.frozenResourceSetSha256??'')||
     typeof fencePort?.readHeldFence!=='function')
    throw forbidden();
  const writerId=writerAuthority.writerId;
  const login=writerAuthority.githubWriterLogin;
  const terminalAuthority=Object.freeze({mode:'SINGLE_WRITER_ADMISSION',exclusiveHostControl:true,
    repository:owner+'/'+repo,writerId,githubWriterLogin:login});
  function heldFenceProofValid(proof,i){
    const observed=typeof proof?.observedAt==='string'?
      Date.parse(proof.observedAt):NaN;
    return proof?.sourceKind===FENCE_READBACK_SOURCE_KIND&&
      proof?.complete===true&&proof?.fenceHeld===true&&
      proof?.repository===fencePort.repository&&
      proof?.writerId===fencePort.writerId&&
      proof?.protectedClaimKey===i.protectedClaimKey&&
      proof?.sourceSha===i.sourceSha&&
      HEX64.test(proof?.frozenResourceSetSha256??'')&&
      proof?.frozenResourceSetSha256===fencePort.frozenResourceSetSha256&&
      Number.isFinite(observed)&&observed<=Date.now()&&
      Date.now()-observed<=FENCE_PROOF_MAX_AGE_MS;
  }
  async function assertHeldFenceBeforePost(i){
    // The context is derived from THIS request with explicit empty foreign
    // projections; only the Host-injected port can answer it, and it binds
    // the answer to its own fixed identity and the durable lease registry.
    // A caller-supplied witness of any shape can never pass this gate.
    let proof=null;
    try{
      proof=await fencePort.readHeldFence(Object.freeze({
        repository:owner+'/'+repo,writerId,
        protectedClaimKey:i.protectedClaimKey,sourceSha:i.sourceSha,
        foreignClaimCount:0,foreignClaims:Object.freeze([]),
        foreignClaimsSha256:EMPTY_FOREIGN_CLAIMS_SHA256
      }));
    }catch(error){
      // The trusted fence library reports typed failures by code (for
      // example HOST_RESOURCE_FENCE_NOT_HELD); those are preserved so the
      // serialized writer can deny with a distinct fence reason. Anything
      // else from an unqualified port is a forbidden-wiring failure.
      if(error instanceof Error&&typeof error.code==='string'&&
         error.code.startsWith('HOST_RESOURCE_FENCE_'))throw error;
      throw forbidden();
    }
    if(!heldFenceProofValid(proof,i))throw forbidden();
  }
  async function publishClaim(request){
    const i=requireReservation(request,owner,repo,writerId);
    const issueUrl=i.issueRef;
    const apiIssueUrl='https://api.github.com/repos/'+owner+'/'+repo+
      '/issues/'+i.issueNumber;
    const body=renderEvent(i);
    // Calling this method is permitted ONLY after a real selected Host CAS
    // reservation AND a currently held exclusive-resource lease, proven by
    // the Host-injected fence port immediately before the POST. The method
    // has no retry and no unprivileged auth fallback.
    await assertHeldFenceBeforePost(i);
    let response;
    try{
      response=await octokit.rest.issues.createComment({
        owner,repo,issue_number:i.issueNumber,body
      });
    }catch{throw ambiguous();}
    const id=response?.data?.id;
    // If POST may have succeeded but response was lost/invalid, preserve
    // Host PENDING reservation and force a separate native reconciliation.
    if(!Number.isSafeInteger(id)||id<1||
       !readbackValid(response.data,id,issueUrl,apiIssueUrl,login,body))
      throw ambiguous();
    let reread;
    try{reread=(await octokit.rest.issues.getComment({
      owner,repo,comment_id:id
    }))?.data;}catch{throw ambiguous();}
    if(!readbackValid(reread,id,issueUrl,apiIssueUrl,login,body))
      throw ambiguous();
    // This is solely a transport ACK. Host CAS persistence and Kernel Task
    // effect admission are independently necessary and NOT indicated here.
    return Object.freeze({
      commentId:id,commentRef:issueUrl+'#issuecomment-'+id,
      commentBodySha256:hash(body),claimPublication:'GITHUB_COMMENT_POST_READBACK',
      claimAccepted:false,authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }
  async function publishDispatchStateChange(reservation){
    const event=terminalEventForReservation(reservation,terminalAuthority);
    const body=marker+'\n'+String.fromCharCode(96).repeat(3)+'yaml\n'+
      stringify({schema:'ads/github/v3',kind:'event',data:event}).trimEnd()+
      '\n'+String.fromCharCode(96).repeat(3)+'\n';
    const apiIssueUrl='https://api.github.com/repos/kaicreator-mm/tally/issues/203';
    let response;
    try{response=await octokit.rest.issues.createComment({owner,repo,issue_number:203,body});}
    catch{throw ambiguous();}
    const id=response?.data?.id;
    if(!Number.isSafeInteger(id)||id<1||!readbackValid(response.data,id,TARGET.issueRef,apiIssueUrl,login,body))throw ambiguous();
    let reread;
    try{reread=(await octokit.rest.issues.getComment({owner,repo,comment_id:id}))?.data;}
    catch{throw ambiguous();}
    if(!readbackValid(reread,id,TARGET.issueRef,apiIssueUrl,login,body))throw ambiguous();
    return Object.freeze({commentId:id,commentRef:`${TARGET.issueRef}#issuecomment-${id}`,
      authorized:false,admitted:false,taskPass:false,releasePass:false});
  }
  return Object.freeze({publishClaim,publishDispatchStateChange});
}
