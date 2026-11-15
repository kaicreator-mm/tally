/** T06m/T06n/T210/T219/J222/E0-#284 sealed Local Codex C boundary. Provider output is never Task evidence. */
import {runLocalCodexAttempt as nativeRunLocalCodexAttempt,
  joinReviewedCandidates as nativeJoinReviewedCandidates,
  publishReviewedCandidate as nativePublishReviewedCandidate,
  hostProjectIntake as nativeHostProjectIntake,
  freezeProductIntake as nativeFreezeProductIntake,
  persistProductDraft312 as nativePersistProductDraft312,
  freezeProductReview312 as nativeFreezeProductReview312,
  persistMaintenanceDraft as nativePersistMaintenanceDraft,
  freezeReviewedMaintenanceCandidate as nativeFreezeReviewedMaintenanceCandidate,
  publishMaintenanceIssue as nativePublishMaintenanceIssue,
  recordTaskLearningOutcome as nativeRecordTaskLearningOutcome,
  hostReleaseDelivery as nativeHostReleaseDelivery,
  addNativeTaskDependency as nativeAddNativeTaskDependency,
  freezeVersionTaskDag as nativeFreezeVersionTaskDag}
  from '@kaicreator/tally-local-codex-host-bundle';

export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-local-codex@v1.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-local-codex';
export const OPERATIONS=Object.freeze(['tally.runLocalCodexAttempt','tally.joinReviewedCandidates','tally.publishReviewedCandidate','tally.freezeProductIntake','tally.hostProjectIntake','tally.persistProductDraft312','tally.freezeProductReview312','tally.persistMaintenanceDraft','tally.freezeReviewedMaintenanceCandidate','tally.publishMaintenanceIssue','tally.recordTaskLearningOutcome','tally.deliverQualifiedRelease','tally.addNativeTaskDependency','tally.freezeVersionTaskDag']);
// Host-owned pins: #172 single Builder plus T06n Builders #177/#178 and the
// separate Reviewer #179. Any other issue is caller authority and refuses.
const ISSUES=new Set([172,177,178,196,197,198]);
const REVIEWER_ISSUE=Number(process.env.TALLY_CONCURRENT_ISSUE_REVIEWER??179);
const OWNER='kaicreator-mm',REPO='tally';
const issueRefFor=issueNumber=>'https://github.com/'+OWNER+'/'+REPO+'/issues/'+issueNumber;
const SHA=/^[0-9a-f]{40}$/;
const ATTEMPT=/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/;
const REASONS=new Set([
  'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','CLAIM_REGISTRY_UNAVAILABLE',
  'CLAIM_REGISTRY_RECORD_NOT_FOUND','CLAIM_REGISTRY_RECORD_INVALID',
  'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH',
  'DAG_WRITESET_NOT_VERIFIED','NATIVE_DEPENDENCIES_BLOCKED','NATIVE_SOURCE_NOT_CURRENT',
  'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','LOCAL_AGENT_UNAVAILABLE',
  'BUILDER_CONFINEMENT_UNAVAILABLE','HIDDEN_VERIFIER_UNAVAILABLE',
  'ATTEMPT_ALREADY_RECONCILED','ATTEMPT_STATUS_UNKNOWN','ATTEMPT_TIMEOUT_RECONCILED',
  'ATTEMPT_CANCELLED','ATTEMPT_PROCESS_FAILED','WRITE_SET_VIOLATION',
  'CANDIDATE_TREE_UNAVAILABLE','LOCAL_AGENT_PROVIDER_THREW',
  'CANDIDATE_WORKTREE_OVERLAP','CANDIDATE_WRITE_GRANT_OVERLAP'
]);
const JOIN_REASONS=new Set([
  ...REASONS,'JOIN_INPUT_INVALID','CANDIDATE_NOT_FOUND','CANDIDATE_NOT_READY',
  'CANDIDATE_STALE','CANDIDATE_READBACK_MISMATCH','CANDIDATE_WRITESET_OVERLAP',
  'REVIEWER_SELF_ASSERTION'
]);
const flags=(admitted,claimAccepted,agentLaunch)=>({claimGeneration:1,claimAccepted,
  admitted,authorized:false,agentLaunch,taskPass:false,releasePass:false});
const common=(input,o,state,reason,fl)=>({state,reason,
  issueRef:issueRefFor(input.issueNumber),sourceSha:input.sourceSha,
  attemptId:input.attemptId,...fl});
const joinCommon=(input,o,state,reason,fl)=>({state,reason,
  issueRef:issueRefFor(input.issueNumber),sourceSha:input.sourceSha,
  joinId:input.joinId,...fl});
const denied=(input,reason)=>Object.freeze({...common(input,null,'NOT_ADMITTED',reason,
  flags(false,false,false)),replayFixture:false,candidateTree:null,candidateCommit:null,process:null});
const joinDenied=(input,reason)=>Object.freeze({...joinCommon(input,null,'NOT_ADMITTED',reason,
  flags(false,false,false)),replayFixture:false,joinedAt:null,candidates:[]});
const validCommon=(input,o)=>o&&typeof o==='object'&&!Array.isArray(o)&&
  o.issueRef===issueRefFor(input.issueNumber)&&o.sourceSha===input.sourceSha&&
  o.attemptId===input.attemptId&&o.claimGeneration===1&&o.authorized===false&&
  o.taskPass===false&&o.releasePass===false;
const validJoinCommon=(input,o)=>o&&typeof o==='object'&&!Array.isArray(o)&&
  o.issueRef===issueRefFor(input.issueNumber)&&o.sourceSha===input.sourceSha&&
  o.joinId===input.joinId&&o.claimGeneration===1&&o.authorized===false&&
  o.taskPass===false&&o.releasePass===false;
const processOut=o=>o.process?Object.freeze({status:o.process.status,
  exitCode:Number.isSafeInteger(o.process.exitCode)?o.process.exitCode:null,
  pid:Number.isSafeInteger(o.process.pid)?o.process.pid:null,
  builderPrincipal:o.process.builderPrincipal,builderSid:o.process.builderSid,
  startedAt:o.process.startedAt,terminatedAt:o.process.terminatedAt}):null;

async function computeLocalCodexAttempt(input){
  if(!input||input.sourceRef!==issueRefFor(input.issueNumber)||
    !ISSUES.has(input.issueNumber)||
    !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.attemptId??''))
    return denied({sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
      issueNumber:ISSUES.has(input?.issueNumber)?input.issueNumber:172,
      attemptId:ATTEMPT.test(input?.attemptId??'')?input.attemptId:null},
      'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  let o;
  try{o=await nativeRunLocalCodexAttempt(input);}catch{return denied(input,'LOCAL_AGENT_PROVIDER_THREW');}
  if(!validCommon(input,o))return denied(input,'LOCAL_AGENT_PROVIDER_THREW');
  if(o.state==='NOT_ADMITTED'&&REASONS.has(o.reason)&&o.admitted===false&&
    o.claimAccepted===false&&o.agentLaunch===false)return denied(input,o.reason);
  if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
    o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&o.replayFixture===true)
    return Object.freeze({...common(input,o,o.state,o.reason,flags(true,true,false)),
      replayFixture:true,candidateTree:null,candidateCommit:null,process:null});
  if(o.state==='LOCAL_CANDIDATE_READY'&&o.reason==='REAL_CODEX_CANDIDATE_RECORDED'&&
    o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===true&&
    o.replayFixture===false&&SHA.test(o.candidateTree??'')&&SHA.test(o.candidateCommit??'')&&
    o.process?.status==='EXITED'&&o.process.exitCode===0&&
    Number.isSafeInteger(o.process.pid)&&typeof o.process.builderPrincipal==='string'&&
    typeof o.process.builderSid==='string')
    return Object.freeze({...common(input,o,o.state,o.reason,flags(true,true,true)),
      replayFixture:false,candidateTree:o.candidateTree,candidateCommit:o.candidateCommit,
      process:processOut(o)});
  if(o.state==='AGENT_NOT_EXECUTED'&&REASONS.has(o.reason)&&o.admitted===true&&
    o.claimAccepted===true&&o.agentLaunch===false)
    return Object.freeze({...common(input,o,o.state,o.reason,flags(true,true,false)),
      replayFixture:false,candidateTree:null,candidateCommit:null,process:null});
  if(['LOCAL_ATTEMPT_UNKNOWN','LOCAL_ATTEMPT_FAILED','LOCAL_ATTEMPT_CANCELLED'].includes(o.state)&&
    REASONS.has(o.reason)&&o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===true&&
    typeof o.process?.status==='string')
    return Object.freeze({...common(input,o,o.state,o.reason,flags(true,true,true)),
      replayFixture:false,candidateTree:SHA.test(o.candidateTree??'')?o.candidateTree:null,
      candidateCommit:SHA.test(o.candidateCommit??'')?o.candidateCommit:null,process:processOut(o)});
  return denied(input,'LOCAL_AGENT_PROVIDER_THREW');
}

async function computeJoinReviewedCandidates(input){
  if(!input||input.sourceRef!==issueRefFor(input.issueNumber)||
    input.issueNumber!==REVIEWER_ISSUE||
    !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.joinId??'')||
    !Array.isArray(input.candidates)||input.candidates.length<2)
    return joinDenied({sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
      issueNumber:REVIEWER_ISSUE,
      joinId:ATTEMPT.test(input?.joinId??'')?input.joinId:null},
      'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH');
  let o;
  try{o=await nativeJoinReviewedCandidates(input);}catch{return joinDenied(input,'LOCAL_AGENT_PROVIDER_THREW');}
  if(!validJoinCommon(input,o))return joinDenied(input,'LOCAL_AGENT_PROVIDER_THREW');
  if(o.state==='NOT_ADMITTED'&&JOIN_REASONS.has(o.reason)&&o.admitted===false&&
    o.claimAccepted===false&&o.agentLaunch===false)return joinDenied(input,o.reason);
  if(o.state==='REPLAY_COMPLETE'&&o.reason==='REPLAY_FIXTURE_ONLY'&&
    o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&o.replayFixture===true)
    return Object.freeze({...joinCommon(input,o,o.state,o.reason,flags(true,true,false)),
      replayFixture:true,joinedAt:null,candidates:[]});
  if(o.state==='REVIEW_JOINED'&&o.reason==='REVIEWED_CANDIDATES_VERIFIED'&&
    o.admitted===true&&o.claimAccepted===true&&o.agentLaunch===false&&o.replayFixture===false&&
    typeof o.joinedAt==='string'&&Array.isArray(o.candidates)&&
    o.candidates.length===input.candidates.length&&
    o.candidates.every((c,i)=>c&&c.attemptId===input.candidates[i].attemptId&&
      SHA.test(c.candidateTree??'')&&SHA.test(c.candidateCommit??'')&&
      typeof c.candidateRelativePath==='string'))
    return Object.freeze({...joinCommon(input,o,o.state,o.reason,flags(true,true,false)),
      replayFixture:false,joinedAt:o.joinedAt,candidates:Object.freeze(o.candidates.map(c=>
        Object.freeze({attemptId:c.attemptId,issueRef:c.issueRef,
          candidateRelativePath:c.candidateRelativePath,candidateTree:c.candidateTree,
          candidateCommit:c.candidateCommit,candidateRef:c.candidateRef??null,
          startedAt:c.startedAt??null,terminatedAt:c.terminatedAt??null})))});
  return joinDenied(input,'LOCAL_AGENT_PROVIDER_THREW');
}

export async function runLocalCodexAttempt(input){
  const output=await computeLocalCodexAttempt(input);
  return Object.freeze({phase:'terminal',externalRef:`tally:${input?.issueNumber??'invalid'}:${input?.attemptId??'invalid'}`,
    outcome:{ok:true,output}});
}

export async function joinReviewedCandidates(input){
  const output=await computeJoinReviewedCandidates(input);
  return Object.freeze({phase:'terminal',externalRef:`tally:${input?.issueNumber??'invalid'}:${input?.joinId??'invalid'}`,
    outcome:{ok:true,output}});
}

const PUBLICATION_REASONS=new Set(['PUBLICATION_TASK_NOT_ALLOWED','PUBLICATION_INPUT_INVALID',
  'GITHUB_PROVIDER_TOKEN_UNAVAILABLE','REVIEW_STORE_UNAVAILABLE','CANDIDATE_REVIEW_NOT_FOUND',
  'CANDIDATE_SUPERSEDED_BY_REPAIR','CANDIDATE_READBACK_MISMATCH','CANDIDATE_REF_MISMATCH',
  'PUBLICATION_IDENTITY_CONFLICT','PUBLICATION_CLOSED_OR_MERGED','GITHUB_ACCESS_DENIED',
  'GITHUB_RATE_LIMITED','GITHUB_PUBLICATION_CONFLICT','GITHUB_PUBLICATION_UNKNOWN',
  'GITHUB_REF_PUBLICATION_UNKNOWN','PUBLICATION_READBACK_MISMATCH','LOCAL_AGENT_PROVIDER_THREW',
  'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','CLAIM_REGISTRY_UNAVAILABLE','CLAIM_REGISTRY_RECORD_NOT_FOUND',
  'CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED','CLAIM_REGISTRY_RESERVATION_MISMATCH',
  'NATIVE_CLAIM_WITNESS_NOT_VERIFIED','NATIVE_SOURCE_NOT_CURRENT','NATIVE_DEPENDENCIES_BLOCKED']);
const publicationDenied=(input,reason,state='NOT_ADMITTED')=>Object.freeze({
  ...common(input,null,state,reason,flags(false,false,false)),replayFixture:false});
export async function publishReviewedCandidate(input){
  let output;
  if(!input||input.sourceRef!==issueRefFor(input.issueNumber)||![197,198].includes(input.issueNumber)||
    !SHA.test(input.sourceSha??'')||!ATTEMPT.test(input.attemptId??''))
    output=publicationDenied({issueNumber:197,sourceSha:null,attemptId:null},'PUBLICATION_INPUT_INVALID');
  else{
    let observed;
    try{observed=await nativePublishReviewedCandidate(input);}catch{observed=null;}
    if(observed?.state==='PUBLISHED_VERIFIED'&&
      observed.reason==='GITHUB_BRANCH_AND_PR_READBACK_VERIFIED'&&validCommon(input,observed)&&
      observed.admitted===true&&observed.claimAccepted===true&&observed.agentLaunch===false&&
      observed.repository===OWNER+'/'+REPO&&observed.baseRef==='version/v2'&&
      SHA.test(observed.candidateCommit??'')&&SHA.test(observed.candidateTree??'')&&
      Number.isSafeInteger(observed.pullNumber)&&observed.pullNumber>0&&
      /^https:\/\/github\.com\/kaicreator-mm\/tally\/pull\/[1-9][0-9]*$/.test(observed.pullRef??'')&&
      observed.externalRef===observed.pullRef)
      output=Object.freeze({...observed,claimGeneration:1,claimAccepted:true,admitted:true,
        authorized:false,agentLaunch:false,taskPass:false,releasePass:false});
    else if(observed&&['NOT_ADMITTED','PUBLICATION_UNKNOWN'].includes(observed.state)&&
      PUBLICATION_REASONS.has(observed.reason))
      output=publicationDenied(input,observed.reason,observed.state);
    else output=publicationDenied(input,'LOCAL_AGENT_PROVIDER_THREW');
  }
  return Object.freeze({phase:'terminal',externalRef:`tally:${input?.issueNumber??'invalid'}:${input?.attemptId??'invalid'}:publication`,
    outcome:{ok:true,output}});
}

// T210 — J01b/J02 governed project intake boundary. The sealed C passes only
// the exact bounded observation shapes; provider garbage or a second writer
// shape fails closed. No Task/Release PASS can cross this boundary.
const INTAKE_ISSUE=210;
const DIGEST=/^[0-9a-f]{64}$/,TITLE=/^.{1,120}$/s;
const PROJECT_URL=/^https:\/\/github\.com\/kaicreator-mm\/tally\/projects\/[1-9][0-9]*$/;
const INTAKE_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'PROJECT_INTAKE_HOST_CONFIG_INVALID','DAG_WRITESET_NOT_VERIFIED',
  'PRODUCT_FREEZE_NOT_VERIFIED','PRODUCT_FREEZE_DIGEST_MISMATCH',
  'PRODUCT_REVIEW_NOT_INDEPENDENT','PRODUCT_FREEZE_STORE_UNAVAILABLE',
  'GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE','GITHUB_PUBLISH_UNKNOWN',
  'PROJECT_INTAKE_READBACK_MISMATCH','PROJECT_INTAKE_INPUT_INVALID']);
const intakeBase=(input,reason,state)=>({state,reason,
  issueRef:issueRefFor(INTAKE_ISSUE),sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  productDigest:DIGEST.test(input?.productDigest??'')?input.productDigest:null,
  claimGeneration:1,claimAccepted:false,admitted:false,authorized:false,
  agentLaunch:false,replayFixture:false,githubEffect:'NOT_RUN',project:null,
  taskPass:false,releasePass:false});
const validIntakeProject=p=>p&&typeof p==='object'&&!Array.isArray(p)&&
  Number.isSafeInteger(p.number)&&p.number>0&&TITLE.test(p.title??'')&&
  PROJECT_URL.test(p.url??'');
const validIntakeOutput=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==issueRefFor(INTAKE_ISSUE)||
    o.sourceSha!==input.sourceSha||o.productDigest!==input.productDigest||
    o.claimGeneration!==1||o.authorized!==false||o.agentLaunch!==false||
    o.taskPass!==false||o.releasePass!==false)return false;
  if(o.state==='NOT_ADMITTED')return INTAKE_REASONS.has(o.reason)&&o.admitted===false&&
    o.claimAccepted===false&&o.githubEffect==='NOT_RUN'&&o.replayFixture===false&&
    o.project===null;
  if(o.state==='REPLAY_COMPLETE')return o.reason==='REPLAY_FIXTURE_ONLY'&&
    o.admitted===true&&o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===true&&o.project===null;
  if(o.state==='PROJECT_INTAKE_UNKNOWN')return INTAKE_REASONS.has(o.reason)&&
    o.admitted===true&&o.claimAccepted===true&&o.githubEffect==='UNKNOWN'&&
    o.replayFixture===false&&o.project===null;
  if(o.state==='PROJECT_INTAKE_ADMITTED')
    return ['PROJECT_INTAKE_PUBLISHED_AND_READBACK_VERIFIED',
      'PROJECT_INTAKE_READBACK_ALREADY_PRESENT'].includes(o.reason)&&
    o.admitted===true&&o.claimAccepted===true&&
    o.githubEffect==='PERFORMED_VERIFIED'&&o.replayFixture===false&&
    o.repository==='kaicreator-mm/tally'&&validIntakeProject(o.project);
  return false;
};
export async function hostProjectIntake(input){
  let output;
  if(!input||typeof input==='object'&&Array.isArray(input)||input.issueNumber!==INTAKE_ISSUE||
    input.sourceRef!==issueRefFor(INTAKE_ISSUE)||!SHA.test(input.sourceSha??'')||
    !DIGEST.test(input.productDigest??'')||!TITLE.test(input.projectTitle??''))
    output=intakeBase(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','NOT_ADMITTED');
  else{
    let observed;
    try{observed=await nativeHostProjectIntake(input);}catch{observed=null;}
    if(validIntakeOutput(input,observed))output=observed;
    else output=intakeBase(input,'LOCAL_AGENT_PROVIDER_THREW','NOT_ADMITTED');
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${INTAKE_ISSUE}:${input?.productDigest??'invalid'}:intake`,
    outcome:{ok:true,output}});
}

// T210/J02 — sealed freeze boundary wrapping the Host CAS freeze operation.
// Only the exact FROZEN_COMPLETE / NOT_FROZEN observation shapes cross; the
// reviewer independence and receipt digest are re-verified here so provider
// garbage can never become a frozen fact or a Task/Release PASS.
const FREEZE_REASONS=new Set(['PRODUCT_FREEZE_INPUT_INVALID','PRODUCT_REVIEW_NOT_INDEPENDENT',
  'PRODUCT_FREEZE_STORE_UNAVAILABLE','PRODUCT_FREEZE_ALREADY_RECORDED',
  'PRODUCT_FREEZE_DIGEST_MISMATCH','LOCAL_AGENT_PROVIDER_THREW']);
const freezeBase=(input,reason)=>({state:'NOT_FROZEN',reason,
  issueRef:issueRefFor(INTAKE_ISSUE),sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  productDigest:DIGEST.test(input?.productDigest??'')?input.productDigest:null,
  claimGeneration:1,frozen:false,replayFixture:false,
  authorized:false,admitted:false,taskPass:false,releasePass:false});
const validFreezeOutput=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==issueRefFor(INTAKE_ISSUE)||
    o.sourceSha!==input.sourceSha||o.productDigest!==input.productDigest||
    o.claimGeneration!==1||o.authorized!==false||o.admitted!==false||
    o.taskPass!==false||o.releasePass!==false)return false;
  if(o.state==='NOT_FROZEN')return o.frozen===false&&o.replayFixture===false&&
    FREEZE_REASONS.has(o.reason);
  if(o.state==='FROZEN_COMPLETE')return o.frozen===true&&o.replayFixture===false&&
    o.receiptRef===`receipt:sha256:${o.productDigest}`&&
    o.review&&typeof o.review==='object'&&!Array.isArray(o.review)&&
    typeof o.review.reviewer==='string'&&o.review.reviewer.length>0&&
    o.review.reviewer!==o.review.author;
  return false;
};
export async function freezeProductIntake(input){
  let output;
  if(!input||typeof input==='object'&&Array.isArray(input)||input.issueNumber!==INTAKE_ISSUE||
    input.sourceRef!==issueRefFor(INTAKE_ISSUE)||!SHA.test(input.sourceSha??'')||
    !DIGEST.test(input.productDigest??'')||typeof input.author!=='string'||
    input.author.length===0||input.author.length>128||
    typeof input.reviewer!=='string'||input.reviewer.length===0||
    input.reviewer.length>128||input.author===input.reviewer)
    output=freezeBase(input,'PRODUCT_REVIEW_NOT_INDEPENDENT');
  else{
    let observed;
    try{observed=await nativeFreezeProductIntake({...input,
      frozenAt:new Date().toISOString()});}catch{observed=null;}
    if(validFreezeOutput(input,observed))output=observed;
    else output=freezeBase(input,'LOCAL_AGENT_PROVIDER_THREW');
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${INTAKE_ISSUE}:${input?.productDigest??'invalid'}:freeze`,
    outcome:{ok:true,output}});
}

// #312 sealed Product ReviewWitness C boundary: never return fabricated
// Host-backed draft, independent review, Task or Release status. The selected
// Host validates accepted roles, real GitHub comment and shared CAS fence.
const PRODUCT312_ISSUE=312,PRODUCT312_REF=issueRefFor(PRODUCT312_ISSUE);
const PRODUCT312_DIGEST=/^[a-f0-9]{64}$/;
const PRODUCT312_REVIEW=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/312#issuecomment-[1-9][0-9]*$/;
const product312Unknown=(input,reason)=>({
  state:'UNKNOWN',reason,issueNumber:PRODUCT312_ISSUE,
  sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  taskPackBlobSha:SHA.test(input?.taskPackBlobSha??'')?
    input.taskPackBlobSha:null,
  writeSetSha256:PRODUCT312_DIGEST.test(input?.writeSetSha256??'')?
    input.writeSetSha256:null,
  frozen:false,authorized:false,admitted:false,
  taskPass:false,releasePass:false});
const product312BaseValid=(input,o)=>o&&typeof o==='object'&&
  !Array.isArray(o)&&o.issueNumber===312&&o.sourceSha===input.sourceSha&&
  o.taskPackBlobSha===input.taskPackBlobSha&&
  o.writeSetSha256===input.writeSetSha256&&o.authorized===false&&
  o.admitted===false&&o.taskPass===false&&o.releasePass===false;
const product312Envelope=(input,output,action)=>Object.freeze({
  phase:'terminal',
  externalRef:'tally:312:'+String(input?.sourceSha??'invalid')+':'+action,
  outcome:{ok:true,output:Object.freeze(output)}});
export async function persistProductDraft312(input){
  const valid=input&&input.issueNumber===312&&input.sourceRef===PRODUCT312_REF&&
    SHA.test(input.sourceSha??'')&&SHA.test(input.taskPackBlobSha??'')&&
    PRODUCT312_DIGEST.test(input.writeSetSha256??'')&&
    input.draft&&typeof input.draft==='object'&&!Array.isArray(input.draft);
  if(!valid)return product312Envelope(input,
    product312Unknown(input,'PRODUCT_DRAFT_INPUT_INVALID'),'draft');
  let observed;
  try{observed=await nativePersistProductDraft312(input);}catch{observed=null;}
  if(product312BaseValid(input,observed)&&
    observed.state==='DRAFT_PENDING_REVIEW'&&
    observed.reason==='HOST_DRAFT_PERSISTED_AND_READ_BACK'&&
    PRODUCT312_DIGEST.test(observed.productDigest??'')&&
    observed.draftRef==='draft:sha256:'+observed.productDigest)
    return product312Envelope(input,observed,'draft');
  return product312Envelope(input,
    product312Unknown(input,'PRODUCT_DRAFT_CAS_UNKNOWN'),'draft');
}
export async function freezeProductReview312(input){
  const valid=input&&input.issueNumber===312&&input.sourceRef===PRODUCT312_REF&&
    SHA.test(input.sourceSha??'')&&SHA.test(input.taskPackBlobSha??'')&&
    PRODUCT312_DIGEST.test(input.writeSetSha256??'')&&
    Number.isSafeInteger(input.reviewCommentId)&&input.reviewCommentId>0&&
    SHA.test(input.reviewedHead??'')&&SHA.test(input.reviewedTree??'')&&
    SHA.test(input.reviewedBase??'');
  if(!valid)return product312Envelope(input,
    product312Unknown(input,'PRODUCT_REVIEW_FREEZE_INPUT_INVALID'),'freeze');
  let observed;
  try{observed=await nativeFreezeProductReview312(input);}catch{observed=null;}
  if(product312BaseValid(input,observed)&&
    observed.state==='FROZEN_COMPLETE'&&observed.frozen===true&&
    observed.reason==='SOURCE_BOUND_HOST_CAS_READBACK_VERIFIED'&&
    PRODUCT312_DIGEST.test(observed.productDigest??'')&&
    observed.draftRef==='draft:sha256:'+observed.productDigest&&
    observed.receiptRef==='receipt:sha256:'+observed.productDigest&&
    PRODUCT312_REVIEW.test(observed.reviewRef??''))
    return product312Envelope(input,observed,'freeze');
  return product312Envelope(input,
    product312Unknown(input,'PRODUCT_REVIEW_FREEZE_CAS_UNKNOWN'),'freeze');
}

// #219 selected CAS operations return only narrow, non-authorizing
// observations. Caller-asserted Task/Release authority is never passed through.
const maintenanceSelectionEnvelope=(input,action,output)=>Object.freeze({
  phase:'terminal',
  externalRef:'tally:219:'+String(input?.candidateDigest??'invalid')+':'+action,
  outcome:{ok:true,output:Object.freeze(output)}});
const maintenanceSelectionDenied=(input,reason)=>({
  state:'NOT_ADMITTED',reason,
  issueRef:issueRefFor(219),sourceSha:SHA.test(input?.sourceSha??'')?
    input.sourceSha:null,
  candidateDigest:/^[a-f0-9]{64}$/.test(input?.candidateDigest??'')?
    input.candidateDigest:null,
  claimGeneration:1,authorized:false,admitted:false,
  taskPass:false,releasePass:false,githubEffect:'NOT_RUN'});
const maintenanceSelectionBase=(input,o)=>o&&typeof o==='object'&&
  !Array.isArray(o)&&o.issueRef===issueRefFor(219)&&
  o.sourceSha===input.sourceSha&&o.candidateDigest===input.candidateDigest&&
  o.claimGeneration===1&&o.authorized===false&&
  o.taskPass===false&&o.releasePass===false;
export async function persistMaintenanceDraft(input){
  const valid=input?.issueNumber===219&&input.sourceRef===issueRefFor(219)&&
    SHA.test(input.sourceSha??'')&&
    /^[a-f0-9]{64}$/.test(input.candidateDigest??'')&&
    input.candidate&&typeof input.candidate==='object'&&
    !Array.isArray(input.candidate);
  if(!valid)return maintenanceSelectionEnvelope(input,'draft',
    maintenanceSelectionDenied(input,'MAINTENANCE_DRAFT_INPUT_INVALID'));
  let observed;
  try{observed=await nativePersistMaintenanceDraft(input);}catch{observed=null;}
  if(maintenanceSelectionBase(input,observed)&&
     observed.state==='DRAFT_PENDING_REVIEW'&&
     ['MAINTENANCE_DRAFT_RECORDED','MAINTENANCE_DRAFT_ALREADY_RECORDED']
       .includes(observed.reason))
    return maintenanceSelectionEnvelope(input,'draft',observed);
  return maintenanceSelectionEnvelope(input,'draft',
    maintenanceSelectionDenied(input,observed?.reason??'MAINTENANCE_DRAFT_CAS_UNKNOWN'));
}
export async function freezeReviewedMaintenanceCandidate(input){
  const valid=input?.issueNumber===219&&input.sourceRef===issueRefFor(219)&&
    SHA.test(input.sourceSha??'')&&
    /^[a-f0-9]{64}$/.test(input.candidateDigest??'')&&
    Number.isSafeInteger(input.reviewCommentId)&&input.reviewCommentId>0&&
    Number.isSafeInteger(input.reviewedPullNumber)&&
    input.reviewedPullNumber>0&&SHA.test(input.reviewedHead??'')&&
    SHA.test(input.reviewedTree??'')&&input.reviewedBase===input.sourceSha;
  if(!valid)return maintenanceSelectionEnvelope(input,'freeze',
    maintenanceSelectionDenied(input,'MAINTENANCE_REVIEW_INPUT_INVALID'));
  let observed;
  try{observed=await nativeFreezeReviewedMaintenanceCandidate(input);}
  catch{observed=null;}
  if(maintenanceSelectionBase(input,observed)&&
     observed.state==='RECORDED_COMPLETE'&&observed.frozen===true&&
     observed.receiptRef==='receipt:sha256:'+input.candidateDigest)
    return maintenanceSelectionEnvelope(input,'freeze',observed);
  return maintenanceSelectionEnvelope(input,'freeze',
    maintenanceSelectionDenied(input,observed?.reason??'MAINTENANCE_REVIEW_FREEZE_UNKNOWN'));
}

// T219 — J08b authorized native Issue publication boundary. Only the exact
// bounded observation shapes cross; a lost acknowledgment stays UNKNOWN with
// a null Issue, a verified publication carries the canonical Issue identity,
// and no shape can ever smuggle a Task/Release PASS or a historical change.
const MAINTENANCE_ISSUE=219;
const CANDIDATE_DIGEST=/^[0-9a-f]{64}$/;
const ISSUE_URL_BOUNDARY=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/[1-9][0-9]*$/;
// Reachable Host UNKNOWN-effect reasons are not a closed enumeration: the
// shared version fence and the reconcile/readback paths surface fresh reason
// tokens (e.g. MAINTENANCE_ISSUE_PENDING_RECONCILIATION,
// MAINTENANCE_PUBLICATION_FENCE_UNKNOWN, MAINTENANCE_ISSUE_CREATE_READBACK_
// MISSING, GITHUB_PUBLICATION_UNKNOWN). A lost acknowledgment must stay
// PUBLICATION_UNKNOWN with githubEffect UNKNOWN — never be rewritten into a
// pre-effect refusal. The boundary contract is the state+effect shape; the
// reason only has to be a bounded machine token.
const MAINTENANCE_TOKEN=/^[A-Z][A-Z0-9_]{0,127}$/;
const validMaintenanceIssue=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==issueRefFor(MAINTENANCE_ISSUE)||
    o.sourceSha!==input.sourceSha||o.candidateDigest!==input.candidateDigest||
    o.claimGeneration!==1||o.authorized!==false||o.agentLaunch!==false||
    o.historicalReleaseMutation!==false||o.taskPass!==false||
    o.releasePass!==false)return false;
  if(o.state==='NOT_ADMITTED')return MAINTENANCE_TOKEN.test(o.reason??'')&&
    o.claimAccepted===false&&o.admitted===false&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===false&&o.issue===null;
  if(o.state==='PUBLICATION_UNKNOWN')return MAINTENANCE_TOKEN.test(o.reason??'')&&
    o.claimAccepted===true&&o.admitted===true&&o.githubEffect==='UNKNOWN'&&
    o.replayFixture===false&&o.issue===null;
  if(o.state==='REPLAY_COMPLETE')return o.reason==='REPLAY_FIXTURE_ONLY'&&
    o.claimAccepted===true&&o.admitted===true&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===true&&o.issue===null;
  if(o.state==='MAINTENANCE_ISSUE_PUBLISHED')
    return ['MAINTENANCE_ISSUE_PUBLISHED_AND_READBACK_VERIFIED',
      'MAINTENANCE_ISSUE_READBACK_ALREADY_PRESENT',
      'MAINTENANCE_ISSUE_RECONCILED_AFTER_LOST_ACK'].includes(o.reason)&&
    o.claimAccepted===true&&o.admitted===true&&
    o.githubEffect==='PERFORMED_VERIFIED'&&o.replayFixture===false&&
    o.repository==='kaicreator-mm/tally'&&o.duplicate===false&&
    o.issue&&typeof o.issue==='object'&&!Array.isArray(o.issue)&&
    Number.isSafeInteger(o.issue.number)&&o.issue.number>0&&
    typeof o.issue.title==='string'&&o.issue.title.length>0&&
    ISSUE_URL_BOUNDARY.test(o.issue.url??'');
  return false;
};
export async function publishMaintenanceIssue(input){
  let output;
  if(!input||typeof input==='object'&&Array.isArray(input)||
    input.issueNumber!==MAINTENANCE_ISSUE||
    input.sourceRef!==issueRefFor(MAINTENANCE_ISSUE)||
    !SHA.test(input.sourceSha??'')||!CANDIDATE_DIGEST.test(input.candidateDigest??''))
    output={state:'NOT_ADMITTED',reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
      issueRef:issueRefFor(MAINTENANCE_ISSUE),
      sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
      candidateDigest:CANDIDATE_DIGEST.test(input?.candidateDigest??'')?
        input.candidateDigest:null,claimGeneration:1,claimAccepted:false,
      admitted:false,authorized:false,agentLaunch:false,replayFixture:false,
      githubEffect:'NOT_RUN',issue:null,historicalReleaseMutation:false,
      taskPass:false,releasePass:false};
  else{
    let observed;
    try{observed=await nativePublishMaintenanceIssue(input);}catch{observed=null;}
    if(validMaintenanceIssue(input,observed))output=observed;
    else output={state:'NOT_ADMITTED',reason:'LOCAL_AGENT_PROVIDER_THREW',
      issueRef:issueRefFor(MAINTENANCE_ISSUE),sourceSha:input.sourceSha,
      candidateDigest:input.candidateDigest,claimGeneration:1,claimAccepted:false,
      admitted:false,authorized:false,agentLaunch:false,replayFixture:false,
      githubEffect:'NOT_RUN',issue:null,historicalReleaseMutation:false,
      taskPass:false,releasePass:false};
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${MAINTENANCE_ISSUE}:${input?.candidateDigest??'invalid'}:maintenance`,
    outcome:{ok:true,output}});
}

// T219 — J08c Task Learning boundary. Only the exact LEARNING_RECORDED /
// NOT_ADMITTED / REPLAY_COMPLETE shapes cross; noAgentRanking and
// qualityGainClaimed=false are re-verified here so a ranking or a quality-gain
// claim can never cross the sealed boundary.
const LEARNING_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'MAINTENANCE_HOST_CONFIG_INVALID','MAINTENANCE_FREEZE_STORE_UNAVAILABLE',
  'LEARNING_PROVENANCE_NOT_VERIFIED','LEARNING_DECISION_REQUIRED',
  'LEARNING_DECISION_MISMATCH','LEARNING_OUTCOME_MISMATCH',
  'MAINTENANCE_INPUT_INVALID','LOCAL_AGENT_PROVIDER_THREW']);
const validLearningOutcome=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==issueRefFor(MAINTENANCE_ISSUE)||
    o.sourceSha!==input.sourceSha||o.learningId!==input.learningId||
    o.claimGeneration!==1||o.authorized!==false||o.agentLaunch!==false||
    o.taskPass!==false||o.releasePass!==false)return false;
  if(o.state==='NOT_ADMITTED')return LEARNING_REASONS.has(o.reason)&&
    o.claimAccepted===false&&o.admitted===false&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===false;
  if(o.state==='REPLAY_COMPLETE')return o.reason==='REPLAY_FIXTURE_ONLY'&&
    o.claimAccepted===true&&o.admitted===true&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===true&&o.measuredEffect==='NOT_MEASURED';
  if(o.state==='LEARNING_RECORDED')
    return ['LEARNING_DECISION_RECORDED_IN_HOST_CAS',
      'LEARNING_DECISION_ALREADY_RECORDED','LEARNING_OUTCOME_RECORDED_IN_HOST_CAS',
      'LEARNING_OUTCOME_ALREADY_RECORDED'].includes(o.reason)&&
    o.claimAccepted===true&&o.admitted===true&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===false&&['ADOPTED','REJECTED'].includes(o.decision)&&
    o.noAgentRanking===true&&o.qualityGainClaimed===false&&
    (o.measuredEffect==='NOT_MEASURED'||o.measuredEffect==='ATTESTED_OUTCOME_ONLY');
  return false;
};
export async function recordTaskLearningOutcome(input){
  let output;
  const malformed=(!input||typeof input==='object'&&Array.isArray(input)||
    input.issueNumber!==MAINTENANCE_ISSUE||
    input.sourceRef!==issueRefFor(MAINTENANCE_ISSUE)||
    !SHA.test(input.sourceSha??'')||!CANDIDATE_DIGEST.test(input.learningId??'')||
    !['ADOPTED','REJECTED'].includes(input.decision));
  const hasOutcome=input?.outcomeAttested!==null&&input?.outcomeAttested!==undefined;
  const outcomeShape=hasOutcome?
    (typeof input.outcomeObservedAt!=='string'||!input.outcomeObservedAt||
      Number.isNaN(Date.parse(input.outcomeObservedAt))||
      typeof input.outcomeNote!=='string'||input.outcomeNote.length===0||
      input.outcomeNote.length>512):
    (input?.outcomeObservedAt!==null&&input?.outcomeObservedAt!==undefined||
      input?.outcomeNote!==null&&input?.outcomeNote!==undefined);
  if(malformed||outcomeShape)
    output={state:'NOT_ADMITTED',reason:'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
      issueRef:issueRefFor(MAINTENANCE_ISSUE),
      sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
      learningId:CANDIDATE_DIGEST.test(input?.learningId??'')?input.learningId:null,
      claimGeneration:1,claimAccepted:false,admitted:false,authorized:false,
      agentLaunch:false,replayFixture:false,githubEffect:'NOT_RUN',
      taskPass:false,releasePass:false};
  else{
    const nativeInput={sourceRef:input.sourceRef,issueNumber:input.issueNumber,
      sourceSha:input.sourceSha,learningId:input.learningId,decision:input.decision,
      ...(hasOutcome?{outcome:{attested:input.outcomeAttested===true,
        observedAt:input.outcomeObservedAt,note:input.outcomeNote}}:{})};
    let observed;
    try{observed=await nativeRecordTaskLearningOutcome(nativeInput);}catch{observed=null;}
    if(validLearningOutcome(nativeInput,observed))output=observed;
    else output={state:'NOT_ADMITTED',reason:'LOCAL_AGENT_PROVIDER_THREW',
      issueRef:issueRefFor(MAINTENANCE_ISSUE),sourceSha:input.sourceSha,
      learningId:input.learningId,claimGeneration:1,claimAccepted:false,
      admitted:false,authorized:false,agentLaunch:false,replayFixture:false,
      githubEffect:'NOT_RUN',taskPass:false,releasePass:false};
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${MAINTENANCE_ISSUE}:${input?.learningId??'invalid'}:learning`,
    outcome:{ok:true,output}});
}

// J222 J07b — sealed qualified Release delivery boundary. Only the exact
// bounded observation shapes cross; identity (issueRef/sourceSha/operation)
// is re-attached and re-verified here so provider garbage, a caller-supplied
// PASS flag or a second writer shape can never become release authority. A
// merged readback stays below Release PASS; a lost acknowledgment stays
// GITHUB_EFFECT_UNKNOWN; a pending Host CAS op stays RECONCILIATION_REQUIRED.
const RELEASE_ISSUE=222;
const RELEASE_OPERATIONS=new Set(['PREFLIGHT','MERGE','TAG']);
const RELEASE_TAG_NAME=/^v[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:-[a-z0-9.-]+)?$/;
const RELEASE_STATES=new Set(['NOT_ADMITTED','GITHUB_EFFECT_UNKNOWN',
  'RECONCILIATION_REQUIRED','HOST_GATES_OBSERVED','EXACT_PREFLIGHT_OBSERVED',
  'MERGED_REQUALIFY_REQUIRED']);
const RELEASE_DENY_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'CLAIM_REGISTRY_UNAVAILABLE','CLAIM_REGISTRY_RECORD_NOT_FOUND',
  'CLAIM_REGISTRY_RECORD_INVALID','CLAIM_REGISTRY_PUBLICATION_NOT_VERIFIED',
  'CLAIM_REGISTRY_RESERVATION_MISMATCH','DAG_WRITESET_NOT_VERIFIED',
  'LOCAL_AGENT_UNAVAILABLE','NATIVE_CLAIM_WITNESS_NOT_VERIFIED',
  'HOST_QUALIFICATION_READ_UNKNOWN','RELEASE_HOST_PORTS_NOT_CONNECTED',
  'RELEASE_SOURCE_NOT_PINNED','NATIVE_CLAIM_NOT_VERIFIED','TASK_WRITESET_NOT_VERIFIED',
  'INDEPENDENT_RQ_NOT_QUALIFIED','OWNER_EXACT_RELEASE_SCOPE_MISSING',
  'RELEASE_PHYSICAL_EFFECT_DISABLED','RELEASE_HOST_CAS_UNAVAILABLE',
  'RELEASE_HOST_CAS_UNAVAILABLE_OR_CONFLICT','TAG_SOURCE_NOT_PINNED',
  'POST_MERGE_RQ_READ_UNKNOWN','POST_MERGE_RQ_OR_GRANT_NOT_VERIFIED',
  'TAG_EFFECT_REQUIRES_HOST_CAS_AND_RELEASE_PUBLICATION_BINDING',
  'RELEASE_CANDIDATE_INPUT_INVALID','EXACT_HEAD_TREE_OR_TARGET_MOVED',
  'MERGEABILITY_NOT_PROVEN','GITHUB_MERGE_TRANSPORT_UNAVAILABLE',
  'TAG_READBACK_INPUT_INVALID','RELEASE_TAG_INPUT_INVALID',
  'TAG_CONFLICT_OR_ANNOTATED_TAG_REQUIRES_REVIEW',
  'POST_MERGE_TARGET_REQUALIFICATION_REQUIRED']);
const RELEASE_UNKNOWN_REASONS=new Set(['GITHUB_UNAUTHENTICATED','GITHUB_ACCESS_DENIED',
  'GITHUB_RESOURCE_NOT_FOUND','GITHUB_CONFLICT','GITHUB_RATE_LIMITED',
  'GITHUB_TRANSPORT_UNKNOWN','GITHUB_MERGE_RECEIPT_INCOMPLETE',
  'MERGE_READBACK_OR_PARENT_MISMATCH','TAG_CREATE_READBACK_UNKNOWN']);
const releaseBase=(input,reason,state)=>({state,reason,
  issueRef:issueRefFor(RELEASE_ISSUE),
  sourceSha:SHA.test(input?.sourceSha??'')?input.sourceSha:null,
  operation:RELEASE_OPERATIONS.has(input?.operation)?input.operation:null,
  claimGeneration:1,claimAccepted:false,admitted:false,authorized:false,
  agentLaunch:false,replayFixture:false,githubEffect:'NOT_RUN',
  releasePublished:false,taskPass:false,releasePass:false});
const releasePin=(input,claimAccepted)=>{
  const mergeShape=Number.isSafeInteger(input?.pullNumber)&&input.pullNumber>0&&
    SHA.test(input?.headSha??'')&&SHA.test(input?.headTree??'')&&
    SHA.test(input?.targetSha??'')&&SHA.test(input?.targetTree??'')&&
    input.tagName==null&&input.commitSha==null&&input.commitTree==null;
  const tagShape=RELEASE_TAG_NAME.test(input?.tagName??'')&&
    SHA.test(input?.commitSha??'')&&SHA.test(input?.commitTree??'')&&
    input.pullNumber==null&&input.headSha==null&&input.targetSha==null;
  return input?.sourceRef===issueRefFor(RELEASE_ISSUE)&&
    input.issueNumber===RELEASE_ISSUE&&SHA.test(input?.sourceSha??'')&&
    RELEASE_OPERATIONS.has(input.operation)&&
    (input.operation==='TAG'?tagShape:mergeShape)&&claimAccepted;
};
const validReleaseOutput=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||
    o.issueRef!==issueRefFor(RELEASE_ISSUE)||o.sourceSha!==input.sourceSha||
    o.operation!==input.operation||o.claimGeneration!==1||
    o.authorized!==false||o.taskPass!==false||o.releasePass!==false||
    o.releasePublished!==false||!RELEASE_STATES.has(o.state))return false;
  if(o.repository!==undefined&&o.repository!=='kaicreator-mm/tally')return false;
  if(o.state==='NOT_ADMITTED')
    return RELEASE_DENY_REASONS.has(o.reason)&&o.admitted===false&&
      (o.githubEffect??'NOT_RUN')==='NOT_RUN'&&o.mergedSha===undefined;
  if(o.state==='GITHUB_EFFECT_UNKNOWN')
    return RELEASE_UNKNOWN_REASONS.has(o.reason)&&o.admitted===true&&
      o.claimAccepted===true&&o.githubEffect==='UNKNOWN'&&
      (o.mergedSha===undefined||o.mergedSha===null);
  if(o.state==='RECONCILIATION_REQUIRED')
    return o.reason==='PREVIOUS_RELEASE_OPERATION_PENDING'&&o.admitted===true&&
      o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
      typeof o.operationKey==='string'&&o.operationKey.length>0;
  if(o.state==='HOST_GATES_OBSERVED')
    return o.reason==='REQUIRES_CURRENT_TARGET_PREFLIGHT'&&o.admitted===true&&
      o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
      SHA.test(o.headSha??'')&&SHA.test(o.targetSha??'');
  if(o.state==='EXACT_PREFLIGHT_OBSERVED')
    return o.reason==='REQUIRES_INDEPENDENT_HOST_RQ_AND_OWNER_GRANT'&&
      o.admitted===true&&o.claimAccepted===true&&o.githubEffect==='NOT_RUN'&&
      o.hostGateRequired===true&&o.repository==='kaicreator-mm/tally'&&
      Number.isSafeInteger(o.pullNumber)&&o.pullNumber===input.pullNumber&&
      o.headSha===input.headSha&&o.headTree===input.headTree&&
      o.targetSha===input.targetSha&&o.targetTree===input.targetTree;
  // MERGED_REQUALIFY_REQUIRED: one verified merge readback, never Release PASS.
  return o.reason==='FINAL_MERGE_TREE_REQUIRES_NEW_INDEPENDENT_RQ'&&
    o.admitted===true&&o.claimAccepted===true&&
    o.githubEffect==='PERFORMED_READBACK_VERIFIED'&&
    SHA.test(o.mergedSha??'')&&SHA.test(o.mergedTree??'')&&
    o.canTag===false&&o.canRelease===false&&
    typeof o.operationKey==='string'&&
    o.next==='REQUALIFY_NEW_MAIN_SHA_AND_TREE_BEFORE_TAG';
};
export async function hostReleaseDelivery(input){
  let output;
  if(!releasePin(input,true)){
    output=releaseBase(input,'CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH','NOT_ADMITTED');
  }else{
    let observed;
    try{observed=await nativeHostReleaseDelivery(input);}catch{observed=null;}
    if(validReleaseOutput(input,observed))output=observed;
    else output=releaseBase(input,'LOCAL_AGENT_PROVIDER_THREW','NOT_ADMITTED');
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${RELEASE_ISSUE}:${input?.operation??'invalid'}:${input?.headSha??input?.commitSha??'invalid'}:release`,
    outcome:{ok:true,output}});
}

// E0 #284 — sealed pre-freeze version Task DAG write boundary. Only the exact
// bounded selected-C observation shapes cross: the official GitHub blocked_by
// readback for ONE proposed PLANNED candidate edge, and the A-owned closeout
// freeze record after all required edges are NATIVE. Identity (#284 + the
// immutable PackBlob ref/SHA) is re-verified here so provider garbage, a
// caller-supplied PASS flag or a post-freeze dependency write can never cross.
const E0_ISSUE=284;
const E0_PACK_REF=/^https:\/\/github\.com\/kaicreator-mm\/tally\/blob\/[a-f0-9]{40}\/[^\s]+$/;
const E0_DEPENDENCY_REASONS=new Set(['EDGE_INPUT_INVALID',
  'VERSION_DAG_ALREADY_FROZEN','TASK_PACK_IDENTITY_INVALID',
  'TASK_PACK_READBACK_FAILED','TASK_PACK_CONTENT_INVALID','EDGE_ISSUE_NOT_IN_PACK',
  'EDGE_NOT_IN_CANDIDATE_PACK','EDGE_ALREADY_NATIVE_IN_PACK','EDGE_CYCLE',
  'ISSUE_READBACK_FAILED','ISSUE_STALE_VS_PACK','GITHUB_UNAUTHENTICATED',
  'GITHUB_ACCESS_DENIED','GITHUB_RESOURCE_NOT_FOUND','GITHUB_VALIDATION_FAILED',
  'GITHUB_DEPENDENCY_READ_FAILED','NATIVE_WRITE_ACK_LOST','NATIVE_READBACK_MISMATCH',
  'NATIVE_DEPENDENCY_WRITE_FAILED']);
const E0_FREEZE_REASONS=new Set([...E0_DEPENDENCY_REASONS,'PLANNED_EDGES_REMAIN',
  'FREEZE_WRITESET_CONFIG_INVALID','FREEZE_WRITESET_CONFLICT',
  'NATIVE_TASK_ROSTER_READBACK_FAILED','NATIVE_TASK_ROSTER_CHANGED_BEFORE_FREEZE',
  'INDEPENDENT_REVIEW_REQUIRED','OWNER_APPROVAL_REQUIRED','FREEZE_TIMESTAMP_INVALID',
  'DIVERGENT_FREEZE_EXISTS','FREEZE_WRITESET_PARTIAL_UNKNOWN',
  'VERSION_DAG_FREEZE_FAILED','FREEZE_READBACK_MISMATCH']);
const dependencyBase=(input,reason)=>({status:'BLOCKED',reason,
  repository:OWNER+'/'+REPO,versionId:'version/v2',githubEffect:'NOT_RUN',
  native:false,authorized:false,admitted:false,taskPass:false,releasePass:false});
const e0Identity=input=>!!input&&typeof input==='object'&&!Array.isArray(input)&&
  input.issueNumber===E0_ISSUE&&input.sourceRef===issueRefFor(E0_ISSUE)&&
  SHA.test(input.sourceSha??'')&&E0_PACK_REF.test(input.taskPackRef??'')&&
  SHA.test(input.taskPackBlobSha??'');
const validDependencyOutput=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||
     o.repository!==OWNER+'/'+REPO||o.versionId!=='version/v2'||
     o.githubEffect===undefined||o.authorized!==false||o.admitted!==false||
     o.taskPass!==false||o.releasePass!==false)return false;
  if(o.status==='BLOCKED')return o.githubEffect==='NOT_RUN'&&o.native===false&&
    E0_DEPENDENCY_REASONS.has(o.reason);
  if(o.status==='UNKNOWN')return o.githubEffect==='UNKNOWN'&&
    o.reconciliationRequired===true&&
    o.retryPolicy==='READ_BACK_BEFORE_RETRY_NO_BLIND_POST'&&
    E0_DEPENDENCY_REASONS.has(o.reason)&&
    o.blockedIssueNumber===input.blockedIssueNumber&&
    o.blockedByIssueNumber===input.blockedByIssueNumber;
  if(o.status==='NATIVE_EDGE_PRESENT')return o.reason===
      'NATIVE_EDGE_ALREADY_PRESENT_READBACK'&&o.native===true&&
    o.githubEffect==='NOT_RUN'&&o.writePerformed===false&&
    Number.isSafeInteger(o.dependencyId)&&o.dependencyId>0&&
    o.blockedIssueNumber===input.blockedIssueNumber&&
    o.blockedByIssueNumber===input.blockedByIssueNumber;
  if(o.status==='NATIVE_EDGE_VERIFIED')return o.reason===
      'OFFICIAL_GITHUB_BLOCKED_BY_READBACK_VERIFIED'&&o.native===true&&
    o.githubEffect==='PERFORMED_VERIFIED'&&o.writePerformed===true&&
    Number.isSafeInteger(o.dependencyId)&&o.dependencyId>0&&
    o.blockedIssueNumber===input.blockedIssueNumber&&
    o.blockedByIssueNumber===input.blockedByIssueNumber;
  return false;
};
export async function addNativeTaskDependency(input){
  let output;
  if(!e0Identity(input)||!Number.isSafeInteger(input.blockedIssueNumber)||
     input.blockedIssueNumber<1||!Number.isSafeInteger(input.blockedByIssueNumber)||
     input.blockedByIssueNumber<1||input.blockedIssueNumber===input.blockedByIssueNumber)
    output=dependencyBase(input,'EDGE_INPUT_INVALID');
  else{
    let observed;
    try{observed=await nativeAddNativeTaskDependency(input);}catch{observed=null;}
    output=validDependencyOutput(input,observed)?observed:
      dependencyBase(input,'LOCAL_AGENT_PROVIDER_THREW');
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${E0_ISSUE}:${input?.taskPackBlobSha??'invalid'}:`+
      `${input?.blockedByIssueNumber??'x'}-blocks-${input?.blockedIssueNumber??'x'}:dependency`,
    outcome:{ok:true,output}});
}
const dagFreezeBase=(input,reason)=>({status:'BLOCKED',reason,
  repository:OWNER+'/'+REPO,versionId:'version/v2',githubEffect:'NOT_RUN',
  authorized:false,admitted:false,taskPass:false,releasePass:false});
const validDagFreezeOutput=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||
     o.repository!==OWNER+'/'+REPO||o.versionId!=='version/v2'||
     o.taskPackBlobSha!==input.taskPackBlobSha||o.githubEffect!=='NOT_RUN'||
     o.authorized!==false||o.admitted!==false||o.taskPass!==false||
     o.releasePass!==false)return false;
  if(o.status==='BLOCKED')return E0_FREEZE_REASONS.has(o.reason);
  if(o.status==='UNKNOWN')return o.reconciliationRequired===true&&
    E0_FREEZE_REASONS.has(o.reason);
  if(o.status==='FROZEN'||o.status==='REPLAY')
    return (o.status==='FROZEN'?o.reason==='FROZEN_VERSION_TASK_DAG_CAS_READBACK_VERIFIED':
        o.reason==='IDENTICAL_FREEZE_REPLAY')&&
      typeof o.frozenAt==='string'&&Number.isFinite(Date.parse(o.frozenAt))&&
      Number.isSafeInteger(o.taskCount)&&o.taskCount>0&&
      Number.isSafeInteger(o.edgeCount)&&o.edgeCount>=0&&
      o.writeSetRecordCount===o.taskCount&&
      o.dependencyWritesAfterFreeze==='REFUSED';
  return false;
};
export async function freezeVersionTaskDag(input){
  let output;
  if(!e0Identity(input))output=dagFreezeBase(input,'TASK_PACK_IDENTITY_INVALID');
  else{
    let observed;
    try{observed=await nativeFreezeVersionTaskDag(input);}catch{observed=null;}
    output=validDagFreezeOutput(input,observed)?observed:
      dagFreezeBase(input,'LOCAL_AGENT_PROVIDER_THREW');
  }
  return Object.freeze({phase:'terminal',
    externalRef:`tally:${E0_ISSUE}:${input?.taskPackBlobSha??'invalid'}:dagfreeze`,
    outcome:{ok:true,output}});
}
