/** T06m/T06n/T210/T219 sealed Local Codex C boundary. Provider output is never Task evidence. */
import {runLocalCodexAttempt as nativeRunLocalCodexAttempt,
  joinReviewedCandidates as nativeJoinReviewedCandidates,
  publishReviewedCandidate as nativePublishReviewedCandidate,
  hostProjectIntake as nativeHostProjectIntake,
  freezeProductIntake as nativeFreezeProductIntake,
  publishMaintenanceIssue as nativePublishMaintenanceIssue,
  recordTaskLearningOutcome as nativeRecordTaskLearningOutcome}
  from '@kaicreator/tally-local-codex-host-bundle';

export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-local-codex@v1.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-local-codex';
export const OPERATIONS=Object.freeze(['tally.runLocalCodexAttempt','tally.joinReviewedCandidates','tally.publishReviewedCandidate','tally.freezeProductIntake','tally.hostProjectIntake','tally.publishMaintenanceIssue','tally.recordTaskLearningOutcome']);
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

// T219 — J08b authorized native Issue publication boundary. Only the exact
// bounded observation shapes cross; a lost acknowledgment stays UNKNOWN with
// a null Issue, a verified publication carries the canonical Issue identity,
// and no shape can ever smuggle a Task/Release PASS or a historical change.
const MAINTENANCE_ISSUE=219;
const CANDIDATE_DIGEST=/^[0-9a-f]{64}$/;
const ISSUE_URL_BOUNDARY=/^https:\/\/github\.com\/kaicreator-mm\/tally\/issues\/[1-9][0-9]*$/;
const MAINTENANCE_REASONS=new Set(['CLAIM_HOST_PIN_OR_IDENTITY_MISMATCH',
  'MAINTENANCE_HOST_CONFIG_INVALID','DAG_WRITESET_NOT_VERIFIED',
  'CANDIDATE_REVIEW_NOT_VERIFIED','MAINTENANCE_REVIEW_DIGEST_MISMATCH',
  'CANDIDATE_REVIEW_NOT_INDEPENDENT','MAINTENANCE_FREEZE_STORE_UNAVAILABLE',
  'MAINTENANCE_ISSUE_DIGEST_MISMATCH','GITHUB_PUBLISH_TRANSPORT_UNAVAILABLE',
  'GITHUB_PUBLISH_UNKNOWN','MAINTENANCE_ISSUE_READBACK_MISMATCH',
  'MAINTENANCE_ISSUE_IDENTITY_CONFLICT','MAINTENANCE_ISSUE_INPUT_INVALID',
  'GITHUB_ACCESS_DENIED','GITHUB_RATE_LIMITED','GITHUB_UNAUTHENTICATED',
  'GITHUB_RESOURCE_NOT_FOUND','GITHUB_PUBLICATION_CONFLICT','LOCAL_AGENT_PROVIDER_THREW']);
const validMaintenanceIssue=(input,o)=>{
  if(!o||typeof o!=='object'||Array.isArray(o)||o.issueRef!==issueRefFor(MAINTENANCE_ISSUE)||
    o.sourceSha!==input.sourceSha||o.candidateDigest!==input.candidateDigest||
    o.claimGeneration!==1||o.authorized!==false||o.agentLaunch!==false||
    o.historicalReleaseMutation!==false||o.taskPass!==false||
    o.releasePass!==false)return false;
  if(o.state==='NOT_ADMITTED')return MAINTENANCE_REASONS.has(o.reason)&&
    o.claimAccepted===false&&o.admitted===false&&o.githubEffect==='NOT_RUN'&&
    o.replayFixture===false&&o.issue===null;
  if(o.state==='PUBLICATION_UNKNOWN')return MAINTENANCE_REASONS.has(o.reason)&&
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
