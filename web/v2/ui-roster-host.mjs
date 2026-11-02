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
const canonical=x=>Array.isArray(x)?x.map(canonical):
  x!==null&&typeof x==='object'?Object.fromEntries(Object.keys(x).sort().map(k=>[k,canonical(x[k])])):x;
const same=(a,b)=>JSON.stringify(canonical(a))===JSON.stringify(canonical(b));
function frozenOk(doc,repository,versionId,expectedSha){
  if(doc?.connected!==true||doc?.source?.repository!==repository||
     doc.source.sourceSha!==expectedSha||doc.source.currentness!=='CURRENT'||
     doc.source.readback!=='VERIFIED'||doc.versionId!==versionId||
     doc.witness?.status!=='INDEPENDENTLY_VERIFIED'||
     doc.witness?.sourceSha!==expectedSha||
     !SHA.test(doc.witness?.reviewedHeadSha??'')||
     typeof doc.witness?.author!=='string'||!doc.witness.author||
     typeof doc.witness?.reviewer!=='string'||!doc.witness.reviewer||
     doc.witness.author===doc.witness.reviewer||
     doc.ownerApproval?.sourceSha!==expectedSha||
     doc.ownerApproval?.owner!==repository.split('/')[0]||
     !SHA.test(doc.witness.blobSha??'')||
     typeof doc.witness.reviewer!=='string'||!doc.witness.reviewer||
     typeof doc.witness.reviewRef!=='string'||
     !doc.witness.reviewRef.startsWith('https://github.com/'+repository+'/')||
     doc.ownerApproval?.status!=='OWNER_APPROVED'||
     typeof doc.ownerApproval.approvalRef!=='string'||
     !doc.ownerApproval.approvalRef.startsWith('https://github.com/'+repository+'/')||
     doc.frozen?.blobSha!==doc.witness.blobSha||
     doc.frozen?.versionId!==versionId||
     typeof doc.frozenAt!=='string'||!Number.isFinite(Date.parse(doc.frozenAt))||
     !Array.isArray(doc.frozen.taskIssues)||doc.frozen.taskIssues.length<1||
     doc.frozen.taskIssues.length>200||
     !Array.isArray(doc.frozen.edges)||doc.frozen.edges.length>2000||
     !Array.isArray(doc.frozen.claimSubjects)||
     doc.frozen.claimSubjects.length!==doc.frozen.taskIssues.length||
     doc.frozen.taskPackRef!==doc.taskPackRef||
     !doc.taskPackRef.startsWith('https://github.com/'+repository+'/blob/')||
     !/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/blob\/[a-f0-9]{40}\//.test(doc.taskPackRef??''))
    return false;
  const tasks=doc.frozen.taskIssues;
  if(tasks.some(n=>!Number.isSafeInteger(n)||n<1)||
     new Set(tasks).size!==tasks.length)return false;
  const known=new Set(tasks.map(String)),edges=new Set(),subjects=new Set();
  for(const s of doc.frozen.claimSubjects){
    if(!s||!known.has(String(s.issueNumber))||subjects.has(String(s.issueNumber))||
       !SHA.test(s.sourceSha??'')||
       !['builder','reviewer','validator'].includes(s.actorRole)||
       s.protectedClaimKey!==repository+'#'+s.issueNumber+':'+s.actorRole+':__default__'||
       !['OBSERVED_COMMENT_SELF_REPORTED_CLAIM','NO_CLAIM_EVENT_EXPECTED'].includes(s.candidateStatus)||
       (s.candidateStatus==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
        (!Number.isSafeInteger(s.commentId)||s.commentId<1||
         !Number.isSafeInteger(s.generation)||s.generation!==1||
         s.commentRef!=='https://github.com/'+repository+'/issues/'+s.issueNumber+
           '#issuecomment-'+s.commentId||
         typeof s.dispatchId!=='string'||!s.dispatchId||
         typeof s.operatorKind!=='string'||!s.operatorKind||
         typeof s.operatorId!=='string'||!s.operatorId)))
      return false;
    if(s.candidateStatus==='NO_CLAIM_EVENT_EXPECTED'&&
       (s.commentId!==null||s.commentRef!==null||s.dispatchId!==null||
        s.protectedClaimKey!==repository+'#'+s.issueNumber+':'+s.actorRole+':__default__))return false;
    subjects.add(String(s.issueNumber));
  }
  if(subjects.size!==known.size)return false;
  for(const e of doc.frozen.edges){
    if(!e||!known.has(String(e.from))||!known.has(String(e.to))||
       String(e.from)===String(e.to)||!['NATIVE','PLANNED'].includes(e.kind))
      return false;
    if(e.evidenceRef!=='https://github.com/'+repository+'/issues/'+e.to)return false;
    const k=edgeId(e.from,e.to);
    if(edges.has(k))return false;
    edges.add(k);
  }
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
const equalFreeze=(a,b)=>same({
  source:{repository:a.source?.repository,sourceSha:a.source?.sourceSha,
    currentness:a.source?.currentness,readback:a.source?.readback},
  versionId:a.versionId,frozenAt:a.frozenAt,taskPackRef:a.taskPackRef,
  frozen:a.frozen,witness:a.witness,ownerApproval:a.ownerApproval
},{
  source:{repository:b.source?.repository,sourceSha:b.source?.sourceSha,
    currentness:b.source?.currentness,readback:b.source?.readback},
  versionId:b.versionId,frozenAt:b.frozenAt,taskPackRef:b.taskPackRef,
  frozen:b.frozen,witness:b.witness,ownerApproval:b.ownerApproval
});
function claimBaseOk(claim,repository,versionId,versionSha,issueNumber,subject,taskPackRef,blobSha){
  return claim?.connected===true&&claim.source?.repository===repository&&
    claim.source.sourceSha===subject.sourceSha&&claim.source.currentness==='PINNED'&&
    claim.source.readback==='VERIFIED'&&claim.versionId===versionId&&
    claim.versionSourceSha===versionSha&&claim.claimSourceSha===subject.sourceSha&&
    claim.taskPackRef===taskPackRef&&claim.taskPackBlobSha===blobSha&&
    claim.issueNumber===issueNumber&&claim.authorized===false&&
    claim.admitted===false&&claim.taskPass===false&&claim.releasePass===false;
}
function validClaim(claim,repository,versionId,versionSha,issueNumber,subject,taskPackRef,blobSha){
  if(!claimBaseOk(claim,repository,versionId,versionSha,issueNumber,subject,taskPackRef,blobSha))
    return false;
  if(claim.claimStatus==='NOT_CLAIMED')
    return subject.candidateStatus==='NO_CLAIM_EVENT_EXPECTED'&&
      claim.claimAccepted===false&&claim.claimPublication==='NOT_VERIFIED'&&
      claim.commentRef===null&&claim.commentId===null;
  if(claim.claimStatus==='NOT_VERIFIED')
    return subject.candidateStatus==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
      claim.claimAccepted===false&&claim.claimPublication==='CANDIDATE_ONLY'&&
      claim.commentId===subject.commentId&&claim.dispatchId===subject.dispatchId&&
      claim.generation===subject.generation&&claim.actorRole===subject.actorRole&&
      claim.operatorKind===subject.operatorKind&&claim.operatorId===subject.operatorId&&
      claim.protectedClaimKey===subject.protectedClaimKey&&
      claim.sourceClass==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
      claim.transportActor===repository.split('/')[0]&&
      claim.commentRef==='https://github.com/'+repository+'/issues/'+issueNumber+
        '#issuecomment-'+subject.commentId&&
      /^[a-f0-9]{64}$/.test(claim.commentBodySha256??'');
  return claim.claimStatus==='ACCEPTED'&&
    subject.candidateStatus==='OBSERVED_COMMENT_SELF_REPORTED_CLAIM'&&
    claim.claimAccepted===true&&
    claim.claimPublication==='REGISTRY_PUBLISHED_VERIFIED_NATIVE_READBACK'&&
    claim.generation===subject.generation&&claim.actorRole===subject.actorRole&&
    claim.operatorKind===subject.operatorKind&&claim.operatorId===subject.operatorId&&
    claim.dispatchId===subject.dispatchId&&
    claim.protectedClaimKey===subject.protectedClaimKey&&
    claim.commentId===subject.commentId&&
    claim.commentRef==='https://github.com/'+repository+'/issues/'+issueNumber+
      '#issuecomment-'+subject.commentId&&
    /^[a-f0-9]{64}$/.test(claim.commentBodySha256??'')&&
    claim.transportActor===repository.split('/')[0]&&
    typeof claim.writerId==='string'&&claim.writerId.length>0&&
    ['VERIFIED','NOT_VERIFIED'].includes(claim.responsibilityStatus)&&
    (claim.responsibilityStatus!=='VERIFIED'||
      claim.activeOperator?.operatorId==='chatgpt-web:tally-v2'&&
      claim.activeOperator?.operatorKind==='chatgpt-web'&&
      claim.activeOperator?.actorRole==='builder'&&
      claim.activeOperator?.executionEnvironment==='WEB');
}
export function createUiRosterHost({repository,versionId,expectedSha,
  frozenHost,nativeC,claimHost}={}){
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
      const native=await nativeC.version({taskIssueNumbers:before.frozen.taskIssues});
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
         native.tasks.some(t=>t.versionId!==versionId))
        return denial('VERSION_ROSTER_NOT_EXHAUSTIVE');
      let claimReadback='NOT_VERIFIED',claimResults=null;
      const subjects=new Map(before.frozen.claimSubjects.map(x=>[String(x.issueNumber),x]));
      const taskPackRef=before.frozen.taskPackRef,taskPackBlobSha=before.frozen.blobSha;
      if(subjects.size!==frozenIds.size||[...frozenIds].some(id=>!subjects.has(id)))
        return denial('FROZEN_CLAIM_SUBJECT_ROSTER_INCOMPLETE');
      if(typeof claimHost?.readAcceptedClaim==='function'){
        claimResults=[];
        for(const task of native.tasks){
          const number=Number(task.id),subject=subjects.get(String(number));
          const args={repository,versionId,versionSourceSha:expectedSha,
            issueNumber:number,claimSourceSha:subject.sourceSha,
            taskPackRef,taskPackBlobSha,claimSubject:subject};
          const first=await claimHost.readAcceptedClaim(args);
          const second=await claimHost.readAcceptedClaim(args);
          if(!validClaim(first,repository,versionId,expectedSha,number,subject,taskPackRef,taskPackBlobSha)||
             !validClaim(second,repository,versionId,expectedSha,number,subject,taskPackRef,taskPackBlobSha)||
             JSON.stringify(first)!==JSON.stringify(second))
            return denial('ACCEPTED_CLAIM_HOST_READBACK_UNQUALIFIED');
          claimResults.push(first);
        }
        claimReadback='VERIFIED_COMPLETE';
      }
      const planned=new Set(before.frozen.edges.filter(e=>e.kind==='NATIVE')
        .map(e=>edgeId(e.from,e.to)));
      const observed=new Set();
      for(const edge of native.dag.edges){
        if(!edge||edge.kind!=='NATIVE'||
           !frozenIds.has(String(edge.from))||!frozenIds.has(String(edge.to))||
           edge.evidenceRef!=='https://github.com/'+repository+'/issues/'+edge.to)
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
      const claimByTask=new Map((claimResults??[]).map(c=>[String(c.issueNumber),c]));
      const tasks=claimResults===null?native.tasks:native.tasks.map(task=>{
        const claim=claimByTask.get(String(task.id));
        return {...task,claimState:claim?.claimStatus??'NOT_VERIFIED',
          claimAccepted:claim?.claimAccepted===true,
          claimRef:claim?.commentRef??null,claimCandidateRef:
            claim?.claimStatus==='NOT_VERIFIED'?claim.commentRef:null,
          claimDispatchId:claim?.dispatchId??null,
          responsibilityStatus:claim?.responsibilityStatus??'NOT_VERIFIED',
          activeOperator:claim?.activeOperator??null};
      });
      const acceptedClaimCount=claimResults===null?null:
        claimResults.filter(c=>c.claimStatus==='ACCEPTED').length;
      const notVerifiedCount=claimResults===null?null:
        claimResults.filter(c=>c.claimStatus==='NOT_VERIFIED').length;
      const notClaimedCount=claimResults===null?null:
        claimResults.filter(c=>c.claimStatus==='NOT_CLAIMED').length;
      const activeWebResponsibilityCount=claimResults===null?null:
        claimResults.filter(c=>c.responsibilityStatus==='VERIFIED').length;
      const claimByTaskState=new Map(tasks.map(task=>[String(task.id),task.claimState??'NOT_VERIFIED']));
      const dagNodes=native.dag.nodes.map(node=>({...node,
        claimState:claimByTaskState.get(String(node.taskId))??'NOT_VERIFIED'}));
      const plannedEdges=before.frozen.edges.filter(e=>e.kind==='PLANNED');
      const dag={...native.dag,edges:[...native.dag.edges,...plannedEdges],
        rosterCoverage:'VERIFIED_COMPLETE',claimReadbackCoverage:claimReadback,
        claimCoverage:claimResults===null?'NOT_VERIFIED':
          acceptedClaimCount===native.tasks.length?'ACCEPTED_ALL':
          acceptedClaimCount>0?'PARTIAL':'NONE_ACCEPTED',
        acceptedClaimCount,notVerifiedCount,notClaimedCount,
        activeWebResponsibilityCount,nodes:dagNodes,
        asOf:{...native.dag.asOf,frozenAt:before.frozenAt??null}};
      return {connected:true,source:native.source,versionId,
        rosterCoverage:'VERIFIED_COMPLETE',
        dependencyCoverage:sameSet(planned,observed)?'NATIVE_COMPLETE':'NATIVE_PARTIAL',
        frozenBlobSha:before.frozen.blobSha,taskPackRef,
        taskCount:native.tasks.length,nativeEdgeCount:observed.size,
        frozenEdgeCount:before.frozen.edges.length,claimReadbackCoverage:claimReadback,
        claimCoverage:claimResults===null?'NOT_VERIFIED':
          acceptedClaimCount===native.tasks.length?'ACCEPTED_ALL':
          acceptedClaimCount>0?'PARTIAL':'NONE_ACCEPTED',
        acceptedClaimCount,notVerifiedCount,notClaimedCount,
        activeWebResponsibilityCount,tasks,dag};
    }catch{return denial('ROSTER_SOURCE_READ_FAILED');}
  }
  return Object.freeze({roster});
}
