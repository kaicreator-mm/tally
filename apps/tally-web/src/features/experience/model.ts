export type ExperienceState='PROPOSED'|'REVIEWED'|'ADOPTED'|'REJECTED';
export type ReportQualification='VERIFIED'|'STALE'|'UNVERIFIED'|'UNKNOWN';
export interface ExperienceCandidate{
  id:string;revision:number;title:string;state:ExperienceState;
  reportQualification:ReportQualification;sourceRef:string;
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
export function eligibleExperienceDecision(snapshot:ExperienceSnapshot|null,
  candidate:ExperienceCandidate|null,repository:string,versionId:string){
  if(snapshot?.connected!==true||snapshot.source?.repository!==repository||
     !/^[a-f0-9]{40}$/.test(snapshot.source.sourceSha??'')||
     snapshot.source.currentness!=='CURRENT'||snapshot.source.readback!=='VERIFIED'||
     snapshot.versionId!==versionId||!Array.isArray(snapshot.candidates)||
     new Set(snapshot.candidates.map(c=>c.id)).size!==snapshot.candidates.length)
    return {eligible:false,reason:'EXPERIENCE_SOURCE_NOT_CURRENT'};
  if(!candidate||!/^[a-f0-9]{64}$/.test(candidate.id??'')||
     !Number.isSafeInteger(candidate.revision)||candidate.revision<1||
     typeof candidate.sourceRef!=='string'||
     !candidate.sourceRef.startsWith('https://github.com/'+repository+'/issues/')||
     !snapshot.candidates.some(c=>c.id===candidate.id&&
       c.revision===candidate.revision)||candidate.state!=='PROPOSED'&&
     candidate.state!=='REVIEWED'||candidate.reportQualification!=='VERIFIED')
    return {eligible:false,reason:'EXPERIENCE_EVIDENCE_NOT_ELIGIBLE'};
  if(candidate.eligibleForNextTask)return {eligible:false,reason:'ALREADY_MARKED_FOR_NEXT_TASK'};
  return {eligible:true,reason:'HUMAN_DECISION_REQUIRED'};
}
export function experienceOutcomeLabel(candidate:ExperienceCandidate){
  if(candidate.state==='ADOPTED')return '已采纳 · 等待下一 Task 实际使用证据';
  if(candidate.state==='REJECTED')return '已拒绝';
  if(candidate.reportQualification!=='VERIFIED')return '来源未验证';
  return '待人工评审';
}
