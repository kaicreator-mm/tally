/** URL identity is advisory; only identities in the Host catalog may be selected. */
export interface ProjectScope{id:string;title:string;repository?:string;}
export interface VersionScope{id:string;projectId:string;label:string;ref?:string;}
export interface ScopeCatalog{
  projects:readonly ProjectScope[]|null;
  versionsByProject:Readonly<Record<string,readonly VersionScope[]>>|null;
}
export interface SelectedScope{projectId:string|null;versionId:string|null;}
export function resolveScope(catalog:ScopeCatalog,requested:Partial<SelectedScope>):SelectedScope{
  if(!catalog.projects||!catalog.versionsByProject)
    return {projectId:null,versionId:null};
  const projectId=requested.projectId??null;
  if(!projectId||!catalog.projects.some(project=>project.id===projectId))
    return {projectId:null,versionId:null};
  const versionId=requested.versionId??null;
  const allowed=(catalog.versionsByProject[projectId]??[])
    .some(version=>version.id===versionId&&version.projectId===projectId);
  return {projectId,versionId:allowed?versionId:null};
}
export function chooseProject(catalog:ScopeCatalog,id:string|null):SelectedScope{
  return resolveScope(catalog,{projectId:id,versionId:null});
}
export function chooseVersion(catalog:ScopeCatalog,current:SelectedScope,
  versionId:string|null):SelectedScope{
  return resolveScope(catalog,{projectId:current.projectId,versionId});
}

/** Canonical Task deep link. Caller must use selected Host-validated scope. */
export function taskRoute(projectId:string,versionId:string,taskId:string|null){
  if(!projectId||!versionId)return null;
  return '#/development/tasks?projectId='+encodeURIComponent(projectId)
    +'&versionId='+encodeURIComponent(versionId)
    +(taskId?'&taskId='+encodeURIComponent(taskId):'');
}
