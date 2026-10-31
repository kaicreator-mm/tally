/** J08a advisory projection: prior release -> defect -> next-version candidate.
 * GitHub remains authoritative; no Issue/Task/Claim write or historical change.
 *
 * Integrates with the existing read-only Engineering B projections (#139/#141/#143/#144)
 * instead of a second history index: same fail-closed UNKNOWN shapes as planning/feedback,
 * same counts-not-percentages coverage as lineage/casebook, and the same advisoryOnly
 * guarantees as index.mjs (never taskPass/releasePass, never an authorized native write).
 * A prior release is carried as an immutable reported identity only; a missing release or
 * task-meta report stays UNKNOWN/NOT_MEASURED and never blocks or fabricates a metric.
 */
const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_][A-Za-z0-9_.-]*\/[A-Za-z0-9_][A-Za-z0-9_.-]*$/;
const TIME=/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;
const iso=x=>typeof x==='string'&&TIME.test(x)&&Number.isFinite(Date.parse(x));
const freeze=x=>Object.freeze(x);
const issue=(repo,value)=>typeof value==='string'&&
  new RegExp(`^https://github\\.com/${repo.replace(/[.*+?^${}()|[\]\\]/g,'\\$&')}/issues/[1-9]\\d*$`).test(value);
const releaseRef=(repo,value)=>typeof value==='string'&&value.length<=2048&&
  value.startsWith(`https://github.com/${repo}/releases/tag/`)&&
  /^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$/.test(value.slice(`https://github.com/${repo}/releases/tag/`.length));
const invalid=reason=>freeze({status:'UNKNOWN',reason,priorRelease:'UNKNOWN',candidates:freeze([]),
  learning:freeze([]),coverage:freeze({comparison:'COUNTS_NOT_PERCENTAGES',activeHours:'NOT_MEASURED',
    metrics:'UNKNOWN',taskPass:false,releasePass:false}),historicalReleaseMutation:false,
  nativeWriteAuthorized:false,advisoryOnly:true});
/** Input must already be collected from GitHub C; the records are never proof of authorization. */
export function projectMaintenanceIntake(input={}){
  const {repository,observedAsOf,priorRelease,defects=[],learning=[]}=input??{};
  if(!REPO.test(repository??'')||!iso(observedAsOf)||!Array.isArray(defects)||
    !Array.isArray(learning)||defects.length>500||learning.length>500)
    return invalid('MAINTENANCE_SOURCE_UNVERIFIED');
  let release='UNKNOWN';
  let previous=null;
  if(priorRelease!==undefined&&priorRelease!==null){
    if(typeof priorRelease!=='object'||Array.isArray(priorRelease)||
      typeof priorRelease.version!=='string'||!/^v?[0-9]+\.[0-9]+(?:\.[0-9]+)?$/.test(priorRelease.version)||
      !SHA.test(priorRelease.commitSha??'')||!releaseRef(repository,priorRelease.evidenceRef))
      return invalid('PRIOR_RELEASE_IDENTITY_UNVERIFIED');
    previous=freeze({version:priorRelease.version,commitSha:priorRelease.commitSha,
      evidenceRef:priorRelease.evidenceRef,immutable:true});
    release='REPORTED_RELEASE_UNVERIFIED';
  }
  const validDefect=row=>row&&typeof row==='object'&&!Array.isArray(row)&&
    issue(repository,row.issueRef)&&issue(repository,row.requirementRef)&&
    SHA.test(row.subjectSha??'')&&iso(row.observedAt)&&Date.parse(row.observedAt)<=Date.parse(observedAsOf)&&
    typeof row.summary==='string'&&row.summary.trim().length>0&&row.summary.length<=512;
  const validLearning=row=>row&&typeof row==='object'&&!Array.isArray(row)&&
    issue(repository,row.taskIssueRef)&&issue(repository,row.targetIssueRef)&&
    SHA.test(row.subjectSha??'')&&iso(row.observedAt)&&Date.parse(row.observedAt)<=Date.parse(observedAsOf)&&
    ['REPORTED_QUALIFIED','UNKNOWN','NOT_RUN'].includes(row.qualification);
  if(!defects.every(validDefect)||!learning.every(validLearning)||
     new Set(defects.map(row=>row.issueRef)).size!==defects.length)
    return invalid('MAINTENANCE_INPUT_UNVERIFIED');
  const candidates=freeze(defects.map(row=>freeze({type:'DefectToVersionCandidate',status:'PROPOSED',
    sourceIssueRef:row.issueRef,requirementRef:row.requirementRef,
    subjectSha:row.subjectSha,summary:row.summary,observedAt:row.observedAt,
    priorRelease:previous,priorReleaseEvidence:release,
    nextAction:'PRODUCT_REVIEW_OR_AUTHORIZED_INTAKE',
    nativeTaskCreated:false,claimAccepted:false,authorized:false,
    mustNot:freeze(['DO_NOT_REWRITE_HISTORICAL_RELEASE','DO_NOT_AUTO_CREATE_NATIVE_TASK'])})));
  const suggestions=freeze(learning.map(row=>freeze({type:'TaskLearningSuggestion',status:
    row.qualification==='REPORTED_QUALIFIED'?'CANDIDATE_UNVERIFIED':row.qualification,
    sourceIssueRef:row.taskIssueRef,targetIssueRef:row.targetIssueRef,subjectSha:row.subjectSha,
    provenance:'GITHUB_REPORTED_NOT_INDEPENDENTLY_ATTESTED',
    adopted:false,measuredEffect:'NOT_MEASURED',cost:'UNKNOWN',noAgentRanking:true})));
  return freeze({status:'CANDIDATES_ONLY',repository,observedAsOf,priorRelease:release,
    previous,candidates,learning:suggestions,coverage:freeze({defectCandidates:candidates.length,
      learningSuggestions:suggestions.length,comparison:'COUNTS_NOT_PERCENTAGES',
      metrics:'UNKNOWN',activeHours:'NOT_MEASURED',taskPass:false,releasePass:false}),
    historicalReleaseMutation:false,nativeWriteAuthorized:false,advisoryOnly:true});
}
