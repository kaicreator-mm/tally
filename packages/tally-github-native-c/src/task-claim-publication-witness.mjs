/** T06g: read-only native GitHub claim publication witness.
 * A verified comment is NOT an accepted CAS Claim or Kernel/Host permit.
 * The privileged Host must independently match its durable reservation.
 */
import {createHash} from 'node:crypto';
import {Octokit} from '@octokit/rest';
import {readClaimEventCandidatesFromClient} from './task-claim-events.mjs';
const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human','github-actions','woodpecker','other']);
const deny=reason=>Object.freeze({status:'UNKNOWN',reason,sourceQualified:false,
  claimPublication:'NOT_VERIFIED',claimAccepted:false,authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const classify=e=>e?.status===401?'GITHUB_UNAUTHENTICATED':
  e?.status===403?'GITHUB_ACCESS_DENIED':
  e?.status===404?'GITHUB_COMMENT_NOT_FOUND':
  e?.status===429?'GITHUB_RATE_LIMITED':'GITHUB_COMMENT_READ_FAILED';
const hash=body=>createHash('sha256').update(body,'utf8').digest('hex');

/** Host-reservation inputs are trusted only when supplied by privileged Host.
 * The injected Octokit client is a test/selected-Host seam, not a B/Agent API.
 */
export async function readClaimPublicationWitnessFromClient({
  octokit,owner,repo,issueNumber,expectedSourceSha,commentId,
  dispatchId,actorRole,operatorKind,operatorId,protectedClaimKey,
  claimGeneration,writerLogin,limit=1000
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||!NAME.test(writerLogin??'')||
     !Number.isSafeInteger(issueNumber)||issueNumber<1||
     !Number.isSafeInteger(commentId)||commentId<1||
     !SHA.test(expectedSourceSha??'')||!ID.test(dispatchId??'')||
     !ID.test(operatorId??'')||!ROLES.has(actorRole)||!KINDS.has(operatorKind)||
     claimGeneration!==1||!Number.isSafeInteger(limit)||limit<1||limit>1000||
     protectedClaimKey!==owner+'/'+repo+'#'+issueNumber+':'+actorRole+':__default__'||
     !octokit?.rest?.issues?.getComment)
    return deny('INVALID_EXACT_CLAIM_WITNESS_REQUEST');
  const found=await readClaimEventCandidatesFromClient({
    octokit,owner,repo,issueNumber,expectedSourceSha,limit
  });
  if(found.reason!=='CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'||
     !Array.isArray(found.candidates))
    return deny('NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED');
  const matchingKey=found.candidates.filter(c=>c.protectedClaimKey===protectedClaimKey);
  if(matchingKey.length!==1)return deny('PROTECTED_CLAIM_KEY_NOT_UNIQUE');
  const c=matchingKey[0];
  if(c.commentId!==commentId||c.dispatchId!==dispatchId||
     c.actorRole!==actorRole||c.operatorKind!==operatorKind||
     c.operatorId!==operatorId||c.transportActor!==writerLogin||
     c.claimGeneration!==claimGeneration||c.sourceSha!==expectedSourceSha||
     c.sourceClass!=='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'||
     c.claimAccepted!==false||!/^[a-f0-9]{64}$/.test(c.commentBodySha256??''))
    return deny('NATIVE_CLAIM_EVENT_RESERVATION_MISMATCH');
  try{
    const current=(await octokit.rest.issues.getComment({
      owner,repo,comment_id:commentId
    }))?.data;
    const issueRef='https://github.com/'+owner+'/'+repo+'/issues/'+issueNumber;
    if(current?.id!==commentId||
       current?.html_url!==issueRef+'#issuecomment-'+commentId||
       current?.issue_url!=='https://api.github.com/repos/'+owner+'/'+repo+'/issues/'+issueNumber||
       current?.user?.login!==writerLogin||typeof current.body!=='string'||
       current.body.length>256*1024||typeof current.created_at!=='string'||
       current.created_at!==current.updated_at||
       hash(current.body)!==c.commentBodySha256)
      return deny('NATIVE_COMMENT_CURRENTNESS_NOT_VERIFIED');
    return Object.freeze({
      status:'PUBLICATION_WITNESSED',
      reason:'COMMENT_BOUND_TO_HOST_RESERVATION_NOT_CAS_ADMISSION',
      sourceQualified:true,issueRef,sourceSha:expectedSourceSha,
      commentId,commentRef:current.html_url,protectedClaimKey,
      generation:claimGeneration,dispatchId,actorRole,operatorKind,operatorId,
      transportActor:writerLogin,claimPublication:'NATIVE_COMMENT_READBACK_ONLY',
      claimAccepted:false,authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }catch(e){return deny(classify(e));}
}
export async function readGitHubClaimPublicationWitness({token,...input}={}){
  if(typeof token!=='string'||token.length<8)
    return deny('GITHUB_PROVIDER_TOKEN_UNAVAILABLE');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return readClaimPublicationWitnessFromClient({octokit,...input});
}
