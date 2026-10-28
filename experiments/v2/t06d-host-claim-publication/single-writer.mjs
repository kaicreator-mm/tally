/**
 * T06d — Tally-owned Host single-writer CAS + native GitHub publish reference.
 *
 * Not a public HTTP endpoint, scheduler or DomainHarness Kernel substitute.
 * All trust ports come from a selected/privileged Host: public v0.10.2
 * CasDocumentStore, verified native GitHub C, an authorized sole publisher.
 * Callers must NEVER be allowed to supply these ports or writer authority.
 *
 * Physical CAS reservation alone is NOT an accepted ADS Task Claim. Only
 * exact native GitHub after-state plus the serialized writer's own reservation
 * may qualify one recorded Start event; Kernel execution admission is separate.
 */
import {createHash} from 'node:crypto';

const NAME=/^[A-Za-z0-9_.-]+$/;
const SHA=/^[0-9a-f]{40}$/;
const ROLES=new Set(['builder','reviewer','validator']);
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*$/;
const ADR=/^#[1-9][0-9]*@[1-9][0-9]*$/;
const allowed=x=>typeof x==='string'&&x.length>=1&&x.length<=128&&
  /^[a-zA-Z0-9][a-zA-Z0-9:._/-]*$/.test(x);
const deny=(reason,extra={})=>Object.freeze({
  status:'BLOCKED',reason,claimPublication:'NOT_ACCEPTED',
  authorized:false,admitted:false,taskPass:false,releasePass:false,
  ...extra
});
const keyOf=(repo,number,role,group='__default__')=>{
  const raw=`${repo}#${number}:${role}:${group}`;
  return 'tally:v2:claim-admission:'+createHash('sha256').update(raw).digest('hex');
};
function safeRequest(q){
  if(!q||typeof q!=='object'||!NAME.test(q.owner??'')||
    !NAME.test(q.repo??'')||!Number.isSafeInteger(q.issueNumber)||q.issueNumber<1||
    !ROLES.has(q.actorRole)||!SHA.test(q.sourceSha??'')||
    !allowed(q.dispatchId)||!allowed(q.operatorId)||!allowed(q.operatorKind)||
    !Number.isSafeInteger(q.expectedGeneration)||q.expectedGeneration!==0||
    (q.compatibilityGroup!==undefined&&q.compatibilityGroup!=='__default__'))
    return null;
  const repository=`${q.owner}/${q.repo}`;
  return Object.freeze({
    repository,issueNumber:q.issueNumber,
    issueRef:`https://github.com/${repository}/issues/${q.issueNumber}`,
    sourceSha:q.sourceSha,actorRole:q.actorRole,
    operatorKind:q.operatorKind,operatorId:q.operatorId,
    dispatchId:q.dispatchId,generation:1,
    protectedClaimKey:`${repository}#${q.issueNumber}:${q.actorRole}:__default__`,
  });
}
function validPreflight(o,i){
  // These are Host-only source-derived facts (never body/client input).
  // Actual GitHub read adapter is not yet qualified as a complete CAS
  // absence witness; production refuses unless that separate seam exists.
  return o&&o.sourceKind==='NATIVE_GITHUB_CURRENT_ADMISSION'&&
    o.sourceQualified===true&&o.complete===true&&
    o.issueRef===i.issueRef&&o.sourceSha===i.sourceSha&&
    o.issueState==='open'&&o.workflowState==='state:ready'&&
    o.dependencyState==='CLEAR'&&o.activeIncompatibleClaimCount===0&&
    o.dispatchSourceBound===true&&o.publisherSourceBound===true;
}
function validReadback(o,i){
  return o&&o.sourceKind==='NATIVE_GITHUB_EXACT_READBACK'&&
    o.complete===true&&o.edited===false&&
    o.issueRef===i.issueRef&&o.sourceSha===i.sourceSha&&
    o.dispatchId===i.dispatchId&&o.actorRole===i.actorRole&&
    o.operatorId===i.operatorId&&o.operatorKind===i.operatorKind&&
    o.generation===i.generation&&o.protectedClaimKey===i.protectedClaimKey&&
    o.event==='DISPATCH_CLAIMED'&&o.dispatchState==='CLAIMED'&&
    Number.isSafeInteger(o.commentId)&&o.commentId>0&&
    o.commentRef===`${i.issueRef}#issuecomment-${o.commentId}`;
}
/**
 * In-process reference for a Host-owned singleton admission writer. Not
 * suitable for multiple hosts without one shared CAS store + sole publisher.
 * Test adapters may simulate source and publisher; they cannot certify a
 * production serialized writer or actual GitHub accepted Claim.
 */
export function createHostSingleWriterClaimController({store,source,publisher,writerAuthority}={}){
  if(!store||typeof store.get!=='function'||typeof store.put!=='function'||
    !source||typeof source.readAdmissionFacts!=='function'||
    typeof source.readPublishedClaim!=='function'||
    !publisher||typeof publisher.publishClaim!=='function'||
    writerAuthority?.mode!=='SINGLE_WRITER_ADMISSION'||
    writerAuthority?.exclusiveHostControl!==true||
    !allowed(writerAuthority?.writerId))
    throw new TypeError('TRUSTED_SINGLE_WRITER_HOST_PORTS_REQUIRED');
  async function reserveAndPublish(request){
    let i;
    try{i=safeRequest(request);}catch{return deny('INVALID_CLAIM_REQUEST');}
    if(!i)return deny('INVALID_CLAIM_REQUEST');
    const key=keyOf(i.repository,i.issueNumber,i.actorRole);
    try{
      const existing=await store.get(key);
      // Ambiguous/crashed publication and old claims MUST NOT be overwritten.
      // Same-dispatch resumption requires a separate durable reconciliation.
      if(existing)return deny(existing.value?.dispatchId===i.dispatchId?
        'EXISTING_CLAIM_NEEDS_NATIVE_RECONCILIATION':'INCOMPATIBLE_OR_PENDING_CLAIM_EXISTS');
      const facts=await source.readAdmissionFacts(i);
      if(!validPreflight(facts,i))return deny('SOURCE_OR_CLAIM_AUTHORITY_UNQUALIFIED');
      const reservation={phase:'PENDING_GITHUB_PUBLISH',generation:i.generation,
        protectedClaimKey:i.protectedClaimKey,repository:i.repository,
        issueRef:i.issueRef,sourceSha:i.sourceSha,dispatchId:i.dispatchId,
        actorRole:i.actorRole,operatorKind:i.operatorKind,
        operatorId:i.operatorId,writerId:writerAuthority.writerId};
      // Exactly ONE physical CAS succeeds for a protected key, even if two
      // requests simultaneously passed their preceding native source read.
      let rev;
      try{({revision:rev}=await store.put(key,reservation,{mustBeAbsent:true}));}
      catch{return deny('CONCURRENT_CLAIM_RESERVATION_LOST');}
      // Always preserve reservation if GitHub publication ACK is lost.
      // Never automatically retry an uncertain publish.
      let ack;
      try{ack=await publisher.publishClaim(Object.freeze({...i,writerId:writerAuthority.writerId}));}
      catch{return deny('GITHUB_CLAIM_PUBLICATION_AMBIGUOUS',{reservationKey:key});}
      const id=ack?.commentId;
      if(!Number.isSafeInteger(id)||id<1)
        return deny('GITHUB_CLAIM_PUBLICATION_AMBIGUOUS',{reservationKey:key});
      let observed;
      try{observed=await source.readPublishedClaim(i,id);}
      catch{return deny('GITHUB_CLAIM_READBACK_UNAVAILABLE',{reservationKey:key});}
      if(!validReadback(observed,i)||observed.commentId!==id)
        return deny('GITHUB_CLAIM_READBACK_NOT_VERIFIED',{reservationKey:key});
      // A durable accepted publication record is **not** Task execution.
      const result={...reservation,phase:'PUBLISHED_VERIFIED',
        commentId:id,commentRef:observed.commentRef};
      try{await store.put(key,result,{mustMatchRevision:rev});}
      catch{return deny('CLAIM_PERSISTENCE_AMBIGUOUS',{reservationKey:key});}
      return Object.freeze({
        status:'RECORDED',reason:'SERIALIZED_GITHUB_PUBLISH_AND_READBACK_CONFIRMED',
        claimPublication:'VERIFIED_PENDING_KERNEL_ADMISSION',
        nativeCommentRef:observed.commentRef,protectedClaimKey:i.protectedClaimKey,
        generation:i.generation,authorized:false,admitted:false,
        taskPass:false,releasePass:false
      });
    }catch{return deny('ADMISSION_STORE_OR_SOURCE_UNAVAILABLE');}
  }
  return Object.freeze({reserveAndPublish});
}
