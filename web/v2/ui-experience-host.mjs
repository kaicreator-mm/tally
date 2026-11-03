import {createHash} from 'node:crypto';
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
const COMMENT_REF=/^https:\/\/github\.com\/([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)\/issues\/([1-9][0-9]*)(?:#issuecomment-([1-9][0-9]*))?$/;
const publisherSet=(repository,extra)=>{
  if(extra!==undefined&&(!Array.isArray(extra)||extra.length>20||
     extra.some(x=>!REPO.test(x))))throw new TypeError('TRUSTED_PUBLISHER_REPOSITORIES_INVALID');
  return new Set([repository,...(extra??[])]);
};
const validCandidate=(v,repo,publishers)=> {
  const ref=COMMENT_REF.exec(v?.sourceRef??'');
  const publisher=v?.publisherRepository??ref?.[1]??null;
  const sameIssue=ref&&publisher===ref[1]&&publishers.has(publisher)&&
    (ref[3]!==undefined||publisher===repo);
  return plain(v)&&ID.test(v.id??'')&&
    Number.isSafeInteger(v.revision)&&v.revision>=1&&
    ['PROPOSED','REVIEWED','ADOPTED','REJECTED'].includes(v.state)&&
    ['VERIFIED','STALE','UNVERIFIED','UNKNOWN'].includes(v.reportQualification)&&
    typeof v.title==='string'&&v.title.length>0&&v.title.length<=160&&
    sameIssue&&
    (v.state!=='ADOPTED'||v.reportQualification==='VERIFIED')&&
    typeof v.eligibleForNextTask==='boolean'&&
    (v.eligibleForNextTask===false||v.state==='ADOPTED'&&v.reportQualification==='VERIFIED')&&
    (v.sourceKind===undefined||['PUBLIC_ADS_REPORT','ISOLATED_E2E_TEST_ONLY'].includes(v.sourceKind));
};
export function createUiExperienceHost({repository,versionId,expectedSha,selectedHost,
  trustedPublisherRepositories}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedSha??''))throw TypeError('EXPERIENCE_SELECTED_SOURCE_REQUIRED');
  const publishers=publisherSet(repository,trustedPublisherRepositories);
  const read=async()=>{
    if(typeof selectedHost?.readExperienceCandidates!=='function')
      return denied('EXPERIENCE_HOST_NOT_CONNECTED');
    try{
      const value=await selectedHost.readExperienceCandidates({repository,versionId,
        sourceSha:expectedSha});
      if(!positive(value,repository,versionId,expectedSha)||
         value.candidates.some(v=>!validCandidate(v,repository,publishers))||
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
        sourceRef:c.sourceRef,publisherRepository:c.publisherRepository??null,
        sourceKind:c.sourceKind??'PUBLIC_ADS_REPORT',
        eligibleForNextTask:c.eligibleForNextTask}))};
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
       (request.decision==='ADOPT'&&candidate.reportQualification!=='VERIFIED'))
      return denied('EXPERIENCE_EVIDENCE_NOT_ELIGIBLE');
    if(typeof selectedHost?.dispatchExperienceDecision!=='function')
      return denied('EXPERIENCE_K_HOST_DISPATCH_NOT_CONNECTED');
    const recheck=await read();
    if(!positive(recheck,repository,versionId,expectedSha))return recheck;
    const fresh=recheck.candidates.find(c=>c.id===candidate.id);
    if(!fresh||fresh.revision!==candidate.revision||
       fresh.state!==candidate.state||fresh.sourceRef!==candidate.sourceRef||
       fresh.reportQualification!==candidate.reportQualification||
       fresh.publisherRepository!==candidate.publisherRepository||
       fresh.eligibleForNextTask!==candidate.eligibleForNextTask)
      return denied('EXPERIENCE_DECISION_STALE');
    try{
      // Only an existing selected K/Host reviewer/Owner may accept this request.
      // No browser-supplied grant, automatic future-Task adoption or fake gain.
      const r=await selectedHost.dispatchExperienceDecision({
        repository,versionId,sourceSha:expectedSha,
        candidateId:candidate.id,revision:candidate.revision,
        decision:request.decision,evidenceRef:candidate.sourceRef,
        reportQualification:candidate.reportQualification,
        publisherRepository:candidate.publisherRepository??null});
      const expectedDecision=request.decision==='ADOPT'?'ADOPTED':'REJECTED';
      const githubReceipt=typeof r?.decisionReceiptRef==='string'&&
        r.decisionReceiptRef.startsWith('https://github.com/'+repository+'/');
      const casReceipt=r?.decisionReceiptAuthority===
        'SELECTED_HOST_K_RESPONSE_AND_HOST_SNAPSHOT'&&
        r.decisionReceiptRef===null;
      if(!plain(r)||r.candidateId!==candidate.id||r.sourceSha!==expectedSha||
         !['OBSERVED','DENIED','UNKNOWN','RECONCILIATION_REQUIRED'].includes(r.status)||
         r.taskPass===true||r.releasePass===true||r.qualityImproved===true||
         (r.status==='OBSERVED'&&r.decision!==expectedDecision)||
         (r.status==='OBSERVED'&&!githubReceipt&&!casReceipt))
        return denied('EXPERIENCE_DECISION_RECEIPT_UNQUALIFIED');
      return {connected:true,ok:r.status==='OBSERVED',status:r.status,
        candidateId:candidate.id,sourceSha:expectedSha,
        receiptRef:r.status==='OBSERVED'&&githubReceipt?r.decisionReceiptRef:null,
        receiptAuthority:r.status==='OBSERVED'&&casReceipt?
          'SELECTED_HOST_K_RESPONSE_AND_HOST_SNAPSHOT':null,
        nextTaskUsage:'NOT_VERIFIED',measuredBenefit:'UNKNOWN'};
    }catch{return {connected:false,ok:false,status:'UNKNOWN',
      reason:'EXPERIENCE_DECISION_ACK_UNKNOWN_RECONCILE'};}
  }
  async function nextTaskUsage(taskIssue){
    if(!Number.isSafeInteger(taskIssue)||taskIssue<1||
       typeof selectedHost?.readNextTaskContextUsage!=='function')
      return denied('NEXT_TASK_CONTEXT_NOT_CONNECTED');
    try{
      const current=await read();
      if(!positive(current,repository,versionId,expectedSha))return current;
      const r=await selectedHost.readNextTaskContextUsage({repository,versionId,
        sourceSha:expectedSha,taskIssue});
      const after=await read();
      if(!positive(after,repository,versionId,expectedSha))return after;
      const afterById=new Map(after.candidates.map(c=>[c.id,c]));
      const stable=current.candidates.length===after.candidates.length&&
        current.candidates.every(c=>{
          const next=afterById.get(c.id);
          return next&&next.revision===c.revision&&next.state===c.state&&
            next.reportQualification===c.reportQualification&&
            next.sourceRef===c.sourceRef&&
            next.publisherRepository===c.publisherRepository&&
            next.sourceKind===c.sourceKind&&
            next.eligibleForNextTask===c.eligibleForNextTask;
        });
      if(!stable)return denied('EXPERIENCE_SOURCE_CHANGED_DURING_USAGE_READ');
      const eligibleIds=new Set(current.candidates.filter(c=>
        c.state==='ADOPTED'&&c.reportQualification==='VERIFIED'&&
        c.eligibleForNextTask===true).map(c=>c.id));
      if(r?.connected!==true||r.source?.repository!==repository||
         r.source?.sourceSha!==expectedSha||r.source?.currentness!=='CURRENT'||
         r.source?.readback!=='VERIFIED'||r.taskIssue!==taskIssue||
         !Array.isArray(r.usedCandidates)||r.usedCandidates.length>100||
         new Set(r.usedCandidates).size!==r.usedCandidates.length||
         r.usedCandidates.some(id=>!ID.test(id)||!eligibleIds.has(id))||
         !['EXPLICITLY_APPLIED','EXPLICITLY_NOT_APPLIED'].includes(r.usageState)||
         (r.usageState==='EXPLICITLY_NOT_APPLIED'&&r.usedCandidates.length!==0)||
         (r.usageState==='EXPLICITLY_APPLIED'&&r.usedCandidates.length===0))
        return denied('NEXT_TASK_CONTEXT_USAGE_UNQUALIFIED');
      return {connected:true,taskIssue,usageState:r.usageState,
        usedCandidates:r.usedCandidates,measuredBenefit:'UNKNOWN'};
    }catch{return denied('NEXT_TASK_CONTEXT_READ_FAILED');}
  }
  return Object.freeze({candidates,decision,nextTaskUsage});
}

const issueRef=(value,repository)=>{
  if(typeof value!=='string')return false;
  const prefix=`https://github.com/${repository}/issues/`;
  return value.startsWith(prefix)&&/^[1-9][0-9]*$/.test(value.slice(prefix.length));
};
function publisherRepositoryFor(namespace,repository){
  if(namespace==='tally')return repository;
  const match=/^task-meta:([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)$/.exec(namespace??'');
  return match?.[1]??null;
}
function learningIdentity(p){
  return createHash('sha256').update(JSON.stringify([
    p.reportId,p.revision,p.commentId,p.kind
  ])).digest('hex');
}
function verifiedSuggestion(row,repository,publishers){
  const p=row?.provenance;
  const publisherRepository=publisherRepositoryFor(p?.namespace,repository);
  const issueUrl=publisherRepository&&Number.isSafeInteger(p?.issueNumber)&&p.issueNumber>0
    ?'https://github.com/'+publisherRepository+'/issues/'+p.issueNumber:null;
  const qualification=['REPORTED_QUALIFIED','STALE','UNVERIFIED','UNKNOWN'].includes(row?.qualification);
  return plain(row)&&plain(p)&&
    typeof p.reportId==='string'&&p.reportId.length>0&&p.reportId.length<=512&&
    Number.isSafeInteger(p.revision)&&p.revision>0&&
    ['EXECUTION_SUMMARY','EXPERIENCE_USE_FEEDBACK'].includes(p.kind)&&
    Number.isSafeInteger(p.commentId)&&p.commentId>0&&
    Number.isSafeInteger(p.issueNumber)&&p.issueNumber>0&&
    typeof p.namespace==='string'&&p.namespace.length>0&&p.namespace.length<=512&&
    p.repository===publisherRepository&&
    /^[a-f0-9]{40}$/.test(p.sourceSha??'')&&p.sourceSha===row.subjectSha&&
    /^[a-f0-9]{40}$/.test(row.subjectSha??'')&&
    row.learningId===learningIdentity(p)&&
    /^[a-f0-9]{64}$/.test(p.dataDigest??'')&&
    /^[a-f0-9]{64}$/.test(row.learningId??'')&&
    qualification&&publishers.has(publisherRepository)&&
    (row.taskIssueRef===null||row.taskIssueRef===undefined||
      issueRef(row.taskIssueRef,publisherRepository))&&
    row.targetIssueRef===issueUrl&&
    p.publicationRef===issueUrl+'#issuecomment-'+p.commentId&&
    ['NOT_MEASURED','ATTESTED_OUTCOME_ONLY'].includes(row.measuredEffect)&&
    row.noAgentRanking===true&&row.qualityGainClaimed===false;
}
/** Reuses the existing J08 maintenance controller: ADS report loading and
 * record-learning already traverse its selected Host reader and sealed B→K→C
 * effect. This adapter adds no registry, scheduler, or GitHub writer.
 */
export function createMaintenanceControllerExperienceAdapter({
  maintenanceController,repository,versionId,expectedSha,
  trustedPublisherRepositories
}={}){
  if(!REPO.test(repository??'')||!VERSION.test(versionId??'')||
     !SHA.test(expectedSha??'')||
     typeof maintenanceController?.dispatch!=='function'||
     typeof maintenanceController?.snapshot!=='function')
    throw new TypeError('SELECTED_MAINTENANCE_CONTROLLER_REQUIRED');
  const publishers=publisherSet(repository,trustedPublisherRepositories);
  async function readExperienceCandidates(request={}){
    if(request.repository!==undefined&&request.repository!==repository||
       request.versionId!==undefined&&request.versionId!==versionId||
       request.sourceSha!==undefined&&request.sourceSha!==expectedSha)
      return {connected:false,reason:'EXPERIENCE_SOURCE_SCOPE_MISMATCH'};
    try{
      const loaded=await maintenanceController.dispatch(expectedSha,'load-learning',{});
      const snapshot=maintenanceController.snapshot();
      if(loaded?.status!=='OK'||loaded?.reason!=='PUBLIC_ADS_REPORTS_OBSERVED'||
         snapshot?.status!=='OBSERVED'||snapshot.sourceSha!==expectedSha||
         !Array.isArray(snapshot.learning)||snapshot.learning.length>100)
        return {connected:false,reason:'PUBLIC_ADS_REPORTS_NOT_CURRENT'};
      const candidates=[];
      for(const row of snapshot.learning){
        if(!verifiedSuggestion(row,repository,publishers))
          return {connected:false,reason:'ADS_LEARNING_PROVENANCE_UNQUALIFIED'};
        const decision=row.decision??null;
        if(decision!==null&&!['ADOPTED','REJECTED'].includes(decision))
          return {connected:false,reason:'ADS_LEARNING_DECISION_UNQUALIFIED'};
        const state=decision==='ADOPTED'?'ADOPTED':
          decision==='REJECTED'?'REJECTED':'PROPOSED';
        const reportQualification=row.qualification==='REPORTED_QUALIFIED'?'VERIFIED':
          row.qualification==='STALE'?'STALE':
          row.qualification==='UNVERIFIED'?'UNVERIFIED':'UNKNOWN';
        const publisherRepository=publisherRepositoryFor(
          row.provenance.namespace,repository);
        candidates.push({
          id:row.learningId,revision:row.provenance.revision,
          title:'Task '+(row.taskIssueRef?.split('/').pop()??row.provenance.issueNumber)+
            ' → Task '+row.targetIssueRef.split('/').pop()+
            ' · '+row.provenance.reportId,
          state,reportQualification,publisherRepository,
          sourceKind:publisherRepository==='kaicreator-mm/task-meta-e2e'?
            'ISOLATED_E2E_TEST_ONLY':'PUBLIC_ADS_REPORT',
          sourceRef:row.provenance.publicationRef,
          // Adoption is not proof that a later Task Context Pack used it.
          eligibleForNextTask:false
        });
      }
      return {connected:true,source:{repository,sourceSha:expectedSha,
        currentness:'CURRENT',readback:'VERIFIED'},
        versionId,candidates};
    }catch{return {connected:false,reason:'ADS_LEARNING_READ_FAILED'};}
  }
  async function dispatchExperienceDecision(request){
    if(request?.repository!==repository||request?.versionId!==versionId||
       request?.sourceSha!==expectedSha||
       !/^[a-f0-9]{64}$/.test(request?.candidateId??'')||
       !Number.isSafeInteger(request?.revision)||request.revision<1||
       !['ADOPT','REJECT'].includes(request?.decision))
      return {status:'DENIED',candidateId:request?.candidateId??null,
        sourceSha:expectedSha,reason:'EXPERIENCE_DECISION_SCOPE_INVALID',
        taskPass:false,releasePass:false};
    try{
      // The selected Host re-loads the public report set immediately before
      // dispatch. UI validation is convenience only; it is never authority.
      const loaded=await maintenanceController.dispatch(expectedSha,'load-learning',{});
      const before=maintenanceController.snapshot();
      if(loaded?.status!=='OK'||loaded?.reason!=='PUBLIC_ADS_REPORTS_OBSERVED'||
         before?.status!=='OBSERVED'||before.sourceSha!==expectedSha||
         !Array.isArray(before.learning))
        return {status:'DENIED',candidateId:request.candidateId,
          sourceSha:expectedSha,reason:'LEARNING_SOURCE_NOT_CURRENT',
          taskPass:false,releasePass:false};
      const candidate=before.learning.find(x=>x.learningId===request.candidateId);
      const publisherRepository=publisherRepositoryFor(
        candidate?.provenance?.namespace,repository);
      const qualification=candidate?.qualification==='REPORTED_QUALIFIED'?'VERIFIED':
        candidate?.qualification==='STALE'?'STALE':
        candidate?.qualification==='UNVERIFIED'?'UNVERIFIED':'UNKNOWN';
      if(!candidate||!verifiedSuggestion(candidate,repository,publishers)||
         candidate.provenance.revision!==request.revision||
         request.evidenceRef!==candidate.provenance.publicationRef||
         request.publisherRepository!==publisherRepository||
         request.reportQualification!==qualification||
         (candidate.decision!==null&&candidate.decision!==undefined))
        return {status:'DENIED',candidateId:request.candidateId,
          sourceSha:expectedSha,reason:'LEARNING_CANDIDATE_STALE_OR_ALREADY_DECIDED',
          taskPass:false,releasePass:false};
      if(request.decision==='ADOPT'&&candidate.qualification!=='REPORTED_QUALIFIED')
        return {status:'DENIED',candidateId:request.candidateId,
          sourceSha:expectedSha,reason:'UNQUALIFIED_REPORT_CANNOT_BE_ADOPTED',
          taskPass:false,releasePass:false};
      const result=await maintenanceController.dispatch(expectedSha,'record-learning',{
        learningId:request.candidateId,
        decision:request.decision==='ADOPT'?'ADOPTED':'REJECTED'});
      const snapshot=maintenanceController.snapshot();
      const row=snapshot?.learning?.find(x=>x.learningId===request.candidateId);
      const expectedDecision=request.decision==='ADOPT'?'ADOPTED':'REJECTED';
      if(snapshot?.status!=='OBSERVED'||snapshot.sourceSha!==expectedSha||
         !Array.isArray(snapshot.learning)||snapshot.learning.length>100||
         !verifiedSuggestion(row,repository,publishers)||
         result?.state!=='LEARNING_RECORDED'||result?.decision!==expectedDecision||
         result?.claimAccepted!==true||result?.admitted!==true||
         result?.authorized!==false||result?.taskPass!==false||
         result?.releasePass!==false||result?.githubEffect!=='NOT_RUN'||
         row?.learningId!==candidate.learningId||
         row?.provenance?.reportId!==candidate.provenance.reportId||
         row?.provenance?.revision!==candidate.provenance.revision||
         row?.provenance?.publicationRef!==candidate.provenance.publicationRef||
         row?.provenance?.dataDigest!==candidate.provenance.dataDigest||
         row?.qualification!==candidate.qualification||
         row?.targetIssueRef!==candidate.targetIssueRef||
         row?.subjectSha!==candidate.subjectSha||
         row?.decision!==expectedDecision||row?.recordState!=='LEARNING_RECORDED'||
         row?.qualityGainClaimed!==false||row?.noAgentRanking!==true||
         row?.measuredEffect!=='NOT_MEASURED')
        return {status:'UNKNOWN',candidateId:request.candidateId,
          sourceSha:expectedSha,reason:'LEARNING_CAS_RESULT_NOT_RECONCILED',
          taskPass:false,releasePass:false};
      return {status:'OBSERVED',candidateId:request.candidateId,
        sourceSha:expectedSha,decision:expectedDecision,
        decisionReceiptAuthority:
          'SELECTED_HOST_K_RESPONSE_AND_HOST_SNAPSHOT',
        decisionReceiptRef:null,taskPass:false,releasePass:false,
        qualityImproved:false};
    }catch{return {status:'UNKNOWN',candidateId:request.candidateId,
      sourceSha:expectedSha,reason:'LEARNING_DECISION_ACK_UNKNOWN',
      taskPass:false,releasePass:false};}
  }
  return Object.freeze({readExperienceCandidates,dispatchExperienceDecision});
}
