/** UI-D #231 — QA / test planning / hidden / installed acceptance view model.
 * Full/Hidden/Critical layers stay independent; self-reported results are
 * never conflated with independent verification.
 */
export type VerificationLayer='fullRegression'|'hiddenValidation'
  |'criticalJourneys'|'installedAcceptance';
export interface QaRecord{
  layer:VerificationLayer;
  /** REPORTED = builder/agent self-report only; INDEPENDENT = verified by a
   * distinct principal/witness; the UI keeps them visually distinct. */
  provenance:'REPORTED'|'INDEPENDENT'|'NOT_RUN'|'UNKNOWN';
  outcome:'PASS'|'FAIL'|'NOT_RUN'|'UNKNOWN';
  witness:string|null;
  evidenceRef:string|null;
  subjectSha:string|null;
  observedAt:string|null;
}
export interface QaGap{code:string;detail:string;}
/** Independent verification gap per layer: reported results never satisfy a
 * layer that requires an independent witness (hidden/installed). */
const independentRequired=new Set<VerificationLayer>(
  ['hiddenValidation','installedAcceptance']);
export function qaGaps(records:readonly QaRecord[]):QaGap[]{
  const byLayer=new Map(records.map(record=>[record.layer,record]));
  const gaps:QaGap[]=[];
  const layers:VerificationLayer[]=['fullRegression','hiddenValidation',
    'criticalJourneys','installedAcceptance'];
  for(const layer of layers){
    const record=byLayer.get(layer);
    if(!record||record.outcome==='NOT_RUN'||record.provenance==='NOT_RUN'){
      gaps.push({code:`${layer}_NOT_RUN`,detail:'该层未执行'});continue;}
    if(record.outcome==='UNKNOWN'||record.provenance==='UNKNOWN'){
      gaps.push({code:`${layer}_UNKNOWN`,detail:'该层结果未知,需对账'});continue;}
    if(record.outcome==='FAIL'){
      gaps.push({code:`${layer}_FAIL`,detail:'该层失败,需修复'});continue;}
    if(independentRequired.has(layer)&&record.provenance!=='INDEPENDENT'){
      gaps.push({code:`${layer}_NEEDS_INDEPENDENT`,
        detail:'该层要求独立见证,自报结果不满足'});continue;}
    if(!record.witness){
      gaps.push({code:`${layer}_NO_WITNESS`,detail:'缺少见证者标识'});continue;}
  }
  return gaps;
}
/** Human QA summary: 1–3 actionable reasons (Owner UX rule), full matrix
 * stays expandable. */
export function qaSummary(records:readonly QaRecord[]):
  {ready:boolean;top:QaGap[];totalGaps:number}{
  const gaps=qaGaps(records);
  return {ready:gaps.length===0,top:gaps.slice(0,3),totalGaps:gaps.length};
}
export function layerTone(record:QaRecord|undefined){
  if(!record||record.outcome==='NOT_RUN')return 'neutral';
  if(record.outcome==='FAIL')return 'danger';
  if(record.outcome==='UNKNOWN'||record.provenance==='UNKNOWN')return 'warn';
  return record.provenance==='INDEPENDENT'?'ok':'info';
}
