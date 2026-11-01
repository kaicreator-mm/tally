/** UI-D #231 — defect intake, next-version planning and measured learning.
 * No direct digest/Claim/agent-rank entry: users describe defects in natural
 * language; dedup and casebook linkage come from the Host read model.
 */
export interface DefectRecord{
  id:string;
  summary:string;
  releaseVersion:string|null;
  releaseCommitSha:string|null;
  requirementRef:string|null;
  state:'OPEN'|'TRIAGED'|'REPAIRED_IN_VERSION'|'DEFERRED_NEXT_VERSION'|'UNKNOWN';
  /** In-version repair or next-version candidate (Owner full-chain rule). */
  routedTo:'repair'|'next_version'|null;
  duplicateOf:string|null;
  casebookRefs:readonly string[];
  observedAt:string;
}
export function dedupeDefects(defects:readonly DefectRecord[]):{
  unique:DefectRecord[];duplicates:DefectRecord[];}{
  const seen=new Map<string,DefectRecord>();
  const unique:DefectRecord[]=[],duplicates:DefectRecord[]=[];
  for(const defect of defects){
    if(defect.duplicateOf){duplicates.push(defect);continue;}
    const key=defect.summary.trim().toLowerCase();
    const existing=seen.get(key);
    if(existing&&existing.id!==defect.id){duplicates.push(defect);continue;}
    seen.set(key,defect);unique.push(defect);
  }
  return {unique,duplicates};
}
/** Route suggestion: repair in current version only when the defect cites the
 * exact release identity; otherwise next-version candidate. */
export function suggestRouting(defect:DefectRecord):
  {routedTo:'repair'|'next_version';reason:string}{
  if(defect.releaseVersion&&defect.releaseCommitSha)
    return {routedTo:'repair',reason:'缺陷绑定精确发布身份,建议本版 Repair'};
  return {routedTo:'next_version',
    reason: defect.releaseCommitSha
      ?'缺少发布版本号,建议进入下一版本候选'
      :'未绑定精确发布身份,建议进入下一版本候选'};
}
export interface LearningRecord{
  id:string;
  decision:'ADOPTED'|'REJECTED';
  outcomeAttested:boolean|null; // null = NOT_MEASURED (never inferred)
  observedAt:string|null;
  note:string|null;
}
/** Quality gain is never inferred: only attested outcomes are shown, with an
 * explicit not-measured state. */
export function learningDisplay(record:LearningRecord):
  {decision:string;effect:'ATTESTED'|'NOT_MEASURED'|'UNKNOWN'}{
  return {decision:record.decision,
    effect:record.outcomeAttested===null?'NOT_MEASURED'
      :record.outcomeAttested?'ATTESTED':'NOT_MEASURED'};
}
/** Next-version candidate intake state machine (UX only; admission is the
 * Host's): draft → submitted → admitted | needs-info. */
export type IntakeState='draft'|'submitted'|'admitted'|'needs_info';
export function nextIntakeState(current:IntakeState,event:'submit'|'admit'
  |'needs_info'|'edit'):IntakeState|null{
  if(current==='draft'&&event==='submit')return 'submitted';
  if(current==='submitted'&&event==='admit')return 'admitted';
  if(current==='submitted'&&event==='needs_info')return 'needs_info';
  if(current==='needs_info'&&event==='edit')return 'draft';
  return null; // illegal transitions refused client-side too
}
