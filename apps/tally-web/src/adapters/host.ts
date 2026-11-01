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
    status:'open'|'in_progress'|'review'|'blocked'|'done'|'failed'|'unknown';
    severity:'P0'|'P1'|'P2'|'UNKNOWN';blockedBy:string[];agentId:string|null;
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
export interface LifecycleRead<T>{
  connected:boolean;versionId:string;records?:T[];
  source?:{repository:string;sourceSha:string;currentness:string;readback:string};
}
export interface VersionAgentsRead{connected:boolean;versionId:string;
  attempts?:import('../features/agents/model.ts').AttemptRecord[];}
export interface VersionRequirementsRead{connected:boolean;versionId:string;
  nodes?:import('../features/requirements/model.ts').RequirementNode[];}
export interface VersionQaRead{connected:boolean;versionId:string;
  records?:import('../features/qa/model.ts').QaRecord[];}
export interface VersionMaintenanceRead{connected:boolean;versionId:string;
  defects?:import('../features/maintenance/model.ts').DefectRecord[];
  learnings?:import('../features/maintenance/model.ts').LearningRecord[];}
export interface VersionDeploymentRead{connected:boolean;versionId:string;
  deployments?:import('../features/deployment/model.ts').DeploymentRecord[];}
export interface GovernanceRead{
  connected:boolean;records?:import('../features/governance/model.ts').GovernanceItem[];
  source?:import('../features/governance/model.ts').GovernanceSnapshot['source'];
}
const scoped=(path:string,projectId:string,versionId:string)=>{
  return '/api/v2/'+path+'?projectId='+encodeURIComponent(projectId)+
    '&versionId='+encodeURIComponent(versionId);
};
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
  agents:(projectId:string,versionId:string)=>
    getJson<VersionAgentsRead>(scoped('agents',projectId,versionId)),
  requirements:(projectId:string,versionId:string)=>
    getJson<VersionRequirementsRead>(scoped('requirements',projectId,versionId)),
  qa:(projectId:string,versionId:string)=>
    getJson<VersionQaRead>(scoped('qa',projectId,versionId)),
  maintenance:(projectId:string,versionId:string)=>
    getJson<VersionMaintenanceRead>(scoped('maintenance',projectId,versionId)),
  deployment:(projectId:string,versionId:string)=>
    getJson<VersionDeploymentRead>(scoped('deployment',projectId,versionId)),
  governance:(kind:'workflow'|'domainPackages'|'experience'|'audit',
    projectId:string)=>
    getJson<GovernanceRead>('/api/v2/'+({
      workflow:'workflow',domainPackages:'domain-packages',
      experience:'experience',audit:'audit'} as const)[kind]+
      '?projectId='+encodeURIComponent(projectId)),
  workbench:()=>getJson<WorkbenchResponse>('/api/v2/workbench'),
  catalog:()=>getJson<Catalog>('/api/v2/catalog'),
  tasks:(projectId:string,versionId:string)=>
    getJson<TasksResponse>(scoped('tasks',projectId,versionId)),
  dag:(projectId:string,versionId:string)=>
    getJson<DagResponse>(scoped('dag',projectId,versionId)),
  settings:()=>getJson<SettingsResponse>('/api/v2/settings'),
  release:(projectId:string,versionId:string)=>
    getJson<ReleaseResponse>(scoped('release',projectId,versionId)),
  metadata:(projectId:string,versionId:string)=>
    getJson<MetadataResponse>(scoped('metadata',projectId,versionId)),
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
