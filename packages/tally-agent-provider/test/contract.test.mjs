import test from 'node:test';
import assert from 'node:assert/strict';
import {
  AGENT_PROVIDER_CONTRACT_VERSION, defineAgentProvider,
  createAgentProviderRegistry, createProviderAttemptPin,
  normalizeProviderObservation,
} from '../src/index.mjs';

const good = (id='codex') => ({id, contractVersion: 1, capabilities: [],
  probe: async () => ({state:'AVAILABLE'}),
  execute: async () => ({state:'COMPLETED', evidence:{agentSummary:'done'}}),
  cancel: async () => ({state:'CANCEL_REQUESTED'}),
  reconcile: async () => ({state:'UNKNOWN',reason:'NOT_RECONCILED'}),
});
const pin = (providerId='codex') => createProviderAttemptPin({
  taskId:'264',attemptId:'attempt:264:r1',sourceSha:'a'.repeat(40),providerId});

test('version and all four operations form one mandatory contract', async () => {
  assert.equal(AGENT_PROVIDER_CONTRACT_VERSION,1);
  const provider=defineAgentProvider(good());
  assert.deepEqual(await provider.probe(),{state:'AVAILABLE',reason:null,evidence:null});
  assert.equal((await provider.execute()).state,'COMPLETED');
  assert.equal((await provider.cancel()).state,'CANCEL_REQUESTED');
  assert.equal((await provider.reconcile()).state,'UNKNOWN');
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
  await assert.rejects(()=>liar.execute(),/UNEXPECTED_FIELD/);
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
    registry.resolve({providerId:t.providerId,enabledProviderIds:['codex','zcode']}).execute(t)));
  assert.deepEqual(seen,[{id:'codex',taskId:'a'},{id:'zcode',taskId:'b'}]);
  assert.deepEqual(outcomes.map(x=>x.state),['COMPLETED','COMPLETED']);
  assert.equal('taskPass' in outcomes[0],false);
});
