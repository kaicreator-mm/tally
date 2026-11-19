import test from 'node:test';
import assert from 'node:assert/strict';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider,
  createAgentProviderRegistry, createProviderAttemptPin,
  normalizeProviderObservation, validateProviderExecutionRequest,
} from '../src/index.mjs';

const good = (id='codex') => ({id, contractVersion: 1, capabilities: [],
  probe: async () => ({state:'AVAILABLE'}),
  execute: async () => ({state:'COMPLETED', evidence:{agentSummary:'done'}}),
  cancel: async () => ({state:'CANCEL_REQUESTED'}),
  reconcile: async () => ({state:'UNKNOWN',reason:'NOT_RECONCILED'}),
});
const pin = (providerId='codex') => createProviderAttemptPin({
  taskId:'264',attemptId:'attempt:264:r1',sourceSha:'a'.repeat(40),providerId});
const request = (over={}) => ({taskId:'264',attemptId:'attempt:264:r1',
  sourceSha:'a'.repeat(40),prompt:'do the work',worktreePath:'C:/tmp/wt',...over});

test('version and all four operations form one mandatory contract', async () => {
  assert.equal(AGENT_PROVIDER_CONTRACT_VERSION,1);
  const provider=defineAgentProvider(good());
  assert.deepEqual(await provider.probe(),{state:'AVAILABLE',reason:null,evidence:null});
  assert.equal((await provider.execute(request())).state,'COMPLETED');
  assert.equal((await provider.cancel(pin())).state,'CANCEL_REQUESTED');
  assert.equal((await provider.reconcile(pin())).state,'UNKNOWN');
  assert.throws(()=>defineAgentProvider({...good(),cancel:null}),/PROVIDER_OPERATION_MISSING_CANCEL/);
  assert.throws(()=>defineAgentProvider({...good(),contractVersion:2}),/UNSUPPORTED/);
});

test('default Codex only, optional providers need explicit enabled selection', () => {
  const registry=createAgentProviderRegistry([good(),good('zcode'),good('dsh')]);
  assert.equal(registry.resolve().id,'codex');
  assert.throws(()=>registry.resolve({providerId:'zcode'}),/PROVIDER_DISABLED/);
  assert.equal(registry.resolve({providerId:'zcode',enabledProviderIds:['codex','zcode']}).id,'zcode');
  assert.throws(()=>registry.resolve({providerId:'grok',enabledProviderIds:['grok']}),/NOT_INSTALLED/);
  assert.throws(()=>createAgentProviderRegistry([good(),good()]),/DUPLICATE/);
});

test('one Attempt pin cannot be silently switched or rebound to another provider', () => {
  const registry=createAgentProviderRegistry([good(),good('zcode')]);
  assert.equal(registry.resolve({pin:pin('zcode'),enabledProviderIds:['zcode']}).id,'zcode');
  assert.throws(()=>registry.resolve({providerId:'codex',pin:pin('zcode')}),/PIN_MISMATCH/);
  assert.throws(()=>pin('ZCODE'),/PROVIDER_ID_INVALID/);
  assert.throws(()=>createProviderAttemptPin({taskId:'264',attemptId:'bad',sourceSha:'bad',providerId:'codex'}),/SOURCE_SHA/);
});

test('provider observations cannot assert authority or a Task PASS', async () => {
  for (const field of ['taskPass','releasePass','claimAccepted','authorized','candidateCommit']) {
    assert.throws(()=>normalizeProviderObservation('execute',{state:'COMPLETED',[field]:true}),
      /UNEXPECTED_FIELD/);
  }
  const liar=defineAgentProvider({...good(),execute:async()=>({state:'COMPLETED',taskPass:true})});
  await assert.rejects(()=>liar.execute(request()),/UNEXPECTED_FIELD/);
  assert.throws(()=>normalizeProviderObservation('execute',{state:'UNKNOWN'}),/UNKNOWN_REASON_REQUIRED/);
  assert.throws(()=>normalizeProviderObservation('execute',{state:'SUCCESS'}),/STATE_INVALID/);
  assert.throws(()=>normalizeProviderObservation('probe',{state:'FAILED'}),/STATE_INVALID/);
});

test('each installed adapter remains independent and does not create a scheduler', async () => {
  const seen=[];
  const provider=id=>({...good(id),execute:async ({taskId})=>{
    seen.push({id,taskId});return {state:'COMPLETED',evidence:{taskId}};
  }});
  const registry=createAgentProviderRegistry([provider('codex'),provider('zcode')]);
  const tasks=[{taskId:'a',providerId:'codex'},{taskId:'b',providerId:'zcode'}];
  const outcomes=await Promise.all(tasks.map(async t=>
    registry.resolve({providerId:t.providerId,enabledProviderIds:['codex','zcode']})
      .execute(request({taskId:t.taskId}))));
  assert.deepEqual(seen,[{id:'codex',taskId:'a'},{id:'zcode',taskId:'b'}]);
  assert.deepEqual(outcomes.map(x=>x.state),['COMPLETED','COMPLETED']);
  assert.equal('taskPass' in outcomes[0],false);
});

test('binding inputs are validated at the provider boundary (review P1-1)', async () => {
  // A valid request passes validation and comes back frozen.
  const validated=validateProviderExecutionRequest(request({timeoutMs:5000}));
  assert.equal(Object.isFrozen(validated),true);
  assert.equal(validated.taskId,'264');
  // Malformed bindings are refused with typed reasons, never silently executed.
  let called=false;
  const spy=defineAgentProvider({...good(),
    execute:async()=>{called=true;return {state:'COMPLETED'};}});
  for (const [name,over] of [
    ['empty taskId',{taskId:''}],
    ['empty attemptId',{attemptId:''}],
    ['garbage sourceSha',{sourceSha:'garbage'}],
    ['non-string prompt',{prompt:42}],
    ['empty worktreePath',{worktreePath:''}],
    ['non-positive timeoutMs',{timeoutMs:0}],
  ]) {
    await assert.rejects(()=>spy.execute(request(over)),
      {name:'TypeError'},'must reject '+name);
    assert.equal(called,false,'adapter must not run for '+name);
  }
  await assert.rejects(()=>spy.execute(undefined),/PROVIDER_REQUEST_INVALID/);
  await assert.rejects(()=>spy.execute(null),/PROVIDER_REQUEST_INVALID/);
  // cancel/reconcile pin shapes are validated with the same discipline.
  let cancelCalled=false;
  const cancellable=defineAgentProvider({...good(),
    cancel:async()=>{cancelCalled=true;return {state:'NOT_RUNNING'};}});
  await assert.rejects(()=>cancellable.cancel({providerId:'codex',
    taskId:'264',attemptId:'attempt:264:r1',sourceSha:'bad'}),/SOURCE_SHA/);
  await assert.rejects(()=>cancellable.cancel(undefined),/PROVIDER_ATTEMPT_PIN_INVALID/);
  assert.equal(cancelCalled,false);
  assert.equal((await cancellable.cancel(pin())).state,'NOT_RUNNING');
});
