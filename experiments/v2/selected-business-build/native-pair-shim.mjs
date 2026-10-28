/**
 * T06f BUILD sealed C: allowlisted, unprivileged Task source pair projection.
 * Host pre-load checks exact provider bytes against NATIVE_PROVIDER_SHA256.
 */
import {observeReadiness as readPair} from '@kaicreator/tally-native-reader-bundle';
export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-source-observation@v2.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-source-observation';
export const OPERATIONS=Object.freeze(['tally.observeReadiness']);
const ISSUE='https://github.com/kaicreator-mm/tally/issues/125';
const SHA=/^[0-9a-f]{40}$/;
const workflow=new Set([
  'state:planned','state:ready','state:claimed','state:implementing',
  'state:review-ready','state:reviewing','state:validation-needed',
  'state:validating','state:changes-requested','state:merge-ready',
  'state:blocked','state:done','state:superseded','state:cancelled',null
]);
const workReasons=new Set([
  'CLAIM_NOT_VERIFIED','CANONICAL_WORK_ITEM_STATE_UNVERIFIED',
  'NATIVE_ISSUE_DEPENDENCIES_BLOCKED','WORK_ITEM_NOT_READY'
]);
const candidateReasons=new Set([
  'NO_DISPATCH_CLAIM_EVENTS','CLAIM_EVENT_CANDIDATES_NOT_CAS_ATTESTED'
]);
const denied=reason=>Object.freeze({
  state:'NATIVE_NOT_CURRENT',status:'UNKNOWN',reason,
  sourceRef:null,sourceSha:null,observationCount:0,
  claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
  taskPass:false,releasePass:false
});
const denies=x=>x?.authorized===false&&x?.admitted===false&&
  x?.taskPass===false&&x?.releasePass===false;
const validWork=(w,sha)=>w&&typeof w==='object'&&!Array.isArray(w)&&
  w.issueRef===ISSUE&&w.sourceSha===sha&&
  workReasons.has(w.reason)&&['CLEAR','BLOCKED'].includes(w.dependencyState)&&
  workflow.has(w.workflowState)&&['type:task','type:planning',null].includes(w.workItemType)&&
  w.claimStatus==='NOT_VERIFIED'&&w.authorized===false&&w.admitted===false&&
  w.taskPass===false&&w.releasePass===false&&
  Number.isSafeInteger(w.blockerCount)&&w.blockerCount>=0&&w.blockerCount<=1000&&
  (w.openBlockerCount===null||
    (Number.isSafeInteger(w.openBlockerCount)&&w.openBlockerCount>=0&&
      w.openBlockerCount<=w.blockerCount));
const validEvents=(c,sha)=>c&&typeof c==='object'&&!Array.isArray(c)&&
  candidateReasons.has(c.reason)&&c.claimStatus==='NOT_VERIFIED'&&
  denies(c)&&Number.isSafeInteger(c.candidateCount)&&c.candidateCount>=0&&
  c.candidateCount<=1000&&Array.isArray(c.candidates)&&
  c.candidates.length===c.candidateCount&&
  c.candidates.every(e=>e&&typeof e==='object'&&!Array.isArray(e)&&
    e.sourceSha===sha&&e.sourceClass==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
    e.claimAccepted===false&&/^observed-comment-[1-9][0-9]{0,15}$/.test(e.dispatchId));
export async function observeReadiness(input){
  try{
    const result=await readPair(input);
    if(result?.status!=='SOURCE_PAIR_OBSERVED'||
       result?.state!=='NATIVE_TASK_SOURCE_ADVISORY'||
       result.reason!=='ACCEPTED_CAS_CLAIM_NOT_VERIFIED'||
       result.sourceRef!==ISSUE||!SHA.test(result.sourceSha??'')||
       result.sourceSha!==input?.sourceSha||
       result.claimStatus!=='NOT_VERIFIED'||!denies(result)||
       !validWork(result.workItem,result.sourceSha)||
       !validEvents(result.claimEvents,result.sourceSha)||
       result.observationCount!==result.claimEvents.candidateCount)
      return denied('UNVERIFIED_NATIVE_PAIR_PROVIDER_OUTPUT');
    // Rebuild all fields from a reviewed finite vocabulary, never spread any
    // untrusted transport object into a Kernel Journal.
    const workItem=Object.freeze({
      issueRef:ISSUE,sourceSha:result.sourceSha,
      reason:result.workItem.reason,issueState:result.workItem.issueState,
      workflowState:result.workItem.workflowState,
      workItemType:result.workItem.workItemType,
      dependencyState:result.workItem.dependencyState,
      blockerCount:result.workItem.blockerCount,
      openBlockerCount:result.workItem.openBlockerCount,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
    const candidates=Object.freeze(result.claimEvents.candidates.map(c=>Object.freeze({
      sourceSha:result.sourceSha,dispatchId:c.dispatchId,
      sourceClass:'OBSERVED_COMMENT_SELF_REPORTED_CLAIM',claimAccepted:false
    })));
    const claimEvents=Object.freeze({
      reason:result.claimEvents.reason,
      candidateCount:candidates.length,candidates,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
    return Object.freeze({
      state:'NATIVE_TASK_SOURCE_ADVISORY',status:'SOURCE_PAIR_OBSERVED',
      reason:'ACCEPTED_CAS_CLAIM_NOT_VERIFIED',
      sourceRef:ISSUE,sourceSha:result.sourceSha,
      observationCount:candidates.length,workItem,claimEvents,
      claimStatus:'NOT_VERIFIED',authorized:false,admitted:false,
      taskPass:false,releasePass:false
    });
  }catch{return denied('NATIVE_PAIR_PROVIDER_THREW');}
}
