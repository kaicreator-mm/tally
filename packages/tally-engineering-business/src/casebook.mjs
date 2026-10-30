/** Bounded Casebook replay/evaluation over existing projections. No analyzer or scoring authority. */
const SHA=/^[0-9a-f]{40}$/;
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull|commit|releases)\/[A-Za-z0-9_.~/#-]+$/;
const iso=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
const array=value=>Array.isArray(value)?value:[];
const validCase=row=>row&&typeof row==='object'&&!Array.isArray(row)&&typeof row.case_id==='string'&&
  row.case_id.length>0&&typeof row.project==='string'&&row.project.length>0&&
  REF.test(row.source_issue_url??'')&&row.source_issue_url.startsWith('https://github.com/kaicreator-mm/')&&
  typeof row.kind==='string'&&row.kind.length>0&&typeof row.version==='string'&&row.version.length>0&&
  iso(row.evidence_cutoff)&&Array.isArray(row.source_refs)&&row.source_refs.every(ref=>REF.test(ref)&&
    ref.startsWith('https://github.com/kaicreator-mm/'))&&
  Array.isArray(row.expected_decisions)&&row.expected_decisions.every(value=>typeof value==='string')&&
  Array.isArray(row.forbidden_decisions)&&row.forbidden_decisions.every(value=>typeof value==='string')&&
  Array.isArray(row.missing_evidence)&&row.missing_evidence.every(value=>typeof value==='string')&&
  row.permitted_inputs&&typeof row.permitted_inputs==='object'&&
  typeof row.permitted_inputs.historicalInputComplete==='boolean'&&iso(row.permitted_inputs.observedAt)&&
  (row.permitted_inputs.sourceSha===null||SHA.test(row.permitted_inputs.sourceSha??''))&&
  Array.isArray(row.permitted_inputs.visibleDecisions)&&
  row.permitted_inputs.visibleDecisions.every(value=>typeof value==='string')&&
  row.joint&&typeof row.joint==='object'&&['SOURCE_MERGED','REAL_COMPLETED','INCOMPLETE','UNKNOWN'].includes(row.joint.timeline)&&
  typeof row.joint.diagnosis==='string'&&typeof row.joint.planning==='string';
function invalidResult(reason){return Object.freeze({status:'UNKNOWN',reason,cases:[],coverage:{
  comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false},advisoryOnly:true});}
/** projections[case_id] = {decisions:string[], observedAt:string, baselineErrors:number}. */
export function replayCasebook(casebook,{mode,asOf,projections={}}={}){
  if(!casebook||casebook.schema!=='tally/casebook-v1'||!Array.isArray(casebook.cases)||
    casebook.cases.length<6||!casebook.cases.every(validCase)||
    new Set(casebook.cases.map(row=>row.case_id)).size!==casebook.cases.length||
    !['RETROSPECTIVE_DIAGNOSIS','HISTORICALLY_BLIND'].includes(mode)||
    !iso(asOf)||!projections||typeof projections!=='object'||Array.isArray(projections))return invalidResult('CASEBOOK_OR_REPLAY_INPUT_INVALID');
  const versions=new Set(casebook.cases.map(row=>row.version)),projects=new Set(casebook.cases.map(row=>row.project??'UNKNOWN'));
  if(versions.size<3||projects.size<2)return invalidResult('CASEBOOK_COVERAGE_INSUFFICIENT');
  const results=[];
  for(const row of casebook.cases){
    const input=row.permitted_inputs,projection=projections[row.case_id];
    const malformedProjection=!projection||!Array.isArray(projection.decisions)||!iso(projection.observedAt)||
      !Number.isSafeInteger(projection.baselineErrors)||projection.baselineErrors<0||
      projection.decisions.some(value=>typeof value!=='string');
    if(malformedProjection){results.push({case_id:row.case_id,status:'UNKNOWN',reason:'PROJECTION_UNAVAILABLE',
      sourceRefs:[row.source_issue_url,...row.source_refs],joint:row.joint});continue;}
    const blindIncomplete=mode==='HISTORICALLY_BLIND'&&(input.historicalInputComplete!==true||
      !SHA.test(input.sourceSha??'')||Date.parse(input.observedAt)>Date.parse(row.evidence_cutoff)||
      Date.parse(projection.observedAt)>Date.parse(row.evidence_cutoff)||Date.parse(row.evidence_cutoff)>Date.parse(asOf)||
      row.expected_decisions.some(value=>!input.visibleDecisions.includes(value)));
    if(blindIncomplete){results.push({case_id:row.case_id,status:'NOT_RECONSTRUCTABLE',
      sourceRefs:[row.source_issue_url,...row.source_refs],joint:row.joint,excludedFromPredictiveCounts:true});continue;}
    const actual=projection.decisions,missing=row.expected_decisions.filter(value=>!actual.includes(value));
    const forbidden=row.forbidden_decisions.filter(value=>actual.includes(value));
    const leaked=mode==='HISTORICALLY_BLIND'?actual.filter(value=>!input.visibleDecisions.includes(value)):[];
    const unknownRequired=row.missing_evidence.length>0&&![...row.expected_decisions,...actual]
      .some(value=>/UNKNOWN|NOT_PROVEN|NOT_RECONSTRUCTABLE|SOURCE_PARTIAL/.test(value));
    const status=missing.length===0&&forbidden.length===0&&leaked.length===0&&!unknownRequired?'PASS':'FAIL';
    results.push({case_id:row.case_id,status,sourceIssue:row.source_issue_url,sourceRefs:row.source_refs,
      kind:row.kind,version:row.version,evidenceCutoff:row.evidence_cutoff,mode,
      joint:{timeline:row.joint.timeline??'UNKNOWN',diagnosis:row.joint.diagnosis??'UNKNOWN',planning:row.joint.planning??'UNKNOWN'},
      expectedDecisions:row.expected_decisions,actualDecisions:actual,missingDecisions:missing,forbiddenDecisions:forbidden,
      leakedDecisions:leaked,missingEvidence:row.missing_evidence,baselineErrors:projection.baselineErrors});
  }
  const evaluated=results.filter(row=>row.status==='PASS'||row.status==='FAIL'),passed=evaluated.filter(row=>row.status==='PASS');
  const count=(predicate)=>passed.filter(predicate).length;
  return Object.freeze({status:'OBSERVED',mode,asOf,cases:results,coverage:{
    caseCount:results.length,evaluatedCount:evaluated.length,passCount:passed.length,failCount:evaluated.length-passed.length,
    notReconstructableCount:results.filter(row=>row.status==='NOT_RECONSTRUCTABLE').length,
    sourceGroundedCorrectCount:passed.length,
    conflictSafetyCount:count(row=>['COLLISION','SECURITY'].includes(row.kind)),
    unknownHonestyCount:count(row=>row.missingEvidence.length>0),
    falseBlockerCount:evaluated.reduce((sum,row)=>sum+row.forbiddenDecisions.length,0),
    denominatorConsistencyCount:count(row=>['CAMPAIGN','REVIEW_CONVERGENCE'].includes(row.kind)),
    nextActionUsefulCount:count(row=>row.joint.planning!=='UNKNOWN'&&row.joint.planning!=='NO_RECOMMENDATION'),
    naiveBaselineErrorCount:evaluated.reduce((sum,row)=>sum+row.baselineErrors,0),candidateErrorCount:evaluated.length-passed.length,
    blindPredictiveCount:mode==='HISTORICALLY_BLIND'?evaluated.length:0,
    comparison:'COUNTS_NOT_PERCENTAGES',activeHours:'NOT_MEASURED',productionRoi:'NOT_MEASURED',
    taskPass:false,releasePass:false},advisoryOnly:true});
}
export function affectedCases(casebook,changedSourceRefs=[]){
  const changed=new Set(array(changedSourceRefs));
  return array(casebook?.cases).filter(row=>changed.has(row.source_issue_url)||array(row.source_refs).some(ref=>changed.has(ref)))
    .map(row=>row.case_id);
}
