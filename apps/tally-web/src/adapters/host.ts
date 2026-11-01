/** UI-E #232 — Host adapter: the ONLY bridge between the shell and the ALM
 * server. Every response keeps the server's honest connected/UNKNOWN shape;
 * transport failures surface as NOT_CONNECTED instead of empty data.
 */
export interface Catalog{
  connected:boolean;reason?:string;
  projects?:{id:string;title:string;repository?:string}[];
  versionsByProject?:Record<string,{id:string;projectId:string;label:string;
    ref?:string}[]>;
  defaultVersionId?:string;
}
export interface TasksResponse{
  connected:boolean;reason?:string;versionId?:string;
  tasks?:{id:string;versionId:string;title:string;
    status:'open'|'in_progress'|'review'|'blocked'|'done'|'failed';
    severity:'P0'|'P1'|'P2';blockedBy:string[];agentId:string|null;
    claimState:string;attemptId:string|null;attemptState:string|null;
    prNumber:number|null;reviewState:string;ciState:string;writeSet:string[];
    requirementRef:string|null;updatedAt:string|null}[];
}
export interface DagResponse{
  connected:boolean;reason?:string;versionId:string;
  asOf:{sourceSha:string;frozenAt:string|null};
  nodes:{taskId:string;versionId:string;title:string;
    status:string;severity:string;requirementGroup:string|null;
    phase:string|null;lane:string|null;agentId:string|null;
    executable:boolean}[];
  edges:{from:string;to:string;kind:'NATIVE'|'PLANNED'|'UNRESOLVED';
    evidenceRef:string|null;amendedAt:string|null}[];
}
export interface SettingsResponse{connected:boolean;
  snapshot?:Record<string,unknown>;}
export interface ReleaseResponse{connected:boolean;
  readiness?:{versionId:string;
    gates:{name:string;status:string;owner:string|null}[];
    permits:{operation:string;state:string;sourceSha:string|null;
      grantedAt:string|null}[]|null;
    rqState:string;ownerGrantState:string};}
export interface MetadataResponse{connected:boolean;versionId:string|null;
  attempts:unknown[];reports:unknown[];lineage:unknown[];
  learnedOutcomes:unknown[];note?:string;}

export interface WorkbenchResponse{
  connected:boolean;source?:{repository:string;sourceSha:string;currentness:string;
    readback:string};
  actions?:import('../features/workbench/model.ts').WorkAction[];
}
export interface GuidedStepResponse{
  connected:boolean;reason?:string;sourceSha?:string;csrfKey?:string;
  step?:import('../features/forms/schema.ts').FormStep;
}
export interface GuidedSubmitResponse{
  connected:boolean;ok?:boolean;reason?:string;
  nextStep?:import('../features/forms/schema.ts').FormStep|null;
}
export type GuidedAnswers=ReturnType<
  typeof import('../features/forms/engine.ts').buildSubmitPayload>;
async function postGuided(path:string,csrfKey:string,payload:unknown):
  Promise<GuidedSubmitResponse>{
  const response=await fetch(path,{method:'POST',cache:'no-store',
    headers:{'content-type':'application/json','x-tally-action-key':csrfKey},
    body:JSON.stringify(payload)});
  const result=await response.json() as GuidedSubmitResponse;
  if(!response.ok||result.ok!==true)return {
    connected:false,ok:false,reason:result.reason??'GUIDED_HOST_DENIED'};
  return result;
}
async function getJson<T>(path:string):Promise<T>{
  const response=await fetch(path,{cache:'no-store'});
  if(!response.ok)throw new Error(`HOST_${response.status}`);
  return await response.json() as T;
}
export const hostAdapter=Object.freeze({
  guidedStep:(projectId:string,versionId:string)=>
    getJson<GuidedStepResponse>(
      '/api/v2/guided/step?projectId='+encodeURIComponent(projectId)+
      '&versionId='+encodeURIComponent(versionId)),
  guidedSubmit:(scope:{projectId:string;versionId:string;sourceSha:string},
    csrfKey:string,payload:GuidedAnswers)=>
    postGuided('/api/v2/guided/submit',csrfKey,{...scope,...payload}),
  workbench:()=>getJson<WorkbenchResponse>('/api/v2/workbench'),
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
