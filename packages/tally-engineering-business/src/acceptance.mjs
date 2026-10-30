/** Derived AcceptanceItem evidence ledger. No Release authority or second Task DB. */
const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const isoTime=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const validSha=value=>SHA.test(value??'');
const validState=(value,expected)=>value===expected;
const realReview=review=>review?.sourceReview==='PASS'&&Number.isSafeInteger(review.pipeline)&&review.pipeline>0;
const baseItem=(vector,behavior,evidence,status,coverage,refs)=>({vector,behavior,
  status,coverage,mustRetain:'PRESERVE',released:false,taskPass:false,hiddenPass:false,
  observedAsOf:evidence.observedAsOf,evidenceRefs:refs});
function sourceReady(status,mergedSha,reviewSha,pipeline){
  return status==='PASS'&&validSha(mergedSha)&&validSha(reviewSha)&&realReview({sourceReview:'PASS',pipeline});
}
function realAttempt(attempt,baseSha){
  return attempt?.state==='LOCAL_CANDIDATE_READY'&&attempt.exitCode===0&&
    ATTEMPT.test(attempt.attemptId??'')&&validSha(attempt.tree)&&validSha(attempt.commit)&&
    validSha(attempt.candidateCommit)&&attempt.commit===attempt.candidateCommit&&validSha(baseSha)&&
    typeof attempt.worktree==='string'&&attempt.worktree.trim().length>0&&
    typeof attempt.writeSet==='string'&&attempt.writeSet.trim().length>0&&
    attempt.baseSha===baseSha&&attempt.confined===true;
}
export function buildAcceptanceItems(evidence){
  const observedAsOf=evidence?.observedAsOf;
  if(!isoTime(observedAsOf))return [baseItem('UNKNOWN','OBSERVATION_TIME_UNAVAILABLE',evidence,
    'UNKNOWN','UNKNOWN',null)];
  const t06m=evidence.t06m??{},t06n=evidence.t06n??{};
  const mSource=sourceReady(t06m.sourceReview,t06m.mergedSha,t06m.reviewSha,t06m.pipeline);
  const nSource=sourceReady(t06n.sourceReview,t06n.mergedSha,t06n.reviewSha,t06n.pipeline);
  const mAttempt=realAttempt(t06m.attempt,t06m.baseSha);
  const mConfined=t06m.confinement?.operatorCredentialsDenied===true&&
    t06m.confinement.remoteWriteDenied===true&&t06m.confinement.builderPrincipal;
  const a=t06n.attemptA,b=t06n.attemptB;
  const disjoint=realAttempt(a,t06n.baseSha)&&realAttempt(b,t06n.baseSha)&&
    a.attemptId!==b.attemptId&&a.worktree!==b.worktree&&a.writeSet!==b.writeSet&&
    a.candidateCommit!==b.candidateCommit;
  const joined=t06n.reviewJoin==='REVIEW_JOINED'&&t06n.repair==='LOCAL_CANDIDATE_READY'&&
    t06n.rejoin==='REVIEW_JOINED'&&Number.isSafeInteger(t06n.overlapMs)&&t06n.overlapMs>0;
  const overlapDenial=t06n.overlapDenial==='CANDIDATE_WRITESET_OVERLAP';
  const realOverlap=disjoint&&joined&&overlapDenial;
  const sourceRefs={pr:t06m.pr,pipeline:t06m.pipeline,mergedSha:t06m.mergedSha,
    reviewSha:t06m.reviewSha,sourceReview:t06m.sourceReview,attempt:t06m.attempt};
  const concurrentRefs={pr:t06n.pr,pipeline:t06n.pipeline,mergedSha:t06n.mergedSha,
    reviewSha:t06n.reviewSha,sourceReview:t06n.sourceReview,attemptA:t06n.attemptA,attemptB:t06n.attemptB,
    overlapMs:t06n.overlapMs,overlapDenial:t06n.overlapDenial,
    reviewJoin:t06n.reviewJoin,repair:t06n.repair,rejoin:t06n.rejoin};
  const items=[
    baseItem('V52','Real installed Codex start, terminal and exit receipt',evidence,
      mSource&&mAttempt&&mConfined?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED',
      'PARTIAL',sourceRefs),
    baseItem('V54','Exact-base disposable worktree and independent candidate tree/commit readback',
      evidence,mSource&&mAttempt?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','PARTIAL',sourceRefs),
    baseItem('V56','Disjoint worktrees/write grants and overlapping-write denial',evidence,
      nSource&&disjoint&&overlapDenial?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','FULL',concurrentRefs),
    baseItem('V63','Two independently claimed real Builder Tasks with actual process overlap and Reviewer Join',
      evidence,nSource&&realOverlap?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','FULL',concurrentRefs),
    baseItem('V66','Least-privilege Builder principal and operator credential/remote-write denial',
      evidence,mSource&&mAttempt&&mConfined?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','PARTIAL',
      {...sourceRefs,hiddenVerifier:'NOT_QUALIFIED'}),
    baseItem('V72','Two real attempts with distinct permitted write sets, accepted Claims, overlap and Reviewer Join',
      evidence,nSource&&realOverlap?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','FULL',concurrentRefs),
    baseItem('V73','Overlapping candidate/write-set Join refused while disjoint pair joined',
      evidence,nSource&&realOverlap?'PHYSICAL_BEHAVIOR_PROVEN':'SOURCE_IMPLEMENTED','FULL',concurrentRefs)
  ];
  if(!evidence.t06m||!evidence.t06n)
    return items.map(item=>Object.freeze({...item,status:'UNKNOWN',coverage:'UNKNOWN'}));
  return items.map(item=>Object.freeze(item));
}
export function acceptanceCoverage(items){
  const counts={};
  for(const item of items)counts[item.status]=(counts[item.status]??0)+1;
  return Object.freeze({items:counts,comparison:'COUNTS_NOT_PERCENTAGES',released:false});
}
export function formatAcceptanceTable(items){
  const lines=['| Vector | Delivered behavior | Status | Coverage | Evidence |','| --- | --- | --- | --- | --- |'];
  for(const item of items){
    const refs=Number.isSafeInteger(item.evidenceRefs?.pr)&&Number.isSafeInteger(item.evidenceRefs?.pipeline)
      ?`PR #${item.evidenceRefs.pr} / pipeline ${item.evidenceRefs.pipeline}`:'UNKNOWN';
    lines.push(`| ${item.vector} | ${item.behavior} | ${item.status} | ${item.coverage} | ${refs} |`);
  }
  return lines.join('\n');
}
