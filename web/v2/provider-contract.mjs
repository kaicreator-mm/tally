/** R105 unified Provider contract (v2.1 candidate; #64; NOT a v2 MVP gate).
 * One Task attempt uses exactly one Provider. This module is the single
 * source of truth for the minimal shared interface every optional provider
 * adapter exposes (providerId / probe / execute / cancel / readback), the
 * unified attempt input/output shapes, and the admission-time selection
 * policy. It defines validators only — no spawn, no store, no scheduler.
 *
 * Evidence semantics: an attempt outcome is EXECUTION EVIDENCE ONLY.
 * taskPass and releasePass are frozen false; no provider outcome may grant
 * Task PASS, Review PASS or Release PASS. There is deliberately NO
 * failure-driven fallback: the default provider is codex (the existing v2
 * path, unchanged), every other provider is DISABLED until explicitly
 * requested AND enabled in trusted config AND admitted by a locally
 * validated capability probe. An explicitly requested but unavailable
 * provider is REJECTED (fail-closed), never silently switched. */
const plain=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const ISO=/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/;

export const PROVIDER_CONTRACT_IDS=Object.freeze({
  defaultProviderId:'codex',
  providerIdPattern:/^[a-z][a-z0-9-]*$/,
  attemptIdPattern:/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/,
  taskIdPattern:/^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/,
  shaPattern:/^[0-9a-f]{40}$/,
  states:Object.freeze(['LOCAL_CANDIDATE_READY','LOCAL_ATTEMPT_FAILED',
    'LOCAL_ATTEMPT_UNKNOWN','LOCAL_ATTEMPT_CANCELLED','AGENT_NOT_EXECUTED','NOT_ADMITTED']),
  processStatuses:Object.freeze(['RUNNING','EXITED','UNKNOWN']),
  probeStages:Object.freeze(['VERSION','CREDENTIALS','CANARY']),
  interfaceMembers:Object.freeze(['providerId','probe','execute','cancel','readback'])
});
const IDS=PROVIDER_CONTRACT_IDS;

const denied=(reason,detail={})=>Object.freeze({valid:false,reason,...detail});
const allowedPath=path=>typeof path==='string'&&path.length>0&&!path.includes('\\')&&
  !path.startsWith('/')&&!path.split('/').includes('..')&&!path.split('/').includes('');

/** Unified attempt input validation. One attempt, one provider, anchored to
 * an exact source SHA inside a disposable worktree with an explicit write
 * set and a Host-authorized claim context. */
export function validateProviderAttemptInput(input){
  if(!plain(input))return denied('ATTEMPT_INPUT_INVALID',{detail:'NOT_AN_OBJECT'});
  if(!IDS.providerIdPattern.test(input.providerId??''))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'PROVIDER_ID_INVALID'});
  if(typeof input.taskId!=='string'||!IDS.taskIdPattern.test(input.taskId))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'TASK_ID_INVALID'});
  if(typeof input.attemptId!=='string'||!IDS.attemptIdPattern.test(input.attemptId))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'ATTEMPT_ID_INVALID'});
  if(typeof input.sourceSha!=='string'||!IDS.shaPattern.test(input.sourceSha))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'SOURCE_SHA_INVALID'});
  if(typeof input.worktreePath!=='string'||!input.worktreePath)
    return denied('ATTEMPT_INPUT_INVALID',{detail:'WORKTREE_PATH_INVALID'});
  if(!Array.isArray(input.allowedWriteSet)||input.allowedWriteSet.length===0||
    !input.allowedWriteSet.every(allowedPath))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'WRITE_SET_INVALID'});
  if(typeof input.taskPrompt!=='string'||!input.taskPrompt.trim())
    return denied('ATTEMPT_INPUT_INVALID',{detail:'TASK_PROMPT_INVALID'});
  if(input.issueRef!==undefined&&typeof input.issueRef!=='string')
    return denied('ATTEMPT_INPUT_INVALID',{detail:'ISSUE_REF_INVALID'});
  if(input.hostAuthorization!==undefined&&!plain(input.hostAuthorization))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'HOST_AUTHORIZATION_INVALID'});
  if(input.providerConfig!==undefined&&!plain(input.providerConfig))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'PROVIDER_CONFIG_INVALID'});
  if(input.signal!==undefined&&
    !(typeof input.signal?.addEventListener==='function'&&typeof input.signal.aborted==='boolean'))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'SIGNAL_INVALID'});
  if(input.timeoutMs!==undefined&&(!Number.isSafeInteger(input.timeoutMs)||input.timeoutMs<=0))
    return denied('ATTEMPT_INPUT_INVALID',{detail:'TIMEOUT_INVALID'});
  const value={providerId:input.providerId,taskId:input.taskId,attemptId:input.attemptId,
    sourceSha:input.sourceSha,worktreePath:input.worktreePath,
    allowedWriteSet:[...input.allowedWriteSet],taskPrompt:input.taskPrompt,
    issueRef:input.issueRef??null,hostAuthorization:input.hostAuthorization??{},
    providerConfig:input.providerConfig??{},signal:input.signal??undefined,
    timeoutMs:input.timeoutMs??undefined};
  return Object.freeze({valid:true,reason:null,value:Object.freeze(value)});
}

/** Unified attempt outcome validation. The evidence-only guard rejects any
 * outcome that tries to grant Task/Release PASS. */
export function validateProviderAttemptOutcome(outcome){
  if(!plain(outcome))return denied('OUTCOME_INVALID',{detail:'NOT_AN_OBJECT'});
  if(!IDS.providerIdPattern.test(outcome.providerId??''))
    return denied('OUTCOME_INVALID',{detail:'PROVIDER_ID_INVALID'});
  if(typeof outcome.attemptId!=='string'||!IDS.attemptIdPattern.test(outcome.attemptId))
    return denied('OUTCOME_INVALID',{detail:'ATTEMPT_ID_INVALID'});
  if(typeof outcome.sourceSha!=='string'||!IDS.shaPattern.test(outcome.sourceSha))
    return denied('OUTCOME_INVALID',{detail:'SOURCE_SHA_INVALID'});
  if(!IDS.states.includes(outcome.state))
    return denied('OUTCOME_INVALID',{detail:'STATE_INVALID',state:outcome.state??null});
  if(typeof outcome.reason!=='string'||!outcome.reason)
    return denied('OUTCOME_INVALID',{detail:'REASON_INVALID'});
  if(typeof outcome.agentLaunch!=='boolean')
    return denied('OUTCOME_INVALID',{detail:'AGENT_LAUNCH_INVALID'});
  const process=outcome.process??null;
  if(!plain(process))
    return denied('OUTCOME_INVALID',{detail:'PROCESS_INVALID'});
  if(!IDS.processStatuses.includes(process.status))
    return denied('OUTCOME_INVALID',{detail:'PROCESS_STATUS_INVALID'});
  for(const field of ['exitCode','pid'])
    if(process[field]!==null&&!Number.isSafeInteger(process[field]))
      return denied('OUTCOME_INVALID',{detail:'PROCESS_'+field.toUpperCase()+'_INVALID'});
  for(const field of ['startedAt','terminatedAt'])
    if(process[field]!==null&&!ISO.test(process[field]))
      return denied('OUTCOME_INVALID',{detail:'PROCESS_'+field.toUpperCase()+'_INVALID'});
  for(const field of ['candidateTree','candidateCommit'])
    if(outcome[field]!==undefined&&outcome[field]!==null&&!IDS.shaPattern.test(outcome[field]))
      return denied('OUTCOME_INVALID',{detail:field.toUpperCase()+'_INVALID'});
  if(outcome.candidateRef!==undefined&&outcome.candidateRef!==null&&
    (typeof outcome.candidateRef!=='string'||!outcome.candidateRef.startsWith('refs/')))
    return denied('OUTCOME_INVALID',{detail:'CANDIDATE_REF_INVALID'});
  if(outcome.candidatePaths!==undefined&&outcome.candidatePaths!==null&&
    (!Array.isArray(outcome.candidatePaths)||!outcome.candidatePaths.every(allowedPath)))
    return denied('OUTCOME_INVALID',{detail:'CANDIDATE_PATHS_INVALID'});
  if(outcome.taskPass!==false||outcome.releasePass!==false)
    return denied('OUTCOME_GRANTS_PASS',{detail:'EVIDENCE_ONLY_VIOLATED'});
  return Object.freeze({valid:true,reason:null,
    value:Object.freeze({providerId:outcome.providerId,attemptId:outcome.attemptId,
      sourceSha:outcome.sourceSha,state:outcome.state,reason:outcome.reason,
      agentLaunch:outcome.agentLaunch,process})});
}

/** Structural conformance check for a TaskProvider implementation. */
export function assertProviderInterface(provider){
  if(!plain(provider))return {conformant:false,missing:['PROVIDER_OBJECT']};
  const missing=[];
  if(!IDS.providerIdPattern.test(provider.providerId??''))missing.push('providerId');
  for(const member of ['probe','execute','cancel','readback'])
    if(typeof provider[member]!=='function')missing.push(member);
  if(Object.isFrozen&&Object.isFrozen(provider)===false)missing.push('FROZEN_INTERFACE');
  return {conformant:missing.length===0,missing};
}

/** Admission-time provider selection. The default provider is codex and it
 * is routed through the EXISTING v2 chain unchanged. An optional provider is
 * only admitted when explicitly requested AND enabled in trusted config AND
 * backed by an available probe result. Anything else is rejected honestly —
 * there is no silent fallback to codex for an explicit non-codex request
 * (that would fabricate capability the operator never validated). */
export function resolveProviderSelection({requested,enabled={},probe}={}){
  const requestedId=requested??IDS.defaultProviderId;
  if(requestedId===IDS.defaultProviderId)
    return {selected:IDS.defaultProviderId,admitted:true,route:'EXISTING_CHAIN',
      reason:requested?'CODEX_EXPLICIT_DEFAULT':'CODEX_DEFAULT',probe:null};
  if(!IDS.providerIdPattern.test(requestedId))
    return {selected:null,admitted:false,reason:'PROVIDER_ID_INVALID',probe:null};
  if(enabled[requestedId]!==true)
    return {selected:null,admitted:false,reason:'PROVIDER_NOT_ENABLED',probe:probe??null};
  if(!plain(probe)||probe.available!==true)
    return {selected:null,admitted:false,reason:'PROVIDER_CAPABILITY_UNVERIFIED',
      probe:probe??null};
  return {selected:requestedId,admitted:true,route:'REGISTRY',
    reason:'PROVIDER_VALIDATED',probe};
}
