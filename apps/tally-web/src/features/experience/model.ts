export type ExperienceState='PROPOSED'|'REVIEWED'|'ADOPTED'|'REJECTED';
export type ReportQualification='VERIFIED'|'STALE'|'UNVERIFIED'|'UNKNOWN';
export interface ExperienceCandidate{
  id:string;revision:number;title:string;state:ExperienceState;
  reportQualification:ReportQualification;sourceRef:string;
  publisherRepository?:string|null;
  sourceKind?:'PUBLIC_ADS_REPORT'|'ISOLATED_E2E_TEST_ONLY';
  eligibleForNextTask:boolean;
}
export interface ExperienceSnapshot{
  connected:boolean;versionId:string;
  source?:{repository:string;sourceSha:string;currentness:string;readback:string};
  qualification?:'SOURCE_OBSERVED_NOT_MEASURED_OUTCOME';
  candidates?:readonly ExperienceCandidate[];
}
export interface ExperienceDecisionReceipt{
  connected:boolean;ok:boolean;status:'OBSERVED'|'DENIED'|'UNKNOWN'|'RECONCILIATION_REQUIRED';
  candidateId?:string;sourceSha?:string;receiptRef?:string|null;
  nextTaskUsage?:'NOT_VERIFIED';measuredBenefit?:'UNKNOWN';
  reason?:string;
}
export interface ExperienceDecisionGate{
  eligible:boolean;canAdopt:boolean;canReject:boolean;reason:string;
}
const deniedDecision=(reason:string):ExperienceDecisionGate=>({
  eligible:false,canAdopt:false,canReject:false,reason
});
export function eligibleExperienceDecision(snapshot:ExperienceSnapshot|null,
  candidate:ExperienceCandidate|null,repository:string,versionId:string):ExperienceDecisionGate{
  if(snapshot?.connected!==true||snapshot.source?.repository!==repository||
     !/^[a-f0-9]{40}$/.test(snapshot.source.sourceSha??'')||
     snapshot.source.currentness!=='CURRENT'||snapshot.source.readback!=='VERIFIED'||
     snapshot.versionId!==versionId||!Array.isArray(snapshot.candidates)||
     new Set(snapshot.candidates.map(c=>c.id)).size!==snapshot.candidates.length)
    return deniedDecision('EXPERIENCE_SOURCE_NOT_CURRENT');
  const sourceRef=typeof candidate?.sourceRef==='string'?
    /^https:\/\/github\.com\/([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)\/issues\/[1-9][0-9]*(?:#issuecomment-[1-9][0-9]*)?$/.exec(candidate.sourceRef):null;
  const publisherRepository=candidate?.publisherRepository??repository;
  if(!candidate||!/^[a-f0-9]{64}$/.test(candidate.id??'')||
     !Number.isSafeInteger(candidate.revision)||candidate.revision<1||
     !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(publisherRepository)||
     !sourceRef||sourceRef[1]!==publisherRepository)
    return deniedDecision('EXPERIENCE_EVIDENCE_NOT_ELIGIBLE');
  const current=snapshot.candidates.find(c=>c.id===candidate.id);
  if(!current||current.revision!==candidate.revision||
     current.state!==candidate.state||
     current.reportQualification!==candidate.reportQualification||
     current.sourceRef!==candidate.sourceRef||
     current.publisherRepository!==candidate.publisherRepository||
     current.sourceKind!==candidate.sourceKind||
     current.eligibleForNextTask!==candidate.eligibleForNextTask||
     !['PROPOSED','REVIEWED'].includes(candidate.state)||
     candidate.eligibleForNextTask)
    return deniedDecision('EXPERIENCE_EVIDENCE_NOT_ELIGIBLE');
  const testOnly=candidate.sourceKind==='ISOLATED_E2E_TEST_ONLY'||
    publisherRepository==='kaicreator-mm/task-meta-e2e';
  const canAdopt=candidate.reportQualification==='VERIFIED'&&!testOnly;
  return {eligible:true,canAdopt,canReject:true,
    reason:testOnly?'ISOLATED_E2E_TEST_ONLY':
      canAdopt?'HUMAN_DECISION_REQUIRED':'UNQUALIFIED_REPORT_REJECT_ONLY'};
}
export function experienceOutcomeLabel(candidate:ExperienceCandidate){
  if(candidate.state==='ADOPTED')return '已采纳 · 等待下一 Task 实际使用证据';
  if(candidate.state==='REJECTED')return '已拒绝';
  if(candidate.reportQualification!=='VERIFIED')return '来源未验证';
  return '待人工评审';
}
