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
import {TARGET} from '../../../packages/tally-github-native-c/src/task-dispatch-state-events.mjs';

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
  async function terminalizePublishedClaim(request){
    const unknown=reason=>Object.freeze({status:'UNKNOWN',reason,claimTerminality:'UNKNOWN',
      authorized:false,admitted:false,taskPass:false,releasePass:false});
    if(request!==undefined||typeof source.readTerminalizationPreflight!=='function'||
      typeof source.readTerminalEvent!=='function'||typeof publisher.publishDispatchStateChange!=='function')
      return unknown('TRUSTED_TERMINAL_HOST_PORTS_REQUIRED');
    try{
      const row=await store.get(TARGET.casKey),v=row?.value;
      if(row?.revision!==1||v?.phase!=='PUBLISHED_VERIFIED'||v.repository!==TARGET.repository||
        v.issueRef!==TARGET.issueRef||v.sourceSha!==TARGET.sourceSha||v.dispatchId!==TARGET.dispatchId||
        v.protectedClaimKey!==TARGET.protectedClaimKey||v.generation!==1||v.actorRole!=='builder'||
        v.operatorKind!=='other'||v.operatorId!=='zcode:candidate-publication:1'||
        v.writerId!=='tally-host:win32:candidate-publication:1'||v.commentId!==TARGET.claimCommentId||
        v.commentRef!==TARGET.claimCommentRef)return unknown('TARGET_CLAIM_NOT_EXACT');
      const facts=await source.readTerminalizationPreflight(row);
      if(facts?.sourceKind!=='HOST_TERMINALIZATION_PREFLIGHT'||facts.complete!==true||
        facts.targetRevision!==row.revision||typeof facts.evidence!=='string'||!facts.evidence||
        !Number.isFinite(Date.parse(facts.occurredAt))||Date.now()-Date.parse(facts.occurredAt)>30000||
        Date.parse(facts.occurredAt)>Date.now())return unknown('TERMINAL_PREFLIGHT_UNQUALIFIED');
      const reservation={...v,phase:'PENDING_GITHUB_TERMINALIZATION',
        terminalWriterId:writerAuthority.writerId,terminalEvidence:facts.evidence,
        terminalOccurredAt:facts.occurredAt,terminalReservationRevision:row.revision+1};
      let revision;
      try{({revision}=await store.put(TARGET.casKey,reservation,{mustMatchRevision:row.revision}));}
      catch{return unknown('TERMINAL_RESERVATION_RACE');}
      const reserved=await store.get(TARGET.casKey);
      if(revision!==reservation.terminalReservationRevision||reserved?.revision!==revision||
        JSON.stringify(reserved.value)!==JSON.stringify(reservation))return unknown('TERMINAL_RESERVATION_UNVERIFIED');
      const recheck=await source.readTerminalizationPreflight(row);
      if(recheck?.complete!==true||recheck.targetRevision!==row.revision||
        !Number.isFinite(Date.parse(recheck.occurredAt))||Date.now()-Date.parse(recheck.occurredAt)>30000||
        Date.parse(recheck.occurredAt)>Date.now()||!facts.fenceId||recheck.fenceId!==facts.fenceId)
        return unknown('TERMINAL_PREFLIGHT_CHANGED');
      const beforePost=await store.get(TARGET.casKey);
      if(beforePost?.revision!==revision||JSON.stringify(beforePost.value)!==JSON.stringify(reservation))
        return unknown('TERMINAL_RESERVATION_CHANGED');
      let ack;
      try{ack=await publisher.publishDispatchStateChange(Object.freeze({...reservation}));}
      catch{return unknown('TERMINAL_PUBLICATION_AMBIGUOUS');}
      if(!Number.isSafeInteger(ack?.commentId)||ack.commentId<1||
        ack.commentRef!==`${TARGET.issueRef}#issuecomment-${ack.commentId}`)
        return unknown('TERMINAL_PUBLICATION_AMBIGUOUS');
      const observed=await source.readTerminalEvent(reservation);
      if(observed?.status!=='EXACT'||observed.complete!==true||observed.edited!==false||
        observed.commentId!==ack.commentId||observed.commentRef!==ack.commentRef)
        return unknown('TERMINAL_READBACK_UNVERIFIED');
      const terminal={...reservation,phase:'TERMINAL_VERIFIED',
        terminalCommentId:ack.commentId,terminalCommentRef:ack.commentRef};
      let finalRevision;
      try{({revision:finalRevision}=await store.put(TARGET.casKey,terminal,{mustMatchRevision:revision}));}
      catch{return unknown('TERMINAL_CONFIRMATION_AMBIGUOUS');}
      const final=await store.get(TARGET.casKey);
      if(final?.revision!==finalRevision||JSON.stringify(final.value)!==JSON.stringify(terminal))
        return unknown('TERMINAL_FINAL_READBACK_UNVERIFIED');
      return Object.freeze({status:'RECORDED',claimTerminality:'VERIFIED',
        terminalCommentRef:ack.commentRef,authorized:false,admitted:false,taskPass:false,releasePass:false});
    }catch{return unknown('TERMINAL_HOST_SOURCE_UNAVAILABLE');}
  }
  return Object.freeze({reserveAndPublish,terminalizePublishedClaim});
}
