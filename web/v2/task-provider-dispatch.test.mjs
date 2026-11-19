import test from 'node:test';
import assert from 'node:assert/strict';
import {createProviderRegistry} from './provider-registry.mjs';
import {createTaskProviderDispatch} from './task-provider-dispatch.mjs';

const sourceSha='a'.repeat(40);
let clock=0;
const now=()=>Date.UTC(2026,10,19,0,0,++clock);

function fakeProvider(providerId,{stall,execute}={}){
  let launched=0;
  const defaultExecute=async input=>{launched++;
    if(stall)await stall;
    return {providerId,provider:providerId,attemptId:input.attemptId,
      sourceSha:input.sourceSha,state:'LOCAL_CANDIDATE_READY',
      reason:'FAKE_READY',agentLaunch:true,
      process:{status:'EXITED',exitCode:0,pid:100+launched,
        startedAt:new Date(now()).toISOString(),
        terminatedAt:new Date(now()).toISOString()},
      candidateTree:'b'.repeat(40),candidateCommit:'c'.repeat(40),
      candidateRef:'refs/tally-'+providerId+'-candidates/'+input.attemptId,
      candidatePaths:input.allowedWriteSet,taskPass:false,releasePass:false};};
  return Object.freeze({providerId,
    probe:async()=>({available:true,provider:providerId,version:'1.0.0'}),
    execute:execute??defaultExecute,
    cancel:async({controller}={})=>{
      if(!controller||typeof controller.abort!=='function')return false;
      if(controller.signal.aborted)return false;
      controller.abort();return true;},
    readback:async()=>({verified:true})});
}
const unknownExecute=state=>reason=>async input=>({providerId:'zcode',
  attemptId:input.attemptId,sourceSha:input.sourceSha,state,reason,agentLaunch:true,
  process:{status:state==='LOCAL_ATTEMPT_UNKNOWN'?'UNKNOWN':'EXITED',exitCode:null,
    pid:null,startedAt:null,terminatedAt:null},taskPass:false,releasePass:false});
function registryWith(...providers){
  const ids=providers.map(provider=>provider.providerId);
  return createProviderRegistry({providers,
    enabled:Object.fromEntries(ids.map(id=>[id,true])),
    probes:Object.fromEntries(ids.map(id=>[id,{available:true,provider:id}]))});
}
function admissionPorts(overrides={}){
  return {dagDependenciesSatisfied:async()=>({ok:true}),
    claimAdmitted:async()=>({ok:true,authorization:{dispatchId:'d1',
      protectedClaimKey:'pk1',admissionGeneration:2}}),
    writeSetConflictFree:async()=>({ok:true}),
    worktreeAndRefIndependent:async()=>({ok:true}),
    capacityAvailable:async()=>({ok:true}),...overrides};
}
function request({taskId='task:1',attemptId='attempt:1',providerId='zcode',...attempt}={}){
  return {taskId,attemptId,requestedProvider:providerId,
    trustedAttemptContext:{providerId,claimAccepted:true},
    attempt:{sourceSha,worktreePath:'C:\\wt\\'+attemptId,
      allowedWriteSet:['docs/x/'+attemptId+'.md'],taskPrompt:'Do it.',
      providerConfig:{},...attempt}};
}

test('registry is required',()=>{
  assert.throws(()=>createTaskProviderDispatch({}),/REGISTRY_REQUIRED/);
});

test('codex route is admitted but executed by the EXISTING chain, not here',async()=>{
  const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request({providerId:'codex'}));
  assert.equal(result.admitted,true);
  assert.equal(result.route,'EXISTING_CHAIN');
  assert.equal(result.executed,false);
  assert.equal(result.outcome,null);
});

test('browser requestedProvider disagreeing with trusted stamp is denied',async()=>{
  const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch({...request({providerId:'zcode'}),
    requestedProvider:'grok'});
  assert.equal(result.admitted,false);
  assert.equal(result.reason,'PROVIDER_MISMATCH_REQUEST_VS_TRUSTED');
});

test('unadmitted claim denies dispatch before any provider work',async()=>{
  const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
    admission:admissionPorts({claimAdmitted:async()=>({ok:false,reason:'CLAIM_STALE'})}),now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.admitted,false);
  assert.equal(result.reason,'CLAIM_STALE');
});

test('missing admission ports fail closed',async()=>{
  const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
    admission:{},now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.admitted,false);
  assert.equal(result.reason,'CLAIM_ADMISSION_PORT_NOT_CONNECTED');
});

test('unavailable provider is rejected, never silently switched',async()=>{
  const disabled=createProviderRegistry({providers:[fakeProvider('zcode')],
    probes:{zcode:{available:true,provider:'zcode'}}});
  const dispatch=createTaskProviderDispatch({registry:disabled,
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.admitted,false);
  assert.equal(result.reason,'PROVIDER_NOT_ENABLED');
  assert.deepEqual(dispatch.describeActive(),[]);
  const unverified=createProviderRegistry({providers:[fakeProvider('zcode')],
    enabled:{zcode:true},probes:{zcode:{available:false,provider:'zcode',
      reason:'LOCAL_AGENT_UNAVAILABLE'}}});
  const dispatch2=createTaskProviderDispatch({registry:unverified,
    admission:admissionPorts(),now});
  const result2=await dispatch2.dispatch(request());
  assert.equal(result2.admitted,false);
  assert.equal(result2.reason,'PROVIDER_CAPABILITY_UNVERIFIED');
});

test('same task cannot run two attempts at once',async()=>{
  let releaseExecute;
  const stall=new Promise(resolve=>{releaseExecute=resolve;});
  const slow=fakeProvider('zcode',{stall});
  const dispatch=createTaskProviderDispatch({registry:registryWith(slow),
    admission:admissionPorts(),now});
  const first=dispatch.dispatch(request({attemptId:'attempt:1'}));
  const second=await dispatch.dispatch(request({attemptId:'attempt:2'}));
  assert.equal(second.admitted,false);
  assert.equal(second.reason,'TASK_ATTEMPT_ALREADY_ACTIVE');
  releaseExecute();
  const done=await first;
  assert.equal(done.outcome.state,'LOCAL_CANDIDATE_READY');
  // terminal state released the slot; a new attempt is admissible again
  const third=await dispatch.dispatch(request({attemptId:'attempt:3'}));
  assert.equal(third.admitted,true);
});

test('unknown outcome keeps the task slot occupied (no blind restart)',async()=>{
  const provider=fakeProvider('zcode',
    {execute:unknownExecute('LOCAL_ATTEMPT_UNKNOWN')('ATTEMPT_STATUS_UNKNOWN')});
  const dispatch=createTaskProviderDispatch({registry:registryWith(provider),
    admission:admissionPorts(),now});
  const first=await dispatch.dispatch(request({attemptId:'attempt:1'}));
  assert.equal(first.outcome.state,'LOCAL_ATTEMPT_UNKNOWN');
  const retry=await dispatch.dispatch(request({attemptId:'attempt:2'}));
  assert.equal(retry.admitted,false);
  assert.equal(retry.reason,'TASK_ATTEMPT_ALREADY_ACTIVE');
});

test('overlapping write set, worktree, capacity and DAG gates each deny',async()=>{
  const cases=[
    ['writeSetConflictFree',async()=>({ok:false,reason:'WRITE_SET_OVERLAP'}),'WRITE_SET_OVERLAP'],
    ['worktreeAndRefIndependent',async()=>({ok:false,reason:'WORKTREE_OR_REF_CONFLICT'}),'WORKTREE_OR_REF_CONFLICT'],
    ['capacityAvailable',async()=>({ok:false,reason:'PROVIDER_CAPACITY_EXHAUSTED'}),'PROVIDER_CAPACITY_EXHAUSTED'],
    ['dagDependenciesSatisfied',async()=>({ok:false,reason:'TASK_DEPENDENCIES_NOT_SATISFIED'}),'TASK_DEPENDENCIES_NOT_SATISFIED']];
  for(const [port,impl,reason] of cases){
    const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
      admission:admissionPorts({[port]:impl}),now});
    const result=await dispatch.dispatch(request());
    assert.equal(result.admitted,false,reason);
    assert.equal(result.reason,reason);
  }
});

test('two non-conflicting tasks on two providers execute in parallel',async()=>{
  const dispatch=createTaskProviderDispatch({
    registry:registryWith(fakeProvider('zcode'),fakeProvider('dsh')),
    admission:admissionPorts(),now});
  const [a,b]=await Promise.all([
    dispatch.dispatch(request({taskId:'task:1',attemptId:'attempt:a',providerId:'zcode'})),
    dispatch.dispatch(request({taskId:'task:2',attemptId:'attempt:b',providerId:'dsh'}))]);
  assert.equal(a.admitted,true);assert.equal(a.selected,'zcode');
  assert.equal(b.admitted,true);assert.equal(b.selected,'dsh');
  assert.equal(a.outcome.state,'LOCAL_CANDIDATE_READY');
  assert.equal(b.outcome.state,'LOCAL_CANDIDATE_READY');
  assert.deepEqual(dispatch.describeActive(),[]);
});

test('provider failure stays on the same provider — no codex fallback',async()=>{
  const failing=fakeProvider('zcode',
    {execute:unknownExecute('LOCAL_ATTEMPT_FAILED')('ATTEMPT_PROCESS_FAILED')});
  const dispatch=createTaskProviderDispatch({registry:registryWith(failing),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.selected,'zcode');
  assert.equal(result.outcome.state,'LOCAL_ATTEMPT_FAILED');
  // slot released (terminal) but the record never claims codex ran instead
  assert.equal(dispatch.describeActive().length,0);
});

test('throwing provider degrades to UNKNOWN and keeps the slot',async()=>{
  const throwing=fakeProvider('zcode',{execute:async()=>{
    throw new Error('provider exploded');}});
  const dispatch=createTaskProviderDispatch({registry:registryWith(throwing),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.executed,true);
  assert.equal(result.outcome.state,'LOCAL_ATTEMPT_UNKNOWN');
  assert.equal(result.outcome.reason,'PROVIDER_EXECUTE_THREW');
  assert.equal(dispatch.describeActive()[0].state,'LOCAL_ATTEMPT_UNKNOWN');
});

test('outcome violating the contract is coerced to UNKNOWN, never trusted',async()=>{
  const liar=fakeProvider('zcode',{execute:async input=>({providerId:'zcode',
    attemptId:input.attemptId,sourceSha:input.sourceSha,
    state:'LOCAL_CANDIDATE_READY',reason:'X',agentLaunch:true,
    process:{status:'EXITED',exitCode:0,pid:1,startedAt:null,terminatedAt:null},
    taskPass:true,releasePass:false})});
  const dispatch=createTaskProviderDispatch({registry:registryWith(liar),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request());
  assert.equal(result.outcome.state,'LOCAL_ATTEMPT_UNKNOWN');
  assert.equal(result.outcome.reason,'PROVIDER_OUTCOME_INVALID');
  assert.equal(result.outcome.taskPass,false);
});

test('invalid attempt input denies before provider execute',async()=>{
  const dispatch=createTaskProviderDispatch({registry:registryWith(fakeProvider('zcode')),
    admission:admissionPorts(),now});
  const result=await dispatch.dispatch(request({sourceSha:'not-a-sha'}));
  assert.equal(result.admitted,false);
  assert.equal(result.reason,'ATTEMPT_INPUT_INVALID');
});

test('cancel routes an AbortController through the provider; idle cancel is honest',async()=>{
  const controller=new AbortController();
  let stallRelease;
  const stall=new Promise(resolve=>{stallRelease=resolve;});
  const provider=fakeProvider('zcode',{execute:async input=>{await stall;
    return {providerId:'zcode',attemptId:input.attemptId,sourceSha:input.sourceSha,
      state:'LOCAL_ATTEMPT_CANCELLED',reason:'ATTEMPT_CANCELLED',agentLaunch:true,
      process:{status:'EXITED',exitCode:null,pid:9,startedAt:null,terminatedAt:null},
      taskPass:false,releasePass:false};}});
  const dispatch=createTaskProviderDispatch({registry:registryWith(provider),
    admission:admissionPorts(),now});
  const inFlight=dispatch.dispatch(request({signal:controller.signal}));
  // the slot reservation happens after the claim microtask resolves — wait
  // for it deterministically instead of racing cancel against admission
  while(dispatch.describeActive().length===0)await Promise.resolve();
  const cancelled=await dispatch.cancel({taskId:'task:1',controller});
  assert.equal(cancelled.cancelled,true);
  assert.ok(controller.signal.aborted);
  stallRelease();
  const done=await inFlight;
  assert.equal(done.outcome.state,'LOCAL_ATTEMPT_CANCELLED');
  const idle=await dispatch.cancel({taskId:'task:none',controller});
  assert.equal(idle.cancelled,false);
  assert.equal(idle.reason,'ATTEMPT_NOT_ACTIVE');
});
