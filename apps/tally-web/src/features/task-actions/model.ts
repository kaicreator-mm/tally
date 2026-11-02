export interface TaskActionOption{
  id:string;kind:string;revision:number;label:string;taskIssue:number|null;
  state:'ELIGIBLE'|'BLOCKED';reason?:string|null;
}
export interface TaskActionCatalog{
  connected:boolean;source?:{repository:string;sourceSha:string;
    currentness:string;readback:string};versionId?:string;
  actions?:readonly TaskActionOption[];
}
export interface TaskActionRequest{
  projectId:string;versionId:string;sourceSha:string;
  actionId:string;revision:number;
}
export function taskActionsFor(catalog:TaskActionCatalog|null,
  projectId:string,versionId:string,taskIssue:number):TaskActionOption[]|null{
  if(catalog?.connected!==true||catalog.source?.repository!==projectId||
     catalog.source.currentness!=='CURRENT'||catalog.source.readback!=='VERIFIED'||
     catalog.versionId!==versionId||!Array.isArray(catalog.actions))return null;
  const selected=catalog.actions.filter(a=>a.taskIssue===taskIssue);
  if(selected.some(a=>!a.id||!Number.isSafeInteger(a.revision)||a.revision<1||
     !['ELIGIBLE','BLOCKED'].includes(a.state)))return null;
  return selected;
}
export function taskActionRequest(catalog:TaskActionCatalog,
  projectId:string,versionId:string,action:TaskActionOption):TaskActionRequest|null{
  if(catalog.connected!==true||!catalog.source||
     catalog.source.repository!==projectId||
     catalog.source.currentness!=='CURRENT'||catalog.source.readback!=='VERIFIED'||
     catalog.versionId!==versionId||action.state!=='ELIGIBLE'||
     !catalog.actions?.some(a=>a.id===action.id&&a.revision===action.revision&&
       a.state==='ELIGIBLE'))return null;
  return {projectId,versionId,sourceSha:catalog.source.sourceSha,
    actionId:action.id,revision:action.revision};
}
