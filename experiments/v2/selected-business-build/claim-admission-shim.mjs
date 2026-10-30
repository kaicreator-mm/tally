/**
 * T06l BUILD-sealed Capability ABI + safe projection for accepted-Claim
 * execution admission (real Claim kaicreator-mm/tally#169).
 *
 * The producer substitutes the source SHA256 into the literal below before
 * calling public BUILD. The trusted Host MUST verify the installed npm
 * provider bytes against that sealed value BEFORE Microkernel.load(), not at
 * or after import (deployment integrity, not arbitrary-JS isolation).
 *
 * The sealed C consumes ONLY the provider's accepted-Claim observation. That
 * observation is derived by the provider from the real DH CAS registry row
 * (phase=PUBLISHED_VERIFIED, generation=1) plus the existing native GitHub
 * publication witness readback — never from a caller flag, a comment field
 * or any input-carried accepted boolean. Positive output admits THIS K
 * intent only: no caller authorization, no Agent launch, no Task PASS, no
 * Release PASS. Every deviation or provider failure fails closed.
 */
import {admitClaimedTask as nativeAdmitClaimedTask}
  from '@kaicreator/tally-native-claim-reader-bundle';

export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-claim-admission@v1.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-claim-admission';
export const OPERATIONS=Object.freeze(['tally.admitClaimedTask']);
const ISSUE='https://github.com/kaicreator-mm/tally/issues/169';
const SHA=/^[0-9a-f]{40}$/;
const POSITIVE_REASON='CLAIM_REGISTRY_PUBLICATION_AND_NATIVE_READBACK_VERIFIED';
const CLAIM_PUBLICATION='REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK';
// Bounded reason vocabulary. Registry-side and pin-mismatch reasons come from
// the provider; GitHub-classified reasons are the existing witness taxonomy.
const deniedReasons=new Set([
  'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','CLAIM_REGISTRY_UNAVAILABLE',
  'CLAIM_REGISTRY_RECORD_NOT_FOUND','CLAIM_REGISTRY_RECORD_INVALID',
  'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH',
  'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','NATIVE_CLAIM_PROVIDER_THREW',
  'UNVERIFIED_NATIVE_CLAIM_PROVIDER_OUTPUT',
  'GITHUB_PROVIDER_TOKEN_UNAVAILABLE','GITHUB_UNAUTHENTICATED',
  'GITHUB_ACCESS_DENIED','GITHUB_COMMENT_NOT_FOUND','GITHUB_RATE_LIMITED',
  'GITHUB_COMMENT_READ_FAILED','INVALID_EXACT_CLAIM_WITNESS_REQUEST',
  'NATIVE_CLAIM_EVENT_SOURCE_UNVERIFIED','PROTECTED_CLAIM_KEY_NOT_UNIQUE',
  'NATIVE_CLAIM_EVENT_RESERVATION_MISMATCH','NATIVE_COMMENT_CURRENTNESS_NOT_VERIFIED'
]);
const deniedFlags=Object.freeze({authorized:false,admitted:false,taskPass:false,
  agentLaunch:false,releasePass:false});
const denied=(reason)=>Object.freeze({
  state:'NOT_ADMITTED',reason,issueRef:ISSUE,sourceSha:null,
  claimPublication:'NOT_VERIFIED',claimAccepted:false,...deniedFlags});
const allDenied=(value)=>value?.authorized===false&&value?.admitted===false&&
  value?.taskPass===false&&value?.agentLaunch===false&&value?.releasePass===false;
const noAuthority=(value)=>value?.authorized===false&&value?.taskPass===false&&
  value?.agentLaunch===false&&value?.releasePass===false;
const claimKey=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+#169:builder:__default__$/;

export async function admitClaimedTask(input) {
  // Provider exceptions (including getters and arbitrary Error.message text)
  // are untrusted. Never allow raw errors, token, or raw comment bodies into
  // Kernel effect journals; rebuild every field from a reviewed vocabulary.
  try {
    const result=await nativeAdmitClaimedTask(input);
    if(result?.state==='ADMITTED_PENDING_AGENT_EFFECT'){
      const positive=
        result.reason===POSITIVE_REASON&&
        result.issueRef===ISSUE&&
        result.sourceRef===undefined&&
        SHA.test(result.sourceSha??'')&&
        typeof result.protectedClaimKey==='string'&&
        claimKey.test(result.protectedClaimKey)&&
        result.generation===1&&
        typeof result.dispatchId==='string'&&result.dispatchId.length>0&&
        result.dispatchId.length<=128&&
        result.actorRole==='builder'&&
        typeof result.operatorKind==='string'&&result.operatorKind.length>0&&
        result.operatorKind.length<=32&&
        typeof result.operatorId==='string'&&result.operatorId.length>0&&
        result.operatorId.length<=128&&
        Number.isSafeInteger(result.commentId)&&result.commentId>0&&
        result.commentRef===`${ISSUE}#issuecomment-${result.commentId}`&&
        result.claimPublication===CLAIM_PUBLICATION&&
        result.claimAccepted===true&&result.admitted===true&&noAuthority(result);
      if(!positive)return denied('UNVERIFIED_NATIVE_CLAIM_PROVIDER_OUTPUT');
      return Object.freeze({
        state:'ADMITTED_PENDING_AGENT_EFFECT',reason:result.reason,
        issueRef:ISSUE,sourceSha:result.sourceSha,
        protectedClaimKey:result.protectedClaimKey,generation:1,
        dispatchId:result.dispatchId,actorRole:'builder',
        operatorKind:result.operatorKind,operatorId:result.operatorId,
        commentId:result.commentId,commentRef:result.commentRef,
        claimPublication:CLAIM_PUBLICATION,claimAccepted:true,
        admitted:true,authorized:false,taskPass:false,
        agentLaunch:false,releasePass:false
      });
    }
    if(result?.state==='NOT_ADMITTED'&&
       deniedReasons.has(result.reason)&&allDenied(result)&&
       (result.sourceSha===null||(SHA.test(result.sourceSha??'')))){
      return denied(result.reason);
    }
    return denied('UNVERIFIED_NATIVE_CLAIM_PROVIDER_OUTPUT');
  } catch {
    return denied('NATIVE_CLAIM_PROVIDER_THREW');
  }
}
