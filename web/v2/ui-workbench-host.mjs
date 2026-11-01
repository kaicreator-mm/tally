/**
 * Partial GitHub-native blocker inbox from the selected v2 Task roster.
 * This is NOT a canonical all-human-actions queue, approval authority,
 * Agent scheduler or accepted CAS Claim. Missing roster ≠ no blockers.
 */
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const SHA=/^[a-f0-9]{40}$/;
const denied=reason=>({connected:false,reason});
export function createUiWorkbenchHost({repository,versionId,expectedSha,nativeC}={}){
  if(!REPO.test(repository??'')||!/^version\/[A-Za-z0-9_.-]+$/.test(versionId??'')||
     !SHA.test(expectedSha??'')||
     typeof nativeC?.catalog!=='function'||typeof nativeC?.version!=='function')
    throw new TypeError('TRUSTED_NATIVE_BLOCKER_SOURCE_REQUIRED');
  const qualified=c=>c?.connected===true&&c.source?.repository===repository&&
    c.source.sourceSha===expectedSha&&c.source.currentness==='CURRENT'&&
    c.source.readback==='VERIFIED'&&
    c.versionsByProject?.[repository]?.some(v=>v?.projectId===repository&&
      v?.id===versionId);
  async function workbench(scope={}){
    if(scope.projectId!==undefined&&scope.projectId!==repository)
      return denied('WORKBENCH_PROJECT_SCOPE_MISMATCH');
    try{
      if(!qualified(await nativeC.catalog()))
        return denied('PINNED_WORKBENCH_SOURCE_UNQUALIFIED');
      const v=await nativeC.version();
      if(v?.connected!==true||v.source?.repository!==repository||
         v.source?.sourceSha!==expectedSha||
         v.source?.currentness!=='CURRENT'||v.source?.readback!=='VERIFIED'||
         v.versionId!==versionId||v.rosterCoverage!=='UNVERIFIED'||
         !Array.isArray(v.tasks)||v.tasks.length>200||
         !Array.isArray(v.dag?.nodes)||!Array.isArray(v.dag?.edges)||
         v.dag.asOf?.sourceSha!==expectedSha)
        return denied('WORKBENCH_TASK_SOURCE_UNQUALIFIED');
      const actions=[],taskIds=new Set();
      for(const task of v.tasks){
        if(!task||!/^\d+$/.test(task.id??'')||taskIds.has(task.id)||
           task.versionId!==versionId||
           !['BLOCKED','CLEAR'].includes(task.dependencyState)||
           !Number.isSafeInteger(task.openBlockerCount)||
           task.openBlockerCount<0||
           !Array.isArray(task.blockedBy)||
           task.blockedBy.length<task.openBlockerCount||
           ((task.openBlockerCount>0)!==(task.dependencyState==='BLOCKED'))||
           task.claimState!=='NOT_VERIFIED')
          return denied('WORKBENCH_NATIVE_DEPENDENCY_UNQUALIFIED');
        taskIds.add(task.id);
        if(task.openBlockerCount>0){
          actions.push({
            id:'native-blocker-'+task.id,taskId:task.id,
            projectId:repository,versionId,
            title:'Task #'+task.id+' · native dependencies blocked',
            kind:'BLOCKER',state:'BLOCKED',ownerRole:'UNASSIGNED',
            sourceRef:'https://github.com/'+repository+'/issues/'+task.id,
            reason:'Observed open GitHub native blockers: '+task.openBlockerCount+
              '. No human assignment or execution authorization inferred.',
            updatedAt:task.updatedAt??null
          });
        }
      }
      if(v.dag.nodes.length!==taskIds.size||
         v.dag.nodes.some(n=>!taskIds.has(n?.taskId)))
        return denied('WORKBENCH_DAG_ROSTER_DRIFT');
      if(!qualified(await nativeC.catalog()))
        return denied('PINNED_WORKBENCH_SOURCE_CHANGED_DURING_READ');
      return {connected:true,source:{
        repository,sourceSha:expectedSha,currentness:'CURRENT',readback:'VERIFIED'},
        queueCoverage:'PARTIAL_UNVERIFIED_ROSTER',
        approvalCoverage:'NOT_CONNECTED',
        actions};
    }catch{return denied('WORKBENCH_NATIVE_READ_FAILED');}
  }
  return Object.freeze({workbench});
}
