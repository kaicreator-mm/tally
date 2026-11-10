/**
 * Tally v2 #275 — source-backed Task Attempt read-only projection.
 *
 * Only selected Host/K roster and public observation source qualify runtime
 * facts. ADS reports are advisory. Neither a terminal runtime event nor an
 * Agent self-report makes an independently accepted Task/Review/Release.
 * Does not scan private SQLite, authorize Claims or dispatch Agents.
 */
const SHA=/^[a-f0-9]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/;
const EVENTS=new Set(['INSTANCE_OPENED','MESSAGE_ACCEPTED','TURN_COMMITTED',
  'TURN_RECOVERY_REQUIRED','RECOVERY_COMMITTED','INSTANCE_TERMINALIZED']);
const ROLES=new Set(['builder','reviewer','validator']);
const OUTCOMES=new Set(['COMPLETED','FAILED','CANCELLED','INTERRUPTED']);
const plain=v=>v!==null&&typeof v==='object'&&!Array.isArray(v);
const iso=v=>typeof v==='string'&&Number.isFinite(Date.parse(v));
const nativeRefFor=(ref,repo,issue)=>typeof ref==='string'&&
  ref.startsWith('https://github.com/'+repo+'/issues/'+issue+'#issuecomment-')&&
  /^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/issues\/[1-9][0-9]*#issuecomment-[1-9][0-9]*$/.test(ref);
const limits=Object.freeze({authorized:false,admitted:false,taskPass:false,releasePass:false});
const invalid=(reason,status='UNKNOWN')=>Object.freeze({status,reason,
  attempts:Object.freeze([]),coverage:Object.freeze({attemptCount:null,
    observedRuntime:0,hostReceipts:0,adsReports:0,metrics:'UNKNOWN'}),
  ...limits});
function matched(row,identity){
  return row.repository===identity.repository&&row.versionId===identity.versionId&&
    row.taskIssueNumber===identity.taskIssueNumber&&row.sourceSha===identity.sourceSha;
}
function validStream(stream) {
  const t=stream?.target,p=stream?.package;
  return plain(stream)&&plain(t)&&plain(p)&&
    typeof t.workflowId==='string'&&t.workflowId.length>0&&t.workflowId.length<=256&&
    typeof t.instanceKey==='string'&&t.instanceKey.length>0&&t.instanceKey.length<=256&&
    typeof p.packageId==='string'&&p.packageId.length>0&&p.packageId.length<=256&&
    typeof p.contentDigest==='string'&&p.contentDigest.length>0&&p.contentDigest.length<=256&&
    typeof p.domainId==='string'&&p.domainId.length>0&&p.domainId.length<=256&&
    typeof p.version==='string'&&p.version.length>0&&p.version.length<=128&&
    typeof p.formatVersion==='string'&&p.formatVersion.length>0&&p.formatVersion.length<=128&&
    Number.isSafeInteger(p.runtimeContractMajor)&&p.runtimeContractMajor>=0&&
    Number.isSafeInteger(p.executionEngineMajor)&&p.executionEngineMajor>=0&&
    Array.isArray(p.requiredCapabilities)&&p.requiredCapabilities.length<=100&&
    p.requiredCapabilities.every(x=>typeof x==='string'&&x.length>0&&x.length<=128)&&
    typeof stream.epochId==='string'&&stream.epochId.length>0&&stream.epochId.length<=128;
}
function sameStream(a,b) {
  if(!validStream(a)||!validStream(b))return false;
  const pa=a.package,pb=b.package;
  return a.target.workflowId===b.target.workflowId&&
    a.target.instanceKey===b.target.instanceKey&&
    a.epochId===b.epochId&&
    pa.domainId===pb.domainId&&pa.version===pb.version&&
    pa.packageId===pb.packageId&&pa.contentDigest===pb.contentDigest&&
    pa.formatVersion===pb.formatVersion&&
    pa.runtimeContractMajor===pb.runtimeContractMajor&&
    pa.executionEngineMajor===pb.executionEngineMajor&&
    JSON.stringify(pa.requiredCapabilities)===JSON.stringify(pb.requiredCapabilities)&&
    (a.runtimeBindingRef??null)===(b.runtimeBindingRef??null)&&
    (a.runtimeActivationRef??null)===(b.runtimeActivationRef??null);
}
function validAttempt(row,identity) {
  return plain(row)&&ID.test(row.attemptId??'')&&ROLES.has(row.role)&&
    matched(row,identity)&&ID.test(row.writerId??'')&&
    validStream(row.runtimeStream)&&
    typeof row.environment==='string'&&row.environment.length>0&&
    row.environment.length<=128&&typeof row.worktree==='string'&&
    row.worktree.length>0&&row.worktree.length<=2048;
}
function runtimeFor(attempt,page) {
  if(!plain(page))return {status:'UNKNOWN',reason:'OBSERVATION_UNAVAILABLE',
    eventCount:0,lastSequence:null,runtimeTerminal:false};
  if(page.attemptId!==attempt.attemptId||
    !['COMPLETE','RETENTION_TRUNCATED','STORAGE_GAP','CURSOR_INVALID','UNKNOWN'].includes(page.status)||
    !Array.isArray(page.observations)||page.observations.length>1000)
    return null;
  let last=0,terminal=false;
  const ids=new Set();
  for(const o of page.observations){
    if(!plain(o)||!sameStream(o.stream,attempt.runtimeStream)||
      !EVENTS.has(o.kind)||!iso(o.observedAt)||
      typeof o.observationId!=='string'||!o.observationId||
      ids.has(o.observationId)||!Number.isSafeInteger(o.sequence)||
      o.sequence!==last+1)return null;
    ids.add(o.observationId);
    last=o.sequence;
    if(o.kind==='INSTANCE_TERMINALIZED')terminal=true;
  }
  if(page.status==='COMPLETE'&&
    (page.cursorComplete!==true||!Number.isSafeInteger(page.highWatermark)||
      page.highWatermark!==last))return null;
  // Exact public RuntimeObservation v1 can be a proven empty stream,
  // but that is not proof an Attempt ran or a Task finished.
  const current=page.status==='COMPLETE'&&last>0;
  return {status:current?'OBSERVED':'INCOMPLETE',
    reason:current?'PUBLIC_RUNTIME_OBSERVATION':'PUBLIC_RUNTIME_'+page.status,
    eventCount:last,lastSequence:last||null,runtimeTerminal:current&&terminal};
}
function hostOutcomeFor(attempt) {
  if(attempt.receipt==null)return {status:'UNKNOWN',outcome:null,ref:null};
  const r=attempt.receipt;
  if(!plain(r)||r.status!=='VERIFIED'||r.attemptId!==attempt.attemptId||
    r.sourceSha!==attempt.sourceSha||!OUTCOMES.has(r.outcome)||!iso(r.observedAt)||
    !nativeRefFor(r.ref,attempt.repository,attempt.taskIssueNumber))
    return null;
  return {status:'HOST_OBSERVED',outcome:r.outcome,ref:r.ref};
}
function adsFor(attempt,reports,identity){
  const matchedReports=reports.filter(r=>r?.attemptId===attempt.attemptId);
  if(matchedReports.length===0)return {status:'UNKNOWN',reportCount:0};
  const ids=new Set();
  for(const r of matchedReports){
    if(!plain(r)||r.status!=='CURRENT_OBSERVED'||!matched(r,identity)||
       !ID.test(r.reportId??'')||!Number.isSafeInteger(r.revision)||
       r.revision<1||ids.has(r.reportId+':'+r.revision)||
       !nativeRefFor(r.nativeRef,identity.repository,identity.taskIssueNumber))
      return null;
    ids.add(r.reportId+':'+r.revision);
  }
  return {status:'OBSERVED_ADVISORY',reportCount:matchedReports.length};
}
/** Qualified read projection. All inputs are observed data, never authority. */
export function projectTaskAttempts(input={}) {
  if(!plain(input))return invalid('INVALID_INPUT');
  const {repository,versionId,taskIssueNumber,sourceSha,roster,
    observationPages=[],adsReports=[]}=input;
  const identity={repository,versionId,taskIssueNumber,sourceSha};
  if(!REPO.test(repository??'')||versionId!=='version/v2'||
    !Number.isSafeInteger(taskIssueNumber)||taskIssueNumber<1||
    !SHA.test(sourceSha??''))return invalid('INVALID_SOURCE_IDENTITY');
  if(!plain(roster)||roster.status!=='VERIFIED'||roster.complete!==true||
    !matched(roster,identity)||!Array.isArray(roster.attempts))
    return invalid('HOST_ATTEMPT_INDEX_UNAVAILABLE');
  if(roster.attempts.length>200||!Array.isArray(observationPages)||
    observationPages.length>200||!Array.isArray(adsReports)||adsReports.length>1000)
    return invalid('UNBOUNDED_EVIDENCE');
  const seen=new Set(),pages=new Map();
  for(const page of observationPages){
    if(!plain(page)||!ID.test(page.attemptId??'')||pages.has(page.attemptId))
      return invalid('AMBIGUOUS_OBSERVATION_PAGE');
    pages.set(page.attemptId,page);
  }
  const attempts=[];
  for(const a of roster.attempts){
    if(!validAttempt(a,identity)||seen.has(a.attemptId))
      return invalid('INVALID_OR_DUPLICATE_ATTEMPT_IDENTITY');
    seen.add(a.attemptId);
    const runtime=runtimeFor(a,pages.get(a.attemptId));
    const hostOutcome=hostOutcomeFor(a);
    const ads=adsFor(a,adsReports,identity);
    if(!runtime||!hostOutcome||!ads)return invalid('CONFLICTING_ATTEMPT_EVIDENCE');
    attempts.push(Object.freeze({attemptId:a.attemptId,role:a.role,
      sourceSha:a.sourceSha,environment:a.environment,
      workflowAddress:Object.freeze({...a.runtimeStream.target}),
      packageId:a.runtimeStream.package.packageId,
      epochId:a.runtimeStream.epochId,writerId:a.writerId,
      // Worktree/paths/secrets are deliberately not returned to the UI.
      runtime:Object.freeze(runtime),hostOutcome:Object.freeze(hostOutcome),
      ads:Object.freeze(ads),metrics:'UNKNOWN',independentPass:'NOT_PROVEN'}));
  }
  if([...pages.keys()].some(id=>!seen.has(id))||
    adsReports.some(r=>!seen.has(r?.attemptId)))
    return invalid('FOREIGN_ATTEMPT_OBSERVATION');
  const observedRuntime=attempts.filter(a=>a.runtime.status==='OBSERVED').length;
  const hostReceipts=attempts.filter(a=>a.hostOutcome.status==='HOST_OBSERVED').length;
  const reportCount=attempts.reduce((sum,a)=>sum+a.ads.reportCount,0);
  return Object.freeze({status:attempts.length===0?'OBSERVED_EMPTY':
    observedRuntime===attempts.length&&hostReceipts===attempts.length?
      'CURRENT_OBSERVED':'PARTIAL',reason:'SELECTED_HOST_READ_ONLY_PROJECTION',
    repository,versionId,taskIssueNumber,sourceSha,
    attempts:Object.freeze(attempts),coverage:Object.freeze({
      attemptCount:attempts.length,observedRuntime,hostReceipts,
      adsReports:reportCount,metrics:'UNKNOWN'}),...limits});
}
