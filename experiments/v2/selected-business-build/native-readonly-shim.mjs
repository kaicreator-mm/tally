/**
 * T04b BUILD-sealed Capability ABI + safe projection.
 *
 * The producer substitutes the source SHA256 into the literal below before
 * calling public BUILD. Trusted Host MUST verify installed npm provider bytes
 * against that sealed value BEFORE Microkernel.load(), not at/after import.
 * This is deployment integrity, not a claim of arbitrary-JS process isolation.
 */
import {observeReadiness as nativeObserveReadiness}
  from '@kaicreator/tally-native-reader-bundle';

export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-source-observation@v2.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-source-observation';
export const OPERATIONS=Object.freeze(['tally.observeReadiness']);
const SOURCE=/^[0-9a-f]{40}$/;
const statuses=new Set(['CURRENT_OBSERVED','UNKNOWN','PARTIAL','STALE','INVALID_OBSERVATION','CONFLICT','UNSUPPORTED_SCHEMA']);
const reasons=new Set([
  'GITHUB_NATIVE_REPORTS_OBSERVED','ADVISORY_REPORTS_NOT_CURRENT',
  'NO_ADS_V5_REPORTS','GITHUB_PROVIDER_TOKEN_UNAVAILABLE',
  'GITHUB_NATIVE_READ_FAILED','GITHUB_ACCESS_DENIED','GITHUB_RESOURCE_NOT_FOUND',
  'GITHUB_UNAUTHENTICATED','GITHUB_RATE_LIMITED','INVALID_NATIVE_QUERY',
  'PUBLISHER_BINDINGS_UNAVAILABLE','NATIVE_ISSUE_OR_COMMIT_MISMATCH',
  'INVALID_GITHUB_PAGE','GITHUB_READ_LIMIT_REACHED',
  'INVALID_OR_UNVERIFIED_REPORT_COMMENT'
]);
const refused=(reason)=>Object.freeze({state:'NATIVE_NOT_CURRENT',status:'UNKNOWN',
  reason,sourceRef:null,sourceSha:null,observationCount:0,
  authorized:false,admitted:false,taskPass:false,releasePass:false});
export async function observeReadiness(input) {
  const result=await nativeObserveReadiness(input);
  if(!result||typeof result!=='object'||Array.isArray(result)||
     !statuses.has(result.status)||!reasons.has(result.reason)||
     result.sourceRef!=='https://github.com/kaicreator-mm/tally/issues/111'||
     !SOURCE.test(result.sourceSha??'')||
     result.sourceSha!==input?.sourceSha||
     !Number.isSafeInteger(result.observationCount)||
     result.observationCount<0||result.observationCount>1000) {
    // Forbid provider-controlled raw results from entering the K Journal.
    // In particular: never forward token, raw comments, or granted booleans.
    return refused('UNVERIFIED_NATIVE_PROVIDER_OUTPUT');
  }
  return Object.freeze({
    state:result.status==='CURRENT_OBSERVED'?'NATIVE_ADVISORY_OBSERVED':'NATIVE_NOT_CURRENT',
    status:result.status,reason:result.reason,
    sourceRef:result.sourceRef,sourceSha:result.sourceSha,
    observationCount:result.observationCount,
    authorized:false,admitted:false,taskPass:false,releasePass:false
  });
}
