/** Read-only cross-project FeedbackCase evidence stages. No cross-repo write authority. */
const SHA=/^[0-9a-f]{40}$/;
const REF=/^https:\/\/github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+\/(?:issues|pull|commit|releases)\/[A-Za-z0-9_.~/#-]+$/;
const array=value=>Array.isArray(value)?value:[];
const validRef=value=>typeof value==='string'&&REF.test(value)&&!value.includes('/../')&&!value.endsWith('/..')&&
  !value.includes('/./')&&!value.endsWith('/.');
const iso=value=>typeof value==='string'&&Number.isFinite(Date.parse(value));
function stage(value,exact=false){
  if(value===undefined||value===null)return {status:'NOT_RUN',sourceRef:null,sourceSha:null};
  const shaValid=value.sourceSha===undefined||SHA.test(value.sourceSha);
  return validRef(value.ref)&&shaValid&&(!exact||SHA.test(value.sourceSha??''))?
    {status:'OBSERVED',sourceRef:value.ref,sourceSha:value.sourceSha??null}:{status:'UNKNOWN',sourceRef:null,sourceSha:null};
}
function validCase(row){
  return row&&typeof row==='object'&&!Array.isArray(row)&&typeof row.id==='string'&&row.id.length>0&&
    typeof row.sourceProject==='string'&&typeof row.sourceVersion==='string'&&validRef(row.sourceIssueRef)&&
    SHA.test(row.sourceSha??'')&&typeof row.classification==='string'&&row.classification.length>0&&
    typeof row.ownerRepo==='string'&&typeof row.ownerVersion==='string'&&validRef(row.ownerIssueRef)&&
    Array.isArray(row.observedEvidence)&&row.observedEvidence.every(validRef)&&row.stages&&typeof row.stages==='object';
}
/** Native/public observations only. CLOSED/merged is not a stage shortcut. */
export function projectFeedbackCases(input={}){
  const {observedAsOf,cases=[]}=input??{};
  if(!iso(observedAsOf)||!Array.isArray(cases)||!cases.every(validCase)||
    new Set(cases.map(row=>row.id)).size!==cases.length)return Object.freeze({status:'UNKNOWN',cases:[],duplicates:[],
    coverage:{comparison:'COUNTS_NOT_PERCENTAGES',taskPass:false,releasePass:false},advisoryOnly:true});
  const projected=cases.map(row=>{
    const available=row.sourceAvailable!==false&&row.observedEvidence.length>0;
    const stages={REPORTED:available?{status:'OBSERVED',sourceRef:row.sourceIssueRef,sourceSha:row.sourceSha}:
      {status:'UNKNOWN',sourceRef:null,sourceSha:null},TRIAGED:stage(row.stages.triaged),
      OWNER_ACCEPTED:stage(row.stages.ownerAccepted),FIX_PR_REVIEWED:stage(row.stages.fixReview,true),
      RELEASED:stage(row.stages.release,true),DOWNSTREAM_RETESTED:stage(row.stages.downstreamRetest,true)};
    if(stages.DOWNSTREAM_RETESTED.status==='OBSERVED'&&row.stages.downstreamRetest.selectedPath!=='PASS')
      stages.DOWNSTREAM_RETESTED={status:'UNKNOWN',sourceRef:null,sourceSha:null};
    let outcome='NOT_PROVEN';
    if(!available)outcome='UNKNOWN';
    else if(row.classification==='LOCAL_MISUSE'&&validRef(row.localMisuseEvidence))outcome='NO_UPSTREAM_ISSUE_REQUIRED';
    else if(validRef(row.stages.deferred?.ref))outcome='DEFERRED_VALIDLY';
    else if(Object.values(stages).every(value=>value.status==='OBSERVED'))outcome='CONSUMER_CONFIRMED';
    return {type:'FeedbackCase',id:row.id,source:{project:row.sourceProject,version:row.sourceVersion,
      issueRef:row.sourceIssueRef,sourceSha:row.sourceSha,evidence:row.observedEvidence},classification:row.classification,
      suggestedOwner:{repo:row.ownerRepo,version:row.ownerVersion,issueRef:row.ownerIssueRef},stages,outcome,
      targetVersionChanged:row.sourceVersion!==row.ownerVersion,
      retestRegression:validRef(row.retestRegressionRef)?row.retestRegressionRef:null,
      duplicateFingerprint:typeof row.fingerprint==='string'?row.fingerprint:null,
      authorizationNeeded:true,mustNot:['DO_NOT_CREATE_CROSS_REPO_ISSUE','DO_NOT_CLOSE_FROM_REFERENCE',
        'DO_NOT_INFER_DOWNSTREAM_PASS_FROM_UPSTREAM_CLOSED'],observedAsOf};
  });
  const duplicates=[];
  for(let left=0;left<projected.length;left++)for(let right=left+1;right<projected.length;right++){
    const fingerprint=projected[left].duplicateFingerprint;
    if(fingerprint&&fingerprint===projected[right].duplicateFingerprint)duplicates.push({
      status:'POTENTIAL_DUPLICATE',fingerprint,cases:[projected[left].id,projected[right].id],autoMerge:false});
  }
  const count=(field,value)=>projected.filter(row=>row[field]===value).length;
  return Object.freeze({status:'OBSERVED',observedAsOf,cases:projected,duplicates,coverage:{
    caseCount:projected.length,consumerConfirmedCount:count('outcome','CONSUMER_CONFIRMED'),
    notProvenCount:count('outcome','NOT_PROVEN'),unknownCount:count('outcome','UNKNOWN'),
    ownerAcceptedCount:projected.filter(row=>row.stages.OWNER_ACCEPTED.status==='OBSERVED').length,
    localMisuseCount:count('outcome','NO_UPSTREAM_ISSUE_REQUIRED'),
    deferredValidlyCount:count('outcome','DEFERRED_VALIDLY'),potentialDuplicateCount:duplicates.length,
    targetVersionChangeCount:projected.filter(row=>row.targetVersionChanged).length,
    retestRegressionCount:projected.filter(row=>row.retestRegression).length,
    comparison:'COUNTS_NOT_PERCENTAGES',efficiencyGain:'NOT_MEASURED',taskPass:false,releasePass:false},advisoryOnly:true});
}
