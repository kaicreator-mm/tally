/** Closeout C/#262: governed Task Learning → next Task Context Pack Host seam.
 * Existing ADS v5 and #221 Business/Kernel own the actual provenance,
 * review, decision and Context Pack mutation; this adapter never grants it.
 */
const SHA=/^[0-9a-f]{40}$/;
const REPO=/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const VERSION=/^version\/[A-Za-z0-9_.-]+$/;
const ID=/^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$/;
const plain=v=>v!==null&&typeof v==='object'&&!Array.isArray(v);
const denied=reason=>({connected:false,ok:false,status:'DENIED',reason});
const positive=(v,repo,versionId,sha)=>v?.connected===true&&
  v.source?.repository===repo&&v.source?.sourceSha===sha&&
  v.source?.currentness==='CURRENT'&&v.source?.readback==='VERIFIED'&&
  v.versionId===versionId&&Array.isArray(v.candidates)&&v.candidates.length<=100;
const validCandidate=(v,repo)=>plain(v)&&ID.test(v.id??'')&&
  Number.isSafeInteger(v.revision)&&v.revision>=1&&
  ['PROPOSED','REVIEWED','ADOPTED','REJECTED'].includes(v.state)&&
  ['VERIFIED','STALE','UNVERIFIED','UNKNOWN'].includes(v.reportQualification)&&
  typeof v.title==='string'&&v.title.length>0&&v.title.length<=160&&
  typeof v.sourceRef==='string'&&
  v.sourceRef.startsWith('https://github.com/'+repo+'/issues/')&&
  (v.state!=='ADOPTED'||v.reportQualification==='VERIFIED')&&
  typeof v.eligibleForNextTask==='boolean'&&
  (v.eligibleForNextTask===false||v.state==='ADOPTED'&&v.reportQualification==='VERIFIED');
export function createUiExperienceHost({repository,versionId,expectedSha,selectedHost}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedSha??''))throw TypeError('EXPERIENCE_SELECTED_SOURCE_REQUIRED');
  const read=async()=>{
    if(typeof selectedHost?.readExperienceCandidates!=='function')
      return denied('EXPERIENCE_HOST_NOT_CONNECTED');
    try{
      const value=await selectedHost.readExperienceCandidates({repository,versionId,
        sourceSha:expectedSha});
      if(!positive(value,repository,versionId,expectedSha)||
         value.candidates.some(v=>!validCandidate(v,repository))||
         new Set(value.candidates.map(v=>v.id)).size!==value.candidates.length)
        return denied('EXPERIENCE_PROVENANCE_UNQUALIFIED');
      return value;
    }catch{return denied('EXPERIENCE_HOST_READ_FAILED');}
  };
  async function candidates(scope={}){
    if(scope.repository!==undefined&&scope.repository!==repository||
       scope.versionId!==undefined&&scope.versionId!==versionId)
      return denied('EXPERIENCE_SCOPE_MISMATCH');
    const v=await read();
    if(!positive(v,repository,versionId,expectedSha))return v;
    return {connected:true,versionId,source:v.source,
      qualification:'SOURCE_OBSERVED_NOT_MEASURED_OUTCOME',
      candidates:v.candidates.map(c=>({id:c.id,revision:c.revision,
        title:c.title,state:c.state,reportQualification:c.reportQualification,
        sourceRef:c.sourceRef,eligibleForNextTask:c.eligibleForNextTask}))};
  }
  async function decision(request){
    if(!plain(request)||Object.keys(request).sort().join(',')!==
      'candidateId,decision,repository,revision,sourceSha,versionId'||
      request.repository!==repository||request.versionId!==versionId||
      request.sourceSha!==expectedSha||!ID.test(request.candidateId??'')||
      !Number.isSafeInteger(request.revision)||request.revision<1||
      !['ADOPT','REJECT'].includes(request.decision))
      return denied('EXPERIENCE_DECISION_INVALID');
    const v=await read();
    if(!positive(v,repository,versionId,expectedSha))return v;
    const candidate=v.candidates.find(c=>c.id===request.candidateId);
    if(!candidate||candidate.revision!==request.revision||
       !['PROPOSED','REVIEWED'].includes(candidate.state)||
       candidate.reportQualification!=='VERIFIED')
      return denied('EXPERIENCE_EVIDENCE_NOT_ELIGIBLE');
    if(typeof selectedHost?.dispatchExperienceDecision!=='function')
      return denied('EXPERIENCE_K_HOST_DISPATCH_NOT_CONNECTED');
    const recheck=await read();
    if(!positive(recheck,repository,versionId,expectedSha))return recheck;
    const fresh=recheck.candidates.find(c=>c.id===candidate.id);
    if(!fresh||fresh.revision!==candidate.revision||
       fresh.state!==candidate.state||fresh.sourceRef!==candidate.sourceRef||
       fresh.reportQualification!=='VERIFIED')
      return denied('EXPERIENCE_DECISION_STALE');
    try{
      // Only an existing selected K/Host reviewer/Owner may accept this request.
      // No browser-supplied grant, automatic future-Task adoption or fake gain.
      const r=await selectedHost.dispatchExperienceDecision({
        repository,versionId,sourceSha:expectedSha,
        candidateId:candidate.id,revision:candidate.revision,
        decision:request.decision,evidenceRef:candidate.sourceRef});
      if(!plain(r)||r.candidateId!==candidate.id||r.sourceSha!==expectedSha||
         !['OBSERVED','DENIED','UNKNOWN','RECONCILIATION_REQUIRED'].includes(r.status)||
         r.taskPass===true||r.releasePass===true||r.qualityImproved===true||
         (r.status==='OBSERVED'&&
           (typeof r.decisionReceiptRef!=='string'||
            !r.decisionReceiptRef.startsWith('https://github.com/'+repository+'/'))))
        return denied('EXPERIENCE_DECISION_RECEIPT_UNQUALIFIED');
      return {connected:true,ok:r.status==='OBSERVED',status:r.status,
        candidateId:candidate.id,sourceSha:expectedSha,
        receiptRef:r.status==='OBSERVED'?r.decisionReceiptRef:null,
        nextTaskUsage:'NOT_VERIFIED',measuredBenefit:'UNKNOWN'};
    }catch{return {connected:false,ok:false,status:'UNKNOWN',
      reason:'EXPERIENCE_DECISION_ACK_UNKNOWN_RECONCILE'};}
  }
  async function nextTaskUsage(taskIssue){
    if(!Number.isSafeInteger(taskIssue)||taskIssue<1||
       typeof selectedHost?.readNextTaskContextUsage!=='function')
      return denied('NEXT_TASK_CONTEXT_NOT_CONNECTED');
    try{
      const r=await selectedHost.readNextTaskContextUsage({repository,versionId,
        sourceSha:expectedSha,taskIssue});
      if(r?.connected!==true||r.source?.repository!==repository||
         r.source?.sourceSha!==expectedSha||r.source?.currentness!=='CURRENT'||
         r.source?.readback!=='VERIFIED'||r.taskIssue!==taskIssue||
         !Array.isArray(r.usedCandidates)||r.usedCandidates.length>100||
         r.usedCandidates.some(id=>!ID.test(id))||
         !['EXPLICITLY_APPLIED','EXPLICITLY_NOT_APPLIED'].includes(r.usageState))
        return denied('NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
      return {connected:true,taskIssue,usageState:r.usageState,
        usedCandidates:r.usedCandidates,measuredBenefit:'UNKNOWN'};
    }catch{return denied('NEXT_TASK_CONTEXT_READ_FAILED');}
  }
  return Object.freeze({candidates,decision,nextTaskUsage});
}
