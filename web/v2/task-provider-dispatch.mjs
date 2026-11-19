/** R105 Task-level provider dispatch (v2.1 candidate; #64).
 * Thin admission + fan-out adapter over the EXISTING v2 chain — not a new
 * scheduler. One Task attempt uses exactly one provider. The provider
 * choice is stamped into the TRUSTED attempt context by Host admission;
 * browser/request fields are advisory only and never grant execution.
 *
 * Parallel execution of different tasks requires ALL of: native DAG
 * prerequisites satisfied, per-attempt Claim/Host admission, no write-set
 * conflict with live resources, independent worktree + candidate ref, and
 * provider/Host capacity. Conflict, cancel, timeout and unknown outcomes
 * reuse the provider Host mechanisms (worktree occupancy, process-tree
 * kill, git replay). Dispatch never blindly restarts and never switches
 * provider on failure. The default codex route is the EXISTING chain,
 * unchanged: dispatch admits it and does not execute it here. */
import {validateProviderAttemptInput,validateProviderAttemptOutcome,
  PROVIDER_CONTRACT_IDS} from './provider-contract.mjs';

const TERMINAL_STATES=new Set(['LOCAL_CANDIDATE_READY','LOCAL_ATTEMPT_FAILED',
  'LOCAL_ATTEMPT_CANCELLED']);
const notConnected=port=>({ok:false,reason:port+'_NOT_CONNECTED'});
const ok=()=>({ok:true});

export function createTaskProviderDispatch({registry,admission={},now=()=>Date.now()}={}){
  if(!registry||typeof registry.select!=='function'||typeof registry.get!=='function')
    throw new TypeError('REGISTRY_REQUIRED');
  const ports={
    dagDependenciesSatisfied:admission.dagDependenciesSatisfied??(async()=>notConnected('TASK_DAG_PORT')),
    claimAdmitted:admission.claimAdmitted??(async()=>notConnected('CLAIM_ADMISSION_PORT')),
    writeSetConflictFree:admission.writeSetConflictFree??(async()=>notConnected('WRITE_SET_PORT')),
    worktreeAndRefIndependent:admission.worktreeAndRefIndependent??(async()=>notConnected('WORKTREE_PORT')),
    capacityAvailable:admission.capacityAvailable??(async()=>notConnected('CAPACITY_PORT'))};
  const active=new Map(); // taskId -> {taskId,attemptId,providerId,dispatchedAt,state}
  const denied=(request,reason,extra={})=>Object.freeze({admitted:false,executed:false,
    reason,selected:null,taskId:request.taskId??null,attemptId:request.attemptId??null,
    ...extra});

  async function dispatch(request){
    if(request===null||typeof request!=='object'||Array.isArray(request))
      return denied(request??{},'DISPATCH_REQUEST_INVALID');
    const {taskId,attemptId}=request;
    if(typeof taskId!=='string'||!PROVIDER_CONTRACT_IDS.taskIdPattern.test(taskId))
      return denied(request,'DISPATCH_REQUEST_INVALID');
    if(typeof attemptId!=='string'||!PROVIDER_CONTRACT_IDS.attemptIdPattern.test(attemptId))
      return denied(request,'DISPATCH_REQUEST_INVALID');
    const trusted=request.trustedAttemptContext??null;
    if(trusted===null||typeof trusted!=='object'||Array.isArray(trusted))
      return denied(request,'TRUSTED_CONTEXT_REQUIRED');
    // Provider identity comes from the trusted context only. An advisory
    // browser field that disagrees with the trusted stamp is denied — the
    // browser never grants execution permission.
    const providerId=trusted.providerId??PROVIDER_CONTRACT_IDS.defaultProviderId;
    if(request.requestedProvider!==undefined&&request.requestedProvider!==providerId)
      return denied(request,'PROVIDER_MISMATCH_REQUEST_VS_TRUSTED',{requestedProvider:request.requestedProvider});
    const claim=await ports.claimAdmitted({taskId,attemptId,
      hostAuthorization:request.attempt?.hostAuthorization??null});
    if(claim?.ok!==true)return denied(request,claim?.reason??'CLAIM_NOT_ADMITTED');
    const hostAuthorization={...(request.attempt?.hostAuthorization??{}),
      ...(claim.authorization??{})};
    const selection=registry.select({requested:providerId});
    if(!selection.admitted)
      return denied(request,selection.reason??'PROVIDER_NOT_ADMITTED',{selected:null,providerId});
    if(selection.route==='EXISTING_CHAIN')
      return Object.freeze({admitted:true,executed:false,selected:PROVIDER_CONTRACT_IDS.defaultProviderId,
        route:'EXISTING_CHAIN',reason:selection.reason,taskId,attemptId,
        outcome:null,dispatchedAt:new Date(now()).toISOString()});
    // Atomic per-task slot reservation BEFORE any async admission port:
    // two concurrent dispatches for one task must never both pass the
    // active check across await boundaries (same race the local-codex host
    // closes with CAS grants). Every denial below releases the slot.
    if(active.has(taskId))return denied(request,'TASK_ATTEMPT_ALREADY_ACTIVE',
      {activeAttempt:active.get(taskId)});
    const dispatchedAt=new Date(now()).toISOString();
    const record={taskId,attemptId,providerId:selection.selected,dispatchedAt,
      state:'ADMITTING'};
    active.set(taskId,record);
    const release=()=>active.delete(taskId);
    const provider=registry.get(selection.selected);
    if(!provider){release();return denied(request,'PROVIDER_NOT_REGISTERED',{providerId:selection.selected});}
    const dag=await ports.dagDependenciesSatisfied({taskId});
    if(dag?.ok!==true){release();return denied(request,dag?.reason??'TASK_DEPENDENCIES_NOT_SATISFIED');}
    const writeSet=request.attempt?.allowedWriteSet??[];
    const conflicts=await ports.writeSetConflictFree({taskId,allowedWriteSet:writeSet,
      activeAttempts:[...active.values()]});
    if(conflicts?.ok!==true){release();return denied(request,conflicts?.reason??'WRITE_SET_OVERLAP');}
    const independence=await ports.worktreeAndRefIndependent({
      worktreePath:request.attempt?.worktreePath??null,attemptId,
      activeAttempts:[...active.values()]});
    if(independence?.ok!==true){release();return denied(request,independence?.reason??'WORKTREE_OR_REF_CONFLICT');}
    const capacity=await ports.capacityAvailable({providerId:selection.selected});
    if(capacity?.ok!==true){release();return denied(request,capacity?.reason??'PROVIDER_CAPACITY_EXHAUSTED');}
    const input=validateProviderAttemptInput({providerId:selection.selected,taskId,attemptId,
      sourceSha:request.attempt?.sourceSha,worktreePath:request.attempt?.worktreePath,
      allowedWriteSet:writeSet,taskPrompt:request.attempt?.taskPrompt,
      issueRef:request.attempt?.issueRef,hostAuthorization,
      providerConfig:request.attempt?.providerConfig??{},
      signal:request.attempt?.signal,timeoutMs:request.attempt?.timeoutMs});
    if(!input.valid){release();return denied(request,'ATTEMPT_INPUT_INVALID',{detail:input.detail});}
    record.state='RUNNING';
    let outcome;
    try{outcome=await provider.execute(input.value);}
    catch(error){
      // A provider that throws is an UNKNOWN execution, never a failure
      // fabricating evidence. The slot stays occupied — no blind restart.
      record.state='LOCAL_ATTEMPT_UNKNOWN';
      return Object.freeze({admitted:true,executed:true,selected:selection.selected,
        route:'REGISTRY',taskId,attemptId,dispatchedAt,
        outcome:{providerId:selection.selected,attemptId,sourceSha:input.value.sourceSha,
          state:'LOCAL_ATTEMPT_UNKNOWN',reason:'PROVIDER_EXECUTE_THREW',
          agentLaunch:true,process:{status:'UNKNOWN',exitCode:null,pid:null,
            startedAt:dispatchedAt,terminatedAt:new Date(now()).toISOString()},
          taskPass:false,releasePass:false,error:String(error?.message??error).slice(0,200)}});
    }
    const checked=validateProviderAttemptOutcome(outcome);
    const finalOutcome=checked.valid?outcome:Object.freeze(
      {...(outcome??{}),providerId:selection.selected,attemptId,
        sourceSha:input.value.sourceSha,state:'LOCAL_ATTEMPT_UNKNOWN',
        reason:'PROVIDER_OUTCOME_INVALID',agentLaunch:true,taskPass:false,releasePass:false});
    record.state=finalOutcome.state??'LOCAL_ATTEMPT_UNKNOWN';
    // Terminal evidence releases the per-task slot; UNKNOWN keeps it
    // occupied for operator reconciliation (no blind restart).
    if(TERMINAL_STATES.has(finalOutcome.state))active.delete(taskId);
    return Object.freeze({admitted:true,executed:true,selected:selection.selected,
      route:'REGISTRY',reason:selection.reason,taskId,attemptId,dispatchedAt,
      outcome:finalOutcome});
  }

  /** Cancel routes an AbortController (owned by the caller) through the
   * provider's cancel channel; the Host's abort listener kills the process
   * tree. AbortSignal cannot be aborted from outside, so the controller —
   * never a bare signal — is the cancellation handle. There is no restart
   * and no provider switch after cancel. */
  async function cancel({taskId,controller}={}){
    const record=active.get(taskId??'');
    if(!record)return {cancelled:false,reason:'ATTEMPT_NOT_ACTIVE'};
    const provider=registry.get(record.providerId);
    if(!provider)return {cancelled:false,reason:'PROVIDER_NOT_REGISTERED'};
    if(!controller||typeof controller.abort!=='function'||!controller.signal)
      return {cancelled:false,reason:'CANCEL_CHANNEL_ABSENT',taskId:record.taskId,
        attemptId:record.attemptId};
    if(controller.signal.aborted)
      return {cancelled:false,reason:'ALREADY_ABORTED',taskId:record.taskId,
        attemptId:record.attemptId};
    const cancelled=await provider.cancel({controller,
      attemptId:record.attemptId});
    return {cancelled:cancelled===true,taskId:record.taskId,
      attemptId:record.attemptId,providerId:record.providerId};
  }
  function describeActive(){return [...active.values()].map(record=>({...record}));}
  return Object.freeze({dispatch,cancel,describeActive});
}

export const TASK_PROVIDER_DISPATCH_IDS=Object.freeze({
  terminalStates:[...TERMINAL_STATES]});
