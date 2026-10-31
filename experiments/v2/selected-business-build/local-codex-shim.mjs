/** T06m/T06n sealed Local Codex C boundary. Provider output is never Task evidence. */
import {runLocalCodexAttempt as nativeRunLocalCodexAttempt,
  joinReviewedCandidates as nativeJoinReviewedCandidates}
  from '@kaicreator/tally-local-codex-host-bundle';

export const NATIVE_PROVIDER_SHA256='__PROVIDER_SHA256__';
export const MODULE_ID='capability-tally-local-codex@v1.0.0';
export const PACKAGE_KIND='capability';
export const ABI_VERSION='v08-capability-abi/1';
export const CAPABILITY_ID='tally-local-codex';
export const OPERATIONS=Object.freeze(['tally.runLocalCodexAttempt','tally.joinReviewedCandidates']);
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
