/** Closeout A/#260: qualify a COMPLETE version roster against a trusted,
 * independently read-back frozen DAG and real GitHub Native C observations.
 * Source and membership truth come from selected Host + Native C, never UI.
 * No Claim, execution grant or Task closure is inferred here.
 */
const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const denial=reason=>({connected:false,reason,rosterCoverage:'UNVERIFIED'});
const ids=items=>items.map(String);
const edgeId=(from,to)=>String(from)+'->'+String(to);
const sameSet=(a,b)=>a.size===b.size&&[...a].every(x=>b.has(x));
function frozenOk(doc,repository,versionId,expectedSha){
  if(doc?.connected!==true||doc?.source?.repository!==repository||
     doc.source.sourceSha!==expectedSha||doc.source.currentness!=='CURRENT'||
     doc.source.readback!=='VERIFIED'||doc.versionId!==versionId||
     doc.witness?.status!=='INDEPENDENTLY_VERIFIED'||
     !SHA.test(doc.witness.blobSha??'')||
     doc.frozen?.blobSha!==doc.witness.blobSha||
     doc.frozen?.versionId!==versionId||
     !Array.isArray(doc.frozen.taskIssues)||doc.frozen.taskIssues.length<1||
     doc.frozen.taskIssues.length>200||
     !Array.isArray(doc.frozen.edges)||doc.frozen.edges.length>2000)return false;
  const tasks=doc.frozen.taskIssues;
  if(tasks.some(n=>!Number.isSafeInteger(n)||n<1)||
     new Set(tasks).size!==tasks.length)return false;
  const known=new Set(tasks.map(String)),edges=new Set();
  for(const e of doc.frozen.edges){
    if(!e||!known.has(String(e.from))||!known.has(String(e.to))||
       String(e.from)===String(e.to))return false;
    const k=edgeId(e.from,e.to);
    if(edges.has(k))return false;
    edges.add(k);
  }
  // Never show a cyclic source as a ready-to-run DAG.
  const adj=new Map([...known].map(id=>[id,[]]));
  for(const e of doc.frozen.edges)adj.get(String(e.from)).push(String(e.to));
  const done=new Set(),visiting=new Set();
  const walk=n=>{
    if(visiting.has(n))return false;
    if(done.has(n))return true;
    visiting.add(n);
    for(const next of adj.get(n)||[])if(!walk(next))return false;
    visiting.delete(n);done.add(n);return true;
  };
  return [...known].every(walk);
}
const equalFreeze=(a,b)=>a.witness.blobSha===b.witness.blobSha&&
  a.versionId===b.versionId&&
  sameSet(new Set(a.frozen.taskIssues),new Set(b.frozen.taskIssues))&&
  sameSet(new Set(a.frozen.edges.map(e=>edgeId(e.from,e.to))),
          new Set(b.frozen.edges.map(e=>edgeId(e.from,e.to))));
export function createUiRosterHost({repository,versionId,expectedSha,
  frozenHost,nativeC}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedSha??''))throw TypeError('PINNED_ROSTER_SCOPE_REQUIRED');
  async function roster(scope={}){
    if(scope.projectId!==undefined&&scope.projectId!==repository||
       scope.versionId!==undefined&&scope.versionId!==versionId)
      return denial('ROSTER_SCOPE_MISMATCH');
    if(typeof frozenHost?.readFrozenDag!=='function'||
       typeof nativeC?.version!=='function')
      return denial('FROZEN_DAG_HOST_NOT_CONNECTED');
    try{
      const params={repository,versionId,sourceSha:expectedSha};
      const before=await frozenHost.readFrozenDag(params);
      if(!frozenOk(before,repository,versionId,expectedSha))
        return denial('FROZEN_DAG_SOURCE_UNQUALIFIED');
      const native=await nativeC.version();
      if(native?.connected!==true||native.source?.repository!==repository||
         native.source.sourceSha!==expectedSha||
         native.source.currentness!=='CURRENT'||
         native.source.readback!=='VERIFIED'||native.versionId!==versionId||
         !Array.isArray(native.tasks)||!Array.isArray(native.dag?.nodes)||
         !Array.isArray(native.dag?.edges)||
         native.dag.asOf?.sourceSha!==expectedSha||
         native.tasks.length!==native.dag.nodes.length)
        return denial('NATIVE_DAG_SOURCE_UNQUALIFIED');
      const frozenIds=new Set(ids(before.frozen.taskIssues));
      const nativeIds=new Set(native.tasks.map(x=>x?.id));
      if(nativeIds.has(undefined)||nativeIds.size!==native.tasks.length||
         !sameSet(frozenIds,nativeIds)||
         !sameSet(nativeIds,new Set(native.dag.nodes.map(n=>n?.taskId)))||
         native.tasks.some(t=>t.versionId!==versionId||
           t.claimState!=='NOT_VERIFIED'))
        return denial('VERSION_ROSTER_NOT_EXHAUSTIVE');
      const planned=new Set(before.frozen.edges.map(e=>edgeId(e.from,e.to)));
      const observed=new Set();
      for(const edge of native.dag.edges){
        if(!edge||edge.kind!=='NATIVE'||
           !frozenIds.has(edge.from)||!frozenIds.has(edge.to)||
           !String(edge.evidenceRef??'').startsWith('https://github.com/'+repository+'/issues/'))
          return denial('UNQUALIFIED_NATIVE_DEPENDENCY');
        const key=edgeId(edge.from,edge.to);
        if(observed.has(key)||!planned.has(key))
          return denial('NATIVE_DEPENDENCY_OUTSIDE_FROZEN_DAG');
        observed.add(key);
      }
      const after=await frozenHost.readFrozenDag(params);
      if(!frozenOk(after,repository,versionId,expectedSha)||
         !equalFreeze(before,after))
        return denial('FROZEN_DAG_CHANGED_DURING_READ');
      return {connected:true,source:native.source,versionId,
        rosterCoverage:'VERIFIED_COMPLETE',
        dependencyCoverage:sameSet(planned,observed)?'NATIVE_COMPLETE':'NATIVE_PARTIAL',
        frozenBlobSha:before.frozen.blobSha,taskCount:native.tasks.length,
        nativeEdgeCount:observed.size,frozenEdgeCount:planned.size,
        // Task execution still requires a separate current accepted Host CAS Claim.
        claimCoverage:'NOT_VERIFIED',tasks:native.tasks,
        dag:{...native.dag,rosterCoverage:'VERIFIED_COMPLETE',
          asOf:{...native.dag.asOf,frozenAt:before.frozen.frozenAt??null}}};
    }catch{return denial('ROSTER_SOURCE_READ_FAILED');}
  }
  return Object.freeze({roster});
}
