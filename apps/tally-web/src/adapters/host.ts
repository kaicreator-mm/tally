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
export interface SettingsResponse{connected:boolean;reason?:string;model?:string;
  snapshot?:Record<string,unknown>;}
export interface ReleaseResponse{connected:boolean;
  readiness?:{versionId:string;
    gates:{name:string;status:string;owner:string|null}[];
    permits:{operation:string;state:string;sourceSha:string|null;
      grantedAt:string|null}[]|null;
    rqState:string;ownerGrantState:string};}
export interface MetadataResponse{connected:boolean;reason?:string;model?:string;
  versionId:string|null;
  attempts?:unknown[];reports?:unknown[];lineage?:unknown[];
  learnedOutcomes?:unknown[];note?:string;}

async function getJson<T>(path:string):Promise<T>{
  const response=await fetch(path,{cache:'no-store'});
  if(!response.ok)throw new Error(`HOST_${response.status}`);
  return await response.json() as T;
}
export const hostAdapter=Object.freeze({
  catalog:()=>getJson<Catalog>('/api/v2/catalog'),
  tasks:(versionId:string)=>getJson<TasksResponse>(
    `/api/v2/tasks?versionId=${encodeURIComponent(versionId)}`),
  dag:(versionId:string)=>getJson<DagResponse>(
    `/api/v2/dag?versionId=${encodeURIComponent(versionId)}`),
  settings:()=>getJson<SettingsResponse>('/api/v2/settings'),
  release:(versionId:string)=>getJson<ReleaseResponse>(
    `/api/v2/release?versionId=${encodeURIComponent(versionId)}`),
  metadata:(versionId:string)=>getJson<MetadataResponse>(
    `/api/v2/metadata?versionId=${encodeURIComponent(versionId)}`),
});
/** Wrap any adapter call: transport failure degrades to NOT_CONNECTED with
 * the model name preserved — the UI never mistakes an outage for emptiness. */
export async function orNotConnected<T>(model:string,
  call:()=>Promise<T>):Promise<T|{connected:false;reason:'NOT_CONNECTED';
  model:string;transportError:string}>{
  try{return await call();}
  catch(error){return {connected:false,reason:'NOT_CONNECTED',model,
    transportError:error instanceof Error?error.message:String(error)};}
}
