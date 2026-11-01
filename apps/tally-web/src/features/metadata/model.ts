/** UI-F #233 — engineering metadata center view model.
 * Metadata is separate-but-linked to task progress: its own read models,
 * per-version scoping, coverage denominators, UNKNOWN ≠ 0, no agent ranking,
 * no private traces, self-reported results never become independent PASS.
 */
export interface AttemptMeta{
  attemptId:string;taskId:string;
  agentRole:'builder'|'reviewer'|'validator';
  agentId:string;
  model:string|null;provider:string|null;
  durationMs:number|null;tokensIn:number|null;tokensOut:number|null;
  costUsd:null; // cost is never asserted without a source-qualified receipt
  startedAt:string|null;terminatedAt:string|null;
}
export interface ReportMeta{
  reportId:string;taskId:string;attemptId:string|null;
  kind:'ads-v5-summary'|'ads-v5-correction'|'feedback';
  subjectSha:string;observedAt:string;evidenceRef:string;
  /** ADS reports are self-reported until independently reviewed. */
  independentlyReviewed:boolean;
}
export interface LineageMeta{
  taskId:string;
  nativeIssueRef:string|null;
  claimCommentRef:string|null;
  claimAccepted:boolean;
  productFreezeRev:number|null;
  l2FreezeRev:number|null;
  packDigest:string|null;
  prNumber:number|null;prHeadSha:string|null;
  reviewState:'JOINED'|'REPAIR'|'NOT_RUN'|'UNKNOWN';
  ciState:'SUCCESS'|'FAILURE'|'RUNNING'|'NOT_RUN'|'UNKNOWN';
  fullHiddenCritical:{full:string;hidden:string;critical:string};
  releaseReceiptRef:string|null;
  deploymentReceiptRef:string|null;
}
export interface VersionMetadata{
  versionId:string;
  attempts:readonly AttemptMeta[];
  reports:readonly ReportMeta[];
  lineage:readonly LineageMeta[];
  learnedOutcomes:readonly {learningId:string;taskId:string;
    decision:'ADOPTED'|'REJECTED';attested:boolean|null}[];
}
export type MetadataTab='reports'|'provenance'|'quality';

/** Aggregate usage ONLY within one version, only over measured attempts;
 * coverage denominator always attached (missing ≠ zero, never merged). */
export interface UsageAggregate{
  attempts:number;measuredDuration:number;
  tokensIn:number;tokensOut:number;
  durationCoverage:number|null;
  tokenInCoverage:number|null;
  tokenOutCoverage:number|null;
  byRole:Record<string,{count:number;measured:number}>;
}
export function aggregateUsage(version:VersionMetadata):UsageAggregate{
  let measuredDuration=0,tokensIn=0,tokensOut=0;
  let durationMeasured=0,inMeasured=0,outMeasured=0;
  const byRole:UsageAggregate['byRole']={};
  for(const attempt of version.attempts){
    const role=byRole[attempt.agentRole]??{count:0,measured:0};
    role.count++;
    if(attempt.durationMs!==null){measuredDuration+=attempt.durationMs;
      durationMeasured++;role.measured++;}
    if(attempt.tokensIn!==null){tokensIn+=attempt.tokensIn;inMeasured++;}
    if(attempt.tokensOut!==null){tokensOut+=attempt.tokensOut;outMeasured++;}
    byRole[attempt.agentRole]=role;
  }
  const total=version.attempts.length;
  return {attempts:total,measuredDuration,tokensIn,tokensOut,
    durationCoverage:total===0?null:durationMeasured/total,
    tokenInCoverage:total===0?null:inMeasured/total,
    tokenOutCoverage:total===0?null:outMeasured/total,byRole};
}
/** Reports view model: stale/duplicate detection and review state. */
export interface ReportRow{
  reportId:string;taskId:string;kind:ReportMeta['kind'];
  reviewed:'REVIEWED'|'SELF_REPORTED';
  staleAgainst:(subjectSha:string)=>boolean;
  observedAt:string;evidenceRef:string;
}
export function reportRows(version:VersionMetadata):ReportRow[]{
  return version.reports.map(report=>({
    reportId:report.reportId,taskId:report.taskId,kind:report.kind,
    reviewed:report.independentlyReviewed?'REVIEWED':'SELF_REPORTED',
    staleAgainst:(subjectSha:string)=>report.subjectSha!==subjectSha,
    observedAt:report.observedAt,evidenceRef:report.evidenceRef,
  }));
}
/** Distinguish a missing report from a present one; ADS v5 absent is a gap. */
export function missingReports(version:VersionMetadata,
  expectedTaskIds:readonly string[]|null):string[]|null{
  if(expectedTaskIds===null)return null;
  const withReports=new Set(version.reports.map(report=>report.taskId));
  return expectedTaskIds.filter(taskId=>!withReports.has(taskId));
}
/** Cross-version casebook references (read-only links; versions never merge
 * their metadata aggregates). */
export function casebookLinks(versions:readonly VersionMetadata[],
  taskId:string):{versionId:string;learningId:string}[]{
  const links:{versionId:string;learningId:string}[]=[];
  for(const version of versions)
    for(const learning of version.learnedOutcomes)
      if(learning.taskId===taskId)
        links.push({versionId:version.versionId,learningId:learning.learningId});
  return links;
}
/** Deep-link helpers both ways: task ↔ metadata. */
export function taskDeepLink(taskId:string,projectId:string,versionId:string){
  return `#/development/tasks?projectId=${encodeURIComponent(projectId)}`
    +`&versionId=${encodeURIComponent(versionId)}`
    +`&taskId=${encodeURIComponent(taskId)}`;
}
export function metadataDeepLink(taskId:string,projectId:string,versionId:string){
  return `#/governance/metadata?projectId=${encodeURIComponent(projectId)}`
    +`&versionId=${encodeURIComponent(versionId)}`
    +`&taskId=${encodeURIComponent(taskId)}`;
}
