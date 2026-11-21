/** UI-E #232 — Host adapter: the ONLY bridge between the shell and the ALM
 * server. Every response keeps the server's honest connected/UNKNOWN shape;
 * transport failures surface as NOT_CONNECTED instead of empty data.
 * #264 E1: Task/DAG models come from the Host-composed selected source
 * (frozen-roster-qualified), so they carry source identity, roster/claim/
 * dependency coverage and honest UNKNOWN severities; absent optional
 * providers (settings/metadata) answer connected:false — never empty data.
 */
export interface SourceIdentity{
  repository:string;sourceSha:string;currentness:string;readback:string;
}
export interface Catalog{
  connected:boolean;reason?:string;
  source?:SourceIdentity;
  projects?:{id:string;title:string;repository?:string}[];
  versionsByProject?:Record<string,{id:string;projectId:string;label:string;
    ref?:string}[]>;
  defaultVersionId?:string;
}
export interface TaskDto{
  id:string;versionId:string;title:string;
  status:'open'|'in_progress'|'review'|'blocked'|'done'|'failed'|'unknown';
  severity:'P0'|'P1'|'P2'|'UNKNOWN';
  issueState?:'open'|'closed';
  closureState?:'OPEN'|'CLOSED_UNVERIFIED';
  dependencyState?:'CLEAR'|'BLOCKED';
  openBlockerCount?:number;
  blockedBy:string[];agentId:string|null;
  claimState:string;claimAccepted?:boolean;claimRef?:string|null;
  claimDispatchId?:string|null;
  responsibilityStatus?:string;activeOperator?:unknown;
  attemptId:string|null;attemptState:string|null;
  prNumber:number|null;reviewState:string;ciState:string;writeSet:string[];
  requirementRef:string|null;updatedAt:string|null;
}
/** Draft Issues are observable, but never frozen/accepted executable Tasks. */
export interface TaskDraftsResponse{connected:boolean;reason?:string;
  repository?:string;versionId?:string;sourceSha?:string;
  coverage?:'LATEST_100_ISSUES_ONLY'|'COMPLETE_SINGLE_PAGE';
  drafts?:{issueNumber:number;title:string;issueUrl:string;
    githubIssueState:'open'|'closed';sourceSha:string;
    sourceCurrentness?:'CURRENT'|'STALE';
    nativeWorkflow?:'PLANNED'|'READY'|'NONCANONICAL';
    taskQualified:false;agentAuthorized:false}[];}
export interface ClaimHostResponse{connected:boolean;reason?:string;
  repository?:string;versionId?:string;sourceSha?:string;
  issueNumber?:number;actionKey?:string;authority?:'HOST_PINNED_ONLY';}
export interface ClaimPublishResult{status:string;reason?:string;
  claimPublication?:string;authorized?:boolean;admitted?:boolean;
  nativeCommentRef?:string;}
export interface CandidateHostResponse{connected:boolean;reason?:string;
  repository?:string;versionId?:string;sourceSha?:string;
  issueNumber?:number;actionKey?:string;}
export interface CandidatePublishResult{status:string;reason?:string;
  pullNumber?:number;pullRef?:string;branch?:string;
  candidateCommit?:string;authorized?:false;taskPass?:false;}
export interface WriteSetHostResponse{connected:boolean;reason?:string;
  repository?:string;versionId?:string;sourceSha?:string;
  issueNumber?:number;candidateRelativePath?:string;actionKey?:string;}
export interface WriteSetFreezeResult{status:string;reason?:string;
  candidateRelativePath?:string;writeSetSha256?:string;
  authorized?:false;admitted?:false;}
export interface TaskReadyHostResponse{connected:boolean;reason?:string;
  repository?:string;versionId?:string;sourceSha?:string;
  actionKey?:string;authority?:'HOST_PINNED_ONLY';}
export interface TaskReadyTransitionResult{status:string;reason?:string;
  githubEffect?:string;admitted?:false;authorized?:false;}
export interface TaskAdmissionResponse{connected:boolean;reason?:string;
  issueNumber?:number;issueUrl?:string;sourceSha?:string;
  stage?:'STALE_SOURCE'|'BLOCKED'|'NEEDS_CANONICAL_LABELS'|
    'PLANNED'|'NATIVE_PREFLIGHT_READY';nativeWorkflow?:string;
  admitted?:false;authorized?:false;agentAuthorized?:false;}
export interface TasksResponse{
  connected:boolean;reason?:string;model?:string;versionId?:string;
  source?:SourceIdentity;rosterCoverage?:string;dependencyCoverage?:string;
  taskCount?:number;nativeEdgeCount?:number;frozenBlobSha?:string|null;
  taskPackRef?:string;claimReadbackCoverage?:string;claimCoverage?:string;
  tasks?:TaskDto[];
}
export interface DagNodeDto{
  taskId:string;versionId:string;title:string;
  status:string;severity:'P0'|'P1'|'P2'|'UNKNOWN';
  issueState?:'open'|'closed';closureState?:string;claimState?:string;
  requirementGroup:string|null;
  phase:string|null;lane:string|null;agentId:string|null;
  executable:boolean;
}
export interface DagResponse{
  connected:boolean;reason?:string;model?:string;versionId:string;
  source?:SourceIdentity;rosterCoverage?:string;
  claimReadbackCoverage?:string;claimCoverage?:string;
  dependencyCoverage?:string;frozenBlobSha?:string|null;
  taskPackRef?:string;
  asOf:{sourceSha:string;frozenAt:string|null};
  nodes:DagNodeDto[];
  edges:{from:string;to:string;kind:'NATIVE'|'PLANNED'|'UNRESOLVED';
    evidenceRef:string|null;amendedAt:string|null}[];
}
export interface RequirementsResponse{connected:boolean;reason?:string;
  versionId?:string;structureCoverage?:string;freezeCoverage?:string;
  nodes?:{id:string;kind:'requirement'|'epic'|'story';title:string;
    parent:string|null;productRevision:number|null;acceptanceRefs:string[];
    l2Decision:string|null;freezeState:'FROZEN'|'STALE'|'NOT_FROZEN'|'UNKNOWN';
    updatedAt:string}[];}
export interface QaResponse{connected:boolean;reason?:string;versionId?:string;
  ciCoverage?:string;independentCoverage?:string;
  records?:{layer:'fullRegression'|'hiddenValidation'|'criticalJourneys'|
      'installedAcceptance';outcome:'PASS'|'FAIL'|'NOT_RUN'|'UNKNOWN';
      provenance:'REPORTED'|'INDEPENDENT'|'NOT_RUN'|'UNKNOWN';
      witness:string|null;evidenceRef:string|null;
      subjectSha:string|null;observedAt:string|null}[];}
export interface SettingsResponse{connected:boolean;reason?:string;model?:string;
  configurationUrl?:string;selectedRepository?:string;selectedRef?:string;
  currentSha?:string;agentHostMounted?:boolean;
  snapshot?:Record<string,unknown>;}
export interface ReleaseResponse{connected:boolean;
  readiness?:{versionId:string;
    gates:{name:string;status:string;reason?:string|null;
      owner:string|null;evidenceRef?:string|null}[];
    gateCounts?:{required:number;reportedPass:number;
      missingOrUnqualified:number;comparison:string};
    blockers?:{name:string;status:string;reason:string|null}[];
    permits:{operation:string;state:string;sourceSha:string|null;
      grantedAt:string|null}[]|null;
    rqState:string;ownerGrantState:string;advisoryOnly?:boolean};}
/** The existing real maintenance controller returns a source-pinned
 * snapshot; no empty-array or fabricated connected state is permitted. */
export interface MaintenanceSnapshotResponse{
  snapshot:{
    status:string;sourceSha:string;
    candidate?:{state:string;summary:string;sourceIssueRef:string}|null;
    review?:{state:string;current:boolean}|null;
    publication?:{state:string;reason?:string;githubEffect?:string}|null;
    learning?:unknown[];history?:unknown[];
    authorized:boolean;taskPass:boolean;releasePass:boolean;
  };
  lastResult?:{kind?:string;status?:string;reason?:string}|null;
}
export interface MetadataResponse{connected:boolean;reason?:string;model?:string;
  versionId:string|null;
  attempts?:unknown[];reports?:unknown[];lineage?:unknown[];
  learnedOutcomes?:unknown[];note?:string;}
/** #264 J02 — dual-epoch projection (S0 frozen historical + live S1).
 * currentnessStatus null means the honest STALE/unwitnessed descendant;
 * observedNativeEdgeCount/edgeAmendmentLedger carry the live-edge truth and
 * the Host-attested per-edge disposition for the frozen-minus-live delta. */
export interface DualEpochResponse{
  connected:boolean;reason?:string;
  frozenSource?:{repository:string;sourceSha:string;
    currentness:string;readback:string};
  currentSource?:{repository:string;sourceSha:string;
    currentness:string;readback:string};
  epoch?:{freezeBaseSha:string;observedVersionSha:string;ancestry:string};
  currentnessStatus:'CURRENT_SAME_SOURCE'|'CURRENT_STRUCTURAL_COMPATIBLE'|null;
  currentness:'CURRENT'|'STALE';
  currentReason:string|null;
  rosterCoverage:string;
  frozenBlobSha?:string|null;taskPackRef?:string|null;
  taskCount?:number;nativeEdgeCount?:number;frozenEdgeCount?:number;
  observedNativeEdgeCount?:number|null;
  edgeAmendmentLedger?:{disposition:string;declared:number;
    applied:{from:number;to:number;evidenceRef:string}[]}|null;
  identityDigests?:{original51IdsDigest:string;native29EdgeDigest:string};
  compatibilityWitness?:{present:boolean;valid:boolean|null}|null;
  claimProjection?:{claimSourceEpoch:string;frozenClaimSourceSha:string;
    currentObservedSourceSha:string;claimReadback:string};
  authorized:boolean;admitted:boolean;taskPass:boolean;releasePass:boolean;}
/** #264 E1 R91 — real observed runs of THIS installed server's /api/actions
 * seam. Structural twin of features/agents AttemptRecord; missing telemetry
 * (duration/tokens/cost/model) is null (UNKNOWN), never zero. */
export interface AgentAttemptRecord{
  attemptId:string;taskIds:readonly string[];agentId:string;
  state:'RUNNING'|'EXITED_OK'|'EXITED_FAIL'|'CANCELLED'|'UNKNOWN';
  startedAt:string|null;terminatedAt:string|null;
  durationMs:number|null;tokensIn:number|null;tokensOut:number|null;
  costUsd:null;model:string|null;provider:string|null;
  role?:'BUILDER'|'REVIEWER'|'REPAIR'|'VALIDATOR'|'ORCHESTRATOR'|'UNKNOWN'|null;
}
export interface AgentDispatchResponse{connected:boolean;reason?:string;
  selectedTaskIssue?:number;sourceSha?:string;provider?:string;
  authority?:'HOST_PINNED_ONLY';actionKey?:string;}
export interface AgentRunResult{status:string;reason?:string|null;
  observedAt?:string|null;exitCode?:number|null;}
export interface AgentAttemptsResponse{connected:boolean;
  attempts?:AgentAttemptRecord[];
  coverage?:{attemptCount:number;observedRuntime:number;hostReceipts:number;
    adsReports:number;metrics:string};}
/** J07c real release evidence snapshot (the controller's observed state). */
export interface ReleaseSnapshotResponse{
  html:string;
  snapshot:{status:string;sourceSha:string;targetSha?:string;
    canMerge?:boolean;canTag?:boolean;canPublish?:boolean;releasePass?:boolean;
    readiness?:{status:string;releasePass?:boolean;ciStatus?:string;
      ciCoverage?:string}|null;};
  lastResult?:{kind:string;status:string;reason?:string}|null;}
export interface ReleaseActionResult{status:string;reason?:string;
  releasePass?:boolean;authorized?:boolean;taskPass?:boolean;
  mutationPerformed?:boolean;}

async function getJson<T>(path:string):Promise<T>{
  const response=await fetch(path,{cache:'no-store'});
  if(!response.ok)throw new Error(`HOST_${response.status}`);
  return await response.json() as T;
}
/** #264 E1 R91: same-origin sealed POST. The operator action key comes from
 * the installed server's ?actionKey= URL contract (same as the legacy
 * consoles); without it the server's CSRF admission denies — never a second
 * credential channel. */
async function postJson<T>(path:string,actionKey:string,
  payload:unknown):Promise<T>{
  const response=await fetch(path,{method:'POST',cache:'no-store',
    headers:{'content-type':'application/json',...(actionKey?
      {'x-tally-action-key':actionKey}:{})},
    body:JSON.stringify(payload)});
  const result=await response.json() as T;
  if(!response.ok&&!(result as {status?:string})?.status)
    throw new Error(`HOST_${response.status}`);
  return result;
}
export const hostAdapter=Object.freeze({
  catalog:()=>getJson<Catalog>('/api/v2/catalog'),
  claimHost:()=>getJson<ClaimHostResponse>('/api/v2/claim-host'),
  writeSetHost:()=>getJson<WriteSetHostResponse>('/api/v2/writeset-host'),
  candidateHost:()=>getJson<CandidateHostResponse>('/api/v2/candidate-host'),
  taskReadyHost:()=>getJson<TaskReadyHostResponse>('/api/v2/task-ready-host'),
  taskAdmission:(versionId:string,issueNumber:number)=>
    getJson<TaskAdmissionResponse>(
      `/api/v2/task-admission?versionId=${encodeURIComponent(versionId)}&issueNumber=${issueNumber}`),
  taskDrafts:(versionId:string)=>getJson<TaskDraftsResponse>(
    `/api/v2/task-drafts?versionId=${encodeURIComponent(versionId)}`),
  tasks:(versionId:string)=>getJson<TasksResponse>(
    `/api/v2/tasks?versionId=${encodeURIComponent(versionId)}`),
  dag:(versionId:string)=>getJson<DagResponse>(
    `/api/v2/dag?versionId=${encodeURIComponent(versionId)}`),
  dualEpoch:(versionId:string)=>getJson<DualEpochResponse>(
    `/api/v2/dual-epoch?versionId=${encodeURIComponent(versionId)}`),
  requirements:(versionId:string)=>getJson<RequirementsResponse>(
    `/api/v2/requirements?versionId=${encodeURIComponent(versionId)}`),
  qa:(versionId:string)=>getJson<QaResponse>(
    `/api/v2/qa?versionId=${encodeURIComponent(versionId)}`),
  maintenance:()=>getJson<MaintenanceSnapshotResponse>('/api/maintenance/snapshot'),
  settings:()=>getJson<SettingsResponse>('/api/v2/settings'),
  release:(versionId:string)=>getJson<ReleaseResponse>(
    `/api/v2/release?versionId=${encodeURIComponent(versionId)}`),
  metadata:(versionId:string)=>getJson<MetadataResponse>(
    `/api/v2/metadata?versionId=${encodeURIComponent(versionId)}`),
  agentDispatch:()=>getJson<AgentDispatchResponse>('/api/v2/agent-dispatch'),
  agentAttempts:()=>getJson<AgentAttemptsResponse>('/api/v2/agent-attempts'),
  releaseSnapshot:()=>getJson<ReleaseSnapshotResponse>('/api/release/snapshot'),
});
/** #264 E1 R91: the real J07c submission flow. The controller admits only a
 * sourceSha matching its pinned source, so the exact SHA comes from the
 * freshly read evidence snapshot — never from a browser-typed field. */
/** One physical Builder task; only the server-selected Host can authorize.
 * The action key is same-origin CSRF material, not an Agent/Claim grant.
 */
/** A recorded GitHub Claim still requires independent accepted-CAS/K admission. */
export async function publishSelectedTaskClaim({
  issueNumber,repository,versionId,sourceSha
}:{issueNumber:number;repository:string;versionId:string;
  sourceSha:string}):Promise<ClaimPublishResult>{
  try{
    const mounted=await hostAdapter.claimHost();
    if(mounted.connected!==true||mounted.repository!==repository||
       mounted.versionId!==versionId||mounted.sourceSha!==sourceSha||
       mounted.issueNumber!==issueNumber||!mounted.actionKey)
      return {status:'BLOCKED',reason:mounted.reason??'CLAIM_HOST_SCOPE_MISMATCH'};
    return await postJson<ClaimPublishResult>('/api/v2/claim-publish',
      mounted.actionKey,{issueNumber,sourceSha});
  }catch{return {status:'UNKNOWN',reason:'CLAIM_PUBLICATION_READBACK_UNKNOWN'};}
}
/** The write set is sealed by the trusted Host; UI sends no path or digest. */
/** A Draft PR requires a real K candidate, never an unreviewed release grant. */
export async function publishSelectedDraftCandidate({
  issueNumber,repository,versionId,sourceSha
}:{issueNumber:number;repository:string;versionId:string;
  sourceSha:string}):Promise<CandidatePublishResult>{
  try{
    const mounted=await hostAdapter.candidateHost();
    if(mounted.connected!==true||mounted.repository!==repository||
       mounted.versionId!==versionId||mounted.sourceSha!==sourceSha||
       mounted.issueNumber!==issueNumber||!mounted.actionKey)
      return {status:'BLOCKED',reason:mounted.reason??'CANDIDATE_HOST_SCOPE_MISMATCH'};
    return await postJson<CandidatePublishResult>(
      '/api/v2/candidate-publish',mounted.actionKey,{issueNumber,sourceSha});
  }catch{return {status:'PUBLICATION_UNKNOWN',
    reason:'DRAFT_PR_PUBLICATION_READBACK_UNKNOWN'};}
}
export async function freezeSelectedTaskWriteSet({
  issueNumber,repository,versionId,sourceSha
}:{issueNumber:number;repository:string;versionId:string;
  sourceSha:string}):Promise<WriteSetFreezeResult>{
  try{
    const host=await hostAdapter.writeSetHost();
    if(host.connected!==true||host.repository!==repository||
       host.versionId!==versionId||host.sourceSha!==sourceSha||
       host.issueNumber!==issueNumber||!host.actionKey)
      return {status:'BLOCKED',reason:host.reason??'WRITESET_HOST_SCOPE_MISMATCH'};
    return await postJson<WriteSetFreezeResult>(
      '/api/v2/writeset-freeze',host.actionKey,{issueNumber,sourceSha});
  }catch{return {status:'UNKNOWN',reason:'WRITESET_FREEZE_READBACK_UNKNOWN'};}
}
export async function transitionSelectedTaskReady({
  issueNumber,repository,versionId,sourceSha
}:{issueNumber:number;repository:string;versionId:string;
  sourceSha:string}):Promise<TaskReadyTransitionResult>{
  try{
    const mounted=await hostAdapter.taskReadyHost();
    if(mounted.connected!==true||mounted.repository!==repository||
       mounted.versionId!==versionId||mounted.sourceSha!==sourceSha||
       !mounted.actionKey)
      return {status:'BLOCKED',reason:mounted.reason??'TASK_READY_HOST_SCOPE_MISMATCH'};
    return await postJson<TaskReadyTransitionResult>(
      '/api/v2/task-ready',mounted.actionKey,{issueNumber,sourceSha});
  }catch{return {status:'UNKNOWN',reason:'TASK_READY_ACTION_READBACK_UNKNOWN'};}
}
export async function startSelectedAgentTask(issueNumber:number,sourceSha:string):
  Promise<AgentRunResult>{
  try{
    const selected=await hostAdapter.agentDispatch();
    if(!selected.connected||selected.selectedTaskIssue!==issueNumber||
       selected.sourceSha!==sourceSha||!selected.actionKey)
      return {status:'DENIED',reason:selected.reason??'HOST_TASK_SCOPE_MISMATCH'};
    return await postJson<AgentRunResult>('/api/actions',selected.actionKey,
      {kind:'start',sourceSha:selected.sourceSha});
  }catch{
    return {status:'UNKNOWN',reason:'AGENT_ACTION_READBACK_UNKNOWN'};
  }
}
export async function submitReleaseAction(actionKey:string,
  kind:'merge'|'tag'|'publish'):Promise<ReleaseActionResult|{
  status:string;reason:string;transportError?:string}>{
  try{
    const snapshot=await getJson<ReleaseSnapshotResponse>('/api/release/snapshot');
    const sourceSha=snapshot.snapshot?.sourceSha;
    if(typeof sourceSha!=='string'||!/^[0-9a-f]{40}$/.test(sourceSha))
      return {status:'DENIED',reason:'NATIVE_RELEASE_EVIDENCE_NOT_READ'};
    return await postJson<ReleaseActionResult>('/api/release/actions',actionKey,
      {kind,sourceSha});
  }catch(error){
    return {status:'DENIED',reason:'TRANSPORT_FAIL',
      transportError:error instanceof Error?error.message:String(error)};
  }
}
/** Wrap any adapter call: transport failure degrades to NOT_CONNECTED with
 * the model name preserved — the UI never mistakes an outage for emptiness. */
export async function orNotConnected<T>(model:string,
  call:()=>Promise<T>):Promise<T|{connected:false;reason:'NOT_CONNECTED';
  model:string;transportError:string}>{
  try{return await call();}
  catch(error){return {connected:false,reason:'NOT_CONNECTED',model,
    transportError:error instanceof Error?error.message:String(error)};}
}
