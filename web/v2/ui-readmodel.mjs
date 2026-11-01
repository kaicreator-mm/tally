/** UI #246: bounded, read-only, selected Host projection gate.
 * This is NOT a GitHub API client, new task authority or runtime config store.
 * Inputs are issued by an injected selected B/K/Host/C adapter, not URL trust.
 */
const SHA=/^[a-f0-9]{40}$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9_.:\/-]{0,191}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const TYPES=new Set(['catalog','tasks','dag','release','settings','metadata',
  'agents','requirements','qa','maintenance','workbench','deployment',
  'workflow','domainPackages','experience','audit']);
const VERSION_MODELS=new Set(['tasks','dag','release','metadata','agents',
  'requirements','qa','maintenance','deployment']);
const SECRET_KEYS=/^(password|secret|apiKey|accessToken|refreshToken|privateKey|authorization|bearer|credentialValue)$/i;
const SECRET_VALUES=/\b(?:gh[pousr]_[A-Za-z0-9_]{12,}|sk-[A-Za-z0-9_-]{16,})\b/;
const plain=v=>!!v&&typeof v==='object'&&!Array.isArray(v);
export function notConnectedV2(model,reason='HOST_READMODEL_UNAVAILABLE'){
  return {connected:false,model,reason};
}
function noSecrets(value,depth=0){
  if(depth>12)return false;
  if(typeof value==='string')return value.length<=4096&&!SECRET_VALUES.test(value);
  if(value===null||typeof value==='boolean'||typeof value==='number')
    return value===null||typeof value==='boolean'||Number.isFinite(value);
  if(Array.isArray(value))return value.length<=2000&&value.every(v=>noSecrets(v,depth+1));
  if(!plain(value))return false;
  return Object.keys(value).length<=100&&Object.entries(value)
    .every(([k,v])=>!SECRET_KEYS.test(k)&&noSecrets(v,depth+1));
}
export function validateV2ReadModel(model,value,scope={}){
  if(!TYPES.has(model)||!plain(value)||value.connected!==true)return false;
  const source=value.source;
  if(!plain(source)||!REPO.test(source.repository)||!SHA.test(source.sourceSha)||
     source.currentness!=='CURRENT'||source.readback!=='VERIFIED')return false;
  if(scope.projectId&&source.repository!==scope.projectId)return false;
  if(VERSION_MODELS.has(model)){
    if(!ID.test(scope.versionId??'')||value.versionId!==scope.versionId)return false;
  }
  if(model==='catalog'){
    if(!Array.isArray(value.projects)||!plain(value.versionsByProject))return false;
    if(value.projects.some(p=>!plain(p)||!REPO.test(p.id)||typeof p.title!=='string'))return false;
    for(const [project,versions] of Object.entries(value.versionsByProject)){
      if(!REPO.test(project)||!Array.isArray(versions)||
         versions.some(v=>!plain(v)||v.projectId!==project||!ID.test(v.id)))return false;
    }
  }
  if(model==='tasks'){
    if(!Array.isArray(value.tasks)||value.tasks.some(task=>
      !plain(task)||!ID.test(task.id)||task.versionId!==scope.versionId||
      typeof task.title!=='string'||!Array.isArray(task.blockedBy)))return false;
  }
  if(model==='dag'){
    if(!plain(value.asOf)||value.asOf.sourceSha!==source.sourceSha||
      !Array.isArray(value.nodes)||!Array.isArray(value.edges))return false;
    const ids=new Set();
    for(const node of value.nodes){
      if(!plain(node)||!ID.test(node.taskId)||node.versionId!==scope.versionId||
        ids.has(node.taskId))return false;
      ids.add(node.taskId);
    }
    for(const edge of value.edges){
      if(!plain(edge)||!ids.has(edge.from)||!ids.has(edge.to)||
        !['NATIVE','PLANNED','UNRESOLVED'].includes(edge.kind)||
        (edge.kind==='NATIVE'&&
          (typeof edge.evidenceRef!=='string'||!edge.evidenceRef.startsWith('https://github.com/'))))
        return false;
    }
  }
  if(model==='metadata'){
    if(!['attempts','reports','lineage','learnedOutcomes']
      .every(key=>Array.isArray(value[key])))return false;
  }
  if(model==='release'){
    if(!plain(value.readiness)||value.readiness.versionId!==scope.versionId||
      !Array.isArray(value.readiness.gates))return false;
  }
  if(model==='workbench'){
    if(!Array.isArray(value.actions))return false;
    const ids=new Set();
    for(const action of value.actions){
      if(!plain(action)||!ID.test(action.id)||ids.has(action.id)||
        action.projectId!==source.repository||
        (action.versionId!==null&&!ID.test(action.versionId))||
        (action.taskId!==null&&!ID.test(action.taskId))||
        typeof action.title!=='string'||!action.title.trim()||
        !['APPROVAL','DECISION','BLOCKER','REVIEW','RECOVERY'].includes(action.kind)||
        !['PENDING','BLOCKED','RESOLVED'].includes(action.state)||
        typeof action.ownerRole!=='string'||!action.ownerRole.trim()||
        typeof action.sourceRef!=='string'||
        !action.sourceRef.startsWith('https://github.com/'+source.repository+'/issues/'))
        return false;
      ids.add(action.id);
    }
  }
  if(model==='settings'&&!plain(value.snapshot))return false;
  return noSecrets(value);
}
export async function readV2HostModel(host,model,scope={}){
  if(!TYPES.has(model))return notConnectedV2(model,'UNKNOWN_MODEL');
  if(scope.projectId!==undefined&&!REPO.test(scope.projectId??''))
    return notConnectedV2(model,'INVALID_PROJECT_SCOPE');
  if(VERSION_MODELS.has(model)&&!ID.test(scope.versionId??''))
    return notConnectedV2(model,'VERSION_SCOPE_REQUIRED');
  if(typeof host?.[model]!=='function')
    return notConnectedV2(model);
  try{
    const result=await host[model]({...scope});
    if(!validateV2ReadModel(model,result,scope))
      return notConnectedV2(model,'SOURCE_UNQUALIFIED');
    return result;
  }catch{
    // Provider exceptions/errors may carry credentials. Never forward them.
    return notConnectedV2(model,'HOST_READ_FAILED');
  }
}
