/**
 * T06j: native GitHub exact Claim readback source port for the existing
 * DomainHarness Host physical single-writer CAS controller.
 *
 * PUBLICATION_WITNESSED is an observational C fact. This maps it into
 * the Host's *source readback* shape ONLY; no durable CAS acceptance,
 * selected K task permit, Agent effect or release is authorized here.
 *
 * Supply real Octokit + selected Host writer login from a privileged Host
 * deployment. Never expose injected ports, token, or writer config to B,
 * an Agent, an LLM, a browser or a user-controlled HTTP endpoint.
 */
import {Octokit} from '@octokit/rest';
import {readClaimPublicationWitnessFromClient} from './task-claim-publication-witness.mjs';

const NAME=/^[A-Za-z0-9_.-]+$/;
const LOGIN=/^[A-Za-z0-9_.-]+(?:\[bot\])?$/;
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9:._/-]{0,127}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const KINDS=new Set(['chatgpt-web','codex','claude-code','human',
  'github-actions','woodpecker','other']);
const fail=()=>new Error('NATIVE_GITHUB_EXACT_CLAIM_READBACK_UNVERIFIED');

function reservationValid(i,owner,repo){
  return i&&typeof i==='object'&&!Array.isArray(i)&&
    i.repository===owner+'/'+repo&&
    Number.isSafeInteger(i.issueNumber)&&i.issueNumber>0&&
    i.issueRef==='https://github.com/'+owner+'/'+repo+'/issues/'+i.issueNumber&&
    SHA.test(i.sourceSha??'')&&
    ROLES.has(i.actorRole)&&KINDS.has(i.operatorKind)&&
    ID.test(i.operatorId??'')&&ID.test(i.dispatchId??'')&&
    i.generation===1&&
    i.protectedClaimKey===owner+'/'+repo+'#'+i.issueNumber+
      ':'+i.actorRole+':__default__';
}
/** Bind immutable Host-owned writer identity to the native source facts. */
export function createHostNativeGitHubClaimReadbackPortFromClient({
  octokit,owner,repo,writerLogin
}={}){
  if(!NAME.test(owner??'')||!NAME.test(repo??'')||
     !LOGIN.test(writerLogin??'')||
     !octokit?.rest?.issues?.getComment)
    throw new TypeError('TRUSTED_HOST_GITHUB_READBACK_PORT_REQUIRED');
  async function readPublishedClaim(i,commentId){
    if(!reservationValid(i,owner,repo)||
       !Number.isSafeInteger(commentId)||commentId<1)
      throw fail();
    let witness;
    try{
      witness=await readClaimPublicationWitnessFromClient({
        octokit,owner,repo,issueNumber:i.issueNumber,
        expectedSourceSha:i.sourceSha,commentId,
        dispatchId:i.dispatchId,actorRole:i.actorRole,
        operatorKind:i.operatorKind,operatorId:i.operatorId,
        protectedClaimKey:i.protectedClaimKey,
        claimGeneration:i.generation,writerLogin
      });
    }catch{throw fail();}
    if(witness?.status!=='PUBLICATION_WITNESSED'||
       witness.sourceQualified!==true||
       witness.claimPublication!=='NATIVE_COMMENT_READBACK_ONLY'||
       witness.claimAccepted!==false||
       witness.authorized!==false||witness.admitted!==false||
       witness.commentId!==commentId||
       witness.commentRef!==i.issueRef+'#issuecomment-'+commentId||
       witness.issueRef!==i.issueRef||witness.sourceSha!==i.sourceSha||
       witness.dispatchId!==i.dispatchId||
       witness.protectedClaimKey!==i.protectedClaimKey||
       witness.generation!==i.generation||
       witness.actorRole!==i.actorRole||
       witness.operatorKind!==i.operatorKind||
       witness.operatorId!==i.operatorId||
       witness.transportActor!==writerLogin)
      throw fail();
    // Existing T06d Host controller independently validates all of these
    // against its *durably reserved* CAS row after the one-shot POST.
    return Object.freeze({
      sourceKind:'NATIVE_GITHUB_EXACT_READBACK',
      complete:true,edited:false,issueRef:i.issueRef,sourceSha:i.sourceSha,
      event:'DISPATCH_CLAIMED',dispatchState:'CLAIMED',
      dispatchId:i.dispatchId,actorRole:i.actorRole,
      operatorKind:i.operatorKind,operatorId:i.operatorId,
      generation:i.generation,protectedClaimKey:i.protectedClaimKey,
      commentId,commentRef:witness.commentRef,
      // Not Task authority; these are source facts only.
      claimAccepted:false,authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }
  return Object.freeze({readPublishedClaim});
}
/** Production read-only Host wiring; token is never returned/journaled. */
export function createGitHubHostClaimReadbackPort({
  token,owner,repo,writerLogin
}={}){
  if(typeof token!=='string'||token.length<8)
    throw new TypeError('TRUSTED_HOST_GITHUB_READBACK_PORT_REQUIRED');
  const octokit=new Octokit({auth:token,request:{timeout:15000}});
  return createHostNativeGitHubClaimReadbackPortFromClient({
    octokit,owner,repo,writerLogin
  });
}
